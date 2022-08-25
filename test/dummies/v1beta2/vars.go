package dummies

import (
	"io/ioutil"
	"os"

	csapi "github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/onsi/gomega"
	"github.com/smallfish/simpleyaml"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// GetYamlVal fetches the values in test/e2e/config/cloudstack.yaml by yaml node. A common config file.
func GetYamlVal(variable string) string {
	val, err := CSConf.Get("variables").Get(variable).String()
	gomega.Ω(err).ShouldNot(gomega.HaveOccurred())
	return val
}

var ( // Declare exported dummy vars.
	AffinityGroup       *cloud.AffinityGroup
	CSAffinityGroup     *infrav1.CloudStackAffinityGroup
	CSCluster           *infrav1.CloudStackCluster
	CAPIMachine         *clusterv1.Machine
	CSMachine1          *infrav1.CloudStackMachine
	CAPICluster         *clusterv1.Cluster
	ClusterLabel        map[string]string
	ClusterName         string
	ClusterNameSpace    string
	CSMachineTemplate1  *infrav1.CloudStackMachineTemplate
	ACSEndpointSecret1  *corev1.Secret
	ACSEndpointSecret2  *corev1.Secret
	Zone1               infrav1.CloudStackZoneSpec
	Zone2               infrav1.CloudStackZoneSpec
	CSFailureDomain1    *infrav1.CloudStackFailureDomain
	CSFailureDomain2    *infrav1.CloudStackFailureDomain
	Net1                infrav1.Network
	Net2                infrav1.Network
	ISONet1             infrav1.Network
	CSISONet1           *infrav1.CloudStackIsolatedNetwork
	Domain              cloud.Domain
	DomainPath          string
	DomainName          string
	DomainID            string
	Level2Domain        cloud.Domain
	Level2DomainPath    string
	Level2DomainName    string
	Level2DomainID      string
	Account             cloud.Account
	AccountName         string
	AccountID           string
	Level2Account       cloud.Account
	Level2AccountName   string
	Level2AccountID     string
	Tags                map[string]string
	Tag1                map[string]string
	Tag2                map[string]string
	Tag1Key             string
	Tag1Val             string
	Tag2Key             string
	Tag2Val             string
	CSApiVersion        string
	CSClusterKind       string
	TestTags            map[string]string
	CSClusterTagKey     string
	CSClusterTagVal     string
	CSClusterTag        map[string]string
	CreatedByCapcKey    string
	CreatedByCapcVal    string
	LBRuleID            string
	PublicIPID          string
	EndPointHost        string
	EndPointPort        int32
	CSConf              *simpleyaml.Yaml
	DiskOffering        infrav1.CloudStackResourceDiskOffering
	BootstrapSecret     *corev1.Secret
	BootstrapSecretName string
)

// SetDummyVars sets/resets all dummy vars.
func SetDummyVars() {
	projDir := os.Getenv("REPO_ROOT")
	source, err := ioutil.ReadFile(projDir + "/test/e2e/config/cloudstack.yaml")
	if err != nil {
		panic(err)
	}
	CSConf, err = simpleyaml.NewYaml(source)
	if err != nil {
		panic(err)
	}

	// These need to be in order as they build upon eachother.
	SetDummyZoneVars()
	SetDiskOfferingVars()
	SetACSEndpointSecretVars()
	SetDummyCAPCClusterVars()
	SetDummyCAPIClusterVars()
	SetDummyCAPIMachineVars()
	SetDummyCSMachineTemplateVars()
	SetDummyCSMachineVars()
	SetDummyTagVars()
	SetDummyBootstrapSecretVar()
	LBRuleID = "FakeLBRuleID"
}

func SetDiskOfferingVars() {
	DiskOffering = infrav1.CloudStackResourceDiskOffering{CloudStackResourceIdentifier: infrav1.CloudStackResourceIdentifier{Name: "Small"},
		MountPath:  "/data",
		Device:     "/dev/vdb",
		Filesystem: "ext4",
		Label:      "data_disk",
	}
}

func CAPCNetToCSAPINet(net *infrav1.Network) *csapi.Network {
	return &csapi.Network{
		Name: net.Name,
		Id:   net.ID,
		Type: net.Type,
	}
}

// SetDummyVars sets/resets tag related dummy vars.
func SetDummyTagVars() {
	CSClusterTagKey = "CAPC_cluster_" + string(CSCluster.ObjectMeta.UID)
	CSClusterTagVal = "1"
	CSClusterTag = map[string]string{CSClusterTagVal: CSClusterTagVal}
	CreatedByCapcKey = "create_by_CAPC"
	CreatedByCapcVal = ""
	Tag1Key = "test_tag1"
	Tag1Val = "arbitrary_value1"
	Tag2Key = "test_tag2"
	Tag2Val = "arbitrary_value2"
	Tag1 = map[string]string{Tag2Key: Tag2Val}
	Tag2 = map[string]string{Tag2Key: Tag2Val}
	Tags = map[string]string{Tag1Key: Tag1Val, Tag2Key: Tag2Val}
}

// SetDummyCSMachineTemplateVars resets the values in each of the exported CloudStackMachinesTemplate dummy variables.
func SetDummyCSMachineTemplateVars() {
	CSMachineTemplate1 = &infrav1.CloudStackMachineTemplate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta2",
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
					Template: infrav1.CloudStackResourceIdentifier{
						Name: GetYamlVal("CLOUDSTACK_TEMPLATE_NAME"),
					},
					Offering: infrav1.CloudStackResourceIdentifier{
						Name: GetYamlVal("CLOUDSTACK_CONTROL_PLANE_MACHINE_OFFERING"),
					},
					DiskOffering: DiskOffering,
					Details: map[string]string{
						"memoryOvercommitRatio": "1.2",
					},
				},
			},
		},
	}
}

// SetDummyCSMachineVars resets the values in each of the exported CloudStackMachine dummy variables.
func SetDummyCSMachineVars() {
	CSMachine1 = &infrav1.CloudStackMachine{
		TypeMeta: metav1.TypeMeta{
			APIVersion: CSApiVersion,
			Kind:       "CloudStackMachine",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine-1",
			Namespace: "default",
			Labels:    ClusterLabel,
		},
		Spec: infrav1.CloudStackMachineSpec{
			Name:       "test-machine-1",
			InstanceID: pointer.String("Instance1"),
			Template: infrav1.CloudStackResourceIdentifier{
				Name: GetYamlVal("CLOUDSTACK_TEMPLATE_NAME"),
			},
			Offering: infrav1.CloudStackResourceIdentifier{
				Name: GetYamlVal("CLOUDSTACK_CONTROL_PLANE_MACHINE_OFFERING"),
			},
			DiskOffering: infrav1.CloudStackResourceDiskOffering{
				CloudStackResourceIdentifier: infrav1.CloudStackResourceIdentifier{
					Name: "DiskOffering",
				},
				MountPath:  "/data",
				Device:     "/dev/vdb",
				Filesystem: "ext4",
				Label:      "data_disk",
			},
			Details: map[string]string{
				"memoryOvercommitRatio": "1.2",
			},
		},
	}
}

func SetDummyZoneVars() {
	Zone1 = infrav1.CloudStackZoneSpec{Network: Net1}
	Zone1.Name = GetYamlVal("CLOUDSTACK_ZONE_NAME")
	Zone2 = infrav1.CloudStackZoneSpec{Network: Net2}
	Zone2.Name = "Zone2"
	Zone2.ID = "FakeZone2ID"
}

// SetDummyCAPCClusterVars resets the values in each of the exported CloudStackCluster related dummy variables.
// It is intended to be called in BeforeEach() functions.
func SetDummyCAPCClusterVars() {
	DomainName = "FakeDomainName"
	DomainID = "FakeDomainID"
	Domain = cloud.Domain{Name: DomainName, ID: DomainID}
	Level2DomainName = "foo/FakeDomainName"
	Level2DomainID = "FakeLevel2DomainID"
	Level2Domain = cloud.Domain{Name: Level2DomainName, ID: Level2DomainID}
	AccountName = "FakeAccountName"
	Account = cloud.Account{Name: AccountName, Domain: Domain}
	AccountName = "FakeLevel2AccountName"
	Level2Account = cloud.Account{Name: Level2AccountName, Domain: Level2Domain}
	CSApiVersion = "infrastructure.cluster.x-k8s.io/v1beta2"
	CSClusterKind = "CloudStackCluster"
	ClusterName = "test-cluster"
	EndPointHost = "EndpointHost"
	EndPointPort = int32(5309)
	PublicIPID = "FakePublicIPID"
	ClusterNameSpace = "default"
	ClusterLabel = map[string]string{clusterv1.ClusterLabelName: ClusterName}
	AffinityGroup = &cloud.AffinityGroup{
		Name: "fakeaffinitygroup",
		Type: cloud.AffinityGroupType,
		ID:   "FakeAffinityGroupID"}
	Net1 = infrav1.Network{Name: GetYamlVal("CLOUDSTACK_NETWORK_NAME"), Type: cloud.NetworkTypeShared}
	Net2 = infrav1.Network{Name: "SharedGuestNet2", Type: cloud.NetworkTypeShared, ID: "FakeSharedNetID2"}
	ISONet1 = infrav1.Network{Name: "isoguestnet1", Type: cloud.NetworkTypeIsolated, ID: "FakeIsolatedNetID1"}
	CSFailureDomain1 = &infrav1.CloudStackFailureDomain{
		TypeMeta: metav1.TypeMeta{
			APIVersion: CSApiVersion,
			Kind:       "CloudStackFailureDomain"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      infrav1.FailureDomainHashedMetaName("fd1", ClusterName),
			Namespace: "default",
			UID:       "0",
			Labels:    ClusterLabel},
		Spec: infrav1.CloudStackFailureDomainSpec{Name: "fd1", Zone: Zone1,
			ACSEndpoint: corev1.SecretReference{
				Namespace: ClusterNameSpace,
				Name:      ACSEndpointSecret1.Name}}}
	CSFailureDomain2 = &infrav1.CloudStackFailureDomain{
		TypeMeta: metav1.TypeMeta{
			APIVersion: CSApiVersion,
			Kind:       "CloudStackFailureDomain"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      infrav1.FailureDomainHashedMetaName("fd2", ClusterName),
			Namespace: "default",
			UID:       "0",
			Labels:    ClusterLabel},
		Spec: infrav1.CloudStackFailureDomainSpec{Name: "fd2", Zone: Zone2,
			ACSEndpoint: corev1.SecretReference{
				Namespace: ClusterNameSpace,
				Name:      ACSEndpointSecret2.Name}}}

	CSAffinityGroup = &infrav1.CloudStackAffinityGroup{
		ObjectMeta: metav1.ObjectMeta{Name: AffinityGroup.Name, Namespace: "default", UID: "0", Labels: ClusterLabel},
		Spec: infrav1.CloudStackAffinityGroupSpec{
			FailureDomainName: CSFailureDomain1.Spec.Name,
			Name:              AffinityGroup.Name,
			Type:              AffinityGroup.Type,
			ID:                AffinityGroup.ID}}

	CSCluster = &infrav1.CloudStackCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: CSApiVersion,
			Kind:       CSClusterKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ClusterName,
			Namespace: "default",
			UID:       "0",
			Labels:    ClusterLabel,
		},
		Spec: infrav1.CloudStackClusterSpec{
			ControlPlaneEndpoint: clusterv1.APIEndpoint{Host: EndPointHost, Port: EndPointPort},
			FailureDomains:       []infrav1.CloudStackFailureDomainSpec{CSFailureDomain1.Spec, CSFailureDomain2.Spec},
		},
		Status: infrav1.CloudStackClusterStatus{},
	}
	CSISONet1 = &infrav1.CloudStackIsolatedNetwork{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ISONet1.Name,
			Namespace: "default",
			UID:       "0",
			Labels:    ClusterLabel,
		},
		Spec: infrav1.CloudStackIsolatedNetworkSpec{
			ControlPlaneEndpoint: CSCluster.Spec.ControlPlaneEndpoint}}
	CSISONet1.Spec.Name = ISONet1.Name
	CSISONet1.Spec.ID = ISONet1.ID
}

func SetACSEndpointSecretVars() {
	ACSEndpointSecret1 = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ClusterNameSpace,
			Name:      "acsendpointsecret1"},
		StringData: map[string]string{
			"api-key":    "someKey1",
			"secret-key": "someSecretKey1",
			"api-url":    "http://someUri1:8080/client/api"},
	}
	ACSEndpointSecret2 = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ClusterNameSpace,
			Name:      "acsendpointsecret2"},
		StringData: map[string]string{
			"api-key":    "someKey2",
			"secret-key": "someSecretKey2",
			"api-url":    "http://someUri2:8080/client/api"},
	}
}

// SetDummyCapiCluster resets the values in each of the exported CAPICluster related dummy variables.
func SetDummyCAPIClusterVars() {
	CAPICluster = &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ClusterName,
			Namespace: ClusterNameSpace,
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

func SetDummyIsoNetToNameOnly() {
	ISONet1.ID = ""
	ISONet1.Type = ""
	Zone1.Network = ISONet1
}

func SetDummyBootstrapSecretVar() {
	BootstrapSecretName := "such-secret-much-wow"
	BootstrapSecret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ClusterNameSpace,
			Name:      BootstrapSecretName},
		Data: map[string][]byte{"value": make([]byte, 0)}}
}

// Sets cluster spec to specified network.
func SetClusterSpecToNet(net *infrav1.Network) {
	Zone1.Network = *net
	CSFailureDomain1 = &infrav1.CloudStackFailureDomain{Spec: infrav1.CloudStackFailureDomainSpec{Zone: Zone1}}
	CSCluster.Spec.FailureDomains = []infrav1.CloudStackFailureDomainSpec{CSFailureDomain1.Spec}
}

func SetDummyCAPIMachineVars() {
	CAPIMachine = &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "capi-test-machine-",
			Namespace:    "default",
			Labels:       ClusterLabel,
		},
		Spec: clusterv1.MachineSpec{
			ClusterName:   ClusterName,
			FailureDomain: pointer.String(Zone1.ID)},
	}
}
