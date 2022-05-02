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
)

type UserCredIFace interface {
	FindDomain(string) (*string, error)
	GetOrCreateDomain(*Domain) error
	GetOrCreateAccount(*Account) error
	DeleteDomain(*Domain) error
	DeleteAccount(*Account) error
	GetOrCreateUser(*User) (*cloudstack.GetUserResponse, error)
	// GetUser(string, string) (*cloudstack.GetUserResponse, error)
	GetUserKeys(*User) error
	GetOrCreateUserKeys(*User) error
}

// Domain contains specifications that identify a domain.
type Domain struct {
	Name string
	Path string
	Id   string
}

// Account contains specifications that identify an account.
type Account struct {
	Name   string
	Domain Domain
	Id     string
}

// Scope details the scope a user is limited to. (Account and Domain)
type Scope struct {
	Account Account
	Domain  Domain
}

// ResolveScope queries CloudStack to fill in scope fields.
func (c *client) ResolveScope() error {
	return nil
}

func (c *client) GetDomain(domain *Domain) error {
	// Parse name from path if name is missing.
	tokens := strings.Split(domain.Path, "/")
	if domain.Name == "" {
		domain.Name = tokens[len(tokens)-1]
	}

	// List Domains.
	p := c.cs.Domain.NewListDomainsParams()
	p.SetListall(true)
	p.SetName(domain.Name)
	resp, retErr := c.cs.Domain.ListDomains(p)
	if retErr != nil {
		return errors.Wrapf(retErr, "error encountered while listing domains in attempt to find domain %s", domain.Name)
	}

	// Prepend ROOT if missing from path.
	if strings.ToUpper(tokens[0]) != rootDomain {
		tokens = append([]string{rootDomain}, tokens...)
	}

	// Search list for desired domain.
	domain.Path = strings.Join([]string{rootDomain, domain.Path}, domainDelimiter)
	for _, foundDomain := range resp.Domains {
		if domain.Path == foundDomain.Path {
			domain.Id = foundDomain.Id
			domain.Name = foundDomain.Name
			return nil
		}
	}
	return errors.Errorf("domain not found for domain path %s", domain)
}

func (c *client) GetOrCreateDomain(domain *Domain) error {
	// Attempt get.
	if err := c.GetDomain(domain); err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return err
		}
	}
	// Not found, attempt create.
	p := c.cs.Domain.NewCreateDomainParams(domain.Name)
	p.SetParentdomainid()
	//c.cs.Domain.

	return nil
}

func (c *client) GetOrCreateAccount(domain *Domain) error {
	return nil
}
func (c *client) GetAccount(domain *Domain) error {
	return nil
}
func (c *client) GetOrCreateUser(domain *Domain) error {
	return nil
}
func (c *client) GetUser(domain *Domain) error {
	return nil
}
func (c *client) GetAnyDomainUser(domain *Domain) error {
	return nil
}
func (c *client) GetDomainUsers(domain *Domain) error {
	return nil
}
func (c *client) CreateDomainCAPCUser(domain *Domain) error {
	return nil
}
func (c *client) GetDomainCAPCUser(domain *Domain) error {
	return nil
}

type User struct {
	Id        string
	ApiKey    string
	ApiSecret string
	Scope
}

// func (c *client) GetUser(account string, domainID string, user *User) (*cloudstack.GetUserResponse, error) {
// 	// p := c.cs.User.NewListUsersParams()
// 	// p.SetAccount(account)
// 	// p.SetDomainid(domainID)
// 	// p.SetListall(true)
// 	// resp, err := c.cs.User.ListUsers(p)
// 	// if err != nil {
// 	// 	return nil, err
// 	// } else if resp.Count != 1 {
// 	// 	return nil, errors.Errorf("expected 1 Account with account name %s in domain ID %s, but got %d",
// 	// 		account, domainID, resp.Count)
// 	// }

// 	// p2 := c.cs.User.NewGetUserParams(resp.Users[0].Apikey)
// 	// user.Id, err := c.cs.User.GetUser(p2)
// 	// if err != nil {
// 	// 	return nil, err
// 	// }

// 	return nil, nil
// }

// func (c *client) GetDomainAndAccount(scope *Scope) {
// 	c.CS
// }

// // GetOrCreateUserKeys gets user's keys from CloudStack or creates them and fills the relevant fields.
// func (c *client) GetOrCreateUserKeys(user *User) error {

// 	// Return if User key found on first try.
// 	if err := c.GetUserKeys(user); err == nil && user.apiKey != "" {
// 		return nil
// 	}

// 	// Create keys instead.
// 	p := c.cs.User.NewRegisterUserKeysParams(user.id)
// 	resp, err := c.cs.User.RegisterUserKeys(p)
// 	if err != nil {
// 		return errors.Wrap(err, "error encountered while registering user keys")
// 	}
// 	user.apiKey = resp.Apikey
// 	user.apiSecret = resp.Secretkey
// 	return nil
// }

// // GetOrCreateUserKeys gets user's keys from CloudStack and fills the relevant fields.
// func (c *client) GetUserKeys(user *User) error {
// 	p := c.cs.User.NewGetUserKeysParams(user.id)
// 	resp, err := c.cs.User.GetUserKeys(p)
// 	if err == nil && resp != nil {
// 		user.apiKey = resp.Apikey
// 		user.apiSecret = resp.Secretkey
// 	}
// 	return err
// }

func (c *client) FindDomain(domain string) (*string, error) {
	p := c.cs.Domain.NewListDomainsParams()
	tokens := strings.Split(domain, "/")
	domainName := tokens[len(tokens)-1]
	p.SetListall(true)
	p.SetName(domainName)

	resp, retErr := c.cs.Domain.ListDomains(p)
	if retErr != nil {
		return nil, retErr
	}

	var domainPath string
	if domain == rootDomain {
		domainPath = rootDomain
	} else {
		domainPath = strings.Join([]string{rootDomain, domain}, "/")
	}
	for _, domain := range resp.Domains {
		if domain.Path == domainPath {
			return &domain.Id, nil
		}
	}
	return nil, errors.Errorf("domain not found for domain path %s", domain)
}
