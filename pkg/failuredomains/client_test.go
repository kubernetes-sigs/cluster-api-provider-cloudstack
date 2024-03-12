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

package failuredomains

import (
	"context"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Client Factory", func() {
	var (
		k8sClient  client.Client
		mockCtrl   *gomock.Controller
		mockClient *cloudstack.CloudStackClient

		us *cloudstack.MockUserServiceIface
		ds *cloudstack.MockDomainServiceIface
		as *cloudstack.MockAccountServiceIface

		ctx                 context.Context
		endpointCredentials *corev1.Secret
		clientConfig        *corev1.ConfigMap
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = cloudstack.NewMockClient(mockCtrl)

		us = mockClient.User.(*cloudstack.MockUserServiceIface)
		ds = mockClient.Domain.(*cloudstack.MockDomainServiceIface)
		as = mockClient.Account.(*cloudstack.MockAccountServiceIface)

		ctx = context.TODO()

		dummies.SetDummyUserVars()
		dummies.AccountName = "test-account"

		endpointCredentials = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "zone-a-creds",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"api-key":    []byte(dummies.Apikey),
				"secret-key": []byte(dummies.SecretKey),
				"api-url":    []byte("http://1.2.3.4:8080/client/api"),
				"verify-ssl": []byte("false"),
			},
		}

		clientConfig = &corev1.ConfigMap{}

		cloud.NewAsyncClient = func(apiurl, apikey, secret string, verifyssl bool, options ...cloudstack.ClientOption) *cloudstack.CloudStackClient {
			return mockClient
		}
		cloud.NewClient = func(apiurl, apikey, secret string, verifyssl bool, options ...cloudstack.ClientOption) *cloudstack.CloudStackClient {
			return mockClient
		}
	})

	BeforeEach(func() {
		asp := &cloudstack.ListAccountsParams{}
		as.EXPECT().NewListAccountsParams().Return(asp).AnyTimes()
		as.EXPECT().ListAccounts(asp).Return(&cloudstack.ListAccountsResponse{Count: 1, Accounts: []*cloudstack.Account{{
			Id:   dummies.AccountID,
			Name: dummies.AccountName,
		}}}, nil).AnyTimes()

		fakeListParams := &cloudstack.ListUsersParams{}
		fakeUser := &cloudstack.User{
			Id:      dummies.UserID,
			Account: dummies.AccountName,
			Domain:  dummies.DomainName,
		}
		us.EXPECT().NewListUsersParams().Return(fakeListParams).AnyTimes()
		us.EXPECT().ListUsers(fakeListParams).Return(&cloudstack.ListUsersResponse{
			Count: 1, Users: []*cloudstack.User{fakeUser},
		}, nil).AnyTimes()

		ukp := &cloudstack.GetUserKeysParams{}
		us.EXPECT().NewGetUserKeysParams(gomock.Any()).Return(ukp).AnyTimes()
		us.EXPECT().GetUserKeys(ukp).Return(&cloudstack.GetUserKeysResponse{
			Apikey:    dummies.Apikey,
			Secretkey: dummies.SecretKey,
		}, nil).AnyTimes()

		dsp := &cloudstack.ListDomainsParams{}
		ds.EXPECT().NewListDomainsParams().Return(dsp).AnyTimes()
		ds.EXPECT().ListDomains(dsp).Return(&cloudstack.ListDomainsResponse{Count: 1, Domains: []*cloudstack.Domain{{
			Id:   dummies.DomainID,
			Name: dummies.DomainName,
			Path: dummies.DomainPath,
		}}}, nil).AnyTimes()
	})

	Context("base factory", func() {
		It("create cloudstack client and user", func() {
			k8sClient = fake.NewClientBuilder().
				WithObjects(endpointCredentials, clientConfig).
				Build()

			fdSpec := &infrav1.CloudStackFailureDomainSpec{Name: "zone-a", ACSEndpoint: corev1.SecretReference{
				Name:      endpointCredentials.Name,
				Namespace: endpointCredentials.Namespace,
			}}

			factory := newBaseClientFactory(k8sClient)

			csClient, csUser, err := factory.GetCloudClientAndUser(ctx, fdSpec)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(csClient).ShouldNot(BeNil())
			Expect(csUser).ShouldNot(BeNil())
		})

		It("create cloudstack client and user for a specific account in the domain", func() {
			k8sClient = fake.NewClientBuilder().
				WithObjects(endpointCredentials, clientConfig).
				Build()

			fdSpec := &infrav1.CloudStackFailureDomainSpec{Name: "zone-a", Account: dummies.AccountName, ACSEndpoint: corev1.SecretReference{
				Name:      endpointCredentials.Name,
				Namespace: endpointCredentials.Namespace,
			}}
			factory := newBaseClientFactory(k8sClient)

			csClient, csUser, err := factory.GetCloudClientAndUser(ctx, fdSpec)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(csClient).ShouldNot(BeNil())
			Expect(csUser).ShouldNot(BeNil())
		})
	})

})
