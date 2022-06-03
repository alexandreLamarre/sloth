// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	"context"
	"time"

	v1 "github.com/alexandreLamarre/sloth/pkg/kubernetes/api/sloth/v1"
	scheme "github.com/alexandreLamarre/sloth/pkg/kubernetes/gen/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// PrometheusServiceLevelsGetter has a method to return a PrometheusServiceLevelInterface.
// A group's client should implement this interface.
type PrometheusServiceLevelsGetter interface {
	PrometheusServiceLevels(namespace string) PrometheusServiceLevelInterface
}

// PrometheusServiceLevelInterface has methods to work with PrometheusServiceLevel resources.
type PrometheusServiceLevelInterface interface {
	Create(ctx context.Context, prometheusServiceLevel *v1.PrometheusServiceLevel, opts metav1.CreateOptions) (*v1.PrometheusServiceLevel, error)
	Update(ctx context.Context, prometheusServiceLevel *v1.PrometheusServiceLevel, opts metav1.UpdateOptions) (*v1.PrometheusServiceLevel, error)
	UpdateStatus(ctx context.Context, prometheusServiceLevel *v1.PrometheusServiceLevel, opts metav1.UpdateOptions) (*v1.PrometheusServiceLevel, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.PrometheusServiceLevel, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.PrometheusServiceLevelList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.PrometheusServiceLevel, err error)
	PrometheusServiceLevelExpansion
}

// prometheusServiceLevels implements PrometheusServiceLevelInterface
type prometheusServiceLevels struct {
	client rest.Interface
	ns     string
}

// newPrometheusServiceLevels returns a PrometheusServiceLevels
func newPrometheusServiceLevels(c *SlothV1Client, namespace string) *prometheusServiceLevels {
	return &prometheusServiceLevels{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the prometheusServiceLevel, and returns the corresponding prometheusServiceLevel object, and an error if there is any.
func (c *prometheusServiceLevels) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.PrometheusServiceLevel, err error) {
	result = &v1.PrometheusServiceLevel{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("prometheusservicelevels").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of PrometheusServiceLevels that match those selectors.
func (c *prometheusServiceLevels) List(ctx context.Context, opts metav1.ListOptions) (result *v1.PrometheusServiceLevelList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.PrometheusServiceLevelList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("prometheusservicelevels").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested prometheusServiceLevels.
func (c *prometheusServiceLevels) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("prometheusservicelevels").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a prometheusServiceLevel and creates it.  Returns the server's representation of the prometheusServiceLevel, and an error, if there is any.
func (c *prometheusServiceLevels) Create(ctx context.Context, prometheusServiceLevel *v1.PrometheusServiceLevel, opts metav1.CreateOptions) (result *v1.PrometheusServiceLevel, err error) {
	result = &v1.PrometheusServiceLevel{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("prometheusservicelevels").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(prometheusServiceLevel).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a prometheusServiceLevel and updates it. Returns the server's representation of the prometheusServiceLevel, and an error, if there is any.
func (c *prometheusServiceLevels) Update(ctx context.Context, prometheusServiceLevel *v1.PrometheusServiceLevel, opts metav1.UpdateOptions) (result *v1.PrometheusServiceLevel, err error) {
	result = &v1.PrometheusServiceLevel{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("prometheusservicelevels").
		Name(prometheusServiceLevel.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(prometheusServiceLevel).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *prometheusServiceLevels) UpdateStatus(ctx context.Context, prometheusServiceLevel *v1.PrometheusServiceLevel, opts metav1.UpdateOptions) (result *v1.PrometheusServiceLevel, err error) {
	result = &v1.PrometheusServiceLevel{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("prometheusservicelevels").
		Name(prometheusServiceLevel.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(prometheusServiceLevel).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the prometheusServiceLevel and deletes it. Returns an error if one occurs.
func (c *prometheusServiceLevels) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("prometheusservicelevels").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *prometheusServiceLevels) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("prometheusservicelevels").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched prometheusServiceLevel.
func (c *prometheusServiceLevels) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.PrometheusServiceLevel, err error) {
	result = &v1.PrometheusServiceLevel{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("prometheusservicelevels").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
