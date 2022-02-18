package test_dummies

import (
	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var ( // Declare exported dummy vars.
	AffinityGroup *cloud.AffinityGroup
	CSMachine     *infrav1.CloudStackMachine
	CSCluster     *infrav1.CloudStackCluster
	CAPIMachine   *capiv1.Machine
	CAPICluster   *clusterv1.Cluster
	Zone1         infrav1.Zone
	Net1          infrav1.Network
	DomainId      string
)

// SetDummyVars sets/resets all dummy vars.
func SetDummyVars() {
	SetDummyCAPCClusterVars()
	SetDummyCAPIClusterVars()
}

// SetDummyClusterSpecVars resets the values in each of the exported CloudStackCluster related dummy variables.
// It is intended to be called in BeforeEach( functions.
func SetDummyCAPCClusterVars() {
	AffinityGroup = &cloud.AffinityGroup{
		Name: "FakeAffinityGroup",
		Type: cloud.AffinityGroupType}
	Net1 = infrav1.Network{Name: "SharedGuestNet1"}
	Zone1 = infrav1.Zone{Name: "Zone1", Network: Net1}
	CSCluster = &infrav1.CloudStackCluster{
		Spec: infrav1.CloudStackClusterSpec{
			Zones: []infrav1.Zone{Zone1}},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "cs-cluster-test1-",
			UID:          "0",
			Namespace:    "default"},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
			Kind:       "CloudStackCluster"}}
	CSMachine = &infrav1.CloudStackMachine{
		Spec: infrav1.CloudStackMachineSpec{
			InstanceID: pointer.StringPtr("instance-id"),
			Offering:   "Medium Instance",
			Template:   "Ubuntu20"}}
	CSMachine.ObjectMeta.SetName("test-vm")
	CAPIMachine = &capiv1.Machine{}
	DomainId = "FakeDomainId"
}

// SetDummyCapiCluster resets the values in each of the exported CAPICluster related dummy variables.
func SetDummyCAPIClusterVars() {
	CAPICluster = &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "capi-cluster-test-",
			Namespace:    "default",
		},
		Spec: clusterv1.ClusterSpec{
			InfrastructureRef: &corev1.ObjectReference{
				APIVersion: infrav1.GroupVersion.String(),
				Kind:       "CloudStackCluster",
				Name:       "somename",
			},
		},
	}
}
