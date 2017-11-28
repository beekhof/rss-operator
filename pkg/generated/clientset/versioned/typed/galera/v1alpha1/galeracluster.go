/*
Copyright 2017 The etcd-operator Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package v1alpha1

import (
	v1alpha1 "github.com/coreos/etcd-operator/pkg/apis/galera/v1alpha1"
	scheme "github.com/coreos/etcd-operator/pkg/generated/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// GaleraClustersGetter has a method to return a GaleraClusterInterface.
// A group's client should implement this interface.
type GaleraClustersGetter interface {
	GaleraClusters(namespace string) GaleraClusterInterface
}

// GaleraClusterInterface has methods to work with GaleraCluster resources.
type GaleraClusterInterface interface {
	Create(*v1alpha1.GaleraCluster) (*v1alpha1.GaleraCluster, error)
	Update(*v1alpha1.GaleraCluster) (*v1alpha1.GaleraCluster, error)
	UpdateStatus(*v1alpha1.GaleraCluster) (*v1alpha1.GaleraCluster, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.GaleraCluster, error)
	List(opts v1.ListOptions) (*v1alpha1.GaleraClusterList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.GaleraCluster, err error)
	GaleraClusterExpansion
}

// galeraClusters implements GaleraClusterInterface
type galeraClusters struct {
	client rest.Interface
	ns     string
}

// newGaleraClusters returns a GaleraClusters
func newGaleraClusters(c *GaleraV1alpha1Client, namespace string) *galeraClusters {
	return &galeraClusters{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the galeraCluster, and returns the corresponding galeraCluster object, and an error if there is any.
func (c *galeraClusters) Get(name string, options v1.GetOptions) (result *v1alpha1.GaleraCluster, err error) {
	result = &v1alpha1.GaleraCluster{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("galeraclusters").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of GaleraClusters that match those selectors.
func (c *galeraClusters) List(opts v1.ListOptions) (result *v1alpha1.GaleraClusterList, err error) {
	result = &v1alpha1.GaleraClusterList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("galeraclusters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested galeraClusters.
func (c *galeraClusters) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("galeraclusters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a galeraCluster and creates it.  Returns the server's representation of the galeraCluster, and an error, if there is any.
func (c *galeraClusters) Create(galeraCluster *v1alpha1.GaleraCluster) (result *v1alpha1.GaleraCluster, err error) {
	result = &v1alpha1.GaleraCluster{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("galeraclusters").
		Body(galeraCluster).
		Do().
		Into(result)
	return
}

// Update takes the representation of a galeraCluster and updates it. Returns the server's representation of the galeraCluster, and an error, if there is any.
func (c *galeraClusters) Update(galeraCluster *v1alpha1.GaleraCluster) (result *v1alpha1.GaleraCluster, err error) {
	result = &v1alpha1.GaleraCluster{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("galeraclusters").
		Name(galeraCluster.Name).
		Body(galeraCluster).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *galeraClusters) UpdateStatus(galeraCluster *v1alpha1.GaleraCluster) (result *v1alpha1.GaleraCluster, err error) {
	result = &v1alpha1.GaleraCluster{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("galeraclusters").
		Name(galeraCluster.Name).
		SubResource("status").
		Body(galeraCluster).
		Do().
		Into(result)
	return
}

// Delete takes name of the galeraCluster and deletes it. Returns an error if one occurs.
func (c *galeraClusters) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("galeraclusters").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *galeraClusters) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("galeraclusters").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched galeraCluster.
func (c *galeraClusters) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.GaleraCluster, err error) {
	result = &v1alpha1.GaleraCluster{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("galeraclusters").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
