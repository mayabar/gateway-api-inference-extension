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

package picker

import (
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/scheduling/plugins"
	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/scheduling/types"
)

func NewLRUPicker() plugins.Picker {
	return &LRUPicker{
		random:    &RandomPicker{},
		podsCache: map[k8stypes.NamespacedName]int{},
	}
}

var _ plugins.Picker = &LRUPicker{}

// RoundRobinPicker picks a pod from the list of candidates in round robin order.
type LRUPicker struct {
	random    *RandomPicker
	podsCache map[k8stypes.NamespacedName]int
}

func (p *LRUPicker) Name() string {
	return "lru"
}

func (p *LRUPicker) Pick(ctx *types.SchedulingContext, scoredPods []*types.ScoredPod) *types.Result {
	// ensure that all scored pods are in the cache
	bestPods := []*types.ScoredPod{}
	minUsages := 100000

	ctx.Logger.Info("LRU picker cache", "pods", p.podsCache)

	// find pod(s) with lowest number of usages
	for _, pod := range scoredPods {
		podUsages, ok := p.podsCache[pod.GetPod().NamespacedName]
		if !ok {
			p.podsCache[pod.GetPod().NamespacedName] = 0
			podUsages = 0
		}

		if podUsages == minUsages {
			// add this pod to list of candidates
			bestPods = append(bestPods, pod)
		} else if podUsages < minUsages {
			minUsages = podUsages
			bestPods = []*types.ScoredPod{pod}
		}
	}

	if len(bestPods) == 1 {
		ctx.Logger.Info("LRU picker", "usages", minUsages, "pod", *bestPods[0])
		p.podsCache[bestPods[0].GetPod().NamespacedName] = p.podsCache[bestPods[0].GetPod().NamespacedName] + 1
		return &types.Result{TargetPod: bestPods[0]}
	}

	res := p.random.Pick(ctx, bestPods) // pick randomly from the less used pods
	p.podsCache[res.TargetPod.GetPod().NamespacedName] = p.podsCache[res.TargetPod.GetPod().NamespacedName] + 1
	ctx.Logger.Info("LRU picker, randomly selected", "usages", minUsages, "pod", res.TargetPod)
	return res
}
