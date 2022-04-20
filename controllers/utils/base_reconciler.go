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
	"time"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ReconcilerBase is the base set of componenets we use in k8s reconcilers.
// These are items that are not copied for each reconciliation request, and must be written to with caution.
type ReconcilerBase struct {
	BaseLogger      logr.Logger
	Scheme          *runtime.Scheme
	Client          client.Client
	CS              cloud.Client
	reconcileDelete CloudStackReconcilerMethod
	reconcile       CloudStackReconcilerMethod
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
	ConditionalResult     bool
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

// SetupLogger sets up the reconciler's logger to log with name and namespace values.
func (r *ReconciliationRunner) SetupLogger() (res ctrl.Result, retErr error) {
	r.Log = r.BaseLogger.WithValues("name", r.Request.Name, "namespace", r.Request.Namespace)
	return ctrl.Result{}, nil
}

func (r *ReconciliationRunner) IfDeletionTimestampIsZero(fn CloudStackReconcilerMethod) CloudStackReconcilerMethod {
	return func() (ctrl.Result, error) {
		if r.ReconciliationSubject.GetDeletionTimestamp().IsZero() {
			return fn()
		}
		r.ConditionalResult = false
		return ctrl.Result{}, nil
	}
}

func (r *ReconciliationRunner) Else(fn CloudStackReconcilerMethod) CloudStackReconcilerMethod {
	return func() (ctrl.Result, error) {
		if !r.ConditionalResult {
			return fn()
		}
		return ctrl.Result{}, nil
	}
}

// Get the CAPI cluster the reconciliation subject belongs to.
func (r *ReconciliationRunner) GetCAPICluster() (ctrl.Result, error) {
	name := r.ReconciliationSubject.GetLabels()[capiv1.ClusterLabelName]
	fmt.Println(r.ReconciliationSubject.GetLabels())
	if name == "" {
		r.Log.Info("Reconciliation Subject is missing cluster label or cluster does not exist.",
			"SubjectKind", r.ReconciliationSubject.GetObjectKind().GroupVersionKind().Kind)
		return ctrl.Result{RequeueAfter: RequeueTimeout}, nil
	}
	r.CAPICluster = &capiv1.Cluster{}
	key := client.ObjectKey{
		Namespace: r.ReconciliationSubject.GetNamespace(),
		Name:      name,
	}
	if err := r.Client.Get(r.RequestCtx, key, r.CAPICluster); err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to get Cluster/%s", name)
	}

	return ctrl.Result{}, nil
}

// CheckOwnedCRDsForReadiness queries for the readiness of CRDs listed in ReconciliationSubject.
func (r *ReconciliationRunner) CheckOwnedCRDsForReadiness() (ctrl.Result, error) {

	for _, ref := range r.ReconciliationSubject.GetOwnerReferences() {
		gv, err := schema.ParseGroupVersion(ref.APIVersion)
		if err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "encountered when parsing group version %s of object ref", ref.APIVersion)
		}
		gvk := gv.WithKind(ref.Kind)

		owned := &unstructured.Unstructured{}
		owned.SetGroupVersionKind(gvk)

		err = r.Client.Get(r.RequestCtx, client.ObjectKey{Namespace: r.ReconciliationSubject.GetNamespace(), Name: ref.Name}, owned)
		if err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "encountered when requesting owned object with ref %s", ref)
		}

		if ready, found, err := unstructured.NestedBool(owned.Object, "status", "ready"); !found || err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "encountered when parsing ready for object %s", owned)
		} else if !ready {
			r.Log.Info("Owned object of kind %s not ready, requeueing", ref.Kind)
			return ctrl.Result{RequeueAfter: RequeueTimeout}, nil
		}
	}
	return ctrl.Result{}, nil
}

// RequeueIfCloudStackClusterNotReady requeues the reconciliation request if the CloudStackCluster is not ready.
func (r *ReconciliationRunner) RequeueIfCloudStackClusterNotReady() (ctrl.Result, error) {
	if !r.CSCluster.Status.Ready {
		r.Log.Info("CloudStackCluster not ready. Requeuing.")
		return ctrl.Result{RequeueAfter: RequeueTimeout}, nil
	}
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

// RunReconciliationStage runs a CloudStackReconcilerMethod and returns a boolean to indicate whether that stage would
// have returned a result that cuts the process short or not.
func (runner *ReconciliationRunner) ShouldReturn(rslt ctrl.Result, err error) bool {
	if err != nil {
		return true
	} else if rslt.Requeue == true || rslt.RequeueAfter != time.Duration(0) {
		return true
	}
	return false
}

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

// InitFromMgr just initiates a ReconcilerBase using given manager's fields/methods.
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

func (r *ReconciliationRunner) GetOwnerByKind(owner client.Object) CloudStackReconcilerMethod {
	return func() (ctrl.Result, error) {
		err := GetOwnerOfKind(r.RequestCtx, r.Client, r.ReconciliationSubject, owner)
		return ctrl.Result{}, err
	}
}
