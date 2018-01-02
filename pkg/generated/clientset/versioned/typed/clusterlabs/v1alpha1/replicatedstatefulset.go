/*
Copyright 2018 The etcd-operator Authors

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
	v1alpha1 "github.com/beekhof/galera-operator/pkg/apis/galera/v1alpha1"
	scheme "github.com/beekhof/galera-operator/pkg/generated/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ReplicatedStatefulSetsGetter has a method to return a ReplicatedStatefulSetInterface.
// A group's client should implement this interface.
type ReplicatedStatefulSetsGetter interface {
	ReplicatedStatefulSets(namespace string) ReplicatedStatefulSetInterface
}

// ReplicatedStatefulSetInterface has methods to work with ReplicatedStatefulSet resources.
type ReplicatedStatefulSetInterface interface {
	Create(*v1alpha1.ReplicatedStatefulSet) (*v1alpha1.ReplicatedStatefulSet, error)
	Update(*v1alpha1.ReplicatedStatefulSet) (*v1alpha1.ReplicatedStatefulSet, error)
	UpdateStatus(*v1alpha1.ReplicatedStatefulSet) (*v1alpha1.ReplicatedStatefulSet, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.ReplicatedStatefulSet, error)
	List(opts v1.ListOptions) (*v1alpha1.ReplicatedStatefulSetList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ReplicatedStatefulSet, err error)
	ReplicatedStatefulSetExpansion
}

// replicatedStatefulSets implements ReplicatedStatefulSetInterface
type replicatedStatefulSets struct {
	client rest.Interface
	ns     string
}

// newReplicatedStatefulSets returns a ReplicatedStatefulSets
func newReplicatedStatefulSets(c *ClusterlabsV1alpha1Client, namespace string) *replicatedStatefulSets {
	return &replicatedStatefulSets{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the replicatedStatefulSet, and returns the corresponding replicatedStatefulSet object, and an error if there is any.
func (c *replicatedStatefulSets) Get(name string, options v1.GetOptions) (result *v1alpha1.ReplicatedStatefulSet, err error) {
	result = &v1alpha1.ReplicatedStatefulSet{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("replicatedstatefulsets").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ReplicatedStatefulSets that match those selectors.
func (c *replicatedStatefulSets) List(opts v1.ListOptions) (result *v1alpha1.ReplicatedStatefulSetList, err error) {
	result = &v1alpha1.ReplicatedStatefulSetList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("replicatedstatefulsets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested replicatedStatefulSets.
func (c *replicatedStatefulSets) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("replicatedstatefulsets").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a replicatedStatefulSet and creates it.  Returns the server's representation of the replicatedStatefulSet, and an error, if there is any.
func (c *replicatedStatefulSets) Create(replicatedStatefulSet *v1alpha1.ReplicatedStatefulSet) (result *v1alpha1.ReplicatedStatefulSet, err error) {
	result = &v1alpha1.ReplicatedStatefulSet{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("replicatedstatefulsets").
		Body(replicatedStatefulSet).
		Do().
		Into(result)
	return
}

// Update takes the representation of a replicatedStatefulSet and updates it. Returns the server's representation of the replicatedStatefulSet, and an error, if there is any.
func (c *replicatedStatefulSets) Update(replicatedStatefulSet *v1alpha1.ReplicatedStatefulSet) (result *v1alpha1.ReplicatedStatefulSet, err error) {
	result = &v1alpha1.ReplicatedStatefulSet{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("replicatedstatefulsets").
		Name(replicatedStatefulSet.Name).
		Body(replicatedStatefulSet).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *replicatedStatefulSets) UpdateStatus(replicatedStatefulSet *v1alpha1.ReplicatedStatefulSet) (result *v1alpha1.ReplicatedStatefulSet, err error) {
	result = &v1alpha1.ReplicatedStatefulSet{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("replicatedstatefulsets").
		Name(replicatedStatefulSet.Name).
		SubResource("status").
		Body(replicatedStatefulSet).
		Do().
		Into(result)
	return
}

// Delete takes name of the replicatedStatefulSet and deletes it. Returns an error if one occurs.
func (c *replicatedStatefulSets) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("replicatedstatefulsets").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *replicatedStatefulSets) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("replicatedstatefulsets").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched replicatedStatefulSet.
func (c *replicatedStatefulSets) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ReplicatedStatefulSet, err error) {
	result = &v1alpha1.ReplicatedStatefulSet{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("replicatedstatefulsets").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
