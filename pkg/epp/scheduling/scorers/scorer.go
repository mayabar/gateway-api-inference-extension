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

package scorers

import (
	"fmt"
	"math/rand/v2"
	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/scheduling/types"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

type PodScore struct {
	Score float64
	Pod   *types.PodMetrics
}

// Scorer defines an interface for scoring pods.
type Scorer interface {
	ScoreTargets(ctx *types.Context, pods []*types.PodMetrics) ([]PodScore, error)
	Name() string
}

// ScorerManager manages a list of scorers and their use.
type ScorerManager struct {
	scorers map[Scorer]float64
}

// NewScorerManager creates a new ScorerManager.
func NewScorerManager() *ScorerManager {
	return &ScorerManager{
		scorers: make(map[Scorer]float64),
	}
}

// AddScorer adds a new scorer to the manager.
func (mgr *ScorerManager) AddScorer(scorer Scorer, weight float64) {
	mgr.scorers[scorer] = weight
}

// ScoreTargets scores the target pods using all registered scorers.
func (mgr *ScorerManager) ScoreTargets(ctx *types.Context, pods []*types.PodMetrics) (*types.PodMetrics, error) {
	logger := log.FromContext(ctx)

	podsTotalScore := make(map[*types.PodMetrics]float64)

	// initialize zero score for all pods
	for _, pod := range pods {
		podsTotalScore[pod] = 0.0
	}

	// add scores from all scorers
	for scorer, weight := range mgr.scorers {
		scoredPods, err := scorer.ScoreTargets(ctx, pods)
		if err != nil {
			logger.Error(err, "Error scoring pods", "scorer", scorer.Name())
			return nil, fmt.Errorf("error scoring pods: %w", err)
		}

		for _, scoredPod := range scoredPods {
			podsTotalScore[scoredPod.Pod] += weight * scoredPod.Score
		}
	}

	// select pod with maximum score, if more than one with the max score - use random pods from the list
	var highestScoreTargets []*types.PodMetrics
	maxScore := -1.0

	for pod, score := range podsTotalScore {
		if score > maxScore {
			maxScore = score
			highestScoreTargets = []*types.PodMetrics{pod}
		} else if score == maxScore {
			highestScoreTargets = append(highestScoreTargets, pod)
		}
	}

	// single pod with max score
	if len(highestScoreTargets) == 1 {
		return highestScoreTargets[0], nil
	}

	// select random pod from list of pods with max score
	return highestScoreTargets[rand.IntN(len(highestScoreTargets))], nil
}
