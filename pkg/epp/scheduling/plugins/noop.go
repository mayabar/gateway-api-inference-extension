/*
Copyright 2025 The Kubernetes Authors.

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

package plugins

import (
	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/scheduling/types"
)

// NoopPlugin provides a default, no-operation implementation of the Plugin interface.
// It can be embedded in other plugin implementations to avoid boilerplate code for
// unused methods.
type NoopPlugin struct{}

func (p *NoopPlugin) Name() string { return "NoopPlugin" }

func (p *NoopPlugin) Score(ctx *types.Context, pod types.Pod) (float64, error) { return 0.0, nil }

func (p *NoopPlugin) Filter(ctx *types.Context, pods []types.Pod) ([]types.Pod, error) {
	return pods, nil
}

func (p *NoopPlugin) PreSchedule(ctx *types.Context) {}

func (p *NoopPlugin) PostSchedule(ctx *types.Context, res *types.Result) {}
