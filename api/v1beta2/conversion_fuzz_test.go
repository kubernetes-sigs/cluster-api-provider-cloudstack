/*
Copyright 2024 The Kubernetes Authors.

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

package v1beta2_test

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	infrav2 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	infrav4 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta4"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/randfill"
)

// machineTemplateFuzzFuncs restricts fuzzing of the v1beta2 machine-template's inner ObjectMeta
// to Labels and Annotations. v1beta2 models template metadata as a full metav1.ObjectMeta, but
// the v1beta3/v1beta4 contract (clusterv1.ObjectMeta) only preserves labels and annotations on
// template metadata, so the other ObjectMeta fields are intentionally not round-trippable.
func machineTemplateFuzzFuncs(_ runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		func(in *infrav2.CloudStackMachineTemplateResource, c randfill.Continue) {
			c.FillNoCustom(in)
			in.ObjectMeta = metav1.ObjectMeta{
				Labels:      in.ObjectMeta.Labels,
				Annotations: in.ObjectMeta.Annotations,
			}
		},
	}
}

// TestFuzzyConversion verifies that converting a v1beta2 spoke object up to the v1beta4 hub
// and back (and the reverse) is lossless, with fields absent from v1beta2 preserved via the
// conversion-data annotation.
func TestFuzzyConversion(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(infrav2.AddToScheme(scheme))
	utilruntime.Must(infrav4.AddToScheme(scheme))

	t.Run("for CloudStackCluster", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme: scheme,
		Hub:    &infrav4.CloudStackCluster{},
		Spoke:  &infrav2.CloudStackCluster{},
	}))

	t.Run("for CloudStackMachine", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme: scheme,
		Hub:    &infrav4.CloudStackMachine{},
		Spoke:  &infrav2.CloudStackMachine{},
	}))

	t.Run("for CloudStackMachineTemplate", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme:      scheme,
		Hub:         &infrav4.CloudStackMachineTemplate{},
		Spoke:       &infrav2.CloudStackMachineTemplate{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{machineTemplateFuzzFuncs},
	}))

	t.Run("for CloudStackFailureDomain", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme: scheme,
		Hub:    &infrav4.CloudStackFailureDomain{},
		Spoke:  &infrav2.CloudStackFailureDomain{},
	}))

	t.Run("for CloudStackIsolatedNetwork", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme: scheme,
		Hub:    &infrav4.CloudStackIsolatedNetwork{},
		Spoke:  &infrav2.CloudStackIsolatedNetwork{},
	}))

	t.Run("for CloudStackAffinityGroup", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme: scheme,
		Hub:    &infrav4.CloudStackAffinityGroup{},
		Spoke:  &infrav2.CloudStackAffinityGroup{},
	}))

	t.Run("for CloudStackMachineStateChecker", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme: scheme,
		Hub:    &infrav4.CloudStackMachineStateChecker{},
		Spoke:  &infrav2.CloudStackMachineStateChecker{},
	}))
}
