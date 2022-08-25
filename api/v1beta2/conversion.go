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

// Hub marks CloudStackCluster as a conversion hub.
func (*CloudStackCluster) Hub() {}

// Hub marks CloudStackClusterList as a conversion hub.
func (*CloudStackClusterList) Hub() {}

// Hub marks CloudStackMachine as a conversion hub.
func (*CloudStackMachine) Hub() {}

// Hub marks CloudStackMachineList as a conversion hub.
func (*CloudStackMachineList) Hub() {}

// Hub marks CloudStackMachineTemplate as a conversion hub.
func (*CloudStackMachineTemplate) Hub() {}

// Hub marks CloudStackMachineTemplateList as a conversion hub.
func (*CloudStackMachineTemplateList) Hub() {}

// Hub marks CloudStackIsolatedNetwork as a conversion hub.
func (*CloudStackIsolatedNetwork) Hub() {}

// Hub marks CloudStackIsolatedNetworkList as a conversion hub.
func (*CloudStackIsolatedNetworkList) Hub() {}

// Hub marks CloudStackFailureDomain as a conversion hub.
func (*CloudStackFailureDomain) Hub() {}

// Hub marks CloudStackFailureDomainList as a conversion hub.
func (*CloudStackFailureDomainList) Hub() {}

// Hub marks CloudStackAffinityGroup as a conversion hub.
func (*CloudStackAffinityGroup) Hub() {}

// Hub marks CloudStackAffinityGroupList as a conversion hub.
func (*CloudStackAffinityGroupList) Hub() {}

// Hub marks CloudStackMachineStateChecker as a conversion hub.
func (*CloudStackMachineStateChecker) Hub() {}

// Hub marks CloudStackMachineStateCheckerList as a conversion hub.
func (*CloudStackMachineStateCheckerList) Hub() {}
