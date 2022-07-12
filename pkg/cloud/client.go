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
	"encoding/json"

	"github.com/apache/cloudstack-go/v2/cloudstack"
)

//go:generate mockgen -destination=../mocks/mock_client.go -package=mocks sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud Client

type Client interface {
	VMIface
	NetworkIface
	AffinityGroupIface
	TagIface
	ZoneIFace
	IsoNetworkIface
	UserCredIFace
}

type client struct {
	cs      *cloudstack.CloudStackClient
	csAsync *cloudstack.CloudStackClient
	config  Config
}

// cloud-config ini structure.
type Config struct {
	APIURL    string `json:"api-url"`
	APIKey    string `json:"api-key"`
	SecretKey string `json:"secret-key"`
	VerifySSL bool   `json:"verify-ssl"`
}

// Creates a new Cloud Client form a map of strings to strings.
func NewClientFromMap(rawCfg map[string]string) (Client, error) {
	cfg := Config{VerifySSL: true} // Set sane defautl for verify-ssl.
	// Use JSON methods to enforce schema in parsing.
	if bytes, err := json.Marshal(rawCfg); err != nil {
		return nil, err
	} else if err := json.Unmarshal(bytes, &cfg); err != nil {
		return nil, err
	}

	// The client returned from NewAsyncClient works in a synchronous way. On the other hand,
	// a client returned from NewClient works in an asynchronous way. Dive into the constructor definition
	// comments for more details
	c := &client{config: cfg}
	c.cs = cloudstack.NewAsyncClient(cfg.APIURL, cfg.APIKey, cfg.SecretKey, cfg.VerifySSL)
	c.csAsync = cloudstack.NewClient(cfg.APIURL, cfg.APIKey, cfg.SecretKey, cfg.VerifySSL)
	return c, nil
}
