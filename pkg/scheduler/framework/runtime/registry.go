/*
Copyright 2019 The Kubernetes Authors.

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

package runtime

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	schedulerqueue "k8s.io/kubernetes/pkg/scheduler/queue"
	"sigs.k8s.io/yaml"
)

// PluginFactory is a function that builds a plugin.
type PluginFactory = func(configuration runtime.Object, f framework.Handle) (framework.Plugin, error)

// DecodeInto decodes configuration whose type is *runtime.Unknown to the interface into.
func DecodeInto(obj runtime.Object, into interface{}) error {
	if obj == nil {
		return nil
	}
	configuration, ok := obj.(*runtime.Unknown)
	if !ok {
		return fmt.Errorf("want args of type runtime.Unknown, got %T", obj)
	}
	if configuration.Raw == nil {
		return nil
	}

	switch configuration.ContentType {
	// If ContentType is empty, it means ContentTypeJSON by default.
	case runtime.ContentTypeJSON, "":
		return json.Unmarshal(configuration.Raw, into)
	case runtime.ContentTypeYAML:
		return yaml.Unmarshal(configuration.Raw, into)
	default:
		return fmt.Errorf("not supported content type %s", configuration.ContentType)
	}
}

// Registry is a collection of all available plugins. The framework uses a
// registry to enable and initialize configured plugins.
// All plugins must be in the registry before initializing the framework.
type Registry struct {
	Pf          map[string]PluginFactory
	CustomQueue schedulerqueue.SchedulingQueue
}

// Register adds a new plugin to the registry. If a plugin with the same name
// exists, it returns an error.
func (r *Registry) Register(name string, factory PluginFactory) error {
	if _, ok := r.Pf[name]; ok {
		return fmt.Errorf("a plugin named %v already exists", name)
	}
	r.Pf[name] = factory
	return nil
}

// Unregister removes an existing plugin from the registry. If no plugin with
// the provided name exists, it returns an error.
func (r *Registry) Unregister(name string) error {
	if _, ok := r.Pf[name]; !ok {
		return fmt.Errorf("no plugin named %v exists", name)
	}
	delete(r.Pf, name)
	return nil
}

// Merge merges the provided registry to the current one.
func (r *Registry) Merge(in Registry) error {
	if in.CustomQueue != nil {
		r.CustomQueue = in.CustomQueue
	}

	for name, factory := range in.Pf {
		if err := r.Register(name, factory); err != nil {
			return err
		}
	}
	return nil
}

// SetCustomQueue sets custom queue to the registry. If the custom queue is already
// set, it returns an error.
func (r *Registry) SetCustomQueue(customQueue schedulerqueue.SchedulingQueue) error {
	if r.CustomQueue != nil {
		return fmt.Errorf("custom queue is registered more than one time")
	}
	r.CustomQueue = customQueue
	return nil
}
