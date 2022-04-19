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
	"time"

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

// ReconcilerBase is the base set of componenets we use in k8s reconcilers.
// These are items that are not copied for each reconciliation request, and must be written to with caution.
type ReconcilerBase struct {
	BaseLogger logr.Logger
	Scheme     *runtime.Scheme
	Client     client.Client
	CS         cloud.Client
}

type CloudStackClusterReconciliationRunner struct {
	ReconciliationRunner
	Zones                 infrav1.CloudStackZoneList
	ReconciliationSubject *infrav1.CloudStackCluster
	CSUser                cloud.Client
}

// CloudStackBaseContext is the base CloudStack data structure created/copied for each reconciliation request to avoid
// concurrent member access.
type CloudStackBaseContext struct {
	Log         logr.Logger
	RequestCtx  context.Context
	Request     ctrl.Request
	CAPICluster *capiv1.Cluster
	CSCluster   *infrav1.CloudStackCluster
	Patcher     *patch.Helper
}

// ReconciliationRunner is the base structure used to run reconciliation methods and implements several.
type ReconciliationRunner struct {
	fn interface{}
	ReconcilerBase
	CloudStackBaseContext
	ReconciliationSubject client.Object // Underlying crd interface.
}

// UsingBaseReconciler sets the ReconciliationRunner to use the same base components as the passed base reconciler.
func (runner *ReconciliationRunner) UsingBaseReconciler(base ReconcilerBase) *ReconciliationRunner {
	runner.ReconcilerBase = base
	return runner
}

// ForRequest sets the reconciliation request.
func (runner *ReconciliationRunner) ForRequest(req ctrl.Request) *ReconciliationRunner {
	runner.Request = req
	return runner
}

// WithRequestCtx sets the request context.
func (runner *ReconciliationRunner) WithRequestCtx(ctx context.Context) *ReconciliationRunner {
	runner.RequestCtx = ctx
	return runner
}

// GetBaseCRDs fetches the CAPI Cluster and the CloudStackCluster. These are the base CRDs required for every
// CloudStack reconciliation.
func (r *ReconciliationRunner) GetBaseCRDs() (res ctrl.Result, reterr error) {
	// Get CloudStack cluster.
	r.CSCluster = &infrav1.CloudStackCluster{}
	if reterr = r.Client.Get(r.RequestCtx, r.Request.NamespacedName, r.CSCluster); reterr != nil {
		if client.IgnoreNotFound(reterr) == nil {
			return r.RequeueWithMessage("CloudStackCluster cluster not found, requeueing.")
		}
		return res, errors.Wrap(reterr, "error encountered while fetching CloudStackCluster CRD")
	}

	// Get CAPI cluster.
	r.CAPICluster, reterr = util.GetOwnerCluster(r.RequestCtx, r.Client, r.CSCluster.ObjectMeta)
	if reterr != nil {
		return ctrl.Result{}, errors.Wrap(reterr, "error encountered while fetching CAPI Cluster CRD")
	} else if r.CAPICluster == nil {
		return r.RequeueWithMessage("CAPI Cluster not found, requeueing.")
	}
	return res, nil
}

// SetupLogger sets up the reconciler's logger to log with name and namespace values.
func (r *ReconciliationRunner) SetupLogger() (res ctrl.Result, retErr error) {
	r.Log = r.BaseLogger.WithValues("name", r.Request.Name, "namespace", r.Request.Namespace)
	return ctrl.Result{}, nil
}

// SetupPatcher initializes the patcher with the ReconciliationSubject.
// This must be done before changes to the ReconciliationSubject for changes to be patched back later.
func (r *ReconciliationRunner) SetupPatcher() (res ctrl.Result, retErr error) {
	r.Log.V(1).Info("Setting up patcher.")
	r.Patcher, retErr = patch.NewHelper(r.ReconciliationSubject, r.Client)
	return res, errors.Wrapf(retErr, "error encountered while setting up patcher")
}

// PatchChangesBackToAPI patches changes to the ReconciliationSubject back to the appropriate API.
func (r *ReconciliationRunner) PatchChangesBackToAPI() (res ctrl.Result, retErr error) {
	r.Log.V(1).Info("Patching changes back to api.")
	err := r.Patcher.Patch(r.RequestCtx, r.ReconciliationSubject)
	return res, errors.Wrapf(err, "error encountered while patching reconciliation subject")
}

// RequeueWithMessage is a convenience method to log requeue message and then return a result with RequeueAfter set.
func (r *ReconciliationRunner) RequeueWithMessage(msg string, keysAndValues ...interface{}) (ctrl.Result, error) {
	r.Log.Info(msg)
	return ctrl.Result{RequeueAfter: RequeueTimeout}, nil
}

func (r *ReconciliationRunner) LogReconciliationSubject() (ctrl.Result, error) {
	r.Log.Info("The subject", "subject", r.ReconciliationSubject)
	return ctrl.Result{}, nil
}

// CloudStackReconcilerMethod is the method type used in RunReconciliationStages. Additional arguments can be added
// by wrapping this type in another function affectively currying them.
type CloudStackReconcilerMethod func() (ctrl.Result, error)

// RunReconciliationStages runs CloudStackReconcilerMethods in order and exits if an error or requeue condition is set.
func (runner *ReconciliationRunner) RunReconciliationStages(fns ...CloudStackReconcilerMethod) (ctrl.Result, error) {
	for _, fn := range fns {
		if rslt, err := fn(); err != nil {
			return rslt, err
		} else if rslt.Requeue == true || rslt.RequeueAfter != time.Duration(0) {
			return rslt, nil
		}
	}
	return ctrl.Result{}, nil
}

// CheckIfPaused returns with reque later set if paused.
func (r *ReconciliationRunner) CheckIfPaused() (ctrl.Result, error) {
	if annotations.IsPaused(r.CAPICluster, r.CSCluster) {
		return r.RequeueWithMessage("Cluster is paused. Refusing to reconcile.")
	}
	return reconcile.Result{}, nil
}

// GetReconcilationSubject gets the reconciliation subject of type defined by the concrete reconciler.
func (r *ReconciliationRunner) GetReconciliationSubject() (res ctrl.Result, reterr error) {
	return ctrl.Result{}, r.Client.Get(r.RequestCtx, r.Request.NamespacedName, r.ReconciliationSubject)
}

// SetReconciliationSubjectToConcreteSubject sets reconciliation subject to passed concrete object.
func (r *ReconciliationRunner) SetReconciliationSubjectToConcreteSubject(subject client.Object) CloudStackReconcilerMethod {
	return func() (ctrl.Result, error) {
		r.ReconciliationSubject = subject
		return ctrl.Result{}, nil
	}
}

func (r *ReconcilerBase) InitFromMgr(mgr ctrl.Manager, client cloud.Client) {
	r.Client = mgr.GetClient()
	r.BaseLogger = ctrl.Log.WithName("controllers").WithName("name")
	r.Scheme = mgr.GetScheme()
	r.CS = client
}

// GetParent returns the object owning the current resource of passed kind.
func (r *ReconciliationRunner) GetParent(child client.Object, parent client.Object) CloudStackReconcilerMethod {
	return func() (ctrl.Result, error) {
		err := GetOwnerOfKind(r.RequestCtx, r.Client, child, parent)
		return ctrl.Result{}, err
	}
}

// // GetChild returns the
// func (r *ReconciliationRunner) GetChild(parent client.Object, child client.Object) CloudStackReconcilerMethod {
// 	return func() (ctrl.Result, error) {
// 		err := GetOwnerOfKind(r.RequestCtx, r.Client, r.ReconciliationSubject, owner)
// 		return ctrl.Result{}, err
// 	}
// }

func (r *ReconciliationRunner) GetOwnerByKind(owner client.Object) CloudStackReconcilerMethod {
	return func() (ctrl.Result, error) {
		err := GetOwnerOfKind(r.RequestCtx, r.Client, r.ReconciliationSubject, owner)
		return ctrl.Result{}, err
	}
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
