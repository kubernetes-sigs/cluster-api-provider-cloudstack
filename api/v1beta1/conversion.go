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
	"fmt"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	conv "k8s.io/apimachinery/pkg/conversion"
	"sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	"strings"
)

//nolint:golint,revive,stylecheck
func Convert_v1beta1_CloudStackCluster_To_v1beta2_CloudStackCluster(in *CloudStackCluster, out *v1beta2.CloudStackCluster, s conv.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	var identifyRef *v1beta2.CloudStackIdentityReference
	if in.Spec.IdentityRef != nil {
		identifyRef = &v1beta2.CloudStackIdentityReference{}
		err := Convert_v1beta1_CloudStackIdentityReference_To_v1beta2_CloudStackIdentityReference(in.Spec.IdentityRef, identifyRef, nil)
		if err != nil {
			return err
		}
	}
	out.Spec = v1beta2.CloudStackClusterSpec{
		ControlPlaneEndpoint: in.Spec.ControlPlaneEndpoint,
		FailureDomains:       getFailureDomains(in),
		Account:              in.Spec.Account,
		Domain:               in.Spec.Domain,
		IdentityRef:          identifyRef,
	}

	zoneStatusMap := v1beta2.ZoneStatusMap{}
	for zoneKey, zone := range in.Status.Zones {
		zone1 := zone
		zonev2 := &v1beta2.Zone{}
		err := Convert_v1beta1_Zone_To_v1beta2_Zone(&zone1, zonev2, nil)
		if err != nil {
			return err
		}
		zoneStatusMap[zoneKey] = *zonev2
	}
	out.Status = v1beta2.CloudStackClusterStatus{
		Zones: zoneStatusMap,
		//CloudStackFailureDomainStatusMap: getFailureDomainsStatusMap(in),
		FailureDomains:    in.Status.FailureDomains,
		PublicIPID:        in.Status.PublicIPID,
		PublicIPNetworkID: in.Status.PublicIPNetworkID,
		LBRuleID:          in.Status.LBRuleID,
		Ready:             in.Status.Ready,
	}
	return nil
}

//nolint:golint,revive,stylecheck
func Convert_v1beta2_CloudStackCluster_To_v1beta1_CloudStackCluster(in *v1beta2.CloudStackCluster, out *CloudStackCluster, scope conv.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	var identifyRef *CloudStackIdentityReference
	if in.Spec.IdentityRef != nil {
		identifyRef = &CloudStackIdentityReference{}
		err := Convert_v1beta2_CloudStackIdentityReference_To_v1beta1_CloudStackIdentityReference(in.Spec.IdentityRef, identifyRef, nil)
		if err != nil {
			return err
		}
	}
	out.Spec = CloudStackClusterSpec{
		Zones:                getZones(in),
		ControlPlaneEndpoint: in.Spec.ControlPlaneEndpoint,
		Account:              in.Spec.Account,
		Domain:               in.Spec.Domain,
		IdentityRef:          identifyRef,
	}

	zoneStatusMap := ZoneStatusMap{}
	for zoneKey, zone := range in.Status.Zones {
		zonev1 := &Zone{}
		zonev2 := zone
		err := Convert_v1beta2_Zone_To_v1beta1_Zone(&zonev2, zonev1, nil)
		if err != nil {
			return err
		}
		zoneStatusMap[zoneKey] = *zonev1
	}
	out.Status = CloudStackClusterStatus{
		Zones:             zoneStatusMap,
		FailureDomains:    in.Status.FailureDomains,
		Ready:             in.Status.Ready,
		DomainID:          in.Status.DomainID,
		PublicIPID:        in.Status.PublicIPID,
		PublicIPNetworkID: in.Status.PublicIPNetworkID,
		LBRuleID:          in.Status.LBRuleID,
	}
	return nil
}

// getZones maps failure domains to zones
func getZones(csCluster *v1beta2.CloudStackCluster) []Zone {
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

//func getZoneMap(csCluster *v1beta2.CloudStackCluster) (map[string]Zone, error) {
//	zoneMap := map[string]Zone{}
//	for key := range csCluster.Status.CloudStackFailureDomainStatusMap {
//		zone, err := getZoneByMetaName(csCluster, key)
//		if err != nil {
//			return nil, err
//		}
//		zoneMap[key] = zone
//	}
//	return zoneMap, nil
//}
//
//func getDomainID(csCluster *v1beta2.CloudStackCluster) (string, error) {
//	var domainID string
//	for _, value := range csCluster.Status.CloudStackFailureDomainStatusMap {
//		if domainID == "" {
//			domainID = value.DomainID
//		} else if domainID != value.DomainID {
//			return "", errors.Errorf("multiple domainId found in cloudstack failure domain status")
//		}
//	}
//	return domainID, nil
//}
//
//func getZoneByMetaName(csCluster *v1beta2.CloudStackCluster, metaName string) (Zone, error) {
//	var zone Zone
//	err := errors.Errorf("zone with meta %s not found", metaName)
//	for _, failureDomain := range csCluster.Spec.FailureDomains {
//		if failureDomain.Zone.MetaName() == metaName {
//			err = nil
//			zone = Zone{
//				ID:   failureDomain.Zone.ID,
//				Name: failureDomain.Zone.Name,
//				Network: Network{
//					ID:   failureDomain.Zone.Network.ID,
//					Name: failureDomain.Zone.Network.Name,
//					Type: failureDomain.Zone.Network.Type,
//				},
//			}
//		}
//	}
//	return zone, err
//}

// getFailureDomains maps v1beta1 zones to v1beta2 failure domains
func getFailureDomains(csCluster *CloudStackCluster) []v1beta2.CloudStackFailureDomainSpec {
	var failureDomains []v1beta2.CloudStackFailureDomainSpec
	index := 0
	for _, zone := range csCluster.Spec.Zones {
		index = index + 1
		name := fmt.Sprintf("%s-%s-%d", csCluster.Name, "failuredomain", index)
		failureDomains = append(failureDomains, v1beta2.CloudStackFailureDomainSpec{
			Name: name,
			Zone: v1beta2.CloudStackZoneSpec{
				ID:   zone.ID,
				Name: zone.Name,
				Network: v1beta2.Network{
					ID:   zone.Network.ID,
					Name: zone.Network.Name,
					Type: zone.Network.Type,
				},
			},
			Domain:  csCluster.Spec.Domain,
			Account: csCluster.Spec.Account,
			ACSEndpoint: corev1.SecretReference{
				Namespace: csCluster.ObjectMeta.Namespace,
				Name:      name,
			},
		})
	}
	return failureDomains
}

//
//func getFailureDomainsStatusMap(csCluster *CloudStackCluster) map[string]v1beta2.CloudStackFailureDomainStatus {
//	failureDomainsStatusMap := map[string]v1beta2.CloudStackFailureDomainStatus{}
//	for _, zone := range csCluster.Spec.Zones {
//		failureDomainsStatusMap[zone.MetaName()] = v1beta2.CloudStackFailureDomainStatus{
//			DomainID: csCluster.Status.DomainID,
//			Ready:    csCluster.Status.Ready,
//		}
//	}
//	return failureDomainsStatusMap
//}

func GetDefaultFailureDomainName(clusterName string, zoneID string, zoneName string) (string, error) {
	zoneMetaName := ""
	if len(zoneName) > 0 {
		zoneMetaName = zoneName
	} else if len(zoneID) > 0 {
		zoneMetaName = zoneID
	}

	if len(zoneMetaName) == 0 {
		return "", errors.Errorf("failed to generate default failureDomainName: zone id and name both empty")
	}
	if len(clusterName) == 0 {
		return "", errors.Errorf("failed to generate default failureDomainName: clusterName empty")
	}

	return strings.ToLower(fmt.Sprintf("%s-failuredomain-%s", clusterName, zoneMetaName)), nil
}
