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
	ginkgo "github.com/onsi/ginkgo/v2"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta1"
	infrav4 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta4"
	fuzz "sigs.k8s.io/cluster-api-provider-cloudstack/test/fuzz"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
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

// CloudStackCluster is intentionally excluded: v1beta1 models its topology as Zones and its
// down-conversion performs external CloudStack/Kubernetes lookups and rejects clusters with fewer
// than one failure domain, none of which is fuzzable. Its conversion is covered by the hand-written
// cases in conversion_test.go.
var _ = ginkgo.DescribeTable("v1beta1 <-> v1beta4 conversion is lossless",
	func(hub conversion.Hub, spoke conversion.Convertible, funcs ...fuzzer.FuzzerFuncs) {
		scheme := runtime.NewScheme()
		utilruntime.Must(infrav1.AddToScheme(scheme))
		utilruntime.Must(infrav4.AddToScheme(scheme))
		fuzz.RoundTrip(scheme, hub, spoke, funcs...)
	},
	ginkgo.Entry("CloudStackMachine", &infrav4.CloudStackMachine{}, &infrav1.CloudStackMachine{}, machineFuzzFuncs),
	ginkgo.Entry("CloudStackMachineTemplate", &infrav4.CloudStackMachineTemplate{}, &infrav1.CloudStackMachineTemplate{}, machineFuzzFuncs, machineTemplateFuzzFuncs),
	ginkgo.Entry("CloudStackIsolatedNetwork", &infrav4.CloudStackIsolatedNetwork{}, &infrav1.CloudStackIsolatedNetwork{}),
	ginkgo.Entry("CloudStackAffinityGroup", &infrav4.CloudStackAffinityGroup{}, &infrav1.CloudStackAffinityGroup{}),
	ginkgo.Entry("CloudStackMachineStateChecker", &infrav4.CloudStackMachineStateChecker{}, &infrav1.CloudStackMachineStateChecker{}),
)
