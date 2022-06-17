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
	capcv1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

func GetYamlVal(variable string) string {
	val, err := CSConf.Get("variables").Get(variable).String()
	gomega.Ω(err).ShouldNot(gomega.HaveOccurred())
	return val
}

var ( // Declare exported dummy vars.
	AffinityGroup      *cloud.AffinityGroup
	CSAffinityGroup    *capcv1.CloudStackAffinityGroup
	CSCluster          *capcv1.CloudStackCluster
	CAPIMachine        *capiv1.Machine
	CSMachine1         *capcv1.CloudStackMachine
	CAPICluster        *capiv1.Cluster
	CSMachineTemplate1 *capcv1.CloudStackMachineTemplate
	Zone1              capcv1.Zone
	Zone2              capcv1.Zone
	CSZone1            *capcv1.CloudStackZone
	CSZone2            *capcv1.CloudStackZone
	Net1               capcv1.Network
	Net2               capcv1.Network
	ISONet1            capcv1.Network
	CSISONet1          *capcv1.CloudStackIsolatedNetwork
	Domain             cloud.Domain
	DomainPath         string
	DomainName         string
	DomainID           string
	Level2Domain       cloud.Domain
	Level2DomainPath   string
	Level2DomainName   string
	Level2DomainID     string
	Account            cloud.Account
	AccountName        string
	AccountID          string
	Level2Account      cloud.Account
	Level2AccountName  string
	Level2AccountID    string
	Tags               map[string]string
	Tag1               map[string]string
	Tag2               map[string]string
	Tag1Key            string
	Tag1Val            string
	Tag2Key            string
	Tag2Val            string
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
	LBRuleID           string
	PublicIPID         string
	EndPointHost       string
	EndPointPort       int32
	CSConf             *simpleyaml.Yaml
	DiskOffering       capcv1.CloudStackResourceDiskOffering
)

// variables:
//   KUBERNETES_VERSION_MANAGEMENT: "v1.20.10"
//   KUBERNETES_VERSION: "v1.20.10"
//   CNI: "./data/cni/kindnet.yaml"
//   IP_FAMILY: "IPv4"
//   NODE_DRAIN_TIMEOUT: "60s"

//   CLOUDSTACK_ZONE_NAME: zone1
//   CLOUDSTACK_INVALID_ZONE_NAME: zoneXXXX
//   CLOUDSTACK_INVALID_NETWORK_NAME: networkXXXX
//   CLOUDSTACK_ACCOUNT_NAME: admin
//   CLOUDSTACK_INVALID_ACCOUNT_NAME: accountXXXX
//   CLOUDSTACK_DOMAIN_NAME: ROOT
//   CLOUDSTACK_INVALID_DOMAIN_NAME: domainXXXX
//   CLOUDSTACK_NETWORK_NAME: isolated-for-e2e-1
//   CLOUDSTACK_NEW_NETWORK_NAME: isolated-for-e2e-new
//   CLOUDSTACK_SHARED_NETWORK_NAME: Shared1
//   CLUSTER_ENDPOINT_IP: 172.16.2.199
//   CLUSTER_ENDPOINT_IP_2: 172.16.2.198
//   CLUSTER_ENDPOINT_NEW_IP: 172.16.2.201
//   CLUSTER_ENDPOINT_PORT: 6443
//   CLUSTER_ENDPOINT_PORT_2: 6443
//   CLOUDSTACK_CONTROL_PLANE_MACHINE_OFFERING: "Large Instance"
//   CLOUDSTACK_INVALID_CONTROL_PLANE_MACHINE_OFFERING: "OfferingXXXX"
//   CLOUDSTACK_EXTREMELY_LARGE_CONTROL_PLANE_MACHINE_OFFERING: "Extremely Large Instance"
//   CLOUDSTACK_WORKER_MACHINE_OFFERING: "Medium Instance"
//   CLOUDSTACK_TEMPLATE_NAME: kube-v1.20.10/ubuntu-2004
//   CLOUDSTACK_INVALID_TEMPLATE_NAME: templateXXXX
//   CLOUDSTACK_SSH_KEY_NAME: CAPCKeyPair6

// SetDummyVars sets/resets all dummy vars.
func SetDummyVars() {
	projDir := os.Getenv("PROJECT_DIR")
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
	SetDummyCAPCClusterVars()
	SetDummyCAPIClusterVars()
	SetDummyCAPIMachineVars()
	SetDummyCSMachineTemplateVars()
	SetDummyCSMachineVars()
	SetDummyTagVars()
	LBRuleID = "FakeLBRuleID"
}

func SetDiskOfferingVars() {
	DiskOffering = capcv1.CloudStackResourceDiskOffering{CloudStackResourceIdentifier: capcv1.CloudStackResourceIdentifier{Name: "Small"},
		MountPath:  "/data",
		Device:     "/dev/vdb",
		Filesystem: "ext4",
		Label:      "data_disk",
	}
}

func CAPCNetToCSAPINet(net *capcv1.Network) *csapi.Network {
	return &csapi.Network{
		Name: net.Name,
		Id:   net.ID,
		Type: net.Type,
	}
}

func CAPCZoneToCSAPIZone(net *capcv1.Zone) *csapi.Zone {
	return &csapi.Zone{
		Name: net.Name,
		Id:   net.ID,
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
	CSMachineTemplate1 = &capcv1.CloudStackMachineTemplate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
			Kind:       "CloudStackMachineTemplate",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machinetemplate-1",
			Namespace: "default",
		},
		Spec: capcv1.CloudStackMachineTemplateSpec{
			Spec: capcv1.CloudStackMachineTemplateResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-machinetemplateresource",
					Namespace: "default",
				},
				Spec: capcv1.CloudStackMachineSpec{
					IdentityRef: &capcv1.CloudStackIdentityReference{
						Kind: "Secret",
						Name: "IdentitySecret",
					},
					Template: capcv1.CloudStackResourceIdentifier{
						Name: GetYamlVal("CLOUDSTACK_TEMPLATE_NAME"),
					},
					Offering: capcv1.CloudStackResourceIdentifier{
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
	CSMachine1 = &capcv1.CloudStackMachine{
		TypeMeta: metav1.TypeMeta{
			APIVersion: CSApiVersion,
			Kind:       "CloudStackMachine",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine-2",
			Namespace: "default",
		},
		Spec: capcv1.CloudStackMachineSpec{
			IdentityRef: &capcv1.CloudStackIdentityReference{
				Kind: "Secret",
				Name: "IdentitySecret",
			},
			InstanceID: pointer.String("Instance1"),
			Template: capcv1.CloudStackResourceIdentifier{
				Name: GetYamlVal("CLOUDSTACK_TEMPLATE_NAME"),
			},
			Offering: capcv1.CloudStackResourceIdentifier{
				Name: GetYamlVal("CLOUDSTACK_CONTROL_PLANE_MACHINE_OFFERING"),
			},
			DiskOffering: capcv1.CloudStackResourceDiskOffering{
				CloudStackResourceIdentifier: capcv1.CloudStackResourceIdentifier{
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
	CSMachine1.ObjectMeta.SetName("test-vm")
}

func SetDummyZoneVars() {
	Zone1 = capcv1.Zone{Network: Net1}
	Zone1.Name = GetYamlVal("CLOUDSTACK_ZONE_NAME")
	Zone2 = capcv1.Zone{Network: Net2}
	Zone2.Name = "Zone2"
	Zone2.ID = "FakeZone2ID"
	CSZone1 = &capcv1.CloudStackZone{Spec: capcv1.CloudStackZoneSpec(Zone1)}
	CSZone2 = &capcv1.CloudStackZone{Spec: capcv1.CloudStackZoneSpec(Zone2)}
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
	CSApiVersion = "infrastructure.cluster.x-k8s.io/v1beta1"
	CSClusterKind = "CloudStackCluster"
	CSClusterName = "test-cluster"
	EndPointHost = "EndpointHost"
	EndPointPort = int32(5309)
	PublicIPID = "FakePublicIPID"

	CSlusterNamespace = "default"
	AffinityGroup = &cloud.AffinityGroup{
		Name: "FakeAffinityGroup",
		Type: cloud.AffinityGroupType,
		ID:   "FakeAffinityGroupID"}
	CSAffinityGroup = &capcv1.CloudStackAffinityGroup{
		Spec: capcv1.CloudStackAffinityGroupSpec{Name: AffinityGroup.Name, Type: AffinityGroup.Type, ID: AffinityGroup.ID}}
	Net1 = capcv1.Network{Name: GetYamlVal("CLOUDSTACK_NETWORK_NAME"), Type: cloud.NetworkTypeShared}
	Net2 = capcv1.Network{Name: "SharedGuestNet2", Type: cloud.NetworkTypeShared, ID: "FakeSharedNetID2"}
	ISONet1 = capcv1.Network{Name: "IsoGuestNet1", Type: cloud.NetworkTypeIsolated, ID: "FakeIsolatedNetID1"}
	CSCluster = &capcv1.CloudStackCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: CSApiVersion,
			Kind:       CSClusterKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      CSClusterName,
			Namespace: "default",
			UID:       "0",
		},
		Spec: capcv1.CloudStackClusterSpec{
			IdentityRef: &capcv1.CloudStackIdentityReference{
				Kind: "Secret",
				Name: "IdentitySecret",
			},
			ControlPlaneEndpoint: capiv1.APIEndpoint{Host: EndPointHost, Port: EndPointPort},
			Zones:                []capcv1.Zone{Zone1, Zone2},
		},
		Status: capcv1.CloudStackClusterStatus{Zones: map[string]capcv1.Zone{}},
	}
	CSISONet1 = &capcv1.CloudStackIsolatedNetwork{
		Spec: capcv1.CloudStackIsolatedNetworkSpec{
			ControlPlaneEndpoint: CSCluster.Spec.ControlPlaneEndpoint}}
	CSISONet1.Spec.Name = ISONet1.Name
	CSISONet1.Spec.ID = ISONet1.ID
}

// SetClusterDummyDomainAndAccount sets domain and account in the CSCluster Spec. This is not the default.
func SetClusterDummyDomainAndAccount() {
	CSCluster.Spec.Account = AccountName
	CSCluster.Spec.Domain = DomainPath
}

// SetClusterDummyDomainAndAccount sets domain and account in the CSCluster Spec. This is not the default.
func SetDummyDomainAndAccount() {
	CSCluster.Spec.Account = AccountName
	CSCluster.Spec.Domain = DomainPath
}

// SetDummyDomainAndAccount sets domainID in the CSCluster Status. This is not the default.
func SetDummyDomainID() {
	CSCluster.Status.DomainID = "FakeDomainID"
}

// SetDummyCapiCluster resets the values in each of the exported CAPICluster related dummy variables.
func SetDummyCAPIClusterVars() {
	CAPICluster = &capiv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "capi-cluster-test-",
			Namespace:    "default",
		},
		Spec: capiv1.ClusterSpec{
			InfrastructureRef: &corev1.ObjectReference{
				APIVersion: capcv1.GroupVersion.String(),
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

// Fills in cluster status vars.
func SetDummyClusterStatus() {
	CSCluster.Status.Zones = capcv1.ZoneStatusMap{Zone1.ID: Zone1, Zone2.ID: Zone2}
	CSCluster.Status.LBRuleID = LBRuleID
}

// Sets cluster spec to specified network.
func SetClusterSpecToNet(net *capcv1.Network) {
	Zone1.Network = *net
	CSCluster.Spec.Zones = []capcv1.Zone{Zone1}
}

func SetDummyCAPIMachineVars() {
	CAPIMachine = &capiv1.Machine{
		Spec: capiv1.MachineSpec{FailureDomain: pointer.String(Zone1.ID)},
	}
}

func SetDummyCSMachineStatuses() {
	CSMachine1.Status = capcv1.CloudStackMachineStatus{ZoneID: Zone1.ID}
}
