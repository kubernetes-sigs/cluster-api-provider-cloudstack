# CloudStack Permissions for CAPC

The account that CAPC runs under must minimally be a Domain Admin type account with a role offering the following permissions

* assignToLoadBalancerRule
* associateIpAddress
* createAffinityGroup
* createEgressFirewallRule
* createLoadBalancerRule
* createNetwork
* createTags
* deleteAffinityGroup
* deleteNetwork
* deleteTags
* deployVirtualMachine
* destroyVirtualMachine
* disassociateIpAddress
* getUserKeys
* listAccounts
* listAffinityGroups
* listDiskOfferings
* listDomains
* listLoadBalancerRuleInstances
* listLoadBalancerRules
* listNetworkOfferings
* listNetworks
* listPublicIpAddresses
* listServiceOfferings
* listSSHKeyPairs
* listTags
* listTemplates
* listUsers
* listVirtualMachines
* listVirtualMachinesMetrics
* listVolumes
* listZones
* queryAsyncJobResult
* startVirtualMachine
* stopVirtualMachine
* updateVMAffinityGroup

This permission set has been verified to successfully run the CAPC E2E test suite.