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

package failuredomains

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"k8s.io/utils/pointer"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Balancer interface {
	Assign(ctx context.Context, csMachine *infrav1.CloudStackMachine, capiMachine *clusterv1.Machine, fds []infrav1.CloudStackFailureDomainSpec) error
}

func NewFailureDomainBalancer(csClientFactory ClientFactory) Balancer {
	return newReassigningFailureDomainBalancer(newFallingBackFailureDomainBalancer(
		newFreeIPValidatingFailureDomainBalancer(csClientFactory),
		newRandomFailureDomainBalancer(),
	))
}

type reassigningFailureDomainBalancer struct {
	delegate Balancer
}

func newReassigningFailureDomainBalancer(delegate Balancer) Balancer {
	return &reassigningFailureDomainBalancer{delegate}
}

func (r *reassigningFailureDomainBalancer) Assign(ctx context.Context, csMachine *infrav1.CloudStackMachine, capiMachine *clusterv1.Machine, fds []infrav1.CloudStackFailureDomainSpec) error {
	if csMachine.DeletionTimestamp != nil {
		return nil
	}

	logger := log.FromContext(ctx)
	logger.Info("Checking failure domain for machine", "machineHasFailed", csMachine.HasFailed(), "currentFailureDomain", csMachine.Spec.FailureDomainName)

	if csMachine.Spec.FailureDomainName != "" && !csMachine.HasFailed() {
		return nil
	}

	if capiMachineHasFailureDomain(capiMachine) && !csMachine.HasFailed() {
		csMachine.Spec.FailureDomainName = *capiMachine.Spec.FailureDomain
		assignFailureDomainLabel(csMachine, capiMachine)
	} else if err := r.delegate.Assign(ctx, csMachine, capiMachine, fds); err != nil {
		return err
	}

	if capiMachineHasFailureDomain(capiMachine) && csMachine.HasFailed() {
		capiMachine.Spec.FailureDomain = pointer.String(csMachine.Spec.FailureDomainName)
	}

	return nil
}

type fallingBackFailureDomainBalancer struct {
	primary  Balancer
	fallback Balancer
}

func newFallingBackFailureDomainBalancer(primary Balancer, fallback Balancer) Balancer {
	return &fallingBackFailureDomainBalancer{primary, fallback}
}

func (f *fallingBackFailureDomainBalancer) Assign(ctx context.Context, csMachine *infrav1.CloudStackMachine, capiMachine *clusterv1.Machine, fds []infrav1.CloudStackFailureDomainSpec) error {
	if err := f.primary.Assign(ctx, csMachine, capiMachine, fds); err != nil {
		logger := log.FromContext(ctx)
		logger.Info("Unable to assign failure domain, falling back to the secondary balancer", "error", err)
		return f.fallback.Assign(ctx, csMachine, capiMachine, fds)
	}

	return nil
}

type zoneIPCounts struct {
	name        string
	zoneName    string
	networkName string
	totalIPs    int
	freeIPs     int
}

type randomFailureDomainBalancer struct {
}

func newRandomFailureDomainBalancer() Balancer {
	return &randomFailureDomainBalancer{}
}

func (r *randomFailureDomainBalancer) Assign(_ context.Context, csMachine *infrav1.CloudStackMachine, capiMachine *clusterv1.Machine, fds []infrav1.CloudStackFailureDomainSpec) error {
	randNum := rand.Int() % len(fds) // #nosec G404 -- weak crypt rand doesn't matter here.
	csMachine.Spec.FailureDomainName = fds[randNum].Name
	assignFailureDomainLabel(csMachine, capiMachine)

	return nil
}

type freeIPValidatingFailureDomainBalancer struct {
	csClientFactory ClientFactory
}

func newFreeIPValidatingFailureDomainBalancer(csClientFactory ClientFactory) Balancer {
	return &freeIPValidatingFailureDomainBalancer{csClientFactory: csClientFactory}
}

type networkKey string

func (b *freeIPValidatingFailureDomainBalancer) Assign(ctx context.Context, csMachine *infrav1.CloudStackMachine, capiMachine *clusterv1.Machine, fds []infrav1.CloudStackFailureDomainSpec) error {
	networkAndZones, err := b.discoverFreeIps(ctx, fds)
	if err != nil {
		return err
	}

	zonesWithMostIps := findNetworkWithFreeIps(networkAndZones)
	if len(zonesWithMostIps) > 0 {
		randNum := rand.Int() % len(zonesWithMostIps) // #nosec G404 -- weak crypt rand doesn't matter here.
		selectedZone := zonesWithMostIps[randNum]
		csMachine.Spec.FailureDomainName = selectedZone.name
		assignFailureDomainLabel(csMachine, capiMachine)
	}

	if csMachine.Spec.FailureDomainName == "" {
		return fmt.Errorf("failed to assign failure domain, no failure domain with free IP addresses found")
	}

	return nil
}

func (b *freeIPValidatingFailureDomainBalancer) discoverFreeIps(ctx context.Context, fds []infrav1.CloudStackFailureDomainSpec) (map[networkKey][]zoneIPCounts, error) {
	counts := map[networkKey][]zoneIPCounts{}

	logger := log.FromContext(ctx)
	logger.Info("Finding failure domains with most free IPs", "fds", fds)

	for _, fd := range fds {
		fdSpec := fd

		_, csUser, err := b.csClientFactory.GetCloudClientAndUser(ctx, &fdSpec)
		if err != nil {
			return nil, fmt.Errorf("failed to get CS client for failure domain %s: %sv", fd.Name, err)
		}

		network := fd.Zone.Network.DeepCopy()
		if network.ID == "" {
			if err := csUser.ResolveNetwork(network); err != nil {
				return nil, fmt.Errorf("failed to resolve failure domain network %s: %sv", fd.Name, err)
			}
		}

		logger.Info("Resolved failure domain network", "network", network)
		if network.Type == cloud.NetworkTypeIsolated {
			continue
		}

		addresses, err := csUser.GetPublicIPs(network)
		if err != nil {
			return nil, fmt.Errorf("failed to determine free IP addressed for failure domain %s: %w", fd.Name, err)
		}

		countSummary := zoneIPCounts{
			name:        fd.Name,
			zoneName:    fd.Zone.Name,
			networkName: network.Name,
			totalIPs:    len(addresses),
			freeIPs:     countFreeIps(addresses),
		}
		logger.Info("Resolved failure domain network IP count summary", "domain", fd.Name, "totalIPs", countSummary.totalIPs, "freeIPs", countSummary.freeIPs)

		key := networkKey(network.Name)
		if _, ok := counts[key]; !ok {
			counts[key] = []zoneIPCounts{}
		}
		counts[key] = append(counts[key], countSummary)
	}

	return counts, nil
}

func findNetworkWithFreeIps(counts map[networkKey][]zoneIPCounts) []zoneIPCounts {
	var keyWithMaxFreeIps networkKey
	var maxFreeIps int

	for key, cs := range counts {
		for _, c := range cs {
			if c.freeIPs > maxFreeIps {
				maxFreeIps = c.freeIPs
				keyWithMaxFreeIps = key
			}
		}
	}

	return counts[keyWithMaxFreeIps]
}

func countFreeIps(addresses []*cloudstack.PublicIpAddress) int {
	free := 0
	for _, address := range addresses {
		if address.State == "Free" {
			free++
		}
	}
	return free
}

func capiMachineHasFailureDomain(capiMachine *clusterv1.Machine) bool {
	return capiMachine != nil && capiMachine.Spec.FailureDomain != nil &&
		(util.IsControlPlaneMachine(capiMachine) || // Is control plane machine -- CAPI will specify.
			*capiMachine.Spec.FailureDomain != "") // Or potentially another machine controller specified.
}

func assignFailureDomainLabel(csMachine *infrav1.CloudStackMachine, capiMachine *clusterv1.Machine) {
	if csMachine.Labels == nil {
		csMachine.Labels = map[string]string{}
	}
	csMachine.Labels[infrav1.FailureDomainLabelName] = infrav1.FailureDomainHashedMetaName(csMachine.Spec.FailureDomainName, capiMachine.Spec.ClusterName)
}
