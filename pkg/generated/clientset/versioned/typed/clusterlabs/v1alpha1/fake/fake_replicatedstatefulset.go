/*
Copyright 2018 The rss-operator Authors

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
	v1alpha1 "github.com/beekhof/rss-operator/pkg/apis/clusterlabs/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeReplicatedStatefulSets implements ReplicatedStatefulSetInterface
type FakeReplicatedStatefulSets struct {
	Fake *FakeClusterlabsV1alpha1
	ns   string
}

var replicatedstatefulsetsResource = schema.GroupVersionResource{Group: "clusterlabs.org", Version: "v1alpha1", Resource: "replicatedstatefulsets"}

var replicatedstatefulsetsKind = schema.GroupVersionKind{Group: "clusterlabs.org", Version: "v1alpha1", Kind: "ReplicatedStatefulSet"}

// Get takes name of the replicatedStatefulSet, and returns the corresponding replicatedStatefulSet object, and an error if there is any.
func (c *FakeReplicatedStatefulSets) Get(name string, options v1.GetOptions) (result *v1alpha1.ReplicatedStatefulSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(replicatedstatefulsetsResource, c.ns, name), &v1alpha1.ReplicatedStatefulSet{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ReplicatedStatefulSet), err
}

// List takes label and field selectors, and returns the list of ReplicatedStatefulSets that match those selectors.
func (c *FakeReplicatedStatefulSets) List(opts v1.ListOptions) (result *v1alpha1.ReplicatedStatefulSetList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(replicatedstatefulsetsResource, replicatedstatefulsetsKind, c.ns, opts), &v1alpha1.ReplicatedStatefulSetList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ReplicatedStatefulSetList{}
	for _, item := range obj.(*v1alpha1.ReplicatedStatefulSetList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested replicatedStatefulSets.
func (c *FakeReplicatedStatefulSets) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(replicatedstatefulsetsResource, c.ns, opts))

}

// Create takes the representation of a replicatedStatefulSet and creates it.  Returns the server's representation of the replicatedStatefulSet, and an error, if there is any.
func (c *FakeReplicatedStatefulSets) Create(replicatedStatefulSet *v1alpha1.ReplicatedStatefulSet) (result *v1alpha1.ReplicatedStatefulSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(replicatedstatefulsetsResource, c.ns, replicatedStatefulSet), &v1alpha1.ReplicatedStatefulSet{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ReplicatedStatefulSet), err
}

// Update takes the representation of a replicatedStatefulSet and updates it. Returns the server's representation of the replicatedStatefulSet, and an error, if there is any.
func (c *FakeReplicatedStatefulSets) Update(replicatedStatefulSet *v1alpha1.ReplicatedStatefulSet) (result *v1alpha1.ReplicatedStatefulSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(replicatedstatefulsetsResource, c.ns, replicatedStatefulSet), &v1alpha1.ReplicatedStatefulSet{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ReplicatedStatefulSet), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeReplicatedStatefulSets) UpdateStatus(replicatedStatefulSet *v1alpha1.ReplicatedStatefulSet) (*v1alpha1.ReplicatedStatefulSet, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(replicatedstatefulsetsResource, "status", c.ns, replicatedStatefulSet), &v1alpha1.ReplicatedStatefulSet{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ReplicatedStatefulSet), err
}

// Delete takes name of the replicatedStatefulSet and deletes it. Returns an error if one occurs.
func (c *FakeReplicatedStatefulSets) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(replicatedstatefulsetsResource, c.ns, name), &v1alpha1.ReplicatedStatefulSet{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeReplicatedStatefulSets) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(replicatedstatefulsetsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.ReplicatedStatefulSetList{})
	return err
}

// Patch applies the patch and returns the patched replicatedStatefulSet.
func (c *FakeReplicatedStatefulSets) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ReplicatedStatefulSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(replicatedstatefulsetsResource, c.ns, name, data, subresources...), &v1alpha1.ReplicatedStatefulSet{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ReplicatedStatefulSet), err
}
