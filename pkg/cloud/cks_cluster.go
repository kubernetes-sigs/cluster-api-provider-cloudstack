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
	GetOrCreateCksCluster(*clusterv1.Cluster, *infrav1.CloudStackCluster, *infrav1.CloudStackFailureDomainSpec) error
	DeleteCksCluster(*infrav1.CloudStackCluster) error
	AddVMToCksCluster(*infrav1.CloudStackCluster, *infrav1.CloudStackMachine) error
	RemoveVMFromCksCluster(*infrav1.CloudStackCluster, *infrav1.CloudStackMachine) error
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

func (c *client) GetOrCreateCksCluster(cluster *clusterv1.Cluster, csCluster *infrav1.CloudStackCluster, fd *infrav1.CloudStackFailureDomainSpec) error {
	// Get cluster
	if csCluster.Status.CloudStackClusterID != "" {
		externalManagedCluster, count, err := c.cs.Kubernetes.GetKubernetesClusterByID(csCluster.Status.CloudStackClusterID, withExternalManaged(), cloudstack.WithProject(c.user.Project.ID))
		if err != nil {
			return err
		} else if count > 0 {
			csCluster.Status.CloudStackClusterID = externalManagedCluster.Id
			return nil
		}
	}

	// Check if a cluster exists with the same name
	clusterName := fmt.Sprintf("%s - %s - %s", cluster.GetName(), csCluster.GetName(), csCluster.GetUID())
	externalManagedCluster, count, err := c.cs.Kubernetes.GetKubernetesClusterByName(clusterName, withExternalManaged(), cloudstack.WithProject(c.user.Project.ID))
	if err != nil && !strings.Contains(err.Error(), "No match found for ") {
		return err
	}
	if count > 0 {
		csCluster.Status.CloudStackClusterID = externalManagedCluster.Id
	} else if err == nil || (err != nil && strings.Contains(err.Error(), "No match found for ")) {
		// Create cluster
		accountName := csCluster.Spec.FailureDomains[0].Account
		if accountName == "" {
			userParams := c.cs.User.NewGetUserParams(c.config.APIKey)
			user, err := c.cs.User.GetUser(userParams)
			if err != nil {
				return err
			}
			accountName = user.Account
		}
		// NewCreateKubernetesClusterParams(description string, kubernetesversionid string, name string, serviceofferingid string, size int64, zoneid string) *CreateKubernetesClusterParams
		params := c.cs.Kubernetes.NewCreateKubernetesClusterParams(fmt.Sprintf("%s managed by CAPC", clusterName), "", clusterName, "", 0, fd.Zone.ID)

		setIfNotEmpty(c.user.Project.ID, params.SetProjectid)
		setIfNotEmpty(accountName, params.SetAccount)
		setIfNotEmpty(c.user.Domain.ID, params.SetDomainid)
		setIfNotEmpty(fd.Zone.Network.ID, params.SetNetworkid)
		setIfNotEmpty(csCluster.Spec.ControlPlaneEndpoint.Host, params.SetExternalloadbalanceripaddress)
		params.ResetKubernetesversionid()
		params.ResetServiceofferingid()
		params.SetClustertype("ExternalManaged")

		cloudStackCKSCluster, err := c.cs.Kubernetes.CreateKubernetesCluster(params)
		if err != nil {
			return err
		}
		csCluster.Status.CloudStackClusterID = cloudStackCKSCluster.Id
	}
	return nil
}

func (c *client) DeleteCksCluster(csCluster *infrav1.CloudStackCluster) error {
	if csCluster.Status.CloudStackClusterID != "" {
		csCksCluster, count, err := c.cs.Kubernetes.GetKubernetesClusterByID(csCluster.Status.CloudStackClusterID, withExternalManaged())
		if err != nil && strings.Contains(err.Error(), " not found") {
			return nil
		}
		if count != 0 {
			params := c.cs.Kubernetes.NewDeleteKubernetesClusterParams(csCksCluster.Id)
			_, err = c.cs.Kubernetes.DeleteKubernetesCluster(params)
			if err != nil {
				return err
			}
		}
		csCluster.Status.CloudStackClusterID = ""
	}
	return nil
}

func (c *client) AddVMToCksCluster(csCluster *infrav1.CloudStackCluster, csMachine *infrav1.CloudStackMachine) error {
	if csCluster.Status.CloudStackClusterID != "" {
		params := c.cs.Kubernetes.NewAddVirtualMachinesToKubernetesClusterParams(csCluster.Status.CloudStackClusterID, []string{*csMachine.Spec.InstanceID})
		if csMachine.Labels != nil {
			_, ok := csMachine.Labels[clusterv1.MachineControlPlaneLabel]
			params.SetIscontrolnode(ok)
		}

		_, err := c.cs.Kubernetes.AddVirtualMachinesToKubernetesCluster(params)
		return err
	}
	return nil
}

func (c *client) RemoveVMFromCksCluster(csCluster *infrav1.CloudStackCluster, csMachine *infrav1.CloudStackMachine) error {
	if csCluster.Status.CloudStackClusterID != "" {
		params := c.cs.Kubernetes.NewRemoveVirtualMachinesFromKubernetesClusterParams(csCluster.Status.CloudStackClusterID, []string{*csMachine.Spec.InstanceID})
		_, err := c.cs.Kubernetes.RemoveVirtualMachinesFromKubernetesCluster(params)
		return err
	}
	return nil
}
