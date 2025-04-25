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

package v1beta2

import (
	conv "k8s.io/apimachinery/pkg/conversion"
	"sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
)

// Convert_v1beta3_Network_To_v1beta2_Network converts from v1beta3.Network to v1beta2.Network
//
//nolint:golint,revive,stylecheck
func Convert_v1beta3_Network_To_v1beta2_Network(in *v1beta3.Network, out *Network, _ conv.Scope) error {
	out.ID = in.ID
	out.Type = in.Type
	out.Name = in.Name
	// Skip Gateway, Netmask, and VPC fields as they do not exist in v1beta2.Network
	return nil
}

// Convert_v1beta3_CloudStackIsolatedNetworkSpec_To_v1beta2_CloudStackIsolatedNetworkSpec converts from v1beta3.CloudStackIsolatedNetworkSpec to v1beta2.CloudStackIsolatedNetworkSpec
//
//nolint:golint,revive,stylecheck
func Convert_v1beta3_CloudStackIsolatedNetworkSpec_To_v1beta2_CloudStackIsolatedNetworkSpec(in *v1beta3.CloudStackIsolatedNetworkSpec, out *CloudStackIsolatedNetworkSpec, _ conv.Scope) error {
	out.Name = in.Name
	out.ID = in.ID
	out.ControlPlaneEndpoint = in.ControlPlaneEndpoint
	out.FailureDomainName = in.FailureDomainName
	// Skip Gateway, Netmask, and VPC fields as they do not exist in v1beta2.CloudStackIsolatedNetworkSpec
	return nil
}

// Convert_v1beta3_CloudStackIsolatedNetworkStatus_To_v1beta2_CloudStackIsolatedNetworkStatus handles manual conversion of CloudStackIsolatedNetworkStatus from v1beta3 to v1beta2
//
//nolint:golint,revive,stylecheck
func Convert_v1beta3_CloudStackIsolatedNetworkStatus_To_v1beta2_CloudStackIsolatedNetworkStatus(in *v1beta3.CloudStackIsolatedNetworkStatus, out *CloudStackIsolatedNetworkStatus, _ conv.Scope) error {
	out.PublicIPID = in.PublicIPID
	out.LBRuleID = in.LBRuleID
	out.Ready = in.Ready
	// RoutingMode field doesn't exist in v1beta2, so we ignore it during conversion
	return nil
}
