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
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"

	"gopkg.in/yaml.v3"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/metrics"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/jellydator/ttlcache/v3"
	"github.com/pkg/errors"
)

//go:generate ../../hack/tools/bin/mockgen -destination=../mocks/mock_client.go -package=mocks sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud Client

type Client interface {
	ClusterIface
	VMIface
	NetworkIface
	AffinityGroupIface
	TagIface
	ZoneIFace
	IsoNetworkIface
	UserCredIFace
	VPCIface
	NewClientInDomainAndAccount(string, string, string) (Client, error)
}

// cloud-config ini structure.
type Config struct {
	APIUrl    string `yaml:"api-url"`
	APIKey    string `yaml:"api-key"`
	SecretKey string `yaml:"secret-key"`
	VerifySSL string `yaml:"verify-ssl"`
}

type client struct {
	cs            *cloudstack.CloudStackClient
	csAsync       *cloudstack.CloudStackClient
	config        Config
	user          *User
	customMetrics metrics.ACSCustomMetrics
}

type SecretConfig struct {
	APIVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Type       string            `yaml:"type"`
	Metadata   map[string]string `yaml:"metadata"`
	StringData Config            `yaml:"stringData"`
}

var clientCache *ttlcache.Cache[string, *client]
var cacheMutex sync.Mutex

var NewAsyncClient = cloudstack.NewAsyncClient
var NewClient = cloudstack.NewClient

const ClientConfigMapName = "capc-client-config"
const ClientConfigMapNamespace = "capc-system"
const ClientCacheTTLKey = "client-cache-ttl"
const DefaultClientCacheTTL = time.Duration(1 * time.Hour)

// UnmarshalAllSecretConfigs parses a yaml document for each secret.
func UnmarshalAllSecretConfigs(in []byte, out *[]SecretConfig) error {
	r := bytes.NewReader(in)
	decoder := yaml.NewDecoder(r)
	for {
		var conf SecretConfig
		if err := decoder.Decode(&conf); err != nil {
			// Break when there are no more documents to decode
			if err != io.EOF {
				return err
			}
			break
		}
		*out = append(*out, conf)
	}
	return nil
}

// NewClientFromK8sSecret returns a client from a k8s secret
func NewClientFromK8sSecret(endpointSecret *corev1.Secret, clientConfig *corev1.ConfigMap, project string) (Client, error) {
	endpointSecretStrings := map[string]string{}
	for k, v := range endpointSecret.Data {
		endpointSecretStrings[k] = string(v)
	}
	bytes, err := yaml.Marshal(endpointSecretStrings)
	if err != nil {
		return nil, err
	}
	return NewClientFromBytesConfig(bytes, clientConfig, project)
}

// NewClientFromBytesConfig returns a client from a bytes array that unmarshals to a yaml config.
func NewClientFromBytesConfig(conf []byte, clientConfig *corev1.ConfigMap, project string) (Client, error) {
	r := bytes.NewReader(conf)
	dec := yaml.NewDecoder(r)
	var config Config
	if err := dec.Decode(&config); err != nil {
		return nil, err
	}

	return NewClientFromConf(config, clientConfig, project)
}

// NewClientFromYamlPath returns a client from a yaml config at path.
func NewClientFromYamlPath(confPath string, secretName string) (Client, error) {
	content, err := os.ReadFile(confPath)
	if err != nil {
		return nil, err
	}
	configs := &[]SecretConfig{}
	if err := UnmarshalAllSecretConfigs(content, configs); err != nil {
		return nil, err
	}
	var conf Config
	for _, config := range *configs {
		if config.Metadata["name"] == secretName {
			conf = config.StringData
			break
		}
	}
	if conf.APIKey == "" {
		return nil, errors.Errorf("config with secret name %s not found", secretName)
	}

	return NewClientFromConf(conf, nil, "")
}

// NewClientFromConf creates a new Cloud Client form a map of strings to strings.
func NewClientFromConf(conf Config, clientConfig *corev1.ConfigMap, project string) (Client, error) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	if clientCache == nil {
		clientCache = newClientCache(clientConfig)
	}

	clientCacheKey := generateClientCacheKey(conf, project)
	if item := clientCache.Get(clientCacheKey); item != nil {
		return item.Value(), nil
	}

	verifySSL := true
	if conf.VerifySSL == "false" {
		verifySSL = false
	}

	// The client returned from NewAsyncClient works in a synchronous way. On the other hand,
	// a client returned from NewClient works in an asynchronous way. Dive into the constructor definition
	// comments for more details
	c := &client{config: conf}
	c.cs = NewClient(conf.APIUrl, conf.APIKey, conf.SecretKey, verifySSL)
	c.csAsync = NewAsyncClient(conf.APIUrl, conf.APIKey, conf.SecretKey, verifySSL)
	c.customMetrics = metrics.NewCustomMetrics()

	p := c.cs.User.NewListUsersParams()
	userResponse, err := c.cs.User.ListUsers(p)
	if err != nil {
		return c, err
	}
	user := &User{
		ID: userResponse.Users[0].Id,
		Account: Account{
			Name: userResponse.Users[0].Account,
			Domain: Domain{
				ID: userResponse.Users[0].Domainid,
			},
		},
		Project: Project{
			Name: project,
		},
	}
	if found, err := c.GetUserWithKeys(user); err != nil {
		return nil, err
	} else if !found {
		return nil, errors.Errorf(
			"could not find sufficient user (with API keys) in domain/account %s/%s", userResponse.Users[0].Domain, userResponse.Users[0].Account)
	}
	c.user = user
	clientCache.Set(clientCacheKey, c, ttlcache.DefaultTTL)

	return c, nil
}

// NewClientInDomainAndAccount returns a new client in the specified domain and account.
func (c *client) NewClientInDomainAndAccount(domain string, account string, project string) (Client, error) {
	user := &User{}
	user.Account.Domain.Path = domain
	user.Account.Name = account
	user.Project.Name = project
	if found, err := c.GetUserWithKeys(user); err != nil {
		return nil, err
	} else if !found {
		return nil, errors.Errorf(
			"could not find sufficient user (with API keys) in domain/account %s/%s", domain, account)
	}
	c.config.APIKey = user.APIKey
	c.config.SecretKey = user.SecretKey
	c.user = user

	return NewClientFromConf(c.config, nil, project)
}

// NewClientFromCSAPIClient creates a client from a CloudStack-Go API client. Used only for testing.
func NewClientFromCSAPIClient(cs *cloudstack.CloudStackClient, user *User) Client {
	if user == nil {
		user = &User{
			Account: Account{
				Domain: Domain{
					CPUAvailable:    "Unlimited",
					MemoryAvailable: "Unlimited",
					VMAvailable:     "Unlimited",
				},
				CPUAvailable:    "Unlimited",
				MemoryAvailable: "Unlimited",
				VMAvailable:     "Unlimited",
			},
		}
	}
	c := &client{
		cs:            cs,
		csAsync:       cs,
		customMetrics: metrics.NewCustomMetrics(),
		user:          user,
	}
	return c
}

// generateClientCacheKey generates a cache key from a Config
func generateClientCacheKey(conf Config, project string) string {
	return fmt.Sprintf("%s-%+v", project, conf)
}

// newClientCache returns a new instance of client cache
func newClientCache(clientConfig *corev1.ConfigMap) *ttlcache.Cache[string, *client] {
	cache := ttlcache.New[string, *client](
		ttlcache.WithTTL[string, *client](GetClientCacheTTL(clientConfig)),
		ttlcache.WithDisableTouchOnHit[string, *client](),
	)

	go cache.Start() // starts automatic expired item deletion

	return cache
}

// GetClientCacheTTL returns a client cache TTL duration from the passed config map
func GetClientCacheTTL(clientConfig *corev1.ConfigMap) time.Duration {
	var cacheTTL time.Duration
	if clientConfig != nil {
		if ttl, exists := clientConfig.Data[ClientCacheTTLKey]; exists {
			cacheTTL, _ = time.ParseDuration(ttl)
		}
	}
	if cacheTTL == 0 {
		cacheTTL = DefaultClientCacheTTL
	}
	return cacheTTL
}
