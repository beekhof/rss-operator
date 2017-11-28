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

// GaleraRestoresGetter has a method to return a GaleraRestoreInterface.
// A group's client should implement this interface.
type GaleraRestoresGetter interface {
	GaleraRestores(namespace string) GaleraRestoreInterface
}

// GaleraRestoreInterface has methods to work with GaleraRestore resources.
type GaleraRestoreInterface interface {
	Create(*v1alpha1.GaleraRestore) (*v1alpha1.GaleraRestore, error)
	Update(*v1alpha1.GaleraRestore) (*v1alpha1.GaleraRestore, error)
	UpdateStatus(*v1alpha1.GaleraRestore) (*v1alpha1.GaleraRestore, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.GaleraRestore, error)
	List(opts v1.ListOptions) (*v1alpha1.GaleraRestoreList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.GaleraRestore, err error)
	GaleraRestoreExpansion
}

// galeraRestores implements GaleraRestoreInterface
type galeraRestores struct {
	client rest.Interface
	ns     string
}

// newGaleraRestores returns a GaleraRestores
func newGaleraRestores(c *GaleraV1alpha1Client, namespace string) *galeraRestores {
	return &galeraRestores{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the galeraRestore, and returns the corresponding galeraRestore object, and an error if there is any.
func (c *galeraRestores) Get(name string, options v1.GetOptions) (result *v1alpha1.GaleraRestore, err error) {
	result = &v1alpha1.GaleraRestore{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("galerarestores").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of GaleraRestores that match those selectors.
func (c *galeraRestores) List(opts v1.ListOptions) (result *v1alpha1.GaleraRestoreList, err error) {
	result = &v1alpha1.GaleraRestoreList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("galerarestores").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested galeraRestores.
func (c *galeraRestores) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("galerarestores").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a galeraRestore and creates it.  Returns the server's representation of the galeraRestore, and an error, if there is any.
func (c *galeraRestores) Create(galeraRestore *v1alpha1.GaleraRestore) (result *v1alpha1.GaleraRestore, err error) {
	result = &v1alpha1.GaleraRestore{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("galerarestores").
		Body(galeraRestore).
		Do().
		Into(result)
	return
}

// Update takes the representation of a galeraRestore and updates it. Returns the server's representation of the galeraRestore, and an error, if there is any.
func (c *galeraRestores) Update(galeraRestore *v1alpha1.GaleraRestore) (result *v1alpha1.GaleraRestore, err error) {
	result = &v1alpha1.GaleraRestore{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("galerarestores").
		Name(galeraRestore.Name).
		Body(galeraRestore).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *galeraRestores) UpdateStatus(galeraRestore *v1alpha1.GaleraRestore) (result *v1alpha1.GaleraRestore, err error) {
	result = &v1alpha1.GaleraRestore{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("galerarestores").
		Name(galeraRestore.Name).
		SubResource("status").
		Body(galeraRestore).
		Do().
		Into(result)
	return
}

// Delete takes name of the galeraRestore and deletes it. Returns an error if one occurs.
func (c *galeraRestores) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("galerarestores").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *galeraRestores) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("galerarestores").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched galeraRestore.
func (c *galeraRestores) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.GaleraRestore, err error) {
	result = &v1alpha1.GaleraRestore{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("galerarestores").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
