package cloud

import (
	_ "github.com/golang/mock/gomock"
	_ "github.com/onsi/gomega"
)

// From start to finish this test takes a while as it actually instantiates and destroys a VM instance.
// func TestCreateInstance2(t *testing.T) {
// 	t.Run("Create Instance", func(t *testing.T) {
// 		cs := newCSClient()

// 		// This cluster should exist.
// 		// None of the tests should ever delete this cluster.
// 		c := Cluster{
// 			CSCluster: &infrav1.CloudStackCluster{
// 				Spec: infrav1.CloudStackClusterSpec{
// 					Zone:    "zone1",
// 					Network: "guestNet1",
// 				},
// 			},
// 		}
// 		err := c.Fetch(cs)
// 		if err != nil {
// 			t.Error(err)
// 		}

// 		// Create instance using cluster fetched above.
// 		I := Instance{
// 			Cluster: &infrav1.CloudStackCluster{Status: c.CSCluster.Status},
// 			Machine: &clusterv1.Machine{},
// 			CSMachine: &infrav1.CloudStackMachine{
// 				Spec: infrav1.CloudStackMachineSpec{
// 					Name:     "Control-Plane-Node",
// 					Offering: "KubeServiceOffering",
// 					Template: "ubuntu20",
// 					SSHKey:   "CAPCKeyPair6",
// 				},
// 				Status: infrav1.CloudStackMachineStatus{}},
// 		}
// 		err = I.Create(cs)
// 		if err != nil {
// 			t.Error(err)
// 		}
// 	})
// }
