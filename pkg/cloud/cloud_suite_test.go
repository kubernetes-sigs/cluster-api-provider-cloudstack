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

package cloud_test

import (
	"os"
	"strings"
	"testing"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/uuid"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta3"
	"sigs.k8s.io/cluster-api-provider-cloudstack/test/helpers"
)

var (
	// cloud.Client is our cloud package used to interact with ACS.
	realCloudClient cloud.Client // Real cloud client is a cloud client connected to a real Apache CloudStack instance.
	client          cloud.Client // client is simply a pointer to a cloud client object intended to be swapped per test.
	realCSClient    *cloudstack.CloudStackClient
	testDomainPath  string // Needed in before and in after suite.
)

func TestCloud(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.BeforeSuite(func() {
		suiteConfig, _ := ginkgo.GinkgoConfiguration()
		if !strings.Contains(suiteConfig.LabelFilter, "!integ") { // Skip if integ tests are filtered out.
			// Create a real cloud client.
			var connectionErr error
			realCSClient, connectionErr = helpers.NewCSClient()
			gomega.Ω(connectionErr).ShouldNot(gomega.HaveOccurred())

			repoRoot := os.Getenv("REPO_ROOT")
			realCloudClient, connectionErr = cloud.NewClientFromYamlPath(
				repoRoot+"/cloud-config.yaml", "myendpoint")
			gomega.Ω(connectionErr).ShouldNot(gomega.HaveOccurred())

			// Create a real CloudStack client.
			realCSClient, connectionErr = helpers.NewCSClient()
			gomega.Ω(connectionErr).ShouldNot(gomega.HaveOccurred())

			// Create a new account and user to run tests that use a real ACS instance.
			uid := string(uuid.NewUUID())
			newAccount := cloud.Account{
				Name:   "TestAccount-" + uid,
				Domain: cloud.Domain{Name: "TestDomain-" + uid, Path: "ROOT/TestDomain-" + uid}}
			newUser := cloud.User{Account: newAccount}
			gomega.Ω(helpers.GetOrCreateUserWithKey(realCSClient, &newUser)).Should(gomega.Succeed())
			testDomainPath = newAccount.Domain.Path

			gomega.Ω(newUser.APIKey).ShouldNot(gomega.BeEmpty())

			// Switch to test account user.
			realCloudClient, connectionErr = realCloudClient.NewClientInDomainAndAccount(
				newAccount.Domain.Name, newAccount.Name, "")
			gomega.Ω(connectionErr).ShouldNot(gomega.HaveOccurred())
		}
	})
	ginkgo.AfterSuite(func() {
		if realCSClient != nil { // Check for nil in case the before suite setup failed.
			// Delete created domain.
			id, err, found := helpers.GetDomainByPath(realCSClient, testDomainPath)
			gomega.Ω(err).ShouldNot(gomega.HaveOccurred())
			gomega.Ω(found).Should(gomega.BeTrue())
			gomega.Ω(helpers.DeleteDomain(realCSClient, id)).Should(gomega.Succeed())
		}
	})
	ginkgo.RunSpecs(t, "Cloud Suite")
}

// FetchIntegTestResources runs through basic CloudStack Client setup methods needed to test others.
func FetchIntegTestResources() {
	gomega.Ω(realCloudClient.ResolveZone(&dummies.CSFailureDomain1.Spec.Zone)).Should(gomega.Succeed())
	gomega.Ω(dummies.CSFailureDomain1.Spec.Zone.ID).ShouldNot(gomega.BeEmpty())
	dummies.CSMachine1.Spec.DiskOffering.Name = ""
	dummies.CSCluster.Spec.ControlPlaneEndpoint.Host = ""
	gomega.Ω(realCloudClient.GetOrCreateIsolatedNetwork(
		dummies.CSFailureDomain1, dummies.CSISONet1, dummies.CSCluster)).Should(gomega.Succeed())
}
