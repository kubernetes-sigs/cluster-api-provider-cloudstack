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

package v1beta4_test

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta4"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta4"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = ginkgo.Describe("CloudStackCluster webhooks", func() {
	var ctx context.Context

	const (
		forbiddenFmt = "admission webhook.*denied the request.*Forbidden\\: %s"
		requiredFmt  = "admission webhook.*denied the request.*Required value\\: %s"
	)

	ginkgo.BeforeEach(func() {
		ctx = context.Background()
		dummies.SetDummyVars()
		_ = k8sClient.Delete(ctx, dummies.CSCluster)
		dummies.SetDummyVars()
	})

	ginkgo.DescribeTable("create validation",
		func(mutate func(), wantErrRegex string) {
			mutate()
			err := k8sClient.Create(ctx, dummies.CSCluster)
			if wantErrRegex != "" {
				gomega.Expect(err).To(gomega.MatchError(gomega.MatchRegexp(wantErrRegex)))
			} else {
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			}
		},
		ginkgo.Entry("accepts cluster with all attributes present", func() {}, ""),
		ginkgo.Entry("rejects cluster missing Zones.Network",
			func() {
				dummies.CSCluster.Spec.FailureDomains = []infrav1.CloudStackFailureDomainSpec{{}}
				dummies.CSCluster.Spec.FailureDomains[0].Zone.Name = "ZoneWNoNetwork"
			},
			fmt.Sprintf(requiredFmt, "each Zone requires a Network specification"),
		),
		ginkgo.Entry("rejects cluster with empty Zone spec",
			func() {
				dummies.CSCluster.Spec.FailureDomains[0].Zone = infrav1.CloudStackZoneSpec{}
			},
			fmt.Sprintf(requiredFmt, "each Zone requires a Network specification"),
		),
	)

	// The typed client omits host: "" and port: 0, so these cases send unstructured objects to match what
	// kubectl or clusterctl send for a template rendered with an empty CLUSTER_ENDPOINT_IP.
	ginkgo.DescribeTable("empty controlPlaneEndpoint is accepted and left for the controller to fill",
		func(endpoint map[string]any, wantEndpoint clusterv1.APIEndpoint) {
			gvk := infrav1.GroupVersion.WithKind("CloudStackCluster")
			dummies.CSCluster.SetGroupVersionKind(gvk)
			content, err := runtime.DefaultUnstructuredConverter.ToUnstructured(dummies.CSCluster)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			obj := &unstructured.Unstructured{Object: content}
			gomega.Expect(unstructured.SetNestedMap(obj.Object, endpoint, "spec", "controlPlaneEndpoint")).To(gomega.Succeed())

			gomega.Expect(k8sClient.Create(ctx, obj)).To(gomega.Succeed())

			got := &infrav1.CloudStackCluster{}
			gomega.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(dummies.CSCluster), got)).To(gomega.Succeed())
			gomega.Expect(got.Spec.ControlPlaneEndpoint).To(gomega.Equal(wantEndpoint))
		},
		ginkgo.Entry("empty host with a port keeps the port",
			map[string]any{"host": "", "port": int64(6443)}, clusterv1.APIEndpoint{Port: 6443}),
		ginkgo.Entry("empty host with a custom port keeps the custom port",
			map[string]any{"host": "", "port": int64(6444)}, clusterv1.APIEndpoint{Port: 6444}),
		ginkgo.Entry("empty host and zero port drop the endpoint",
			map[string]any{"host": "", "port": int64(0)}, clusterv1.APIEndpoint{}),
		ginkgo.Entry("empty host alone drops the endpoint",
			map[string]any{"host": ""}, clusterv1.APIEndpoint{}),
		ginkgo.Entry("empty endpoint object is dropped",
			map[string]any{}, clusterv1.APIEndpoint{}),
		ginkgo.Entry("host with zero port keeps the host",
			map[string]any{"host": "10.0.0.1", "port": int64(0)}, clusterv1.APIEndpoint{Host: "10.0.0.1"}),
	)

	ginkgo.Describe("update validation", func() {
		ginkgo.BeforeEach(func() {
			gomega.Expect(k8sClient.Create(ctx, dummies.CSCluster)).Should(gomega.Succeed())
		})

		ginkgo.DescribeTable("immutable fields are rejected",
			func(mutate func(), wantErrRegex string) {
				mutate()
				gomega.Expect(k8sClient.Update(ctx, dummies.CSCluster)).
					To(gomega.MatchError(gomega.MatchRegexp(wantErrRegex)))
			},
			ginkgo.Entry("rejects FailureDomain name change",
				func() { dummies.CSCluster.Spec.FailureDomains[0].Zone.Name = "SomeRandomUpdate" },
				fmt.Sprintf(forbiddenFmt, "Cannot change FailureDomain"),
			),
			ginkgo.Entry("rejects Zone Network name change",
				func() { dummies.CSCluster.Spec.FailureDomains[0].Zone.Network.Name = "ArbitraryUpdateNetworkName" },
				fmt.Sprintf(forbiddenFmt, "Cannot change FailureDomain"),
			),
			ginkgo.Entry("rejects controlplaneendpoint.host change",
				func() { dummies.CSCluster.Spec.ControlPlaneEndpoint.Host = "1.1.1.1" },
				fmt.Sprintf(forbiddenFmt, "controlplaneendpoint\\.host"),
			),
			ginkgo.Entry("rejects controlplaneendpoint.port change",
				func() { dummies.CSCluster.Spec.ControlPlaneEndpoint.Port = int32(1234) },
				fmt.Sprintf(forbiddenFmt, "controlplaneendpoint\\.port"),
			),
		)
	})
})
