/*
Copyright 2022.

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

package cloud

import (
	"strings"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/pkg/errors"
	"gopkg.in/ini.v1"
)

//go:generate mockgen -destination=../mocks/mock_client.go -package=mocks github.com/aws/cluster-api-provider-cloudstack/pkg/cloud Client

type Client interface {
	ClusterIface
	VMIface
	ResolveNetwork(*infrav1.CloudStackCluster) error
	GetOrCreateNetwork(*infrav1.CloudStackCluster) error
	OpenFirewallRules(*infrav1.CloudStackCluster) error
	ResolvePublicIPDetails(*infrav1.CloudStackCluster) (*cloudstack.PublicIpAddress, error)
	ResolveLoadBalancerRuleDetails(*infrav1.CloudStackCluster) error
	GetOrCreateLoadBalancerRule(*infrav1.CloudStackCluster) error
	AffinityGroupIFace
}

type client struct {
	cs *cloudstack.CloudStackClient
	// This is a placeholder for sending non-blocking requests.
	// csA *cloudstack.CloudStackClient
}

// cloud-config ini structure.
type config struct {
	ApiUrl    string `ini:"api-url"`
	ApiKey    string `ini:"api-key"`
	SecretKey string `ini:"secret-key"`
	VerifySSL bool   `ini:"verify-ssl"`
}

func NewClient(cc_path string) (Client, error) {
	c := &client{}
	cfg := &config{VerifySSL: true}
	rawCfg, err := ini.Load(cc_path)
	if err != nil {
		return nil, errors.Wrapf(err, "Error encountered while reading config at path: %s", cc_path)
	}
	if err = rawCfg.Section("Global").StrictMapTo(cfg); err != nil {
		return nil, errors.Wrapf(err, "Error encountered while parsing [Global] section from config at path: %s", cc_path)
	}

	// This is a placeholder for sending non-blocking requests.
	// c.csA = cloudstack.NewClient(apiUrl, apiKey, secretKey, false)
	// TODO: attempt a less clunky client liveliness check (not just listing zones).
	c.cs = cloudstack.NewAsyncClient(cfg.ApiUrl, cfg.ApiKey, cfg.SecretKey, cfg.VerifySSL)
	_, err = c.cs.Zone.ListZones(c.cs.Zone.NewListZonesParams())
	if err != nil && strings.Contains(err.Error(), "i/o timeout") {
		return c, errors.Wrap(err, "Timeout while checking CloudStack API Client connectivity.")
	}
	return c, errors.Wrap(err, "Error encountered while checking CloudStack API Client connectivity.")
}

func NewClientFromCSAPIClient(cs *cloudstack.CloudStackClient) Client {
	c := &client{cs: cs}
	return c
}
