package commands

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	openslov1alpha "github.com/OpenSLO/oslo/pkg/manifest/v1alpha"
	prometheusmodel "github.com/prometheus/common/model"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/alexandreLamarre/core/alert"
	"github.com/alexandreLamarre/core/app/generate"
	"github.com/alexandreLamarre/core/info"
	"github.com/alexandreLamarre/core/k8sprometheus"
	"github.com/alexandreLamarre/core/log"
	"github.com/alexandreLamarre/core/openslo"
	"github.com/alexandreLamarre/core/prometheus"
	kubernetesv1 "github.com/alexandreLamarre/pkg/kubernetes/api/sloth/v1"
	prometheusv1 "github.com/alexandreLamarre/pkg/prometheus/api/v1"
)

type generateCommand struct {
	slosInput             string
	slosOut               string
	slosExcludeRegex      string
	slosIncludeRegex      string
	disableRecordings     bool
	disableAlerts         bool
	disableOptimizedRules bool
	extraLabels           map[string]string
	sliPluginsPaths       []string
	sloPeriodWindowsPath  string
	sloPeriod             string
}

// NewGenerateCommand returns the generate command.
func NewGenerateCommand(app *kingpin.Application) Command {
	c := &generateCommand{extraLabels: map[string]string{}}
	cmd := app.Command("generate", "Generates Prometheus SLOs.")
	cmd.Flag("input", "SLO spec input file path or directory (if directory is used, slos will be discovered recursively and out must be a directory).").Short('i').StringVar(&c.slosInput)
	cmd.Flag("out", "Generated rules output file path or directory. If `-` it will use stdout (if input is a directory this must be a directory).").Default("-").Short('o').StringVar(&c.slosOut)
	cmd.Flag("fs-exclude", "Filter regex to ignore matched discovered SLO file paths (used with directory based input/output).").Short('e').StringVar(&c.slosExcludeRegex)
	cmd.Flag("fs-include", "Filter regex to include matched discovered SLO file paths, everything else will be ignored. Exclude has preference (used with directory based input/output).").Short('n').StringVar(&c.slosIncludeRegex)

	cmd.Flag("extra-labels", "Extra labels that will be added to all the generated Prometheus rules ('key=value' form, can be repeated).").Short('l').StringMapVar(&c.extraLabels)
	cmd.Flag("disable-recordings", "Disables recording rules generation.").BoolVar(&c.disableRecordings)
	cmd.Flag("disable-alerts", "Disables alert rules generation.").BoolVar(&c.disableAlerts)
	cmd.Flag("sli-plugins-path", "The path to SLI plugins (can be repeated), if not set it disable plugins support.").Short('p').StringsVar(&c.sliPluginsPaths)
	cmd.Flag("slo-period-windows-path", "The directory path to custom SLO period windows catalog (replaces default ones).").StringVar(&c.sloPeriodWindowsPath)
	cmd.Flag("default-slo-period", "The default SLO period windows to be used for the SLOs.").Default("30d").StringVar(&c.sloPeriod)
	cmd.Flag("disable-optimized-rules", "If enabled it will disable optimized generated rules.").BoolVar(&c.disableOptimizedRules)

	return c
}

func (g generateCommand) Name() string { return "generate" }
func (g generateCommand) Run(ctx context.Context, config RootConfig) error {
	logger := config.Logger.WithValues(log.Kv{"window": g.sloPeriod})

	// Check input and output.
	inputInfo, err := os.Stat(g.slosInput)
	if err != nil {
		return err
	}
	if inputInfo.IsDir() {
		// If input is a dir, output must be a directory.
		outInfo, err := os.Stat(g.slosOut)
		if err != nil {
			return err
		}
		if !outInfo.IsDir() {
			return fmt.Errorf("the path %q is not a directory, however input is a directory", g.slosOut)
		}

		// Check input and output are not the same.
		ia, err := filepath.Abs(g.slosInput)
		if err != nil {
			return err
		}
		oa, err := filepath.Abs(g.slosOut)
		if err != nil {
			return err
		}
		if ia == oa {
			return fmt.Errorf("input and output can't be the same directory: %s", ia)
		}
	}

	// SLO period.
	sp, err := prometheusmodel.ParseDuration(g.sloPeriod)
	if err != nil {
		return fmt.Errorf("invalid SLO period duration: %w", err)
	}
	sloPeriod := time.Duration(sp)

	ctx = logger.SetValuesOnCtx(ctx, log.Kv{
		"out": g.slosOut,
	})

	// Load plugins
	pluginRepo, err := createPluginLoader(ctx, logger, g.sliPluginsPaths)
	if err != nil {
		return err
	}

	// Windows repository.
	var wfs fs.FS
	if g.sloPeriodWindowsPath != "" {
		wfs = os.DirFS(g.sloPeriodWindowsPath)
	}
	windowsRepo, err := alert.NewFSWindowsRepo(alert.FSWindowsRepoConfig{
		FS:     wfs,
		Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("could not load SLO period windows repository: %w", err)
	}

	// Check if the default slo period is supported by our windows repo.
	_, err = windowsRepo.GetWindows(ctx, sloPeriod)
	if err != nil {
		return fmt.Errorf("invalid default slo period: %w", err)
	}

	// Create Spec loaders.
	promYAMLLoader := prometheus.NewYAMLSpecLoader(pluginRepo, sloPeriod)
	kubeYAMLLoader := k8sprometheus.NewYAMLSpecLoader(pluginRepo, sloPeriod)
	openSLOYAMLLoader := openslo.NewYAMLSpecLoader(sloPeriod)

	// Get SLO targets.
	genTargets := []generateTarget{}

	// FIle based input/outputs.
	if !inputInfo.IsDir() {
		// Get SLO spec data.
		f, err := os.Open(g.slosInput)
		if err != nil {
			return fmt.Errorf("could not open SLOs spec file: %w", err)
		}
		defer f.Close()

		slxData, err := io.ReadAll(f)
		if err != nil {
			return fmt.Errorf("could not read SLOs spec file data: %w", err)
		}

		// Split YAMLs in case we have multiple yaml files in a single file.
		splittedSLOsData := splitYAML(slxData)

		// Prepare store output.
		var out io.Writer = config.Stdout
		if g.slosOut != "-" {
			outFile, err := os.Create(g.slosOut)
			if err != nil {
				return fmt.Errorf("could not create out file: %w", err)
			}
			defer f.Close()
			out = outFile
		}
		for _, s := range splittedSLOsData {
			genTargets = append(genTargets, generateTarget{
				SLOData: s,
				Out:     out,
			})
		}
	} else {
		// Directory based input/outpus.
		var excludeRegex *regexp.Regexp
		var includeRegex *regexp.Regexp
		if g.slosExcludeRegex != "" {
			r, err := regexp.Compile(g.slosExcludeRegex)
			if err != nil {
				return fmt.Errorf("invalid exclude regex: %w", err)
			}
			excludeRegex = r
		}
		if g.slosIncludeRegex != "" {
			r, err := regexp.Compile(g.slosIncludeRegex)
			if err != nil {
				return fmt.Errorf("invalid include regex: %w", err)
			}
			includeRegex = r
		}

		sloPaths, err := discoverSLOManifests(logger, excludeRegex, includeRegex, g.slosInput)
		if err != nil {
			return fmt.Errorf("could not discover files: %w", err)
		}
		if len(sloPaths) == 0 {
			return fmt.Errorf("0 slo specs have been discovered")
		}

		for _, sloPath := range sloPaths {
			f, err := os.Open(sloPath)
			if err != nil {
				return fmt.Errorf("could not open SLOs spec file: %w", err)
			}
			defer f.Close()

			slxData, err := io.ReadAll(f)
			if err != nil {
				return fmt.Errorf("could not read SLOs spec file data: %w", err)
			}

			// Infer output path.
			outputPath := strings.TrimPrefix(path.Clean(sloPath), strings.TrimPrefix(g.slosInput, "./"))
			outputPath = path.Join(g.slosOut, outputPath)

			// Ensure the file path is ready.
			err = os.MkdirAll(path.Dir(outputPath), os.ModePerm)
			if err != nil {
				return err
			}

			// Create the target file.
			outFile, err := os.Create(outputPath)
			if err != nil {
				return fmt.Errorf("could not create out file: %w", err)
			}
			defer outFile.Close()

			// Split YAMLs in case we have multiple yaml files in a single file.
			splittedSLOsData := splitYAML(slxData)
			for _, s := range splittedSLOsData {
				genTargets = append(genTargets, generateTarget{
					SLOData: s,
					Out:     outFile,
				})
			}
		}
	}

	for _, genTarget := range genTargets {
		dataB := []byte(genTarget.SLOData)

		// Match the spec type to know how to generate.
		switch {
		case promYAMLLoader.IsSpecType(ctx, dataB):
			slos, err := promYAMLLoader.LoadSpec(ctx, dataB)
			if err != nil {
				return fmt.Errorf("tried loading raw prometheus SLOs spec, it couldn't: %w", err)
			}

			err = generatePrometheus(ctx, logger, windowsRepo, g.disableRecordings, g.disableAlerts, g.disableOptimizedRules, g.extraLabels, *slos, genTarget.Out)
			if err != nil {
				return fmt.Errorf("could not generate Prometheus format rules: %w", err)
			}

		case kubeYAMLLoader.IsSpecType(ctx, dataB):
			sloGroup, err := kubeYAMLLoader.LoadSpec(ctx, dataB)
			if err != nil {
				return fmt.Errorf("tried loading Kubernetes prometheus SLOs spec, it couldn't: %w", err)
			}

			err = generateKubernetes(ctx, logger, windowsRepo, g.disableRecordings, g.disableAlerts, g.disableOptimizedRules, g.extraLabels, *sloGroup, genTarget.Out)
			if err != nil {
				return fmt.Errorf("could not generate Kubernetes format rules: %w", err)
			}

		case openSLOYAMLLoader.IsSpecType(ctx, dataB):
			slos, err := openSLOYAMLLoader.LoadSpec(ctx, dataB)
			if err != nil {
				return fmt.Errorf("tried loading OpenSLO SLOs spec, it couldn't: %w", err)
			}

			err = generateOpenSLO(ctx, logger, windowsRepo, g.disableRecordings, g.disableAlerts, g.disableOptimizedRules, g.extraLabels, *slos, genTarget.Out)
			if err != nil {
				return fmt.Errorf("could not generate OpenSLO format rules: %w", err)
			}

		default:
			return fmt.Errorf("invalid spec, could not load with any of the supported spec types")
		}
	}

	return nil
}

// generatePrometheus generates the SLOs based on a raw regular Prometheus spec format input and
// outs a Prometheus raw yaml.
func generatePrometheus(ctx context.Context, logger log.Logger, windowsRepo alert.WindowsRepo, disableRecs, disableAlerts, disableOptimizedRules bool, extraLabels map[string]string, slos prometheus.SLOGroup, out io.Writer) error {
	logger.Infof("Generating from Prometheus spec")
	info := info.Info{
		Version: info.Version,
		Mode:    info.ModeCLIGenPrometheus,
		Spec:    prometheusv1.Version,
	}

	result, err := generateRules(ctx, logger, info, windowsRepo, disableRecs, disableAlerts, disableOptimizedRules, extraLabels, slos)
	if err != nil {
		return err
	}

	repo := prometheus.NewIOWriterGroupedRulesYAMLRepo(out, logger)
	storageSLOs := make([]prometheus.StorageSLO, 0, len(result.PrometheusSLOs))
	for _, s := range result.PrometheusSLOs {
		storageSLOs = append(storageSLOs, prometheus.StorageSLO{
			SLO:   s.SLO,
			Rules: s.SLORules,
		})
	}

	err = repo.StoreSLOs(ctx, storageSLOs)
	if err != nil {
		return fmt.Errorf("could not store SLOS: %w", err)
	}

	return nil
}

// generateKubernetes generates the SLOs based on a Kuberentes spec format input and
// outs a Kubernetes prometheus operator CRD yaml.
func generateKubernetes(ctx context.Context, logger log.Logger, windowsRepo alert.WindowsRepo, disableRecs, disableAlerts, disableOptimizedRules bool, extraLabels map[string]string, sloGroup k8sprometheus.SLOGroup, out io.Writer) error {
	logger.Infof("Generating from Kubernetes Prometheus spec")

	info := info.Info{
		Version: info.Version,
		Mode:    info.ModeCLIGenKubernetes,
		Spec:    fmt.Sprintf("%s/%s", kubernetesv1.SchemeGroupVersion.Group, kubernetesv1.SchemeGroupVersion.Version),
	}
	result, err := generateRules(ctx, logger, info, windowsRepo, disableRecs, disableAlerts, disableOptimizedRules, extraLabels, sloGroup.SLOGroup)
	if err != nil {
		return err
	}

	repo := k8sprometheus.NewIOWriterPrometheusOperatorYAMLRepo(out, logger)
	storageSLOs := make([]k8sprometheus.StorageSLO, 0, len(result.PrometheusSLOs))
	for _, s := range result.PrometheusSLOs {
		storageSLOs = append(storageSLOs, k8sprometheus.StorageSLO{
			SLO:   s.SLO,
			Rules: s.SLORules,
		})
	}

	err = repo.StoreSLOs(ctx, sloGroup.K8sMeta, storageSLOs)
	if err != nil {
		return fmt.Errorf("could not store SLOS: %w", err)
	}

	return nil
}

// generateOpenSLO generates the SLOs based on a OpenSLO spec format input and
// outs a Prometheus raw yaml.
func generateOpenSLO(ctx context.Context, logger log.Logger, windowsRepo alert.WindowsRepo, disableRecs, disableAlerts, disableOptimizedRules bool, extraLabels map[string]string, slos prometheus.SLOGroup, out io.Writer) error {
	logger.Infof("Generating from OpenSLO spec")
	info := info.Info{
		Version: info.Version,
		Mode:    info.ModeCLIGenOpenSLO,
		Spec:    openslov1alpha.APIVersion,
	}

	result, err := generateRules(ctx, logger, info, windowsRepo, disableRecs, disableAlerts, disableOptimizedRules, extraLabels, slos)
	if err != nil {
		return err
	}

	repo := prometheus.NewIOWriterGroupedRulesYAMLRepo(out, logger)
	storageSLOs := make([]prometheus.StorageSLO, 0, len(result.PrometheusSLOs))
	for _, s := range result.PrometheusSLOs {
		storageSLOs = append(storageSLOs, prometheus.StorageSLO{
			SLO:   s.SLO,
			Rules: s.SLORules,
		})
	}

	err = repo.StoreSLOs(ctx, storageSLOs)
	if err != nil {
		return fmt.Errorf("could not store SLOS: %w", err)
	}

	return nil
}

// generate is the main generator logic that all the spec types and storers share. Mainly
// has the logic of the generate app service.
func generateRules(ctx context.Context, logger log.Logger, info info.Info, windowsRepo alert.WindowsRepo, disableRecs, disableAlerts, disableOptimizedRules bool, extraLabels map[string]string, slos prometheus.SLOGroup) (*generate.Response, error) {
	// Disable recording rules if required.
	var sliRuleGen generate.SLIRecordingRulesGenerator = generate.NoopSLIRecordingRulesGenerator
	var metaRuleGen generate.MetadataRecordingRulesGenerator = generate.NoopMetadataRecordingRulesGenerator
	if !disableRecs {
		// Disable optimized rules if required.
		sliRuleGen = prometheus.OptimizedSLIRecordingRulesGenerator
		if disableOptimizedRules {
			sliRuleGen = prometheus.SLIRecordingRulesGenerator
		}
		metaRuleGen = prometheus.MetadataRecordingRulesGenerator
	}

	// Disable alert rules if required.
	var alertRuleGen generate.SLOAlertRulesGenerator = generate.NoopSLOAlertRulesGenerator
	if !disableAlerts {
		alertRuleGen = prometheus.SLOAlertRulesGenerator
	}

	// Generate.
	controller, err := generate.NewService(generate.ServiceConfig{
		AlertGenerator:              alert.NewGenerator(windowsRepo),
		SLIRecordingRulesGenerator:  sliRuleGen,
		MetaRecordingRulesGenerator: metaRuleGen,
		SLOAlertRulesGenerator:      alertRuleGen,
		Logger:                      logger,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create application service: %w", err)
	}

	result, err := controller.Generate(ctx, generate.Request{
		ExtraLabels: extraLabels,
		Info:        info,
		SLOGroup:    slos,
	})
	if err != nil {
		return nil, fmt.Errorf("could not generate prometheus rules: %w", err)
	}

	return result, nil
}

type generateTarget struct {
	Out     io.Writer
	SLOData string
}
