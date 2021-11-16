package cloud

import (
	"github.com/apache/cloudstack-go/v2/cloudstack"
	infrav1 "gitlab.aws.dev/ce-pike/merida/cluster-api-provider-capc/api/v1alpha4"
)

func FetchClusterInfo(cs *cloudstack.CloudStackClient, csCluster *infrav1.CloudStackCluster) (retErr error) {
	csCluster.Status.ZoneID, _, retErr = cs.Zone.GetZoneID(csCluster.Spec.Zone)
	if retErr != nil {
		return retErr
	}
	csCluster.Status.NetworkID, _, retErr = cs.Network.GetNetworkID(csCluster.Spec.Network)
	if retErr != nil {
		return retErr
	}
	csCluster.Status.Ready = true
	return retErr
}
