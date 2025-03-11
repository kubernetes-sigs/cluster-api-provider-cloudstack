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

package utils_test

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	"sigs.k8s.io/cluster-api-provider-cloudstack/controllers/utils"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta3"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// mockConcreteRunner is a minimal implementation of the runner interface
type mockConcreteRunner struct{}

func (m *mockConcreteRunner) ReconcileDelete() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (m *mockConcreteRunner) Reconcile() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

var _ = Describe("ReconciliationRunner", func() {
	var (
		mockCtrl   *gomock.Controller
		k8sClient  client.Client
		scheme     *runtime.Scheme
		baseRunner *utils.ReconciliationRunner
		ctx        context.Context
		mockRunner *mockConcreteRunner
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		scheme = runtime.NewScheme()
		Expect(infrav1.AddToScheme(scheme)).To(Succeed())
		Expect(clusterv1.AddToScheme(scheme)).To(Succeed())

		k8sClient = fake.NewClientBuilder().WithScheme(scheme).Build()
		ctx = context.Background()
		mockRunner = &mockConcreteRunner{}

		// Create the base reconciler
		base := utils.ReconcilerBase{
			K8sClient:  k8sClient,
			Scheme:     scheme,
			BaseLogger: logr.Discard(),
			Recorder:   record.NewFakeRecorder(10),
		}

		// Create a reconciliation runner with our mock concrete runner
		baseRunner = utils.NewRunner(mockRunner, &infrav1.CloudStackMachine{}, "TestController")
		baseRunner.UsingBaseReconciler(base)
		baseRunner.WithRequestCtx(ctx)

		// Setup a fake Request
		baseRunner.ForRequest(ctrl.Request{
			NamespacedName: client.ObjectKey{
				Namespace: "default",
				Name:      "test-machine",
			},
		})
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("GetParent", func() {
		var (
			child *infrav1.CloudStackMachine
		)

		BeforeEach(func() {
			dummies.SetDummyVars()

			// Set up child object
			child = &infrav1.CloudStackMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-child",
					Namespace: "default",
				},
			}
		})

		Context("when parent exists and is correctly referenced", func() {
			var parent *clusterv1.Machine
			BeforeEach(func() {
				parent = &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-parent",
						Namespace: "default",
					},
				}
				// Set up owner reference
				child.OwnerReferences = []metav1.OwnerReference{
					{
						APIVersion: clusterv1.GroupVersion.String(),
						Kind:       "Machine",
						Name:       parent.Name,
						UID:        "test-uid",
					},
				}

				// Create the objects in the fake client
				Expect(k8sClient.Create(ctx, parent)).To(Succeed())
				Expect(k8sClient.Create(ctx, child)).To(Succeed())
			})

			It("should find the parent successfully", func() {
				// Create an empty parent object to be filled
				parentToFind := &clusterv1.Machine{}

				// Call GetParent
				result, err := baseRunner.GetParent(child, parentToFind)()

				// Check results
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{}))
				Expect(parentToFind.Name).To(Equal(parent.Name))
				Expect(parentToFind.Namespace).To(Equal(parent.Namespace))
			})
		})

		Context("when parent doesn't exist", func() {
			BeforeEach(func() {
				// Set up owner reference to non-existent parent
				child.OwnerReferences = []metav1.OwnerReference{
					{
						APIVersion: clusterv1.GroupVersion.String(),
						Kind:       "Machine",
						Name:       "non-existent-parent",
						UID:        "test-uid",
					},
				}

				// Create only the child in the fake client
				Expect(k8sClient.Create(ctx, child)).To(Succeed())
			})

			It("should return an error", func() {
				// Create an empty parent object to be filled
				parentToFind := &clusterv1.Machine{}

				// Call GetParent
				_, err := baseRunner.GetParent(child, parentToFind)()

				// Check results
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not found"))
			})
		})

		Context("when no owner reference of requested kind exists", func() {
			BeforeEach(func() {
				// Set up owner reference to different kind
				child.OwnerReferences = []metav1.OwnerReference{
					{
						APIVersion: "different.api/v1",
						Kind:       "DifferentKind",
						Name:       "different-name",
						UID:        "test-uid",
					},
				}

				// Create only the child in the fake client
				Expect(k8sClient.Create(ctx, child)).To(Succeed())
			})

			It("should requeue with error message", func() {
				// Create an empty parent object to be filled
				parentToFind := &clusterv1.Machine{}

				// Call GetParent
				result, err := baseRunner.GetParent(child, parentToFind)()

				// Check results
				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(Equal(utils.RequeueTimeout))
			})
		})
	})
})
