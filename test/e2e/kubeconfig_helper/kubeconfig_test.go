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

package kubeconfig_helper_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"sigs.k8s.io/cluster-api-provider-cloudstack-staging/test/e2e/kubeconfig_helper"
)

var _ = Describe("Test kubeconfig helper methods", func() {
	It("should work", func() {
		kubeconfig := kubeconfig_helper.NewKubeconfig()

		var kubeconfigPath string = "./data/kubeconfig"
		var unmodifiedKubeconfigPath string = "/tmp/unmodifiedKubeconfig"
		Ω(kubeconfig.Load(kubeconfigPath)).Should(Succeed())
		Ω(kubeconfig.Save(unmodifiedKubeconfigPath)).Should(Succeed())

		originalKubeconfig, err := ioutil.ReadFile(kubeconfigPath)
		Ω(err).ShouldNot(HaveOccurred())
		rewrittenKubeconfig, err := ioutil.ReadFile(unmodifiedKubeconfigPath)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(rewrittenKubeconfig).Should(Equal(originalKubeconfig))

		currentContextName, err := kubeconfig.GetCurrentContextName()
		Ω(err).ShouldNot(HaveOccurred())
		Ω(currentContextName).Should(Equal("kind-capi-test"))

		_, err = kubeconfig.GetCurrentContext()
		Ω(err).ShouldNot(HaveOccurred())

		currentClusterName, err := kubeconfig.GetCurrentClusterName()
		Ω(err).ShouldNot(HaveOccurred())
		Ω(currentClusterName).Should(Equal("kind-capi-test"))

		currentCluster, err := kubeconfig.GetCurrentCluster()
		Ω(err).ShouldNot(HaveOccurred())
		Ω(currentCluster).ShouldNot(BeEmpty())

		currentServer, err := kubeconfig.GetCurrentServer()
		Ω(err).ShouldNot(HaveOccurred())
		Ω(currentServer).Should(Equal("https://127.0.0.1:64927"))

		var newServerUrl string = "\"https://myTestServer:12345\""
		kubeconfig.SetCurrentServer(newServerUrl)

		var modifiedKubeconfigPath string = "/tmp/modifiedKubeconfig.yaml"
		Ω(kubeconfig.Save(modifiedKubeconfigPath)).Should(Succeed())
		Ω(modifiedKubeconfigPath).Should(BeAnExistingFile())

		modifiedKubeconfigContent, err := ioutil.ReadFile(modifiedKubeconfigPath)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(string(modifiedKubeconfigContent)).Should(ContainSubstring(newServerUrl))

	})
})
