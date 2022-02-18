package test_dummies

import (
	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var ( // Declare shared vars.
	AffinityGroup *cloud.AffinityGroup
	Cluster       *infrav1.CloudStackCluster
	Machine       *infrav1.CloudStackMachine
	CapiMachine   *capiv1.Machine
	Client        cloud.Client
	TestZone1     infrav1.Zone
	TestNet1      infrav1.Network
)

// SetDummyClusterSpecVars resets the values in each of the exported dummy variables.
// It is intended to be called in BeforeEach( functions.
func SetDummyClusterSpecVars() {
	AffinityGroup = &cloud.AffinityGroup{
		Name: "FakeAffinityGroup",
		Type: cloud.AffinityGroupType}
	TestNet1 = infrav1.Network{Name: "SharedGuestNet1"}
	TestZone1 = infrav1.Zone{Name: "Zone1", Network: TestNet1}
	Cluster = &infrav1.CloudStackCluster{
		Spec: infrav1.CloudStackClusterSpec{
			Zones: []infrav1.Zone{TestZone1}}}
	Machine = &infrav1.CloudStackMachine{Spec: infrav1.CloudStackMachineSpec{
		Offering: "Medium Instance", Template: "Ubuntu20"}}
	Machine.ObjectMeta.SetName("rejoshed-affinity-group-test-vm")
	CapiMachine = &capiv1.Machine{}
}
