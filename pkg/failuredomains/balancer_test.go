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
	"strings"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/mocks"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var _ = Describe("Load Balancer", func() {

	var (
		mockCtrl     *gomock.Controller
		mockCSClient *mocks.MockClient

		ctx         context.Context
		capiMachine *clusterv1.Machine
		csMachine   *infrav1.CloudStackMachine
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockCSClient = mocks.NewMockClient(mockCtrl)

		ctx = context.TODO()
		capiMachine = &clusterv1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "a-machine",
				Namespace: "default",
			},
			Spec: clusterv1.MachineSpec{},
		}
		csMachine = &infrav1.CloudStackMachine{
			Spec: infrav1.CloudStackMachineSpec{},
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("reassigning balancer", func() {
		It("assign a failure domain when NOT already assigned", func() {
			balancer := newReassigningFailureDomainBalancer(newRandomFailureDomainBalancer())

			fds := []infrav1.CloudStackFailureDomainSpec{
				{Name: "zone-a"},
			}

			Expect(balancer.Assign(ctx, csMachine, capiMachine, fds)).Should(Succeed())
			Expect(csMachine.Spec.FailureDomainName).ShouldNot(BeEmpty())
		})

		It("re-assign a failure domain when already assigned but machine has failed to launch", func() {
			delegate := &fakeBalancer{"zone-b"}
			balancer := newReassigningFailureDomainBalancer(delegate)

			fds := []infrav1.CloudStackFailureDomainSpec{
				{Name: "zone-a"},
				{Name: "zone-b"},
			}

			csMachine.Spec.FailureDomainName = fds[0].Name
			csMachine.MarkAsFailed()

			Expect(balancer.Assign(ctx, csMachine, capiMachine, fds)).Should(Succeed())
			Expect(csMachine.Spec.FailureDomainName).Should(Equal(fds[1].Name))
		})

		It("re-assign a failure domain AND update CAPI machine when already assigned but machine has failed to launch", func() {
			delegate := &fakeBalancer{"zone-b"}

			balancer := newReassigningFailureDomainBalancer(delegate)

			fds := []infrav1.CloudStackFailureDomainSpec{
				{Name: "zone-a"},
				{Name: "zone-b"},
			}

			capiMachine.Spec.FailureDomain = pointer.String(fds[0].Name)
			csMachine.Spec.FailureDomainName = fds[0].Name
			csMachine.MarkAsFailed()

			Expect(balancer.Assign(ctx, csMachine, capiMachine, fds)).Should(Succeed())
			Expect(csMachine.Spec.FailureDomainName).Should(Equal(fds[1].Name))
		})

		It("should NOT re-assign a failure domain when already assigned", func() {
			balancer := newReassigningFailureDomainBalancer(newRandomFailureDomainBalancer())

			fds := []infrav1.CloudStackFailureDomainSpec{
				{Name: "zone-a"},
				{Name: "zone-b"},
				{Name: "zone-c"},
			}

			csMachine.Spec.FailureDomainName = fds[0].Name

			Expect(balancer.Assign(ctx, csMachine, capiMachine, fds)).Should(Succeed())
			Expect(csMachine.Spec.FailureDomainName).Should(Equal(fds[0].Name))
		})
	})

	Context("falling back balancer", func() {
		It("fallback to secondary balancer", func() {
			balancer := newFallingBackFailureDomainBalancer(
				newFailingDomainBalancer(fmt.Errorf("failed to assign a failure domain")),
				newRandomFailureDomainBalancer(),
			)

			fds := []infrav1.CloudStackFailureDomainSpec{
				{Name: "zone-a"},
				{Name: "zone-b"},
			}

			Expect(balancer.Assign(ctx, csMachine, capiMachine, fds)).Should(Succeed())
			Expect(csMachine.Spec.FailureDomainName).ShouldNot(BeEmpty())
		})

		It("should NOT assign a failure domain because all balancers failed", func() {
			balancer := newFallingBackFailureDomainBalancer(
				newFailingDomainBalancer(fmt.Errorf("primary failed to assign a failure domain")),
				newFailingDomainBalancer(fmt.Errorf("secondary failed to assign a failure domain")),
			)

			fds := []infrav1.CloudStackFailureDomainSpec{
				{Name: "zone-a"},
				{Name: "zone-b"},
			}

			Expect(balancer.Assign(ctx, csMachine, capiMachine, fds)).Should(MatchError("secondary failed to assign a failure domain"))
		})
	})

	Context("free ip addresses validating balancer", func() {
		network1 := "172.16.0.0/24"
		network2 := "172.16.0.1/24"

		It("assign a failure domain with more free IPs", func() {
			balancer := newFreeIPValidatingFailureDomainBalancer(&fakeFailureDomainClientFactory{mockCSClient})

			fds := []infrav1.CloudStackFailureDomainSpec{
				{Name: "zone-a", Zone: infrav1.CloudStackZoneSpec{Network: infrav1.Network{Name: network1}}},
				{Name: "zone-b", Zone: infrav1.CloudStackZoneSpec{Network: infrav1.Network{Name: network2}}},
			}

			// zone-a
			mockCSClient.EXPECT().ResolveNetwork(&fds[0].Zone.Network)
			mockCSClient.EXPECT().GetPublicIPs(&fds[0].Zone.Network).Return([]*cloudstack.PublicIpAddress{
				{State: "Allocated"},
				{State: "Free"},
			}, nil)

			// zone-b
			mockCSClient.EXPECT().ResolveNetwork(&fds[1].Zone.Network)
			mockCSClient.EXPECT().GetPublicIPs(&fds[1].Zone.Network).Return([]*cloudstack.PublicIpAddress{
				{State: "Allocated"},
				{State: "Free"},
				{State: "Free"},
			}, nil)

			Expect(balancer.Assign(ctx, csMachine, capiMachine, fds)).Should(Succeed())
			Expect(csMachine.Spec.FailureDomainName).Should(Equal("zone-b"))
		})

		It("assign a failure domain when there are zones with shared networks", func() {
			balancer := newFreeIPValidatingFailureDomainBalancer(&fakeFailureDomainClientFactory{mockCSClient})

			fds := []infrav1.CloudStackFailureDomainSpec{
				{Name: "zone-a-net-1", Zone: infrav1.CloudStackZoneSpec{Network: infrav1.Network{Name: network1}}},
				{Name: "zone-a-net-2", Zone: infrav1.CloudStackZoneSpec{Network: infrav1.Network{Name: network2}}},
				{Name: "zone-b-net-1", Zone: infrav1.CloudStackZoneSpec{Network: infrav1.Network{Name: network1}}},
				{Name: "zone-b-net-2", Zone: infrav1.CloudStackZoneSpec{Network: infrav1.Network{Name: network2}}},
			}

			// zone-a
			mockCSClient.EXPECT().ResolveNetwork(&fds[0].Zone.Network).Times(2)
			mockCSClient.EXPECT().GetPublicIPs(&fds[0].Zone.Network).Return([]*cloudstack.PublicIpAddress{
				{State: "Allocated"},
				{State: "Free"},
			}, nil).Times(2)

			// zone-b
			mockCSClient.EXPECT().ResolveNetwork(&fds[1].Zone.Network).Times(2)
			mockCSClient.EXPECT().GetPublicIPs(&fds[1].Zone.Network).Return([]*cloudstack.PublicIpAddress{
				{State: "Allocated"},
				{State: "Free"},
				{State: "Free"},
			}, nil).Times(2)

			Expect(balancer.Assign(ctx, csMachine, capiMachine, fds)).Should(Succeed())
			Expect(strings.HasSuffix(csMachine.Spec.FailureDomainName, "net-2")).Should(BeTrue())
		})

		It("assign the first non-zero failure domain with free IPs", func() {
			balancer := newFreeIPValidatingFailureDomainBalancer(&fakeFailureDomainClientFactory{mockCSClient})

			fds := []infrav1.CloudStackFailureDomainSpec{
				{Name: "zone-a", Zone: infrav1.CloudStackZoneSpec{Network: infrav1.Network{Name: network1}}},
				{Name: "zone-b", Zone: infrav1.CloudStackZoneSpec{Network: infrav1.Network{Name: network2}}},
			}

			// zone-a
			mockCSClient.EXPECT().ResolveNetwork(&fds[0].Zone.Network)
			mockCSClient.EXPECT().GetPublicIPs(&fds[0].Zone.Network).Return([]*cloudstack.PublicIpAddress{
				{State: "Free"},
			}, nil)

			// zone-b
			mockCSClient.EXPECT().ResolveNetwork(&fds[1].Zone.Network)
			mockCSClient.EXPECT().GetPublicIPs(&fds[1].Zone.Network).Return([]*cloudstack.PublicIpAddress{
				{State: "Free"},
			}, nil)

			Expect(balancer.Assign(ctx, csMachine, capiMachine, fds)).Should(Succeed())
			Expect(csMachine.Spec.FailureDomainName).Should(Equal("zone-a"))
		})

		It("should NOT assign a failure domain because all IPs are allocated", func() {
			balancer := newFreeIPValidatingFailureDomainBalancer(&fakeFailureDomainClientFactory{mockCSClient})

			fds := []infrav1.CloudStackFailureDomainSpec{
				{Name: "zone-a", Zone: infrav1.CloudStackZoneSpec{Network: infrav1.Network{Name: network1}}},
				{Name: "zone-b", Zone: infrav1.CloudStackZoneSpec{Network: infrav1.Network{Name: network2}}},
			}

			// zone-a
			mockCSClient.EXPECT().ResolveNetwork(&fds[0].Zone.Network)
			mockCSClient.EXPECT().GetPublicIPs(&fds[0].Zone.Network).Return([]*cloudstack.PublicIpAddress{
				{State: "Allocated"},
			}, nil)

			// zone-b
			mockCSClient.EXPECT().ResolveNetwork(&fds[1].Zone.Network)
			mockCSClient.EXPECT().GetPublicIPs(&fds[1].Zone.Network).Return([]*cloudstack.PublicIpAddress{
				{State: "Allocated"},
			}, nil)

			Expect(balancer.Assign(ctx, csMachine, capiMachine, fds)).
				Should(MatchError("failed to assign failure domain, no failure domain with free IP addresses found"))
		})
	})

	Context("random balancer", func() {
		It("assign random failure domain", func() {
			balancer := newRandomFailureDomainBalancer()

			fds := []infrav1.CloudStackFailureDomainSpec{
				{Name: "zone-a"},
			}

			Expect(balancer.Assign(ctx, csMachine, capiMachine, fds)).Should(Succeed())
			Expect(csMachine.Spec.FailureDomainName).Should(Equal("zone-a"))
		})
	})

})

type fakeFailureDomainClientFactory struct {
	cloud.Client
}

func (f *fakeFailureDomainClientFactory) GetCloudClientAndUser(_ context.Context, _ *infrav1.CloudStackFailureDomainSpec) (csClient cloud.Client, csUser cloud.Client, err error) {
	return f, f, nil
}

type failingDomainBalancer struct {
	failure error
}

func newFailingDomainBalancer(failure error) Balancer {
	return &failingDomainBalancer{failure}
}

func (n *failingDomainBalancer) Assign(_ context.Context, _ *infrav1.CloudStackMachine, _ *clusterv1.Machine, _ []infrav1.CloudStackFailureDomainSpec) error {
	return n.failure
}

type fakeBalancer struct {
	failureDomainName string
}

func (f *fakeBalancer) Assign(_ context.Context, csMachine *infrav1.CloudStackMachine, _ *clusterv1.Machine, _ []infrav1.CloudStackFailureDomainSpec) error {
	csMachine.Spec.FailureDomainName = f.failureDomainName

	return nil
}
