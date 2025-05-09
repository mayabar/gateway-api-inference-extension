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
	"fmt"
	"math/rand"

	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/scheduling/types"
	logutil "sigs.k8s.io/gateway-api-inference-extension/pkg/epp/util/logging"
)

type RandomPicker struct{}

func (rp *RandomPicker) Name() string {
	return "random"
}

func (rp *RandomPicker) Pick(ctx *types.SchedulingContext, pods []types.Pod) *types.Result {
	ctx.Logger.V(logutil.DEBUG).Info(fmt.Sprintf("Selecting a random pod from %d candidates: %+v", len(pods), pods))
	i := rand.Intn(len(pods))
	return &types.Result{TargetPod: pods[i]}
}
