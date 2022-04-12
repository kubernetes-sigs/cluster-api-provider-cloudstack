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

package utils

import (
	"context"
	"fmt"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type CloudStackBaseReconciler struct {
	CAPICluster           *capiv1.Cluster
	CSCluster             *infrav1.CloudStackCluster
	Log                   logr.Logger
	Scheme                *runtime.Scheme
	CS                    cloud.Client
	Client                client.Client
	ReconciliationSubject client.Object
}

type CloudStackReconcilerMethod func(context.Context, ctrl.Request) (ctrl.Result, error)

func (r *CloudStackBaseReconciler) RunWith(
	ctx context.Context, req ctrl.Request, fns ...CloudStackReconcilerMethod) (ctrl.Result, error) {
	for _, fn := range fns {
		if rslt, err := fn(ctx, req); err != nil || rslt.Requeue == true || rslt.RequeueAfter != 0 {
			return rslt, err
		}
	}
	return ctrl.Result{}, nil
}

// UsingConcreteSubject sets up the base reconciler to use passed concrete reconciler subject.
func (r *CloudStackBaseReconciler) UsingConcreteSubject(subject client.Object) {
	r.ReconciliationSubject = subject
}

// SetupLogger sets up the reconciler's logger to log with cluster and namespace values.
func (r *CloudStackBaseReconciler) SetupLogger(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Setup loger with reconciler specifics.
	// r.Log = r.Log.WithValues(r.ReconciliationSubject.GetObjectKind(), req.Name, "namespace", req.Namespace)

	return ctrl.Result{}, nil
}

// CheckIfPaused returns with reque later set if paused.
func (r *CloudStackBaseReconciler) CheckIfPaused(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if annotations.IsPaused(r.CAPICluster, r.CSCluster) {
		r.Log.Info("Cluster is paused. Refusing to reconcile.")
		return reconcile.Result{RequeueAfter: RequeueTimeout}, nil
	}
	return reconcile.Result{}, nil
}

// patchChangesBackToAPI writes the changes made to the Reconciler's local copy of the reconcilation subject back
// to the API.
func (r *CloudStackBaseReconciler) PatchChangesBackToAPI(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	patchHelper, err := patch.NewHelper(r.ReconciliationSubject, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err = patchHelper.Patch(ctx, r.ReconciliationSubject); err != nil {
		err = errors.Wrapf(
			err,
			"error patching %s %s/%s",
			r.ReconciliationSubject.GetObjectKind(),
			r.ReconciliationSubject.GetNamespace(),
			r.ReconciliationSubject.GetName(),
		)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *CloudStackBaseReconciler) GetBaseCRDs(ctx context.Context, req ctrl.Request) (res ctrl.Result, reterr error) {
	r.CSCluster = &infrav1.CloudStackCluster{}
	r.CAPICluster = &capiv1.Cluster{}

	// Get CloudStack cluster.
	if reterr = r.Client.Get(ctx, req.NamespacedName, r.CSCluster); reterr != nil {
		if client.IgnoreNotFound(reterr) == nil {
			r.Log.Info("Cluster not found.")
			return ctrl.Result{}, nil
		}
		return res, errors.Wrap(reterr, "error encountered while fetching CloudStackCluster crd")
	}

	// Get CAPI cluster.
	r.CAPICluster, reterr = util.GetOwnerCluster(ctx, r.Client, r.CSCluster.ObjectMeta)
	if reterr != nil {
		return ctrl.Result{}, errors.Wrap(reterr, "error encountered while fetching CAPI Cluster crd")
	} else if r.CAPICluster == nil {
		return res, errors.New("CAPI Cluster not found")
	}

	return res, nil
}

// FetchReconcilationSubject fetches the reconciliation subject of type defined by the concrete reconciler.
func (r *CloudStackBaseReconciler) FetchReconcilationSubject(ctx context.Context, req ctrl.Request) (res ctrl.Result, reterr error) {
	fmt.Println("blah")
	fmt.Println(r.ReconciliationSubject)
	fmt.Println(r.ReconciliationSubject)
	fmt.Println("blah")
	return ctrl.Result{}, r.Client.Get(ctx, req.NamespacedName, r.ReconciliationSubject)
}

// Base returns the base cloudstack reconciler itself. This is to satisfy additional CRD interfaces.
func (r *CloudStackBaseReconciler) Base() CloudStackBaseReconciler {
	return *r
}

// Subject returns the ReconciliationSubject. This is to satisfy additional CRD interfaces.
func (r *CloudStackBaseReconciler) Subject() client.Object {
	return r.ReconciliationSubject
}

// FetchReconcilationSubject logs the reconcilation subject in its entirety.
func (r *CloudStackBaseReconciler) LogReconcilationSubject(ctx context.Context, req ctrl.Request) (res ctrl.Result, reterr error) {
	fmt.Println("tooblah")
	fmt.Println(fmt.Sprintln(r.ReconciliationSubject))
	fmt.Println(r.ReconciliationSubject)
	fmt.Println("tooblah")
	return ctrl.Result{}, nil
}

type CloudStackIsoNetUser struct {
	CloudStackBaseReconciler
	Zones *infrav1.CloudStackZoneList
}

// // GenerateIsolatedNetwork generates a CloudStackIsolatedNetwork CRD owned by the ReconcilationSubject.
// func (r *CloudStackIsoNetUser) GenerateIsolatedNetwork(
// 	ctx context.Context, zone *infrav1.CloudStackZone, csCluster *infrav1.CloudStackCluster) error {

// 	csIsoNet := &infrav1.CloudStackIsolatedNetwork{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      zone.Spec.Name,
// 			Namespace: zone.Namespace,
// 			// Labels:      internal.ControlPlaneMachineLabelsForCluster(csCluster, csCluster.Name),
// 			Annotations: map[string]string{},
// 			OwnerReferences: []metav1.OwnerReference{
// 				*metav1.NewControllerRef(zone, controlplanev1.GroupVersion.WithKind("CloudStackZone")),
// 				*metav1.NewControllerRef(csCluster, controlplanev1.GroupVersion.WithKind("CloudStackCluster")),
// 			},
// 		},
// 		Spec: infrav1.CloudStackIsolatedNetworkSpec{Name: zone.Spec.Network.Name},
// 	}

// 	if err := r.Client.Create(ctx, csIsoNet); err != nil {
// 		return errors.Wrap(err, "failed to create machine")
// 	}
// 	return nil
// }
