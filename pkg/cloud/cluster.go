/*
Copyright 2023 The Kubernetes Authors.

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

package cloud

import (
	"fmt"
	"strings"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/pkg/errors"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

type ClusterIface interface {
	GetOrCreateCluster(*clusterv1.Cluster, *infrav1.CloudStackCluster, *infrav1.CloudStackFailureDomainSpec) error
	DeleteCluster(*infrav1.CloudStackCluster) error
	AddVMToCluster(*infrav1.CloudStackCluster, *infrav1.CloudStackMachine) error
	RemoveVMFromCluster(*infrav1.CloudStackCluster, *infrav1.CloudStackMachine) error
}

type ClustertypeSetter interface {
	SetClustertype(string)
}

func withExternalManaged() cloudstack.OptionFunc {
	return func(cs *cloudstack.CloudStackClient, p interface{}) error {
		ps, ok := p.(ClustertypeSetter)
		if !ok {
			return errors.New("invalid params type")
		}
		ps.SetClustertype("ExternalManaged")
		return nil
	}
}

func (c *client) GetOrCreateCluster(cluster *clusterv1.Cluster, csCluster *infrav1.CloudStackCluster, fd *infrav1.CloudStackFailureDomainSpec) error {
	// Get cluster
	if csCluster.Status.CloudStackClusterID != "" {
		_, count, err := c.cs.Kubernetes.GetKubernetesClusterByID(csCluster.Status.CloudStackClusterID, withExternalManaged())
		if err != nil {
			return err
		}
		if count == 1 {
			return nil
		}
	}

	// Check if a cluster exists with the same name
	clusterName := fmt.Sprintf("%s - %s", cluster.GetName(), csCluster.GetName())
	csUnmanagedCluster, count, err := c.cs.Kubernetes.GetKubernetesClusterByName(clusterName, withExternalManaged())
	if err != nil && !strings.Contains(err.Error(), "No match found for ") {
		return err
	}
	if count <= 0 {
		// Create cluster
		domain := Domain{Path: rootDomain}
		if csCluster.Spec.FailureDomains[0].Domain != "" {
			domain.Path = fd.Domain
		}
		_ = c.ResolveDomain(&domain)

		accountName := csCluster.Spec.FailureDomains[0].Account
		if accountName == "" {
			userParams := c.cs.User.NewGetUserParams(c.config.APIKey)
			user, err := c.cs.User.GetUser(userParams)
			if err != nil {
				return err
			}
			accountName = user.Account
		}
		params := c.cs.Kubernetes.NewCreateKubernetesClusterParams(fmt.Sprintf("%s managed by CAPC", clusterName), clusterName, fd.Zone.ID)

		setIfNotEmpty(accountName, params.SetAccount)
		setIfNotEmpty(domain.ID, params.SetDomainid)
		setIfNotEmpty(fd.Zone.Network.ID, params.SetNetworkid)
		setIfNotEmpty(csCluster.Spec.ControlPlaneEndpoint.Host, params.SetExternalloadbalanceripaddress)
		params.SetClustertype("ExternalManaged")

		r, err := c.cs.Kubernetes.CreateKubernetesCluster(params)
		if err != nil {
			return err
		}
		csUnmanagedCluster, count, err = c.cs.Kubernetes.GetKubernetesClusterByID(r.Id)
		if err != nil {
			return err
		}
		if count == 0 {
			return errors.New("cluster not found")
		}
	}
	csCluster.Status.CloudStackClusterID = csUnmanagedCluster.Id
	return nil
}

func (c *client) DeleteCluster(csCluster *infrav1.CloudStackCluster) error {
	if csCluster.Status.CloudStackClusterID != "" {
		csUnmanagedCluster, count, err := c.cs.Kubernetes.GetKubernetesClusterByID(csCluster.Status.CloudStackClusterID, withExternalManaged())
		if err != nil {
			return err
		}
		if count != 0 {
			params := c.cs.Kubernetes.NewDeleteKubernetesClusterParams(csUnmanagedCluster.Id)
			_, err = c.cs.Kubernetes.DeleteKubernetesCluster(params)
			if err != nil {
				return err
			}
		}
		csCluster.Status.CloudStackClusterID = ""
		return nil
	}
	return nil
}

func (c *client) AddVMToCluster(csCluster *infrav1.CloudStackCluster, csMachine *infrav1.CloudStackMachine) error {
	if csCluster.Status.CloudStackClusterID != "" {
		params := c.cs.Kubernetes.NewAddVirtualMachinesToKubernetesClusterParams(csCluster.Status.CloudStackClusterID, []string{*csMachine.Spec.InstanceID})
		_, err := c.cs.Kubernetes.AddVirtualMachinesToKubernetesCluster(params)
		return err
	}
	return nil
}

func (c *client) RemoveVMFromCluster(csCluster *infrav1.CloudStackCluster, csMachine *infrav1.CloudStackMachine) error {
	if csCluster.Status.CloudStackClusterID != "" {
		params := c.cs.Kubernetes.NewRemoveVirtualMachinesFromKubernetesClusterParams(csCluster.Status.CloudStackClusterID, []string{*csMachine.Spec.InstanceID})
		_, err := c.cs.Kubernetes.RemoveVirtualMachinesFromKubernetesCluster(params)
		return err
	}
	return nil
}
