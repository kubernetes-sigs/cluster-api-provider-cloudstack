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
	cs  *cloudstack.CloudStackClient
	csA *cloudstack.CloudStackClient
}

func NewClient(cc_path string) (Client, error) {
	c := &client{}
	apiUrl, apiKey, secretKey, err := readAPIConfig(cc_path)
	if err != nil {
		return nil, errors.Wrapf(err, "Error encountered while reading config at path: %s", cc_path)
	}

	// TODO: attempt a less clunky client liveliness check (not just listing zones).
	c.csA = cloudstack.NewClient(apiUrl, apiKey, secretKey, false)
	c.cs = cloudstack.NewAsyncClient(apiUrl, apiKey, secretKey, false)
	_, err = c.cs.Zone.ListZones(c.cs.Zone.NewListZonesParams())
	if err != nil && strings.Contains(err.Error(), "i/o timeout") {
		return c, errors.Wrap(err, "Timeout while checking CloudStack API Client connectivity.")
	}
	return c, errors.Wrap(err, "Error encountered while checking CloudStack API Client connectivity.")
}

// CloudStack API config reader.
func readAPIConfig(cc_path string) (string, string, string, error) {
	cfg, err := ini.Load(cc_path)
	if err != nil {
		return "", "", "", err
	}
	g := cfg.Section("Global")
	if len(g.Keys()) == 0 {
		return "", "", "", errors.New("section Global not found")
	}
	return g.Key("api-url").Value(), g.Key("api-key").Value(), g.Key("secret-key").Value(), err
}

func NewClientFromCSAPIClient(cs *cloudstack.CloudStackClient) Client {
	c := &client{cs: cs}
	return c
}
