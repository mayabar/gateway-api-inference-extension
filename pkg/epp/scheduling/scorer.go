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

package scheduling

import (
	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/scheduling/types"
)

type PodScore struct {
	score float64
	pod   *types.PodMetrics
}

// Scorer is the interface that scorers must implement
type Scorer interface {
	ScoreTargets(ctx *types.Context, pods []*types.PodMetrics, req *types.LLMRequest) ([]PodScore, error)
}

// sessionAffinity is a routing scorer that routes subsequent
// requests in a session to the same pod as the first request in the
// session was sent to, by giving that pod the specified weight and assigning
// zero score to the rest of the targets
type SessionAffinityScorer struct {
	weight    float64
	datastore Datastore
}

func NewSessionAffinityScorer(weight float64, datastore Datastore) Scorer {
	return SessionAffinityScorer{
		weight:    weight,
		datastore: datastore,
	}
}

// ScoreTargets does the actual scoring of the target pods by the session affinity.
func (s SessionAffinityScorer) ScoreTargets(ctx *types.Context, pods []*types.PodMetrics, req *types.LLMRequest) ([]PodScore, error) {
	scoredPods := make([]PodScore, len(pods))
	selectedPodFullName := ""

	if req.SessionID != "" {
		selectedPod := s.datastore.GetPodForSession(req.SessionID)
		if selectedPod != nil {
			selectedPodFullName = selectedPod.NamespacedName.String()
		}
	}

	// session is not defined - no score for all pods
	for i, pod := range pods {
		if selectedPodFullName == pod.NamespacedName.String() {
			scoredPods[i].score = s.weight
		}
		scoredPods[i].pod = pod
	}

	return scoredPods, nil
}
