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

package v1beta4_test

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	infrav3 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta3"
	infrav4 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta4"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

var _ = ginkgo.Describe("v1beta3 to v1beta4 conversion", func() {

	// ── CloudStackCluster ──────────────────────────────────────────────────────

	ginkgo.Context("CloudStackCluster", func() {
		ginkgo.It("ConvertToV4 preserves all spec and status fields", func() {
			src := &infrav3.CloudStackCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster-1", Namespace: "default"},
				Spec: infrav3.CloudStackClusterSpec{
					ControlPlaneEndpoint: capiv1beta1.APIEndpoint{Host: "1.2.3.4", Port: 6443},
					SyncWithACS:          ptr.To(true),
					FailureDomains: []infrav3.CloudStackFailureDomainSpec{{
						Name:        "fd-1",
						Account:     "admin",
						Domain:      "ROOT",
						Project:     "proj1",
						ACSEndpoint: corev1.SecretReference{Name: "cs-secret", Namespace: "default"},
						Zone: infrav3.CloudStackZoneSpec{
							Name: "zone-1",
							ID:   "zone-id-1",
							Network: infrav3.Network{
								ID:          "net-id-1",
								Type:        infrav3.NetworkTypeShared,
								Name:        "net-1",
								Gateway:     "10.0.0.1",
								Netmask:     "255.255.255.0",
								Offering:    "DefaultOff",
								RoutingMode: "Static",
								VPC:         &infrav3.VPC{ID: "vpc-1", Name: "vpc-name", CIDR: "10.0.0.0/16", Offering: "DefaultVPC"},
							},
						},
					}},
				},
				Status: infrav3.CloudStackClusterStatus{
					Ready:               true,
					CloudStackClusterID: "cid-1",
					FailureDomains: infrav3.FailureDomains{
						"fd-1": {ControlPlane: true, Attributes: map[string]string{"key": "val"}},
					},
				},
			}
			expected := &infrav4.CloudStackCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster-1", Namespace: "default"},
				Spec: infrav4.CloudStackClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{Host: "1.2.3.4", Port: 6443},
					SyncWithACS:          ptr.To(true),
					FailureDomains: []infrav4.CloudStackFailureDomainSpec{{
						Name:        "fd-1",
						Account:     "admin",
						Domain:      "ROOT",
						Project:     "proj1",
						ACSEndpoint: corev1.SecretReference{Name: "cs-secret", Namespace: "default"},
						Zone: infrav4.CloudStackZoneSpec{
							Name: "zone-1",
							ID:   "zone-id-1",
							Network: infrav4.Network{
								ID:          "net-id-1",
								Type:        infrav4.NetworkTypeShared,
								Name:        "net-1",
								Gateway:     "10.0.0.1",
								Netmask:     "255.255.255.0",
								Offering:    "DefaultOff",
								RoutingMode: "Static",
								VPC:         &infrav4.VPC{ID: "vpc-1", Name: "vpc-name", CIDR: "10.0.0.0/16", Offering: "DefaultVPC"},
							},
						},
					}},
				},
				Status: infrav4.CloudStackClusterStatus{
					Ready:               true,
					CloudStackClusterID: "cid-1",
					FailureDomains: []infrav4.FailureDomain{
						{Name: "fd-1", ControlPlane: ptr.To(true), Attributes: map[string]string{"key": "val"}},
					},
					// Initialization is v1beta4-only; must be nil after up-conversion from v1beta3.
				},
			}

			dst := &infrav4.CloudStackCluster{}
			err := infrav3.ConvertCloudStackClusterToV4(src, dst)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(dst).Should(gomega.Equal(expected))
		})

		ginkgo.It("ConvertToV4 represents ControlPlane: false as ptr.To(false), not nil", func() {
			// ptr.To(false) and nil are semantically different: nil means "unknown",
			// false means "explicitly not a control-plane domain".
			src := &infrav3.CloudStackCluster{
				Status: infrav3.CloudStackClusterStatus{
					FailureDomains: infrav3.FailureDomains{
						"fd-worker": {ControlPlane: false},
					},
				},
			}
			expected := &infrav4.CloudStackCluster{
				Status: infrav4.CloudStackClusterStatus{
					FailureDomains: []infrav4.FailureDomain{
						{Name: "fd-worker", ControlPlane: ptr.To(false)},
					},
				},
			}

			dst := &infrav4.CloudStackCluster{}
			err := infrav3.ConvertCloudStackClusterToV4(src, dst)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(dst).Should(gomega.Equal(expected))
		})

		ginkgo.It("ConvertToV4 leaves nil failure domain collections as nil", func() {
			dst := &infrav4.CloudStackCluster{}
			err := infrav3.ConvertCloudStackClusterToV4(&infrav3.CloudStackCluster{}, dst)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(dst.Spec.FailureDomains).Should(gomega.BeNil())
			gomega.Expect(dst.Status.FailureDomains).Should(gomega.BeNil())
		})

		ginkgo.It("ConvertToV4 does not populate Initialization (v1beta4-only field)", func() {
			src := &infrav3.CloudStackCluster{
				Status: infrav3.CloudStackClusterStatus{Ready: true},
			}
			dst := &infrav4.CloudStackCluster{}
			err := infrav3.ConvertCloudStackClusterToV4(src, dst)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(dst.Status.Initialization).Should(gomega.BeNil())
		})

		ginkgo.It("ConvertFromV4 preserves all spec and status fields", func() {
			src := &infrav4.CloudStackCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster-2", Namespace: "ns1"},
				Spec: infrav4.CloudStackClusterSpec{
					ControlPlaneEndpoint: clusterv1.APIEndpoint{Host: "5.6.7.8", Port: 443},
					SyncWithACS:          ptr.To(false),
					FailureDomains: []infrav4.CloudStackFailureDomainSpec{{
						Name:        "fd-a",
						Account:     "acct",
						Domain:      "dom",
						Project:     "proj",
						ACSEndpoint: corev1.SecretReference{Name: "sec", Namespace: "ns1"},
						Zone: infrav4.CloudStackZoneSpec{
							Name: "zone-a",
							ID:   "zid-a",
							Network: infrav4.Network{
								Name:        "net-a",
								Gateway:     "172.16.0.1",
								RoutingMode: "Dynamic",
								VPC:         &infrav4.VPC{ID: "vpc-a", Name: "vpc-a-name"},
							},
						},
					}},
				},
				Status: infrav4.CloudStackClusterStatus{
					Ready:               true,
					CloudStackClusterID: "cid-2",
					FailureDomains: []infrav4.FailureDomain{
						{Name: "fd-a", ControlPlane: ptr.To(true), Attributes: map[string]string{"x": "y"}},
					},
					Initialization: &infrav4.ClusterInitializationStatus{Provisioned: ptr.To(true)},
				},
			}
			expected := &infrav3.CloudStackCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster-2", Namespace: "ns1"},
				Spec: infrav3.CloudStackClusterSpec{
					ControlPlaneEndpoint: capiv1beta1.APIEndpoint{Host: "5.6.7.8", Port: 443},
					SyncWithACS:          ptr.To(false),
					FailureDomains: []infrav3.CloudStackFailureDomainSpec{{
						Name:        "fd-a",
						Account:     "acct",
						Domain:      "dom",
						Project:     "proj",
						ACSEndpoint: corev1.SecretReference{Name: "sec", Namespace: "ns1"},
						Zone: infrav3.CloudStackZoneSpec{
							Name: "zone-a",
							ID:   "zid-a",
							Network: infrav3.Network{
								Name:        "net-a",
								Gateway:     "172.16.0.1",
								RoutingMode: "Dynamic",
								VPC:         &infrav3.VPC{ID: "vpc-a", Name: "vpc-a-name"},
							},
						},
					}},
				},
				Status: infrav3.CloudStackClusterStatus{
					Ready:               true,
					CloudStackClusterID: "cid-2",
					FailureDomains: infrav3.FailureDomains{
						"fd-a": {ControlPlane: true, Attributes: map[string]string{"x": "y"}},
					},
					// Initialization is v1beta4-only; dropped on down-conversion.
				},
			}

			dst := &infrav3.CloudStackCluster{}
			err := infrav3.ConvertCloudStackClusterFromV4(src, dst)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(dst).Should(gomega.Equal(expected))
		})

		ginkgo.It("ConvertFromV4 maps nil and ptr.To(false) ControlPlane to false", func() {
			src := &infrav4.CloudStackCluster{
				Status: infrav4.CloudStackClusterStatus{
					FailureDomains: []infrav4.FailureDomain{
						{Name: "fd-nil", ControlPlane: nil},
						{Name: "fd-false", ControlPlane: ptr.To(false)},
					},
				},
			}
			expected := &infrav3.CloudStackCluster{
				Status: infrav3.CloudStackClusterStatus{
					FailureDomains: infrav3.FailureDomains{
						"fd-nil":   {ControlPlane: false},
						"fd-false": {ControlPlane: false},
					},
				},
			}

			dst := &infrav3.CloudStackCluster{}
			err := infrav3.ConvertCloudStackClusterFromV4(src, dst)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(dst).Should(gomega.Equal(expected))
		})

		ginkgo.It("ConvertFromV4 leaves nil failure domain collections as nil", func() {
			dst := &infrav3.CloudStackCluster{}
			err := infrav3.ConvertCloudStackClusterFromV4(&infrav4.CloudStackCluster{}, dst)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(dst.Spec.FailureDomains).Should(gomega.BeNil())
			gomega.Expect(dst.Status.FailureDomains).Should(gomega.BeNil())
		})
	})

	// ── CloudStackMachine ──────────────────────────────────────────────────────

	ginkgo.Context("CloudStackMachine", func() {
		ginkgo.It("ConvertToV4 preserves all spec and status fields", func() {
			instanceID := "inst-123"
			providerID := "cloudstack:///inst-123"
			statusMsg := "running"
			reason := "ok"

			src := &infrav3.CloudStackMachine{
				ObjectMeta: metav1.ObjectMeta{Name: "machine-1", Namespace: "default"},
				Spec: infrav3.CloudStackMachineSpec{
					Name:       "machine-name",
					ID:         "machine-id",
					InstanceID: &instanceID,
					Offering:   infrav3.CloudStackResourceIdentifier{ID: "off-id", Name: "off-name"},
					Template:   infrav3.CloudStackResourceIdentifier{ID: "tmpl-id", Name: "tmpl-name"},
					DiskOffering: infrav3.CloudStackResourceDiskOffering{
						CloudStackResourceIdentifier: infrav3.CloudStackResourceIdentifier{ID: "disk-id", Name: "disk-name"},
						CustomSize:                   50,
						MountPath:                    "/data",
						Device:                       "/dev/vdb",
						Filesystem:                   "ext4",
						Label:                        "data-disk",
					},
					Networks:             []infrav3.NetworkSpec{{Name: "net-a", IP: "10.0.0.5", ID: "nid-a"}},
					SSHKey:               "my-ssh-key",
					Details:              map[string]string{"cpuNumber": "4"},
					AffinityGroupIDs:     []string{"ag-1", "ag-2"},
					Affinity:             "pro",
					AffinityGroupRef:     &corev1.ObjectReference{Name: "aff-ref", Namespace: "default"},
					ProviderID:           &providerID,
					FailureDomainName:    "fd-1",
					UncompressedUserData: ptr.To(true),
				},
				Status: infrav3.CloudStackMachineStatus{
					Addresses:     []corev1.NodeAddress{{Type: corev1.NodeExternalIP, Address: "1.2.3.4"}},
					InstanceState: "Running",
					Ready:         true,
					Status:        &statusMsg,
					Reason:        &reason,
				},
			}
			expected := &infrav4.CloudStackMachine{
				ObjectMeta: metav1.ObjectMeta{Name: "machine-1", Namespace: "default"},
				Spec: infrav4.CloudStackMachineSpec{
					Name:       "machine-name",
					ID:         "machine-id",
					InstanceID: &instanceID,
					Offering:   infrav4.CloudStackResourceIdentifier{ID: "off-id", Name: "off-name"},
					Template:   infrav4.CloudStackResourceIdentifier{ID: "tmpl-id", Name: "tmpl-name"},
					DiskOffering: infrav4.CloudStackResourceDiskOffering{
						CloudStackResourceIdentifier: infrav4.CloudStackResourceIdentifier{ID: "disk-id", Name: "disk-name"},
						CustomSize:                   50,
						MountPath:                    "/data",
						Device:                       "/dev/vdb",
						Filesystem:                   "ext4",
						Label:                        "data-disk",
					},
					Networks:             []infrav4.NetworkSpec{{Name: "net-a", IP: "10.0.0.5", ID: "nid-a"}},
					SSHKey:               "my-ssh-key",
					Details:              map[string]string{"cpuNumber": "4"},
					AffinityGroupIDs:     []string{"ag-1", "ag-2"},
					Affinity:             "pro",
					AffinityGroupRef:     &corev1.ObjectReference{Name: "aff-ref", Namespace: "default"},
					ProviderID:           &providerID,
					FailureDomainName:    "fd-1",
					UncompressedUserData: ptr.To(true),
				},
				Status: infrav4.CloudStackMachineStatus{
					Addresses:     []corev1.NodeAddress{{Type: corev1.NodeExternalIP, Address: "1.2.3.4"}},
					InstanceState: "Running",
					Ready:         true,
					Status:        &statusMsg,
					Reason:        &reason,
					// Initialization is v1beta4-only; must be nil after up-conversion from v1beta3.
				},
			}

			dst := &infrav4.CloudStackMachine{}
			err := infrav3.ConvertCloudStackMachineToV4(src, dst)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(dst).Should(gomega.Equal(expected))
		})

		ginkgo.It("ConvertToV4 leaves nil Networks as nil", func() {
			dst := &infrav4.CloudStackMachine{}
			err := infrav3.ConvertCloudStackMachineToV4(&infrav3.CloudStackMachine{}, dst)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(dst.Spec.Networks).Should(gomega.BeNil())
		})

		ginkgo.It("ConvertToV4 does not populate Initialization (v1beta4-only field)", func() {
			dst := &infrav4.CloudStackMachine{}
			err := infrav3.ConvertCloudStackMachineToV4(&infrav3.CloudStackMachine{
				Status: infrav3.CloudStackMachineStatus{Ready: true},
			}, dst)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(dst.Status.Initialization).Should(gomega.BeNil())
		})

		ginkgo.It("ConvertFromV4 preserves all spec and status fields", func() {
			instanceID := "inst-456"
			providerID := "cloudstack:///inst-456"

			src := &infrav4.CloudStackMachine{
				ObjectMeta: metav1.ObjectMeta{Name: "machine-2", Namespace: "ns2"},
				Spec: infrav4.CloudStackMachineSpec{
					InstanceID:           &instanceID,
					Offering:             infrav4.CloudStackResourceIdentifier{ID: "o1", Name: "oname"},
					Template:             infrav4.CloudStackResourceIdentifier{ID: "t1", Name: "tname"},
					FailureDomainName:    "fd-2",
					ProviderID:           &providerID,
					Networks:             []infrav4.NetworkSpec{{Name: "net-b", IP: "10.0.1.5", ID: "nid-b"}},
					SSHKey:               "key-2",
					UncompressedUserData: ptr.To(false),
				},
				Status: infrav4.CloudStackMachineStatus{
					InstanceState:  "Running",
					Ready:          true,
					Initialization: &infrav4.MachineInitializationStatus{Provisioned: ptr.To(true)},
				},
			}
			expected := &infrav3.CloudStackMachine{
				ObjectMeta: metav1.ObjectMeta{Name: "machine-2", Namespace: "ns2"},
				Spec: infrav3.CloudStackMachineSpec{
					InstanceID:           &instanceID,
					Offering:             infrav3.CloudStackResourceIdentifier{ID: "o1", Name: "oname"},
					Template:             infrav3.CloudStackResourceIdentifier{ID: "t1", Name: "tname"},
					FailureDomainName:    "fd-2",
					ProviderID:           &providerID,
					Networks:             []infrav3.NetworkSpec{{Name: "net-b", IP: "10.0.1.5", ID: "nid-b"}},
					SSHKey:               "key-2",
					UncompressedUserData: ptr.To(false),
				},
				Status: infrav3.CloudStackMachineStatus{
					InstanceState: "Running",
					Ready:         true,
					// Initialization is v1beta4-only; dropped on down-conversion.
				},
			}

			dst := &infrav3.CloudStackMachine{}
			err := infrav3.ConvertCloudStackMachineFromV4(src, dst)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(dst).Should(gomega.Equal(expected))
		})
	})

	// ── CloudStackMachineTemplate ──────────────────────────────────────────────

	ginkgo.Context("CloudStackMachineTemplate", func() {
		ginkgo.It("ConvertToV4 preserves template spec via machine conversion", func() {
			src := &infrav3.CloudStackMachineTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "tmpl-1", Namespace: "ns"},
				Spec: infrav3.CloudStackMachineTemplateSpec{
					Template: infrav3.CloudStackMachineTemplateResource{
						Spec: infrav3.CloudStackMachineSpec{
							Offering: infrav3.CloudStackResourceIdentifier{Name: "large"},
							Template: infrav3.CloudStackResourceIdentifier{Name: "ubuntu-20"},
							SSHKey:   "tmpl-key",
							DiskOffering: infrav3.CloudStackResourceDiskOffering{
								CloudStackResourceIdentifier: infrav3.CloudStackResourceIdentifier{Name: "fast-disk"},
								MountPath:                    "/mnt/data",
							},
						},
					},
				},
			}
			expected := &infrav4.CloudStackMachineTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "tmpl-1", Namespace: "ns"},
				Spec: infrav4.CloudStackMachineTemplateSpec{
					Template: infrav4.CloudStackMachineTemplateResource{
						Spec: infrav4.CloudStackMachineSpec{
							Offering: infrav4.CloudStackResourceIdentifier{Name: "large"},
							Template: infrav4.CloudStackResourceIdentifier{Name: "ubuntu-20"},
							SSHKey:   "tmpl-key",
							DiskOffering: infrav4.CloudStackResourceDiskOffering{
								CloudStackResourceIdentifier: infrav4.CloudStackResourceIdentifier{Name: "fast-disk"},
								MountPath:                    "/mnt/data",
							},
						},
					},
				},
			}

			dst := &infrav4.CloudStackMachineTemplate{}
			err := infrav3.ConvertCloudStackMachineTemplateToV4(src, dst)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(dst).Should(gomega.Equal(expected))
		})

		ginkgo.It("ConvertFromV4 preserves template spec via machine conversion", func() {
			src := &infrav4.CloudStackMachineTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "tmpl-2"},
				Spec: infrav4.CloudStackMachineTemplateSpec{
					Template: infrav4.CloudStackMachineTemplateResource{
						Spec: infrav4.CloudStackMachineSpec{
							Offering: infrav4.CloudStackResourceIdentifier{Name: "small"},
							SSHKey:   "other-key",
							Affinity: "anti",
						},
					},
				},
			}
			expected := &infrav3.CloudStackMachineTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "tmpl-2"},
				Spec: infrav3.CloudStackMachineTemplateSpec{
					Template: infrav3.CloudStackMachineTemplateResource{
						Spec: infrav3.CloudStackMachineSpec{
							Offering: infrav3.CloudStackResourceIdentifier{Name: "small"},
							SSHKey:   "other-key",
							Affinity: "anti",
						},
					},
				},
			}

			dst := &infrav3.CloudStackMachineTemplate{}
			err := infrav3.ConvertCloudStackMachineTemplateFromV4(src, dst)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(dst).Should(gomega.Equal(expected))
		})
	})

	// ── CloudStackFailureDomain ────────────────────────────────────────────────

	ginkgo.Context("CloudStackFailureDomain", func() {
		ginkgo.It("ConvertToV4 preserves all fields including VPC and RoutingMode", func() {
			src := &infrav3.CloudStackFailureDomain{
				ObjectMeta: metav1.ObjectMeta{Name: "fd-1", Namespace: "ns"},
				Spec: infrav3.CloudStackFailureDomainSpec{
					Name:        "fd-name",
					Account:     "admin",
					Domain:      "ROOT",
					Project:     "proj",
					ACSEndpoint: corev1.SecretReference{Name: "acs-secret", Namespace: "ns"},
					Zone: infrav3.CloudStackZoneSpec{
						Name: "zone-1",
						ID:   "zone-id-1",
						Network: infrav3.Network{
							ID:          "nid",
							Type:        infrav3.NetworkTypeIsolated,
							Name:        "isolated-net",
							Gateway:     "192.168.1.1",
							Netmask:     "255.255.255.0",
							Offering:    "IsolatedOff",
							RoutingMode: "Dynamic",
							VPC:         &infrav3.VPC{ID: "v1", Name: "vpc-name", CIDR: "192.168.0.0/16", Offering: "DefaultVPC"},
						},
					},
				},
				Status: infrav3.CloudStackFailureDomainStatus{Ready: true},
			}
			expected := &infrav4.CloudStackFailureDomain{
				ObjectMeta: metav1.ObjectMeta{Name: "fd-1", Namespace: "ns"},
				Spec: infrav4.CloudStackFailureDomainSpec{
					Name:        "fd-name",
					Account:     "admin",
					Domain:      "ROOT",
					Project:     "proj",
					ACSEndpoint: corev1.SecretReference{Name: "acs-secret", Namespace: "ns"},
					Zone: infrav4.CloudStackZoneSpec{
						Name: "zone-1",
						ID:   "zone-id-1",
						Network: infrav4.Network{
							ID:          "nid",
							Type:        infrav4.NetworkTypeIsolated,
							Name:        "isolated-net",
							Gateway:     "192.168.1.1",
							Netmask:     "255.255.255.0",
							Offering:    "IsolatedOff",
							RoutingMode: "Dynamic",
							VPC:         &infrav4.VPC{ID: "v1", Name: "vpc-name", CIDR: "192.168.0.0/16", Offering: "DefaultVPC"},
						},
					},
				},
				Status: infrav4.CloudStackFailureDomainStatus{Ready: true},
			}

			dst := &infrav4.CloudStackFailureDomain{}
			err := infrav3.ConvertCloudStackFailureDomainToV4(src, dst)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(dst).Should(gomega.Equal(expected))
		})

		ginkgo.It("ConvertToV4 preserves nil VPC as nil", func() {
			src := &infrav3.CloudStackFailureDomain{
				Spec: infrav3.CloudStackFailureDomainSpec{
					Zone: infrav3.CloudStackZoneSpec{
						Network: infrav3.Network{Name: "no-vpc-net", VPC: nil},
					},
				},
			}

			dst := &infrav4.CloudStackFailureDomain{}
			err := infrav3.ConvertCloudStackFailureDomainToV4(src, dst)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(dst.Spec.Zone.Network.VPC).Should(gomega.BeNil())
		})

		ginkgo.It("ConvertFromV4 preserves all fields", func() {
			src := &infrav4.CloudStackFailureDomain{
				ObjectMeta: metav1.ObjectMeta{Name: "fd-2"},
				Spec: infrav4.CloudStackFailureDomainSpec{
					Name:    "fd-name-2",
					Account: "acct",
					Zone: infrav4.CloudStackZoneSpec{
						Name: "zone-2",
						Network: infrav4.Network{
							Name: "net-2",
							VPC:  &infrav4.VPC{Name: "vpc-2", CIDR: "172.16.0.0/12"},
						},
					},
				},
				Status: infrav4.CloudStackFailureDomainStatus{Ready: true},
			}
			expected := &infrav3.CloudStackFailureDomain{
				ObjectMeta: metav1.ObjectMeta{Name: "fd-2"},
				Spec: infrav3.CloudStackFailureDomainSpec{
					Name:    "fd-name-2",
					Account: "acct",
					Zone: infrav3.CloudStackZoneSpec{
						Name: "zone-2",
						Network: infrav3.Network{
							Name: "net-2",
							VPC:  &infrav3.VPC{Name: "vpc-2", CIDR: "172.16.0.0/12"},
						},
					},
				},
				Status: infrav3.CloudStackFailureDomainStatus{Ready: true},
			}

			dst := &infrav3.CloudStackFailureDomain{}
			err := infrav3.ConvertCloudStackFailureDomainFromV4(src, dst)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(dst).Should(gomega.Equal(expected))
		})
	})

	// ── CloudStackIsolatedNetwork ──────────────────────────────────────────────

	ginkgo.Context("CloudStackIsolatedNetwork", func() {
		ginkgo.It("ConvertToV4 preserves all spec and status fields", func() {
			src := &infrav3.CloudStackIsolatedNetwork{
				ObjectMeta: metav1.ObjectMeta{Name: "isonet-1", Namespace: "ns"},
				Spec: infrav3.CloudStackIsolatedNetworkSpec{
					Name:                 "isolated-net-1",
					ID:                   "isonet-id-1",
					ControlPlaneEndpoint: capiv1beta1.APIEndpoint{Host: "10.0.0.10", Port: 6443},
					FailureDomainName:    "fd-iso",
					Gateway:              "10.0.0.1",
					Netmask:              "255.255.255.0",
					Offering:             "IsolatedOff",
					VPC:                  &infrav3.VPC{ID: "vpc-iso", Name: "iso-vpc"},
				},
				Status: infrav3.CloudStackIsolatedNetworkStatus{
					PublicIPID:          "pip-1",
					LBRuleID:            "lb-1",
					RoutingMode:         "Static",
					FirewallRulesOpened: true,
					Ready:               true,
				},
			}
			expected := &infrav4.CloudStackIsolatedNetwork{
				ObjectMeta: metav1.ObjectMeta{Name: "isonet-1", Namespace: "ns"},
				Spec: infrav4.CloudStackIsolatedNetworkSpec{
					Name:                 "isolated-net-1",
					ID:                   "isonet-id-1",
					ControlPlaneEndpoint: clusterv1.APIEndpoint{Host: "10.0.0.10", Port: 6443},
					FailureDomainName:    "fd-iso",
					Gateway:              "10.0.0.1",
					Netmask:              "255.255.255.0",
					Offering:             "IsolatedOff",
					VPC:                  &infrav4.VPC{ID: "vpc-iso", Name: "iso-vpc"},
				},
				Status: infrav4.CloudStackIsolatedNetworkStatus{
					PublicIPID:          "pip-1",
					LBRuleID:            "lb-1",
					RoutingMode:         "Static",
					FirewallRulesOpened: true,
					Ready:               true,
				},
			}

			dst := &infrav4.CloudStackIsolatedNetwork{}
			err := infrav3.ConvertCloudStackIsolatedNetworkToV4(src, dst)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(dst).Should(gomega.Equal(expected))
		})

		ginkgo.It("ConvertFromV4 preserves all spec and status fields", func() {
			src := &infrav4.CloudStackIsolatedNetwork{
				ObjectMeta: metav1.ObjectMeta{Name: "isonet-2"},
				Spec: infrav4.CloudStackIsolatedNetworkSpec{
					Name:              "iso-2",
					FailureDomainName: "fd-2",
					VPC:               &infrav4.VPC{Name: "vpc-2"},
				},
				Status: infrav4.CloudStackIsolatedNetworkStatus{
					PublicIPID:  "pip-2",
					RoutingMode: "Dynamic",
					Ready:       true,
				},
			}
			expected := &infrav3.CloudStackIsolatedNetwork{
				ObjectMeta: metav1.ObjectMeta{Name: "isonet-2"},
				Spec: infrav3.CloudStackIsolatedNetworkSpec{
					Name:              "iso-2",
					FailureDomainName: "fd-2",
					VPC:               &infrav3.VPC{Name: "vpc-2"},
				},
				Status: infrav3.CloudStackIsolatedNetworkStatus{
					PublicIPID:  "pip-2",
					RoutingMode: "Dynamic",
					Ready:       true,
				},
			}

			dst := &infrav3.CloudStackIsolatedNetwork{}
			err := infrav3.ConvertCloudStackIsolatedNetworkFromV4(src, dst)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(dst).Should(gomega.Equal(expected))
		})
	})

	// ── CloudStackAffinityGroup ────────────────────────────────────────────────

	ginkgo.Context("CloudStackAffinityGroup", func() {
		ginkgo.It("ConvertToV4 preserves all fields", func() {
			src := &infrav3.CloudStackAffinityGroup{
				ObjectMeta: metav1.ObjectMeta{Name: "aff-1"},
				Spec: infrav3.CloudStackAffinityGroupSpec{
					Type:              "host anti-affinity",
					Name:              "aff-group-1",
					ID:                "aff-id-1",
					FailureDomainName: "fd-1",
				},
				Status: infrav3.CloudStackAffinityGroupStatus{Ready: true},
			}
			expected := &infrav4.CloudStackAffinityGroup{
				ObjectMeta: metav1.ObjectMeta{Name: "aff-1"},
				Spec: infrav4.CloudStackAffinityGroupSpec{
					Type:              "host anti-affinity",
					Name:              "aff-group-1",
					ID:                "aff-id-1",
					FailureDomainName: "fd-1",
				},
				Status: infrav4.CloudStackAffinityGroupStatus{Ready: true},
			}

			dst := &infrav4.CloudStackAffinityGroup{}
			err := infrav3.ConvertCloudStackAffinityGroupToV4(src, dst)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(dst).Should(gomega.Equal(expected))
		})

		ginkgo.It("ConvertFromV4 preserves all fields", func() {
			src := &infrav4.CloudStackAffinityGroup{
				ObjectMeta: metav1.ObjectMeta{Name: "aff-2"},
				Spec: infrav4.CloudStackAffinityGroupSpec{
					Type:              "host affinity",
					Name:              "aff-group-2",
					ID:                "aff-id-2",
					FailureDomainName: "fd-2",
				},
				Status: infrav4.CloudStackAffinityGroupStatus{Ready: false},
			}
			expected := &infrav3.CloudStackAffinityGroup{
				ObjectMeta: metav1.ObjectMeta{Name: "aff-2"},
				Spec: infrav3.CloudStackAffinityGroupSpec{
					Type:              "host affinity",
					Name:              "aff-group-2",
					ID:                "aff-id-2",
					FailureDomainName: "fd-2",
				},
				Status: infrav3.CloudStackAffinityGroupStatus{Ready: false},
			}

			dst := &infrav3.CloudStackAffinityGroup{}
			err := infrav3.ConvertCloudStackAffinityGroupFromV4(src, dst)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(dst).Should(gomega.Equal(expected))
		})
	})

	// ── CloudStackMachineStateChecker ─────────────────────────────────────────

	ginkgo.Context("CloudStackMachineStateChecker", func() {
		ginkgo.It("ConvertToV4 preserves InstanceID and Ready", func() {
			src := &infrav3.CloudStackMachineStateChecker{
				ObjectMeta: metav1.ObjectMeta{Name: "msc-1"},
				Spec:       infrav3.CloudStackMachineStateCheckerSpec{InstanceID: "inst-abc"},
				Status:     infrav3.CloudStackMachineStateCheckerStatus{Ready: true},
			}
			expected := &infrav4.CloudStackMachineStateChecker{
				ObjectMeta: metav1.ObjectMeta{Name: "msc-1"},
				Spec:       infrav4.CloudStackMachineStateCheckerSpec{InstanceID: "inst-abc"},
				Status:     infrav4.CloudStackMachineStateCheckerStatus{Ready: true},
			}

			dst := &infrav4.CloudStackMachineStateChecker{}
			err := infrav3.ConvertCloudStackMachineStateCheckerToV4(src, dst)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(dst).Should(gomega.Equal(expected))
		})

		ginkgo.It("ConvertFromV4 preserves InstanceID and Ready", func() {
			src := &infrav4.CloudStackMachineStateChecker{
				ObjectMeta: metav1.ObjectMeta{Name: "msc-2"},
				Spec:       infrav4.CloudStackMachineStateCheckerSpec{InstanceID: "inst-def"},
				Status:     infrav4.CloudStackMachineStateCheckerStatus{Ready: false},
			}
			expected := &infrav3.CloudStackMachineStateChecker{
				ObjectMeta: metav1.ObjectMeta{Name: "msc-2"},
				Spec:       infrav3.CloudStackMachineStateCheckerSpec{InstanceID: "inst-def"},
				Status:     infrav3.CloudStackMachineStateCheckerStatus{Ready: false},
			}

			dst := &infrav3.CloudStackMachineStateChecker{}
			err := infrav3.ConvertCloudStackMachineStateCheckerFromV4(src, dst)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(dst).Should(gomega.Equal(expected))
		})
	})
})
