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

package v1alpha3

// Hub marks CloudStackCluster as a conversion hub.
func (*CloudStackCluster) Hub() {}

// Hub marks CloudStackClusterList as a conversion hub.
func (*CloudStackClusterList) Hub() {}

// Hub marks CloudStackMachine as a conversion hub.
func (*CloudStackMachine) Hub() {}

// Hub marks CloudStackMachineList as a conversion hub.
func (*CloudStackMachineList) Hub() {}
