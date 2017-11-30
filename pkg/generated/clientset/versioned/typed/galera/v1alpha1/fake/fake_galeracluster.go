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
package fake

import (
	v1alpha1 "github.com/beekhof/galera-operator/pkg/apis/galera/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeGaleraClusters implements GaleraClusterInterface
type FakeGaleraClusters struct {
	Fake *FakeGaleraV1alpha1
	ns   string
}

var galeraclustersResource = schema.GroupVersionResource{Group: "galera.db.beekhof.net", Version: "v1alpha1", Resource: "galeraclusters"}

var galeraclustersKind = schema.GroupVersionKind{Group: "galera.db.beekhof.net", Version: "v1alpha1", Kind: "GaleraCluster"}

// Get takes name of the galeraCluster, and returns the corresponding galeraCluster object, and an error if there is any.
func (c *FakeGaleraClusters) Get(name string, options v1.GetOptions) (result *v1alpha1.GaleraCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(galeraclustersResource, c.ns, name), &v1alpha1.GaleraCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.GaleraCluster), err
}

// List takes label and field selectors, and returns the list of GaleraClusters that match those selectors.
func (c *FakeGaleraClusters) List(opts v1.ListOptions) (result *v1alpha1.GaleraClusterList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(galeraclustersResource, galeraclustersKind, c.ns, opts), &v1alpha1.GaleraClusterList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.GaleraClusterList{}
	for _, item := range obj.(*v1alpha1.GaleraClusterList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested galeraClusters.
func (c *FakeGaleraClusters) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(galeraclustersResource, c.ns, opts))

}

// Create takes the representation of a galeraCluster and creates it.  Returns the server's representation of the galeraCluster, and an error, if there is any.
func (c *FakeGaleraClusters) Create(galeraCluster *v1alpha1.GaleraCluster) (result *v1alpha1.GaleraCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(galeraclustersResource, c.ns, galeraCluster), &v1alpha1.GaleraCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.GaleraCluster), err
}

// Update takes the representation of a galeraCluster and updates it. Returns the server's representation of the galeraCluster, and an error, if there is any.
func (c *FakeGaleraClusters) Update(galeraCluster *v1alpha1.GaleraCluster) (result *v1alpha1.GaleraCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(galeraclustersResource, c.ns, galeraCluster), &v1alpha1.GaleraCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.GaleraCluster), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeGaleraClusters) UpdateStatus(galeraCluster *v1alpha1.GaleraCluster) (*v1alpha1.GaleraCluster, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(galeraclustersResource, "status", c.ns, galeraCluster), &v1alpha1.GaleraCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.GaleraCluster), err
}

// Delete takes name of the galeraCluster and deletes it. Returns an error if one occurs.
func (c *FakeGaleraClusters) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(galeraclustersResource, c.ns, name), &v1alpha1.GaleraCluster{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeGaleraClusters) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(galeraclustersResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.GaleraClusterList{})
	return err
}

// Patch applies the patch and returns the patched galeraCluster.
func (c *FakeGaleraClusters) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.GaleraCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(galeraclustersResource, c.ns, name, data, subresources...), &v1alpha1.GaleraCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.GaleraCluster), err
}
