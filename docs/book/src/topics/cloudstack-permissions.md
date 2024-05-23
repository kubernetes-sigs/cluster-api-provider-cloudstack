# CloudStack Permissions for CAPC

The account that CAPC runs under must minimally be a User type account with a role offering the following permissions

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

> Note: If the user doesn't have permissions to expunge the VM, it will be left in a destroyed state. The user will need to manually expunge the VM.

This permission set has been verified to successfully run the CAPC E2E test suite (Oct 11, 2022).