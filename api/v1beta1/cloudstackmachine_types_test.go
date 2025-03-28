/*
Copyright 2022 The Kubernetes Authors.

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
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta1"
)

var _ = ginkgo.Describe("CloudStackMachine types", func() {
	var cloudStackMachine infrav1.CloudStackMachine

	ginkgo.BeforeEach(func() { // Reset test vars to initial state.
		cloudStackMachine = infrav1.CloudStackMachine{}
	})

	ginkgo.Context("When calculating time since state change", func() {
		ginkgo.It("Return the correct value when the last state update time is known", func() {
			delta := time.Duration(10 * time.Minute)
			lastUpdated := time.Now().Add(-delta)
			cloudStackMachine.Status.InstanceStateLastUpdated = metav1.NewTime(lastUpdated)
			gomega.Expect(cloudStackMachine.Status.TimeSinceLastStateChange()).Should(gomega.BeNumerically("~", delta, time.Second))
		})

		ginkgo.It("Return a negative value when the last state update time is unknown", func() {
			gomega.Expect(cloudStackMachine.Status.TimeSinceLastStateChange()).Should(gomega.BeNumerically("<", 0))
		})
	})
})
