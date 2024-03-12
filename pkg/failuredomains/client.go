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

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClientFactory interface {
	GetCloudClientAndUser(ctx context.Context, fdSpec *infrav1.CloudStackFailureDomainSpec) (csClient cloud.Client, csUser cloud.Client, err error)
}

func NewClientFactory(k8sClient client.Client) ClientFactory {
	return newBaseClientFactory(k8sClient)
}

type baseClientFactory struct {
	client.Client
}

func newBaseClientFactory(k8sClient client.Client) ClientFactory {
	return &baseClientFactory{k8sClient}
}

func (f *baseClientFactory) GetCloudClientAndUser(ctx context.Context, fdSpec *infrav1.CloudStackFailureDomainSpec) (csClient cloud.Client, csUser cloud.Client, err error) {
	endpointCredentials := &corev1.Secret{}
	key := client.ObjectKey{Name: fdSpec.ACSEndpoint.Name, Namespace: fdSpec.ACSEndpoint.Namespace}
	if err := f.Get(ctx, key, endpointCredentials); err != nil {
		return nil, nil, errors.Wrapf(err, "getting ACSEndpoint secret with ref: %v", fdSpec.ACSEndpoint)
	}

	clientConfig := &corev1.ConfigMap{}
	key = client.ObjectKey{Name: cloud.ClientConfigMapName, Namespace: cloud.ClientConfigMapNamespace}
	_ = f.Get(ctx, key, clientConfig)

	csClient, err = cloud.NewClientFromK8sSecret(endpointCredentials, clientConfig)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "parsing ACSEndpoint secret with ref: %v", fdSpec.ACSEndpoint)
	}

	if fdSpec.Account != "" { // Get CloudStack Client per Account and Domain.
		csClientInDomain, err := csClient.NewClientInDomainAndAccount(fdSpec.Domain, fdSpec.Account)
		if err != nil {
			return nil, nil, err
		}
		csUser = csClientInDomain
	} else { // Set r.CSUser CloudStack Client to r.CSClient since Account & Domain weren't provided.
		csUser = csClient
	}

	return csClient, csUser, nil
}
