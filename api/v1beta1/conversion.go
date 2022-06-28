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
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	conv "k8s.io/apimachinery/pkg/conversion"
	"sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	"strings"
)

//nolint:golint,revive,stylecheck
func Convert_v1beta1_CloudStackCluster_To_v1beta2_CloudStackCluster(in *CloudStackCluster, out *v1beta2.CloudStackCluster, s conv.Scope) error {
	out.Kind = in.Kind
	out.APIVersion = strings.ReplaceAll(in.APIVersion, "v1beta1", "v1beta2")
	out.Finalizers = in.Finalizers
	out.Labels = in.Labels
	out.Name = in.Name
	out.Namespace = in.Namespace
	out.OwnerReferences = in.OwnerReferences

	out.Spec = v1beta2.CloudStackClusterSpec{
		ControlPlaneEndpoint: in.Spec.ControlPlaneEndpoint,
		FailureDomains:       getFailureDomains(in),
		Account:              in.Spec.Account,
		Domain:               in.Spec.Domain,
		IdentityRef: &v1beta2.CloudStackIdentityReference{
			Kind: in.Spec.IdentityRef.Kind,
			Name: in.Spec.IdentityRef.Name,
		},
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
	out.Kind = in.Kind
	out.APIVersion = strings.ReplaceAll(in.APIVersion, "v1beta2", "v1beta1")
	out.Finalizers = in.Finalizers
	out.Labels = in.Labels
	out.Name = in.Name
	out.Namespace = in.Namespace
	out.OwnerReferences = in.OwnerReferences

	if len(in.Spec.FailureDomains) > 0 {
		var zones []Zone
		for _, failureDomains := range in.Spec.FailureDomains {
			zones = append(zones, Zone{
				Name: failureDomains.Spec.Zone.Name,
				ID:   failureDomains.Spec.Zone.ID,
				Network: Network{
					Name: failureDomains.Spec.Zone.Network.Name,
					ID:   failureDomains.Spec.Zone.Network.ID,
					Type: failureDomains.Spec.Zone.Network.Type,
				},
			})
		}
		out.Spec = CloudStackClusterSpec{
			zones,
			in.Spec.ControlPlaneEndpoint,
			in.Spec.FailureDomains[0].Spec.Account,
			in.Spec.FailureDomains[0].Spec.Domain,
			&CloudStackIdentityReference{
				Kind: in.Spec.FailureDomains[0].Spec.CredentialsRef.Kind,
				Name: in.Spec.FailureDomains[0].Spec.CredentialsRef.Name,
			},
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
	}

	return nil
}

func getZoneMap(src *v1beta2.CloudStackCluster) (map[string]Zone, error) {
	zoneMap := map[string]Zone{}
	for key := range src.Status.CloudStackFailureDomainStatusMap {
		zone, err := getZoneByMetaName(src, key)
		if err != nil {
			return nil, err
		}
		zoneMap[key] = zone
	}
	return zoneMap, nil
}

func getDomainID(src *v1beta2.CloudStackCluster) (string, error) {
	var domainID string
	for _, value := range src.Status.CloudStackFailureDomainStatusMap {
		if domainID == "" {
			domainID = value.DomainID
		} else if domainID != value.DomainID {
			return "", errors.Errorf("multiple domainId found in cloudstack failure domain status")
		}
	}
	return domainID, nil
}

func getZoneByMetaName(src *v1beta2.CloudStackCluster, metaName string) (Zone, error) {
	var zone Zone
	err := errors.Errorf("zone with meta %s not found", metaName)
	for _, failureDomains := range src.Spec.FailureDomains {
		if failureDomains.Spec.Zone.MetaName() == metaName {
			err = nil
			zone = Zone{
				ID:   failureDomains.Spec.Zone.ID,
				Name: failureDomains.Spec.Zone.Name,
				Network: Network{
					ID:   failureDomains.Spec.Zone.Network.ID,
					Name: failureDomains.Spec.Zone.Network.Name,
					Type: failureDomains.Spec.Zone.Network.Type,
				},
			}
		}
	}
	return zone, err
}

func getFailureDomains(src *CloudStackCluster) []v1beta2.CloudStackFailureDomain {
	failureDomains := make([]v1beta2.CloudStackFailureDomain, len(src.Spec.Zones))
	for _, zone := range src.Spec.Zones {
		typeMeta := metav1.TypeMeta{
			Kind:       src.Kind,
			APIVersion: strings.ReplaceAll(src.APIVersion, "v1beta1", "v1beta2"),
		}
		objectMeta := metav1.ObjectMeta{
			Finalizers:      src.Finalizers,
			Labels:          src.Labels,
			Name:            src.Name,
			Namespace:       src.Namespace,
			OwnerReferences: src.OwnerReferences,
		}
		failureDomains = append(failureDomains, v1beta2.CloudStackFailureDomain{
			TypeMeta:   typeMeta,
			ObjectMeta: objectMeta,
			Spec: v1beta2.CloudStackFailureDomainSpec{
				Zone: v1beta2.Zone{
					Name: zone.Name,
					ID:   zone.ID,
					Network: v1beta2.Network{
						ID:   zone.Network.ID,
						Name: zone.Network.Name,
						Type: zone.Network.Type,
					},
				},
				Account: src.Spec.Account,
				Domain:  src.Spec.Domain,
				CredentialsRef: &corev1.ObjectReference{
					Kind: src.Spec.IdentityRef.Kind,
					Name: src.Spec.IdentityRef.Name,
				},
			},
			Status: v1beta2.CloudStackFailureDomainStatus{
				DomainID: src.Status.DomainID,
				Ready:    src.Status.Ready,
			},
		})
	}
	return failureDomains
}

func getFailureDomainsStatusMap(src *CloudStackCluster) map[string]v1beta2.CloudStackFailureDomainStatus {
	failureDomainsStatusMap := map[string]v1beta2.CloudStackFailureDomainStatus{}
	for _, zone := range src.Spec.Zones {
		failureDomainsStatusMap[zone.MetaName()] = v1beta2.CloudStackFailureDomainStatus{
			DomainID: src.Status.DomainID,
			Ready:    src.Status.Ready,
		}
	}
	return failureDomainsStatusMap
}
