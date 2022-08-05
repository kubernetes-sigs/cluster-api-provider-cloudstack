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
	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	conv "k8s.io/apimachinery/pkg/conversion"
	"sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const DefaultEndpointCredential = "global"

//nolint:golint,revive,stylecheck
func Convert_v1beta1_CloudStackCluster_To_v1beta2_CloudStackCluster(in *CloudStackCluster, out *v1beta2.CloudStackCluster, s conv.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	failureDomains, err := getFailureDomains(in)
	if err != nil {
		return err
	}
	out.Spec = v1beta2.CloudStackClusterSpec{
		ControlPlaneEndpoint: in.Spec.ControlPlaneEndpoint,
		FailureDomains:       failureDomains,
		Account:              in.Spec.Account,
		Domain:               in.Spec.Domain,
	}

	out.Status = v1beta2.CloudStackClusterStatus{
		FailureDomains: in.Status.FailureDomains,
		Ready:          in.Status.Ready,
	}
	return nil
}

//nolint:golint,revive,stylecheck
func Convert_v1beta2_CloudStackCluster_To_v1beta1_CloudStackCluster(in *v1beta2.CloudStackCluster, out *CloudStackCluster, scope conv.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	out.Spec = CloudStackClusterSpec{
		Zones:                getZones(in),
		ControlPlaneEndpoint: in.Spec.ControlPlaneEndpoint,
		Account:              in.Spec.Account,
		Domain:               in.Spec.Domain,
	}

	out.Status = CloudStackClusterStatus{
		FailureDomains: in.Status.FailureDomains,
		Ready:          in.Status.Ready,
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

// getFailureDomains maps v1beta1 zones to v1beta2 failure domains.
func getFailureDomains(csCluster *CloudStackCluster) ([]v1beta2.CloudStackFailureDomainSpec, error) {
	var failureDomains []v1beta2.CloudStackFailureDomainSpec
	namespace := csCluster.Namespace
	for _, zone := range csCluster.Spec.Zones {
		name, err := GetDefaultFailureDomainName(namespace, csCluster.Name, zone.ID, zone.Name)
		if err != nil {
			return nil, err
		}
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
func GetDefaultFailureDomainName(namespace string, clusterName string, zoneID string, zoneName string) (string, error) {
	if len(zoneID) > 0 {
		return zoneID + "-" + clusterName, nil
	}

	// try fetch zoneID using zoneName through cloudstack client
	zoneID, err := fetchZoneIDUsingCloudStack(namespace, zoneName)
	if err == nil {
		return zoneID + "-" + clusterName, nil
	}

	zoneID, err = fetchZoneIDUsingK8s(namespace, clusterName, zoneName)
	if err != nil {
		return "", nil
	}
	return zoneID + "-" + clusterName, nil
}

func fetchZoneIDUsingK8s(namespace string, clusterName string, zoneName string) (string, error) {
	zones := &CloudStackZoneList{}
	capiClusterLabel := map[string]string{clusterv1.ClusterLabelName: clusterName}
	if err := v1beta2.K8sClient.List(
		context.TODO(),
		zones,
		client.InNamespace(namespace),
		client.MatchingLabels(capiClusterLabel),
	); err != nil {
		return "", err
	}

	for _, zone := range zones.Items {
		if zone.Spec.Name == zoneName {
			return zone.Spec.ID, nil
		}
	}

	return "", errors.Errorf("failed to generate default failureDomainName: zone id not found for zone name: %s", zoneName)
}

func fetchZoneIDUsingCloudStack(namespace string, zoneName string) (string, error) {
	config, err := GetCloudStackConfig(namespace)
	if err != nil {
		return "", err
	}

	csClient := cloudstack.NewAsyncClient(fmt.Sprint(config["api-url"]), fmt.Sprint(config["api-key"]), fmt.Sprint(config["secret-key"]), fmt.Sprint(config["verify-ssl"]) == "true")

	if zoneID, count, err := csClient.Zone.GetZoneID(zoneName); err != nil {
		return "", err
	} else if count != 1 {
		return "", errors.Errorf("%v zones found for zone name %s", count, zoneName)
	} else {
		return zoneID, nil
	}
}

func GetCloudStackConfig(namespace string) (map[string]interface{}, error) {
	endpointCredentials := &corev1.Secret{}
	key := client.ObjectKey{Name: DefaultEndpointCredential, Namespace: namespace}
	if err := v1beta2.K8sClient.Get(context.TODO(), key, endpointCredentials); err != nil {
		return nil, err
	}

	config := map[string]interface{}{}
	for k, v := range endpointCredentials.Data {
		config[k] = string(v)
	}
	// TODO change secret parsing manner.
	if val, present := config["verify-ssl"]; present {
		if val == "true" {
			config["verify-ssl"] = true
		} else if val == "false" {
			config["verify-ssl"] = false
		}
	}
	return config, nil
}
