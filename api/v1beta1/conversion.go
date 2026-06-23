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

package v1beta1

import (
	"context"
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
	conv "k8s.io/apimachinery/pkg/conversion"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta4"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const DefaultEndpointCredential = "global"

// Convert_v1beta1_CloudStackCluster_To_v1beta4_CloudStackCluster converts a v1beta1 (zone-based)
// cluster to the v1beta4 hub, mapping zones to failure domains and the status failure-domain map
// to the v1beta4 slice.
//
//nolint:golint,revive,stylecheck
func Convert_v1beta1_CloudStackCluster_To_v1beta4_CloudStackCluster(in *CloudStackCluster, out *v1beta4.CloudStackCluster, s conv.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	failureDomains, err := GetFailureDomains(in)
	if err != nil {
		return err
	}
	out.Spec = v1beta4.CloudStackClusterSpec{
		ControlPlaneEndpoint: in.Spec.ControlPlaneEndpoint,
		FailureDomains:       failureDomains,
	}

	out.Status = v1beta4.CloudStackClusterStatus{
		Ready: in.Status.Ready,
	}
	if len(in.Status.FailureDomains) > 0 {
		out.Status.FailureDomains = make([]v1beta4.FailureDomain, 0, len(in.Status.FailureDomains))
		names := make([]string, 0, len(in.Status.FailureDomains))
		for name := range in.Status.FailureDomains {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fd := in.Status.FailureDomains[name]
			out.Status.FailureDomains = append(out.Status.FailureDomains, v1beta4.FailureDomain{
				Name:         name,
				ControlPlane: ptr.To(fd.ControlPlane),
				Attributes:   fd.Attributes,
			})
		}
	}
	return nil
}

// Convert_v1beta4_CloudStackCluster_To_v1beta1_CloudStackCluster converts a v1beta4 hub cluster
// down to the v1beta1 (zone-based) representation.
//
//nolint:golint,revive,stylecheck
func Convert_v1beta4_CloudStackCluster_To_v1beta1_CloudStackCluster(in *v1beta4.CloudStackCluster, out *CloudStackCluster, scope conv.Scope) error {
	if len(in.Spec.FailureDomains) < 1 {
		return fmt.Errorf("v1beta4 to v1beta1 conversion not supported when < 1 failure domain is provided. Input CloudStackCluster spec %v", in.Spec)
	}
	out.ObjectMeta = in.ObjectMeta
	out.Spec = CloudStackClusterSpec{
		Account:              in.Spec.FailureDomains[0].Account,
		Domain:               in.Spec.FailureDomains[0].Domain,
		Zones:                getZones(in),
		ControlPlaneEndpoint: in.Spec.ControlPlaneEndpoint,
	}

	out.Status = CloudStackClusterStatus{
		Ready: in.Status.Ready,
	}
	if len(in.Status.FailureDomains) > 0 {
		out.Status.FailureDomains = make(FailureDomains, len(in.Status.FailureDomains))
		for _, fd := range in.Status.FailureDomains {
			cp := false
			if fd.ControlPlane != nil {
				cp = *fd.ControlPlane
			}
			out.Status.FailureDomains[fd.Name] = FailureDomainSpec{ControlPlane: cp, Attributes: fd.Attributes}
		}
	}
	return nil
}

// getZones maps v1beta4 failure domains to v1beta1 zones.
func getZones(csCluster *v1beta4.CloudStackCluster) []Zone {
	var zones []Zone
	for _, failureDomain := range csCluster.Spec.FailureDomains {
		zone := failureDomain.Zone
		zones = append(zones, Zone{
			Name: zone.Name,
			ID:   zone.ID,
			Network: Network{
				Name: zone.Network.Name,
				ID:   zone.Network.ID,
				Type: zone.Network.Type,
			},
		})
	}
	return zones
}

// GetFailureDomains maps v1beta1 zones to v1beta4 failure domains.
func GetFailureDomains(csCluster *CloudStackCluster) ([]v1beta4.CloudStackFailureDomainSpec, error) {
	var failureDomains []v1beta4.CloudStackFailureDomainSpec
	namespace := csCluster.Namespace
	for _, zone := range csCluster.Spec.Zones {
		name, err := GetDefaultFailureDomainName(namespace, csCluster.Name, zone.ID, zone.Name)
		if err != nil {
			return nil, err
		}
		failureDomains = append(failureDomains, v1beta4.CloudStackFailureDomainSpec{
			Name: name,
			Zone: v1beta4.CloudStackZoneSpec{
				ID:   zone.ID,
				Name: zone.Name,
				Network: v1beta4.Network{
					ID:   zone.Network.ID,
					Name: zone.Network.Name,
					Type: zone.Network.Type,
				},
			},
			Domain:  csCluster.Spec.Domain,
			Account: csCluster.Spec.Account,
			ACSEndpoint: corev1.SecretReference{
				Namespace: namespace,
				Name:      DefaultEndpointCredential,
			},
		})
	}
	return failureDomains, nil
}

// GetDefaultFailureDomainName return zoneID as failuredomain name.
// Default failure domain name is used when migrating an old cluster to a multiple-endpoints supported cluster, that
// requires to convert each zone to a failure domain.
// When upgrading cluster using eks-a, a secret named global will be created by eks-a, and it is used by following
// method to get zoneID by calling cloudstack API.
// When upgrading cluster using clusterctl directly, zoneID is fetched directly from kubernetes cluster in cloudstackzones.
func GetDefaultFailureDomainName(namespace string, _ string, zoneID string, zoneName string) (string, error) {
	if len(zoneID) > 0 {
		return zoneID, nil
	}

	secret, err := GetK8sSecret(DefaultEndpointCredential, namespace)
	if err != nil {
		return "", err
	}

	// try fetch zoneID using zoneName through cloudstack client
	zoneID, err = fetchZoneIDUsingCloudStack(secret, zoneName)
	if err == nil {
		return zoneID, nil
	}

	zoneID, err = fetchZoneIDUsingK8s(namespace, zoneName)
	if err != nil {
		return "", nil
	}
	return zoneID, nil
}

func fetchZoneIDUsingK8s(namespace string, zoneName string) (string, error) {
	zone := &CloudStackZone{}
	key := client.ObjectKey{Name: zoneName, Namespace: namespace}
	if err := v1beta4.K8sClient.Get(context.TODO(), key, zone); err != nil {
		return "", err
	}

	return zone.Spec.ID, nil
}

func fetchZoneIDUsingCloudStack(secret *corev1.Secret, zoneName string) (string, error) {
	client, err := cloud.NewClientFromK8sSecret(secret, nil, "")
	if err != nil {
		return "", err
	}
	zone := &v1beta4.CloudStackZoneSpec{Name: zoneName}
	err = client.ResolveZone(zone)
	return zone.ID, err
}

func GetK8sSecret(name, namespace string) (*corev1.Secret, error) {
	endpointCredentials := &corev1.Secret{}
	key := client.ObjectKey{Name: name, Namespace: namespace}
	if err := v1beta4.K8sClient.Get(context.TODO(), key, endpointCredentials); err != nil {
		return nil, err
	}
	return endpointCredentials, nil
}

// Convert_v1beta4_Network_To_v1beta1_Network converts from v1beta4.Network to v1beta1.Network,
// dropping the fields (Gateway, Netmask, Offering, RoutingMode, VPC) that do not exist in v1beta1.
//
//nolint:golint,revive,stylecheck
func Convert_v1beta4_Network_To_v1beta1_Network(in *v1beta4.Network, out *Network, _ conv.Scope) error {
	out.ID = in.ID
	out.Type = in.Type
	out.Name = in.Name
	return nil
}
