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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/uuid"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta2"
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
	RegisterFailHandler(Fail)
	BeforeSuite(func() {
		suiteConfig, _ := GinkgoConfiguration()
		if !strings.Contains(suiteConfig.LabelFilter, "!integ") { // Skip if integ tests are filtered out.
			// Create a real cloud client.
			projDir := os.Getenv("PROJECT_DIR")
			var connectionErr error
			realCloudClient, connectionErr = cloud.NewClient(projDir + "/cloud-config")
			Ω(connectionErr).ShouldNot(HaveOccurred())

			// Create a real CloudStack client.
			realCSClient, connectionErr = helpers.NewCSClient()
			Ω(connectionErr).ShouldNot(HaveOccurred())

			// Create a new account and user to run tests that use a real ACS instance.
			uid := string(uuid.NewUUID())
			newAccount := cloud.Account{
				Name:   "TestAccount-" + uid,
				Domain: cloud.Domain{Name: "TestDomain-" + uid, Path: "ROOT/TestDomain-" + uid}}
			newUser := cloud.User{Account: newAccount}
			Ω(helpers.GetOrCreateUserWithKey(realCSClient, &newUser)).Should(Succeed())
			testDomainPath = newAccount.Domain.Path

			Ω(newUser.APIKey).ShouldNot(BeEmpty())

			// Switch to test account user.
			cfg := cloud.Config{APIKey: newUser.APIKey, SecretKey: newUser.SecretKey}
			realCloudClient, connectionErr = realCloudClient.NewClientFromSpec(cfg)
			Ω(connectionErr).ShouldNot(HaveOccurred())
		}
	})
	AfterSuite(func() {
		if realCSClient != nil { // Check for nil in case the before suite setup failed.
			// Delete created domain.
			id, err, found := helpers.GetDomainByPath(realCSClient, testDomainPath)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(found).Should(BeTrue())
			Ω(helpers.DeleteDomain(realCSClient, id)).Should(Succeed())
		}
	})
	RunSpecs(t, "Cloud Suite")
}

// FetchIntegTestResources runs through basic CloudStack Client setup methods needed to test others.
func FetchIntegTestResources() {
	Ω(realCloudClient.ResolveZone(&dummies.CSZone1.Spec)).Should(Succeed())
	Ω(dummies.CSZone1.Spec.ID).ShouldNot(BeEmpty())
	dummies.CSMachine1.Status.ZoneID = dummies.CSZone1.Spec.ID
	dummies.CSMachine1.Spec.DiskOffering.Name = ""
	dummies.CSCluster.Spec.ControlPlaneEndpoint.Host = ""
	Ω(realCloudClient.GetOrCreateIsolatedNetwork(dummies.CSZone1, dummies.CSISONet1, dummies.CSCluster)).Should(Succeed())
}
