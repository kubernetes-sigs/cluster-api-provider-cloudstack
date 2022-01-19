/*
Copyright 2022.

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

import infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1alpha3"

type AffinityGroup struct {
	AntiAffinity bool
	Name         string
}

type AffinityGroupIFace interface {
	GetOrCreateAffinityGroup(*infrav1.CloudStackCluster, AffinityGroup) error
	DeleteAffinityGroup(*infrav1.CloudStackCluster, AffinityGroup) error
	AssociateAffinityGroup(*infrav1.CloudStackMachine, AffinityGroup) error
	DissassociateAffinityGroup(*infrav1.CloudStackMachine, AffinityGroup) error
}

func (c *client) GetOrCreateAffinityGroup(csCluster *infrav1.CloudStackCluster, group AffinityGroup) (retErr error) {
	return nil
}

func (c *client) DeleteAffinityGroup(csCluster *infrav1.CloudStackCluster, group AffinityGroup) (retErr error) {
	return nil
}

func (c *client) AssociateAffinityGroup(csCluster *infrav1.CloudStackMachine, group AffinityGroup) (retErr error) {
	return nil
}

func (c *client) DissassociateAffinityGroup(csCluster *infrav1.CloudStackMachine, group AffinityGroup) (retErr error) {
	return nil
}
