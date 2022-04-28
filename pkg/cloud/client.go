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

package cloud

import (
	"strings"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/pkg/errors"
	"gopkg.in/ini.v1"
)

//go:generate mockgen -destination=../mocks/mock_client.go -package=mocks github.com/aws/cluster-api-provider-cloudstack/pkg/cloud Client

type Client interface {
	ClusterIface
	VMIface
	NetworkIface
	AffinityGroupIface
	TagIface
	ZoneIFace
	IsoNetworkIface
}

type client struct {
	cs      *cloudstack.CloudStackClient
	csAsync *cloudstack.CloudStackClient
}

// cloud-config ini structure.
type config struct {
	APIURL    string `ini:"api-url"`
	APIKey    string `ini:"api-key"`
	SecretKey string `ini:"secret-key"`
	VerifySSL bool   `ini:"verify-ssl"`
}

func NewClient(ccPath string) (Client, error) {
	c := &client{}
	cfg := &config{VerifySSL: true}
	if rawCfg, err := ini.Load(ccPath); err != nil {
		return nil, errors.Wrapf(err, "error encountered while reading config at path: %s", ccPath)
	} else if g := rawCfg.Section("Global"); len(g.Keys()) == 0 {
		return nil, errors.New("section Global not found")
	} else if err = rawCfg.Section("Global").StrictMapTo(cfg); err != nil {
		return nil, errors.Wrapf(err, "error encountered while parsing [Global] section from config at path: %s", ccPath)
	}

	// The client returned from NewAsyncClient works in a synchronous way. On the other hand,
	// a client returned from NewClient works in an asynchronous way. Dive into the constructor definition
	// comments for more details
	c.cs = cloudstack.NewAsyncClient(cfg.APIURL, cfg.APIKey, cfg.SecretKey, cfg.VerifySSL)
	c.csAsync = cloudstack.NewClient(cfg.APIURL, cfg.APIKey, cfg.SecretKey, cfg.VerifySSL)
	
	_, err := c.cs.APIDiscovery.ListApis(c.cs.APIDiscovery.NewListApisParams())
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "i/o timeout") {
		return c, errors.Wrap(err, "timeout while checking CloudStack API Client connectivity")
	}
	return c, errors.Wrap(err, "error encountered while checking CloudStack API Client connectivity")
}

func NewClientFromCSAPIClient(cs *cloudstack.CloudStackClient) Client {
	c := &client{cs: cs, csAsync: cs}
	return c
}
