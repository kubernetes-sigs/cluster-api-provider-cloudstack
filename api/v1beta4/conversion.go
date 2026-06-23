/*
Copyright 2024 The Kubernetes Authors.

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

package v1beta4

// Hub marks CloudStackCluster as the conversion hub.
func (*CloudStackCluster) Hub() {}

// Hub marks CloudStackClusterList as the conversion hub.
func (*CloudStackClusterList) Hub() {}

// Hub marks CloudStackMachine as the conversion hub.
func (*CloudStackMachine) Hub() {}

// Hub marks CloudStackMachineList as the conversion hub.
func (*CloudStackMachineList) Hub() {}

// Hub marks CloudStackMachineTemplate as the conversion hub.
func (*CloudStackMachineTemplate) Hub() {}

// Hub marks CloudStackMachineTemplateList as the conversion hub.
func (*CloudStackMachineTemplateList) Hub() {}

// Hub marks CloudStackIsolatedNetwork as the conversion hub.
func (*CloudStackIsolatedNetwork) Hub() {}

// Hub marks CloudStackIsolatedNetworkList as the conversion hub.
func (*CloudStackIsolatedNetworkList) Hub() {}

// Hub marks CloudStackFailureDomain as the conversion hub.
func (*CloudStackFailureDomain) Hub() {}

// Hub marks CloudStackFailureDomainList as the conversion hub.
func (*CloudStackFailureDomainList) Hub() {}

// Hub marks CloudStackAffinityGroup as the conversion hub.
func (*CloudStackAffinityGroup) Hub() {}

// Hub marks CloudStackAffinityGroupList as the conversion hub.
func (*CloudStackAffinityGroupList) Hub() {}

// Hub marks CloudStackMachineStateChecker as the conversion hub.
func (*CloudStackMachineStateChecker) Hub() {}

// Hub marks CloudStackMachineStateCheckerList as the conversion hub.
func (*CloudStackMachineStateCheckerList) Hub() {}
