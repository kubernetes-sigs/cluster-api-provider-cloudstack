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

	out.Status = v1beta2.CloudStackClusterStatus{
		CloudStackFailureDomainStatusMap: getFailureDomainsStatusMap(in),
		FailureDomains:                   in.Status.FailureDomains,
		PublicIPID:                       in.Status.PublicIPID,
		PublicIPNetworkID:                in.Status.PublicIPNetworkID,
		LBRuleID:                         in.Status.LBRuleID,
		Ready:                            in.Status.Ready,
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
		getZones(in),
		in.Spec.ControlPlaneEndpoint,
		in.Spec.Account,
		in.Spec.Domain,
		identifyRef,
	}
	zoneMap, err := getZoneMap(in)
	if err != nil {
		return err
	}
	domainId, err := getDomainID(in)
	if err != nil {
		return err
	}
	out.Status = CloudStackClusterStatus{
		zoneMap,
		in.Status.FailureDomains,
		in.Status.Ready,
		domainId,
		in.Status.PublicIPID,
		in.Status.PublicIPNetworkID,
		in.Status.LBRuleID,
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

func getZoneMap(csCluster *v1beta2.CloudStackCluster) (map[string]Zone, error) {
	zoneMap := map[string]Zone{}
	for key := range csCluster.Status.CloudStackFailureDomainStatusMap {
		zone, err := getZoneByMetaName(csCluster, key)
		if err != nil {
			return nil, err
		}
		zoneMap[key] = zone
	}
	return zoneMap, nil
}

func getDomainID(csCluster *v1beta2.CloudStackCluster) (string, error) {
	var domainID string
	for _, value := range csCluster.Status.CloudStackFailureDomainStatusMap {
		if domainID == "" {
			domainID = value.DomainID
		} else if domainID != value.DomainID {
			return "", errors.Errorf("multiple domainId found in cloudstack failure domain status")
		}
	}
	return domainID, nil
}

func getZoneByMetaName(csCluster *v1beta2.CloudStackCluster, metaName string) (Zone, error) {
	var zone Zone
	err := errors.Errorf("zone with meta %s not found", metaName)
	for _, failureDomain := range csCluster.Spec.FailureDomains {
		if failureDomain.Zone.MetaName() == metaName {
			err = nil
			zone = Zone{
				ID:   failureDomain.Zone.ID,
				Name: failureDomain.Zone.Name,
				Network: Network{
					ID:   failureDomain.Zone.Network.ID,
					Name: failureDomain.Zone.Network.Name,
					Type: failureDomain.Zone.Network.Type,
				},
			}
		}
	}
	return zone, err
}

// getFailureDomains maps v1beta1 zones to v1beta2 failure domains
func getFailureDomains(csCluster *CloudStackCluster) []v1beta2.FailureDomain {
	var failureDomains []v1beta2.FailureDomain
	index := 0
	for _, zone := range csCluster.Spec.Zones {
		index = index + 1
		name := fmt.Sprintf("%s-%s-%d", csCluster.Name, "failuredomain", index)
		//objectMeta := metav1.ObjectMeta{
		//	Finalizers: []string{v1beta2.FailureDomainFinalizer},
		//	Labels: map[string]string{
		//		"cloudstackcluster.infrastructure.cluster.x-k8s.io/name": csCluster.Name,
		//		"cluster.x-k8s.io/cluster-name":                          csCluster.Name,
		//	},
		//	Name:      fmt.Sprintf("%s-%s-%d", csCluster.Name, "failuredomain", index),
		//	Namespace: csCluster.Namespace,
		//	OwnerReferences: []metav1.OwnerReference{
		//		{
		//			APIVersion:         "infrastructure.cluster.x-k8s.io/v1beta2",
		//			BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
		//			Controller:         func(b bool) *bool { return &b }(true),
		//			Kind:               "CloudStackCluster",
		//			Name:               csCluster.Name,
		//		},
		//	},
		//}
		failureDomains = append(failureDomains, v1beta2.FailureDomain{
			Zone: v1beta2.Zone{
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
			CredentialsRef: &corev1.ObjectReference{
				Kind: DefaultIdentityRefKind,
				Name: name,
			},
			//Spec: v1beta2.CloudStackFailureDomainSpec{
			//	Zone: v1beta2.Zone{
			//		Name: zone.Name,
			//		ID:   zone.ID,
			//		Network: v1beta2.Network{
			//			ID:   zone.Network.ID,
			//			Name: zone.Network.Name,
			//			Type: zone.Network.Type,
			//		},
			//	},
			//	Account: csCluster.Spec.Account,
			//	Domain:  csCluster.Spec.Domain,
			//	CredentialsRef: &corev1.ObjectReference{
			//		Kind: DefaultIdentityRefKind,
			//		Name: objectMeta.Name,
			//	},
			//},
			//Status: v1beta2.CloudStackFailureDomainStatus{
			//	DomainID: csCluster.Status.DomainID,
			//	Ready:    csCluster.Status.Ready,
			//},
		})
	}
	return failureDomains
}

func getFailureDomainsStatusMap(csCluster *CloudStackCluster) map[string]v1beta2.CloudStackFailureDomainStatus {
	failureDomainsStatusMap := map[string]v1beta2.CloudStackFailureDomainStatus{}
	for _, zone := range csCluster.Spec.Zones {
		failureDomainsStatusMap[zone.MetaName()] = v1beta2.CloudStackFailureDomainStatus{
			DomainID: csCluster.Status.DomainID,
			Ready:    csCluster.Status.Ready,
		}
	}
	return failureDomainsStatusMap
}
