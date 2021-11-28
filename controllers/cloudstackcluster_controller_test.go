/*
Copyright 2021.

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

package controllers_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	infrav1 "gitlab.aws.dev/ce-pike/merida/cluster-api-provider-capc/api/v1alpha4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getCloudStackCluster() *infrav1.CloudStackCluster {
	return &infrav1.CloudStackCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
			Kind:       "CloudStackCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: infrav1.CloudStackClusterSpec{
			Zone:    "zone",
			Network: "network",
		},
	}
}

var _ = Describe("CloudStackClusterReconciler", func() {
	ctx := context.Background()
	csCluster := getCloudStackCluster()

	BeforeEach(func() {

	})

	It("does things", func() {
		Ω(k8sClient.Create(ctx, csCluster)).Should(Succeed())
	})
})
