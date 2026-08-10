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

package v1beta3_test

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	infrav3 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	infrav4 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta4"
	fuzz "sigs.k8s.io/cluster-api-provider-cloudstack/test/fuzz"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

var _ = ginkgo.DescribeTable("v1beta3 <-> v1beta4 conversion is lossless",
	func(hub conversion.Hub, spoke conversion.Convertible, funcs ...fuzzer.FuzzerFuncs) {
		scheme := runtime.NewScheme()
		utilruntime.Must(infrav3.AddToScheme(scheme))
		utilruntime.Must(infrav4.AddToScheme(scheme))
		fuzz.RoundTrip(scheme, hub, spoke, funcs...)
	},
	ginkgo.Entry("CloudStackCluster", &infrav4.CloudStackCluster{}, &infrav3.CloudStackCluster{}),
	ginkgo.Entry("CloudStackMachine", &infrav4.CloudStackMachine{}, &infrav3.CloudStackMachine{}),
	ginkgo.Entry("CloudStackMachineTemplate", &infrav4.CloudStackMachineTemplate{}, &infrav3.CloudStackMachineTemplate{}),
	ginkgo.Entry("CloudStackFailureDomain", &infrav4.CloudStackFailureDomain{}, &infrav3.CloudStackFailureDomain{}),
	ginkgo.Entry("CloudStackIsolatedNetwork", &infrav4.CloudStackIsolatedNetwork{}, &infrav3.CloudStackIsolatedNetwork{}),
	ginkgo.Entry("CloudStackAffinityGroup", &infrav4.CloudStackAffinityGroup{}, &infrav3.CloudStackAffinityGroup{}),
	ginkgo.Entry("CloudStackMachineStateChecker", &infrav4.CloudStackMachineStateChecker{}, &infrav3.CloudStackMachineStateChecker{}),
)
