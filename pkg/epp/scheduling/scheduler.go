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

// Package scheduling implements request scheduling algorithms.
package scheduling

import (
	"context"
	"fmt"
	"github.com/neuralmagic/kvcache-manager/pkg/kvcache"
	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/scheduling/scorers"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/scheduling/types"
	envutil "sigs.k8s.io/gateway-api-inference-extension/pkg/epp/util/env"
	errutil "sigs.k8s.io/gateway-api-inference-extension/pkg/epp/util/error"
	logutil "sigs.k8s.io/gateway-api-inference-extension/pkg/epp/util/logging"
)

// Config holds all the configuration values for the scheduler
type Config struct {
	KVCacheThreshold       float64
	QueueThresholdCritical int
	QueueingThresholdLoRA  int
	LoraAffinityThreshold  float64
}

const (
	// Default values to use if environment variables are not set
	defaultKVCacheThreshold       = 0.8
	defaultQueueThresholdCritical = 5
	defaultQueueingThresholdLoRA  = 128
	defaultLoraAffinityThreshold  = 0.999
)

// LoadConfig loads configuration from environment variables
func LoadConfig() Config {
	// Use a default logger for initial configuration loading
	baseLogger := log.Log.WithName("scheduling-config")

	config := Config{
		KVCacheThreshold:       envutil.GetEnvFloat("KV_CACHE_THRESHOLD", defaultKVCacheThreshold, baseLogger),
		QueueThresholdCritical: envutil.GetEnvInt("QUEUE_THRESHOLD_CRITICAL", defaultQueueThresholdCritical, baseLogger),
		QueueingThresholdLoRA:  envutil.GetEnvInt("QUEUING_THRESHOLD_LORA", defaultQueueingThresholdLoRA, baseLogger),
		LoraAffinityThreshold:  envutil.GetEnvFloat("LORA_AFFINITY_THRESHOLD", defaultLoraAffinityThreshold, baseLogger),
	}

	baseLogger.V(logutil.DEFAULT).Info("Scheduler configuration loaded", "config", config)

	return config
}

var config = LoadConfig()

var (
	lowLatencyFilter = &decisionTreeFilter{
		current: lowQueueFilter,
		nextOnSuccess: &decisionTreeFilter{
			current: loRAAffinityFilter,
			nextOnSuccessOrFailure: &decisionTreeFilter{
				current: leastQueueFilter,
				nextOnSuccessOrFailure: &decisionTreeFilter{
					current: leastKVCacheFilter,
				},
			},
		},
		nextOnFailure: &decisionTreeFilter{
			current: leastQueueFilter,
			nextOnSuccessOrFailure: &decisionTreeFilter{
				current: loRAAffinityFilter,
				nextOnSuccessOrFailure: &decisionTreeFilter{
					current: leastKVCacheFilter,
				},
			},
		},
	}

	sheddableRequestFilter = &decisionTreeFilter{
		// When there is at least one model server that's not queuing requests, and still has KV
		// cache below a certain threshold, we consider this model server has capacity to handle
		// a sheddable request without impacting critical requests.
		current:       hasCapacityFilter,
		nextOnSuccess: lowLatencyFilter,
		// If all pods are queuing or running above the KVCache threshold, we drop the sheddable
		// request to make room for critical requests.
		nextOnFailure: dropRequestFilter,
	}

	hasCapacityFilter = &basicFilter{
		name:   "has capacity for sheddable requests",
		filter: toFilterFunc(queueThresholdPredicate(config.QueueThresholdCritical).and(kvCacheThresholdPredicate(config.KVCacheThreshold))),
	}

	dropRequestFilter = &basicFilter{
		name: "drop request",
		filter: func(ctx *types.Context, pods []*types.PodMetrics) ([]*types.PodMetrics, error) {
			ctx.Logger.V(logutil.DEFAULT).Info("Request dropped", "request", ctx.Req)
			return []*types.PodMetrics{}, errutil.Error{
				Code: errutil.InferencePoolResourceExhausted, Msg: "dropping request due to limited backend resources",
			}
		},
	}
)

func NewScheduler(ctx context.Context, datastore types.Datastore) (*Scheduler, error) {
	sMgr := scorers.NewScorerManager()

	// TODO: make scorers configuration

	kvCacheAwareIndexerConfig := kvcache.NewDefaultConfig()
	kvCacheAwareIndexerConfig.KVBlockIndexerConfig.RedisAddr = "vllm-p2p-lookup-server-service.default.svc.cluster.local:8100"
	kvCacheAwareScorer, err := scorers.NewKVCacheAwareScorer(ctx, kvCacheAwareIndexerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create KVCacheAwareScorer: %w", err)
	}

	sMgr.AddScorer(scorers.NewSessionAffinityScorer(datastore), 1.0)
	sMgr.AddScorer(kvCacheAwareScorer, 5.0)

	return &Scheduler{
		datastore:              datastore,
		criticalRequestFilter:  lowLatencyFilter,
		sheddableRequestFilter: sheddableRequestFilter,
		scorerMng:              sMgr,
	}, nil
}

type Scheduler struct {
	datastore              types.Datastore
	criticalRequestFilter  Filter
	sheddableRequestFilter Filter
	scorerMng              *scorers.ScorerManager
}

// Schedule finds the target pod based on metrics and the requested lora adapter.
func (s *Scheduler) Schedule(ctx context.Context, req *types.LLMRequest) (targetPod types.Pod, err error) {
	logger := log.FromContext(ctx).WithValues("request", req)

	// Snapshot pod metrics from the datastore to:
	// 1. Reduce concurrent access to the datastore.
	// 2. Ensure consistent data during the scheduling operation of a request.
	sCtx := types.NewContext(ctx, req, types.ToSchedulerPodMetrics(s.datastore.PodGetAll()))
	logger.V(logutil.DEBUG).Info(fmt.Sprintf("Scheduling a request. Metrics: %+v", sCtx.PodsSnapshot))

	var filter Filter
	if req.Critical {
		filter = s.criticalRequestFilter
	} else {
		filter = s.sheddableRequestFilter
	}

	pods, err := filter.Filter(sCtx, sCtx.PodsSnapshot)
	if err != nil || len(pods) == 0 {
		return nil, fmt.Errorf("failed to apply filter, resulted %v pods, this should never happen: %w", len(pods), err)
	}

	selectedPod, err := s.scorerMng.ScoreTargets(sCtx, pods)
	if err != nil {
		return nil, fmt.Errorf("failed to apply scorers: %w", err)
	}

	return selectedPod, nil
}
