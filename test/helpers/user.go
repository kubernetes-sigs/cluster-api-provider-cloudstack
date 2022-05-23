package helpers

import (
	"fmt"
	"strings"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
)

const tempUserName = "TemporaryUser"

// GetDomainByPath fetches a domain by its path.
func GetDomainByPath(csClient *cloudstack.CloudStackClient, path string) (string, error, bool) {
	// Split path and get name.
	path = strings.Trim(path, "/")
	tokens := strings.Split(path, "/")

	// Ensure the path begins with ROOT.
	if !strings.EqualFold(tokens[0], "ROOT") {
		tokens = append([]string{"ROOT"}, tokens...)
	} else {
		tokens[0] = "ROOT"
	}
	path = strings.Join(tokens, "/")

	// Set present search/list parameters.
	p := csClient.Domain.NewListDomainsParams()
	p.SetListall(true)

	// If path was provided also use level narrow the search for domain.
	if level := len(tokens) - 1; level >= 0 {
		p.SetLevel(level)
	}

	if resp, err := csClient.Domain.ListDomains(p); err != nil {
		return "", err, false
	} else {
		for _, domain := range resp.Domains {
			if domain.Path == path {
				return domain.Id, nil, true
			}
		}
	}

	return "", nil, false
}

// CreateDomainUnderParent creates a domain as a sub-domain of the passed parent.
func CreateDomainUnderParent(csClient *cloudstack.CloudStackClient, parentID string, domainName string) (string, error) {
	p := csClient.Domain.NewCreateDomainParams(domainName)
	p.SetParentdomainid(parentID)
	resp, err := csClient.Domain.CreateDomain(p)
	if err != nil {
		return "", err
	}
	return resp.Id, nil
}

// GetOrCreateDomain gets or creates a domain as specified in the passed domain object.
func GetOrCreateDomain(csClient *cloudstack.CloudStackClient, domain *cloud.Domain) error {
	// Split the specified domain path and prepend ROOT/ if it's missing.
	domain.Path = strings.Trim(domain.Path, "/")
	tokens := strings.Split(domain.Path, "/")
	if strings.EqualFold(tokens[0], "root") {
		tokens[0] = "ROOT"
	} else {
		tokens = append([]string{"ROOT"}, tokens...)
	}
	domain.Path = strings.Join(tokens, "/")

	// Fetch ROOT domain ID.
	rootID, err, _ := GetDomainByPath(csClient, "ROOT")
	if err != nil {
		return err
	}

	// Iteratively create the domain from its path.
	parentID := rootID
	currPath := "ROOT"
	for _, nextDomainName := range tokens[1:] {
		currPath = currPath + "/" + nextDomainName
		if nextId, err, found := GetDomainByPath(csClient, currPath); err != nil {
			return err
		} else if !found {
			if nextId, err := CreateDomainUnderParent(csClient, parentID, nextDomainName); err != nil {
				return err
			} else {
				parentID = nextId
			}
		} else {
			parentID = nextId
		}
	}
	domain.ID = parentID
	domain.Name = tokens[len(tokens)-1]
	domain.Path = strings.Join(tokens, "/")
	return nil
}

// DeleteDomain deletes a domain by ID.
func DeleteDomain(csClient *cloudstack.CloudStackClient, domainID string) error {
	p := csClient.Domain.NewDeleteDomainParams(domainID)
	p.SetCleanup(true)
	resp, err := csClient.Domain.DeleteDomain(p)
	if !resp.Success {
		return fmt.Errorf("unsuccessful deletion of domain with ID %s", domainID)
	}
	return err
}

// GetOrCreateAccount creates a domain as specified in the passed account object.
func GetOrCreateAccount(csClient *cloudstack.CloudStackClient, account *cloud.Account) error {
	if err := GetOrCreateDomain(csClient, &account.Domain); err != nil {
		return err
	}

	// Attempt to fetch account.
	if resp, count, err := csClient.Account.GetAccountByName(
		account.Name, cloudstack.WithDomain(account.Domain.ID)); err != nil && !strings.Contains(err.Error(), "No match found") {
		return err
	} else if count > 1 {
		return fmt.Errorf("expected exactly 1 account, but got %d", count)
	} else if count == 1 {
		account.ID = resp.Id
		return nil
	} // Account not found, do account creation.

	// Get role for account creation.
	roleDetails, count, err := csClient.Role.GetRoleByName("Domain Admin")
	if err != nil {
		return err
	} else if count != 1 {
		return fmt.Errorf("expected exactly one role with name 'Domain Admin', found %d", count)
	}

	p := csClient.Account.NewCreateAccountParams("blah@someDomain.net", "first", "last", "temp123", tempUserName)
	p.SetDomainid(account.Domain.ID)
	p.SetRoleid(roleDetails.Id)
	resp, err := csClient.Account.CreateAccount(p)
	if err != nil {
		return err
	}
	account.Name = resp.Name
	account.ID = resp.Id

	return nil
}

// GetOrCreateUserWithKey creates a domain as specified in the passed account object.
// Right now only works with a default TemporaryUser name. This function was only built to get a testing user built.
func GetOrCreateUserWithKey(csClient *cloudstack.CloudStackClient, user *cloud.User) error {
	if err := GetOrCreateAccount(csClient, &user.Account); err != nil {
		return err
	}

	p := csClient.User.NewListUsersParams()
	p.SetAccount(user.Account.Name)
	p.SetDomainid(user.Account.Domain.ID)
	if resp, err := csClient.User.ListUsers(p); err != nil {
		return err
	} else if resp.Count > 1 {
		return fmt.Errorf("expected exactly one User with name %s, found %d", user.Name, resp.Count)
	} else if resp.Count == 1 {
		user.ID = resp.Users[0].Id
	} else { // User not found, create user.
		// TODO: If ever needed, actually implement user creation here.
		// For now we only care about the default account since this is a testing infrastructure method.
		return fmt.Errorf("User not found for %s", user.Name)
	}

	pGKey := csClient.User.NewGetUserKeysParams(user.ID)
	if resp, err := csClient.User.GetUserKeys(pGKey); err != nil {
		return err
	} else if user.APIKey != "" {
		user.APIKey = resp.Apikey
		user.SecretKey = resp.Secretkey
		return nil
	}

	pKey := csClient.User.NewRegisterUserKeysParams(user.ID)
	if resp, err := csClient.User.RegisterUserKeys(pKey); err != nil {
		return err
	} else {
		user.APIKey = resp.Apikey
		user.SecretKey = resp.Secretkey
	}

	return nil
}
