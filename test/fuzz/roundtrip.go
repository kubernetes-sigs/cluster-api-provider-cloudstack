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

// Package fuzz provides a Ginkgo-friendly conversion round-trip assertion.
package fuzz

import (
	"github.com/google/go-cmp/cmp"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// iterations matches the fuzz count used by
// sigs.k8s.io/cluster-api/util/conversion.FuzzTestFunc.
const iterations = 10000

// RoundTrip asserts that converting between the given spoke and hub is lossless in both
// directions (spoke→hub→spoke and hub→spoke→hub), with hub-only fields preserved through the
// conversion-data annotation.
//
// It mirrors sigs.k8s.io/cluster-api/util/conversion.FuzzTestFunc, which can't be reused here
// because it requires a *testing.T and is therefore incompatible with Ginkgo specs. Call it from
// within a Ginkgo spec; it relies on the global Gomega registered by RunSpecs.
func RoundTrip(scheme *runtime.Scheme, hub conversion.Hub, spoke conversion.Convertible, funcs ...fuzzer.FuzzerFuncs) {
	ginkgo.GinkgoHelper()
	fz := utilconversion.GetFuzzer(scheme, funcs...)

	// spoke → hub → spoke
	for range iterations {
		spokeBefore := spoke.DeepCopyObject().(conversion.Convertible)
		fz.Fill(spokeBefore)

		hubCopy := hub.DeepCopyObject().(conversion.Hub)
		gomega.Expect(spokeBefore.ConvertTo(hubCopy)).To(gomega.Succeed())

		spokeAfter := spoke.DeepCopyObject().(conversion.Convertible)
		gomega.Expect(spokeAfter.ConvertFrom(hubCopy)).To(gomega.Succeed())

		// Drop the annotation ConvertFrom added so it doesn't skew the comparison.
		if obj, ok := spokeAfter.(metav1.Object); ok {
			delete(obj.GetAnnotations(), utilconversion.DataAnnotation)
		}

		gomega.Expect(apiequality.Semantic.DeepEqual(spokeBefore, spokeAfter)).To(gomega.BeTrue(),
			func() string { return "spoke→hub→spoke is lossy:\n" + cmp.Diff(spokeBefore, spokeAfter) })
	}

	// hub → spoke → hub
	for range iterations {
		hubBefore := hub.DeepCopyObject().(conversion.Hub)
		fz.Fill(hubBefore)

		// DeepCopy hubBefore because ConvertFrom may mutate its input while marshalling the
		// conversion-data annotation.
		spokeCopy := spoke.DeepCopyObject().(conversion.Convertible)
		gomega.Expect(spokeCopy.ConvertFrom(hubBefore.DeepCopyObject().(conversion.Hub))).To(gomega.Succeed())

		hubAfter := hub.DeepCopyObject().(conversion.Hub)
		gomega.Expect(spokeCopy.ConvertTo(hubAfter)).To(gomega.Succeed())

		gomega.Expect(apiequality.Semantic.DeepEqual(hubBefore, hubAfter)).To(gomega.BeTrue(),
			func() string { return "hub→spoke→hub is lossy:\n" + cmp.Diff(hubBefore, hubAfter) })
	}
}
