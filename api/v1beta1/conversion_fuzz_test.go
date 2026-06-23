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

package v1beta1_test

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta1"
	infrav4 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta4"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/randfill"
)

// machineFuzzFuncs zeroes the v1beta1-only machine fields that are marked +k8s:conversion-gen=false
// (spec ZoneID/ZoneName/IdentityRef and status ZoneID). These legacy fields have no v1beta4
// equivalent and are intentionally dropped on up-conversion, so they cannot survive a round trip.
func machineFuzzFuncs(_ runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		func(in *infrav1.CloudStackMachineSpec, c randfill.Continue) {
			c.FillNoCustom(in)
			in.ZoneID = ""
			in.ZoneName = ""
			in.IdentityRef = nil
		},
		func(in *infrav1.CloudStackMachineStatus, c randfill.Continue) {
			c.FillNoCustom(in)
			in.ZoneID = ""
		},
	}
}

// machineTemplateFuzzFuncs restricts fuzzing of the v1beta1 machine-template's inner ObjectMeta to
// Labels and Annotations. v1beta1 models template metadata as a full metav1.ObjectMeta, but the
// v1beta4 contract (clusterv1.ObjectMeta) only preserves labels and annotations on template
// metadata, so the other ObjectMeta fields are intentionally not round-trippable.
func machineTemplateFuzzFuncs(_ runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		func(in *infrav1.CloudStackMachineTemplateResource, c randfill.Continue) {
			c.FillNoCustom(in)
			in.ObjectMeta = metav1.ObjectMeta{
				Labels:      in.ObjectMeta.Labels,
				Annotations: in.ObjectMeta.Annotations,
			}
		},
	}
}

// TestFuzzyConversion verifies lossless round-trip conversion between v1beta1 spokes and the
// v1beta4 hub. CloudStackCluster is intentionally excluded: v1beta1 models its topology as Zones
// and its down-conversion performs external CloudStack/Kubernetes lookups and rejects clusters
// with fewer than one failure domain, none of which is fuzzable. Its conversion is covered by the
// hand-written cases in conversion_test.go.
func TestFuzzyConversion(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(infrav1.AddToScheme(scheme))
	utilruntime.Must(infrav4.AddToScheme(scheme))

	t.Run("for CloudStackMachine", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme:      scheme,
		Hub:         &infrav4.CloudStackMachine{},
		Spoke:       &infrav1.CloudStackMachine{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{machineFuzzFuncs},
	}))

	t.Run("for CloudStackMachineTemplate", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme:      scheme,
		Hub:         &infrav4.CloudStackMachineTemplate{},
		Spoke:       &infrav1.CloudStackMachineTemplate{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{machineFuzzFuncs, machineTemplateFuzzFuncs},
	}))

	t.Run("for CloudStackIsolatedNetwork", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme: scheme,
		Hub:    &infrav4.CloudStackIsolatedNetwork{},
		Spoke:  &infrav1.CloudStackIsolatedNetwork{},
	}))

	t.Run("for CloudStackAffinityGroup", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme: scheme,
		Hub:    &infrav4.CloudStackAffinityGroup{},
		Spoke:  &infrav1.CloudStackAffinityGroup{},
	}))

	t.Run("for CloudStackMachineStateChecker", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme: scheme,
		Hub:    &infrav4.CloudStackMachineStateChecker{},
		Spoke:  &infrav1.CloudStackMachineStateChecker{},
	}))
}
