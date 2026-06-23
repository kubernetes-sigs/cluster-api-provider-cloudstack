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
	ginkgo "github.com/onsi/ginkgo/v2"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	infrav2 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	infrav4 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta4"
	fuzz "sigs.k8s.io/cluster-api-provider-cloudstack/test/fuzz"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
	"sigs.k8s.io/randfill"
)

// machineTemplateFuzzFuncs restricts fuzzing of the v1beta2 machine-template's inner ObjectMeta to
// Labels and Annotations. v1beta2 models template metadata as a full metav1.ObjectMeta, but the
// v1beta3/v1beta4 contract (clusterv1.ObjectMeta) only preserves labels and annotations on template
// metadata, so the other ObjectMeta fields are intentionally not round-trippable.
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

var _ = ginkgo.DescribeTable("v1beta2 <-> v1beta4 conversion is lossless",
	func(hub conversion.Hub, spoke conversion.Convertible, funcs ...fuzzer.FuzzerFuncs) {
		scheme := runtime.NewScheme()
		utilruntime.Must(infrav2.AddToScheme(scheme))
		utilruntime.Must(infrav4.AddToScheme(scheme))
		fuzz.RoundTrip(scheme, hub, spoke, funcs...)
	},
	ginkgo.Entry("CloudStackCluster", &infrav4.CloudStackCluster{}, &infrav2.CloudStackCluster{}),
	ginkgo.Entry("CloudStackMachine", &infrav4.CloudStackMachine{}, &infrav2.CloudStackMachine{}),
	ginkgo.Entry("CloudStackMachineTemplate", &infrav4.CloudStackMachineTemplate{}, &infrav2.CloudStackMachineTemplate{}, machineTemplateFuzzFuncs),
	ginkgo.Entry("CloudStackFailureDomain", &infrav4.CloudStackFailureDomain{}, &infrav2.CloudStackFailureDomain{}),
	ginkgo.Entry("CloudStackIsolatedNetwork", &infrav4.CloudStackIsolatedNetwork{}, &infrav2.CloudStackIsolatedNetwork{}),
	ginkgo.Entry("CloudStackAffinityGroup", &infrav4.CloudStackAffinityGroup{}, &infrav2.CloudStackAffinityGroup{}),
	ginkgo.Entry("CloudStackMachineStateChecker", &infrav4.CloudStackMachineStateChecker{}, &infrav2.CloudStackMachineStateChecker{}),
)
