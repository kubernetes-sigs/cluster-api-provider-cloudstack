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

//go:generate ../../hack/tools/bin/mockgen -destination=../mocks/mock_client.go -package=mocks sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud Client

const GLOBAL = "Global"

type Client interface {
	VMIface
	NetworkIface
	AffinityGroupIface
	TagIface
	ZoneIFace
	IsoNetworkIface
	UserCredIFace
	NewClientFromSpec(Config) (Client, error)
}

type client struct {
	cs      *cloudstack.CloudStackClient
	csAsync *cloudstack.CloudStackClient
	config  Config
}

// cloud-config ini structure.
type Config struct {
	APIURL    string `ini:"api-url"`
	APIKey    string `ini:"api-key"`
	SecretKey string `ini:"secret-key"`
	VerifySSL bool   `ini:"verify-ssl"`
}

func NewClient(ccPath string) (Client, error) {
	c := &client{config: Config{VerifySSL: true}}
	if rawCfg, err := ini.Load(ccPath); err != nil {
		return nil, errors.Wrapf(err, "reading config at path %s", ccPath)
	} else if g := rawCfg.Section(GLOBAL); len(g.Keys()) == 0 {
		return nil, errors.New("section Global not found")
	} else if err = rawCfg.Section(GLOBAL).StrictMapTo(&c.config); err != nil {
		return nil, errors.Wrapf(err, "parsing [Global] section from config at path %s", ccPath)
	}

	// The client returned from NewAsyncClient works in a synchronous way. On the other hand,
	// a client returned from NewClient works in an asynchronous way. Dive into the constructor definition
	// comments for more details
	c.cs = cloudstack.NewAsyncClient(c.config.APIURL, c.config.APIKey, c.config.SecretKey, c.config.VerifySSL)
	c.csAsync = cloudstack.NewClient(c.config.APIURL, c.config.APIKey, c.config.SecretKey, c.config.VerifySSL)

	_, err := c.cs.APIDiscovery.ListApis(c.cs.APIDiscovery.NewListApisParams())
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "i/o timeout") {
		return c, errors.Wrap(err, "timeout while checking CloudStack API Client connectivity")
	}
	return c, errors.Wrap(err, "checking CloudStack API Client connectivity")
}

// NewClientFromSpec generates a new client from an existing client.
// Unless the passed config contains a new API URL the original one will be used.
// VerifySSL will be set to true if either the old or new configs is true.
func (origC *client) NewClientFromSpec(cfg Config) (Client, error) {
	newC := &client{config: cfg}
	newC.config.VerifySSL = cfg.VerifySSL || origC.config.VerifySSL // Prefer the most secure setting given.
	if newC.config.APIURL == "" {
		newC.config.APIURL = origC.config.APIURL
	}

	// The client returned from NewAsyncClient works in a synchronous way. On the other hand,
	// a client returned from NewClient works in an asynchronous way. Dive into the constructor definition
	// comments for more details
	newC.cs = cloudstack.NewAsyncClient(newC.config.APIURL, newC.config.APIKey, newC.config.SecretKey, newC.config.VerifySSL)
	newC.csAsync = cloudstack.NewClient(newC.config.APIURL, newC.config.APIKey, newC.config.SecretKey, newC.config.VerifySSL)

	_, err := newC.cs.APIDiscovery.ListApis(newC.cs.APIDiscovery.NewListApisParams())
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "i/o timeout") {
		return newC, errors.Wrap(err, "timeout while checking CloudStack API Client connectivity")
	}
	return newC, errors.Wrap(err, "checking CloudStack API Client connectivity")
}

func NewClientFromCSAPIClient(cs *cloudstack.CloudStackClient) Client {
	c := &client{cs: cs, csAsync: cs}
	return c
}
