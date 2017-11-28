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
	v1alpha1 "github.com/coreos/etcd-operator/pkg/apis/galera/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeGaleraRestores implements GaleraRestoreInterface
type FakeGaleraRestores struct {
	Fake *FakeGaleraV1alpha1
	ns   string
}

var galerarestoresResource = schema.GroupVersionResource{Group: "galera.database.beekhof.net", Version: "v1alpha1", Resource: "galerarestores"}

var galerarestoresKind = schema.GroupVersionKind{Group: "galera.database.beekhof.net", Version: "v1alpha1", Kind: "GaleraRestore"}

// Get takes name of the galeraRestore, and returns the corresponding galeraRestore object, and an error if there is any.
func (c *FakeGaleraRestores) Get(name string, options v1.GetOptions) (result *v1alpha1.GaleraRestore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(galerarestoresResource, c.ns, name), &v1alpha1.GaleraRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.GaleraRestore), err
}

// List takes label and field selectors, and returns the list of GaleraRestores that match those selectors.
func (c *FakeGaleraRestores) List(opts v1.ListOptions) (result *v1alpha1.GaleraRestoreList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(galerarestoresResource, galerarestoresKind, c.ns, opts), &v1alpha1.GaleraRestoreList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.GaleraRestoreList{}
	for _, item := range obj.(*v1alpha1.GaleraRestoreList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested galeraRestores.
func (c *FakeGaleraRestores) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(galerarestoresResource, c.ns, opts))

}

// Create takes the representation of a galeraRestore and creates it.  Returns the server's representation of the galeraRestore, and an error, if there is any.
func (c *FakeGaleraRestores) Create(galeraRestore *v1alpha1.GaleraRestore) (result *v1alpha1.GaleraRestore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(galerarestoresResource, c.ns, galeraRestore), &v1alpha1.GaleraRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.GaleraRestore), err
}

// Update takes the representation of a galeraRestore and updates it. Returns the server's representation of the galeraRestore, and an error, if there is any.
func (c *FakeGaleraRestores) Update(galeraRestore *v1alpha1.GaleraRestore) (result *v1alpha1.GaleraRestore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(galerarestoresResource, c.ns, galeraRestore), &v1alpha1.GaleraRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.GaleraRestore), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeGaleraRestores) UpdateStatus(galeraRestore *v1alpha1.GaleraRestore) (*v1alpha1.GaleraRestore, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(galerarestoresResource, "status", c.ns, galeraRestore), &v1alpha1.GaleraRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.GaleraRestore), err
}

// Delete takes name of the galeraRestore and deletes it. Returns an error if one occurs.
func (c *FakeGaleraRestores) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(galerarestoresResource, c.ns, name), &v1alpha1.GaleraRestore{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeGaleraRestores) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(galerarestoresResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.GaleraRestoreList{})
	return err
}

// Patch applies the patch and returns the patched galeraRestore.
func (c *FakeGaleraRestores) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.GaleraRestore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(galerarestoresResource, c.ns, name, data, subresources...), &v1alpha1.GaleraRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.GaleraRestore), err
}
