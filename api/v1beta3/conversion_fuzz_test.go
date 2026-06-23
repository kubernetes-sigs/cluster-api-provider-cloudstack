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

package v1beta3_test

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	infrav3 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	infrav4 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta4"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
)

// TestFuzzyConversion verifies that converting a v1beta3 spoke object up to the v1beta4 hub
// and back (and the reverse) is lossless, with v1beta4-only fields preserved via the
// conversion-data annotation.
func TestFuzzyConversion(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(infrav3.AddToScheme(scheme))
	utilruntime.Must(infrav4.AddToScheme(scheme))

	t.Run("for CloudStackCluster", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme: scheme,
		Hub:    &infrav4.CloudStackCluster{},
		Spoke:  &infrav3.CloudStackCluster{},
	}))

	t.Run("for CloudStackMachine", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme: scheme,
		Hub:    &infrav4.CloudStackMachine{},
		Spoke:  &infrav3.CloudStackMachine{},
	}))

	t.Run("for CloudStackMachineTemplate", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme: scheme,
		Hub:    &infrav4.CloudStackMachineTemplate{},
		Spoke:  &infrav3.CloudStackMachineTemplate{},
	}))

	t.Run("for CloudStackFailureDomain", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme: scheme,
		Hub:    &infrav4.CloudStackFailureDomain{},
		Spoke:  &infrav3.CloudStackFailureDomain{},
	}))

	t.Run("for CloudStackIsolatedNetwork", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme: scheme,
		Hub:    &infrav4.CloudStackIsolatedNetwork{},
		Spoke:  &infrav3.CloudStackIsolatedNetwork{},
	}))

	t.Run("for CloudStackAffinityGroup", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme: scheme,
		Hub:    &infrav4.CloudStackAffinityGroup{},
		Spoke:  &infrav3.CloudStackAffinityGroup{},
	}))

	t.Run("for CloudStackMachineStateChecker", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme: scheme,
		Hub:    &infrav4.CloudStackMachineStateChecker{},
		Spoke:  &infrav3.CloudStackMachineStateChecker{},
	}))
}
