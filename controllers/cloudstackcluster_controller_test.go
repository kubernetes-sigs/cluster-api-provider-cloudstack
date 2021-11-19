/*
Copyright 2021.

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

package controllers

import (
	"bytes"
	"flag"
	"fmt"
	"testing"

	"github.com/pkg/errors"

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	infrav1 "gitlab.aws.dev/ce-pike/merida/cluster-api-provider-capc/api/v1alpha4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func getCloudStackCluster() *infrav1.CloudStackCluster {
	return &infrav1.CloudStackCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
			Kind:       "CloudStackCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: infrav1.CloudStackClusterSpec{
			Zone:    "zone",
			Network: "network",
		},
	}
}

func TestCloudStackClusterReconciler(t *testing.T) {

	var (
		reconciler        CloudStackClusterReconciler
		mockCtrl          *gomock.Controller
		cloudStackCluster *infrav1.CloudStackCluster
		client            client.Client
		mockClient        *cloudstack.CloudStackClient
	)

	err := infrav1.AddToScheme(scheme.Scheme)
	if err != nil {
		t.Error()
	}
	err = clusterv1.AddToScheme(scheme.Scheme)
	if err != nil {
		t.Error()
	}

	klog.InitFlags(nil)
	if err := flag.Set("logtostderr", "false"); err != nil {
		_ = fmt.Errorf("Error setting logtostderr flag")
	}

	if err := flag.Set("v", "2"); err != nil {
		_ = fmt.Errorf("Error setting v flag")
	}
	flag.Parse()

	setup := func(cloudStackCluster *infrav1.CloudStackCluster, t *testing.T, g *WithT) {

		client = fake.NewClientBuilder().WithObjects(cloudStackCluster).Build()

		mockCtrl = gomock.NewController(t)
		mockClient = cloudstack.NewMockClient(mockCtrl)

		reconciler = CloudStackClusterReconciler{
			Client: client,
			CS:     mockClient,
		}
	}
	teardown := func(t *testing.T, g *WithT) {
		mockCtrl.Finish()
	}

	t.Run("Exit if required resources can not be found", func(t *testing.T) {
		t.Run("Zone not found", func(t *testing.T) {
			g := NewWithT(t)
			cloudStackCluster = getCloudStackCluster()
			setup(cloudStackCluster, t, g)
			defer teardown(t, g)

			buf := new(bytes.Buffer)
			klog.SetOutput(buf)
			logger := klogr.New()

			zoneName := "zone"
			expectedErr := fmt.Errorf("Not found")
			zs := mockClient.Zone.(*cloudstack.MockZoneServiceIface)
			zs.EXPECT().GetZoneID(zoneName).Return("", -1, expectedErr)

			_, err := reconciler.reconcile(logger, cloudStackCluster)
			g.Expect(errors.Cause(err)).To(MatchError(expectedErr))
		})

		t.Run("Network not found", func(t *testing.T) {
			g := NewWithT(t)
			cloudStackCluster = getCloudStackCluster()
			setup(cloudStackCluster, t, g)
			defer teardown(t, g)

			buf := new(bytes.Buffer)
			klog.SetOutput(buf)
			logger := klogr.New()

			zoneName := "zone"
			zoneID := "zone-id"
			zs := mockClient.Zone.(*cloudstack.MockZoneServiceIface)
			zs.EXPECT().GetZoneID(zoneName).Return(zoneID, 1, nil)

			networkName := "network"
			expectedErr := fmt.Errorf("Not found")
			ns := mockClient.Network.(*cloudstack.MockNetworkServiceIface)
			ns.EXPECT().GetNetworkID(networkName).Return("", -1, expectedErr)

			_, err := reconciler.reconcile(logger, cloudStackCluster)
			g.Expect(errors.Cause(err)).To(MatchError(expectedErr))
		})
	})
}
