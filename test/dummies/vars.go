package dummies

import (
	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var ( // Declare exported dummy vars.
	AffinityGroup      *cloud.AffinityGroup
	CSCluster          *infrav1.CloudStackCluster
	CAPIMachine        *capiv1.Machine
	CSMachine1         *infrav1.CloudStackMachine
	CAPICluster        *clusterv1.Cluster
	CSMachineTemplate1 *infrav1.CloudStackMachineTemplate
	Zone1              infrav1.Zone
	Zone2              infrav1.Zone
	Net1               infrav1.Network
	Net2               infrav1.Network
	DomainID           string
	Tags               map[string]string
	Tag1Key            string
	Tag1Val            string
	CSApiVersion       string
	CSClusterKind      string
	CSClusterName      string
	CSlusterNamespace  string
	TestTags           map[string]string
	CSClusterTagKey    string
	CSClusterTagVal    string
	CSClusterTag       map[string]string
	CreatedByCapcKey   string
	CreatedByCapcVal   string
)

// SetDummyVars sets/resets tag related dummy vars.
func SetTestTags() {
	TestTags = map[string]string{"TestTagKey": "TestTagValue"}
	CSClusterTagKey = "CAPC_cluster_" + string(CSCluster.ObjectMeta.UID)
	CSClusterTagVal = "1"
	CSClusterTag = map[string]string{CSClusterTagVal: CSClusterTagVal}
	CreatedByCapcKey = "create_by_CAPC"
	CreatedByCapcVal = ""
}

// SetDummyVars sets/resets all dummy vars.
func SetDummyVars() {
	// These need to be in order as they build upon eachother.
	SetDummyCAPCClusterVars()
	SetDummyCSMachineVars()
	SetDummyCAPIClusterVars()
	SetDummyTagVars()
}

// SetDummyClusterSpecVars resets the values in each of the exported CloudStackMachines related dummy variables.
func SetDummyCSMachineTemplateVars() {
	CSMachineTemplate1 = &infrav1.CloudStackMachineTemplate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
			Kind:       "CloudStackMachineTemplate",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machinetemplate-1",
			Namespace: "default",
		},
		Spec: infrav1.CloudStackMachineTemplateSpec{
			Spec: infrav1.CloudStackMachineTemplateResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-machinetemplateresource",
					Namespace: "default",
				},
				Spec: infrav1.CloudStackMachineSpec{
					IdentityRef: &infrav1.CloudStackIdentityReference{
						Kind: "Secret",
						Name: "IdentitySecret",
					},
					Template: "Template",
					Offering: "Offering",
					Details: map[string]string{
						"memoryOvercommitRatio": "1.2",
					},
				},
			},
		},
	}
}

// SetDummyClusterSpecVars resets the values in each of the exported CloudStackMachines related dummy variables.
func SetDummyCSMachineVars() {
	CSMachine1 = &infrav1.CloudStackMachine{
		TypeMeta: metav1.TypeMeta{
			APIVersion: CSApiVersion,
			Kind:       "CloudStackMachine",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine-2",
			Namespace: "default",
		},
		Spec: infrav1.CloudStackMachineSpec{
			IdentityRef: &infrav1.CloudStackIdentityReference{
				Kind: "Secret",
				Name: "IdentitySecret",
			},
			Template:         "Template",
			Offering:         "Offering",
			AffinityGroupIDs: []string{"41eeb6e4-946f-4a18-b543-b2184815f1e4"},
			Details: map[string]string{
				"memoryOvercommitRatio": "1.2",
			},
		},
	}
	CSMachine1.ObjectMeta.SetName("test-vm")
}

// SetDummyClusterSpecVars resets the values in each of the exported CloudStackCluster related dummy variables.
// It is intended to be called in BeforeEach( functions.
func SetDummyCAPCClusterVars() {
	CSApiVersion = "infrastructure.cluster.x-k8s.io/v1beta1"
	CSClusterKind = "CloudStackCluster"
	CSClusterName = "test-cluster"
	CSlusterNamespace = "default"
	AffinityGroup = &cloud.AffinityGroup{
		Name: "FakeAffinityGroup",
		Type: cloud.AffinityGroupType}
	Net1 = infrav1.Network{Name: "SharedGuestNet1"}
	Zone1 = infrav1.Zone{Name: "Zone1", Network: Net1}
	Zone2 = infrav1.Zone{Name: "Zone2", Network: Net2}
	CSCluster = &infrav1.CloudStackCluster{
		Spec: infrav1.CloudStackClusterSpec{
			IdentityRef: &infrav1.CloudStackIdentityReference{
				Kind: "Secret",
				Name: "IdentitySecret",
			},
			Zones: []infrav1.Zone{Zone1, Zone2}},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "cs-cluster-test1-",
			UID:          "0",
			Namespace:    "default"},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
			Kind:       "CloudStackCluster"}}
	CAPIMachine = &capiv1.Machine{}
	DomainID = "FakeDomainId"
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

// SetDummyTagVars resets the values in each of the exported Tag related dummy variables.
func SetDummyTagVars() {
	Tag1Key = "test_tag"
	Tag1Val = "arbitrary_value"
	Tags = map[string]string{Tag1Key: Tag1Val}
}
