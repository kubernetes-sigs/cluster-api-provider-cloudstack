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
	"strings"
	"time"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	BaseLogger logr.Logger
	Scheme     *runtime.Scheme
	Client     client.Client
	CS         cloud.Client
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
	ReconcilerBase
	CloudStackBaseContext
	ReconciliationSubject client.Object // Underlying crd interface.
	ConditionalResult     bool          // Stores a conidtinal result for stringing if else type methods.
	returnEarly           bool          // A signal that the reconcile should return early.
	ReconcileDelete       CloudStackReconcilerMethod
	Reconcile             CloudStackReconcilerMethod
}

type ConcreteRunner interface {
	ReconcileDelete() (ctrl.Result, error)
	Reconcile() (ctrl.Result, error)
	GetReconcilationSubject() client.Object
}

func NewRunner(concreteRunner ConcreteRunner, subject client.Object) ReconciliationRunner {
	r := ReconciliationRunner{}
	r.CSCluster = &infrav1.CloudStackCluster{}
	r.CAPICluster = &capiv1.Cluster{}
	r.ReconciliationSubject = subject
	r.Reconcile = concreteRunner.Reconcile
	r.ReconcileDelete = concreteRunner.ReconcileDelete
	return r
}

func (r *ReconciliationRunner) GetReconcilationSubject() client.Object {
	return r.ReconciliationSubject
}

func (r *ReconciliationRunner) UsingBaseReconciler(base ReconcilerBase) *ReconciliationRunner {
	r.ReconcilerBase = base
	return r
}

// ForRequest sets the reconciliation request.
func (r *ReconciliationRunner) ForRequest(req ctrl.Request) *ReconciliationRunner {
	r.Request = req
	return r
}

// WithRequestCtx sets the request context.
func (r *ReconciliationRunner) WithRequestCtx(ctx context.Context) *ReconciliationRunner {
	r.RequestCtx = ctx
	return r
}

// SetupLogger sets up the reconciler's logger to log with name and namespace values.
func (r *ReconciliationRunner) SetupLogger() (res ctrl.Result, retErr error) {
	r.Log = r.BaseLogger.WithValues("name", r.Request.Name, "namespace", r.Request.Namespace)
	return ctrl.Result{}, nil
}

func (r *ReconciliationRunner) IfDeletionTimestampIsZero(fn CloudStackReconcilerMethod) CloudStackReconcilerMethod {
	return func() (ctrl.Result, error) {
		if r.ConditionalResult = r.ReconciliationSubject.GetDeletionTimestamp().IsZero(); r.ConditionalResult {
			return fn()
		}
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

// GetCAPICluster gets the CAPI cluster the reconciliation subject belongs to.
func (r *ReconciliationRunner) GetCAPICluster() (ctrl.Result, error) {
	name := r.ReconciliationSubject.GetLabels()[capiv1.ClusterLabelName]
	if name == "" {
		r.Log.V(1).Info("Reconciliation Subject is missing cluster label or cluster does not exist. Skipping CAPI Cluster fetch.",
			"SubjectKind", r.ReconciliationSubject.GetObjectKind().GroupVersionKind().Kind)
		return ctrl.Result{}, nil
	}
	r.CAPICluster = &capiv1.Cluster{}
	key := client.ObjectKey{
		Namespace: r.ReconciliationSubject.GetNamespace(),
		Name:      name,
	}
	if err := r.Client.Get(r.RequestCtx, key, r.CAPICluster); err != nil {
		return ctrl.Result{}, errors.Wrap(client.IgnoreNotFound(err), "error encountered while getting CAPI Cluster "+name)
	} else if r.CAPICluster.Name == "" {
		return r.RequeueWithMessage("Cluster not fetched.")
	}
	return ctrl.Result{}, nil
}

// GetCSCluster gets the CAPI cluster the reconciliation subject belongs to.
func (r *ReconciliationRunner) GetCSCluster() (ctrl.Result, error) {
	name := r.ReconciliationSubject.GetLabels()[capiv1.ClusterLabelName]
	if name == "" {
		r.Log.V(1).Info("Reconciliation Subject is missing cluster label or cluster does not exist. Skipping CloudStackCluster fetch.",
			"SubjectKind", r.ReconciliationSubject.GetObjectKind().GroupVersionKind().Kind)
		return ctrl.Result{}, nil
	}
	r.CSCluster = &infrav1.CloudStackCluster{}
	key := client.ObjectKey{
		Namespace: r.ReconciliationSubject.GetNamespace(),
		Name:      name,
	}
	err := r.Client.Get(r.RequestCtx, key, r.CSCluster)
	return ctrl.Result{}, errors.Wrap(client.IgnoreNotFound(err), "error encountered while getting CAPI Cluster "+name)
}

// CheckOwnedCRDsForReadiness queries for the readiness of CRDs of GroupVersionKind passed.
func (r *ReconciliationRunner) CheckOwnedCRDsForReadiness(gvks ...schema.GroupVersionKind) CloudStackReconcilerMethod {
	return func() (ctrl.Result, error) {
		// For each GroupVersionKind...
		for _, gvk := range gvks {
			// Query to find objects of this kind.
			// TODO: Filter use owner label to filter. Will need to build generic owner labeling system too.
			potentiallyOnwedObjs := &unstructured.UnstructuredList{}
			potentiallyOnwedObjs.SetGroupVersionKind(gvk)
			err := r.Client.List(r.RequestCtx, potentiallyOnwedObjs)
			if err != nil {
				return ctrl.Result{}, errors.Wrapf(err, "encountered when requesting owned objects with gvk %s", gvk)
			}

			// Filter objects not actually owned by reconciliation subject via owner reference UID.
			ownedObjs := []unstructured.Unstructured{}
			for _, pOwned := range potentiallyOnwedObjs.Items {
				refs := pOwned.GetOwnerReferences()
				for _, ref := range refs {
					if ref.UID == r.ReconciliationSubject.GetUID() {
						ownedObjs = append(ownedObjs, pOwned)
					}
				}

			}

			// Check that found objects are ready.
			for _, owned := range ownedObjs {
				if ready, found, err := unstructured.NestedBool(owned.Object, "status", "ready"); err != nil {
					return ctrl.Result{}, errors.Wrapf(err, "encountered when parsing ready for object %s", owned)
				} else if !found || !ready {
					if name, found, err := unstructured.NestedString(owned.Object, "metadata", "name"); err != nil {
						return ctrl.Result{}, errors.Wrapf(err, "encountered when parsing name for object %s", owned)
					} else if !found {
						r.Log.Info(fmt.Sprintf("Owned object of kind %s not ready, requeueing", gvk.Kind))
						return ctrl.Result{RequeueAfter: RequeueTimeout}, nil
					} else {
						r.Log.Info(fmt.Sprintf("Owned object %s of kind %s not ready, requeueing", name, gvk.Kind))
						return ctrl.Result{RequeueAfter: RequeueTimeout}, nil
					}
				}
			}
		}
		return ctrl.Result{}, nil
	}
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

// RequeueWithMessage is a convenience method to log requeue message and then return a result with RequeueAfter set.
func (r *ReconciliationRunner) ReturnWrappedError(err error, msg string) (ctrl.Result, error) {
	return ctrl.Result{}, errors.Wrap(err, msg)
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
func (r *ReconciliationRunner) ShouldReturn(rslt ctrl.Result, err error) bool {
	if err != nil {
		return true
	} else if rslt.Requeue || rslt.RequeueAfter != time.Duration(0) {
		return true
	}
	return false
}

// RunReconciliationStages runs CloudStackReconcilerMethods in order and exits if an error or requeue condition is set.
// On exit patches changes back to API.
func (r *ReconciliationRunner) RunReconciliationStages(fns ...CloudStackReconcilerMethod) (ctrl.Result, error) {
	for _, fn := range fns {
		if rslt, err := fn(); err != nil {
			return rslt, err
		} else if rslt.Requeue || rslt.RequeueAfter != time.Duration(0) || r.returnEarly {
			return rslt, nil
		}
	}
	return ctrl.Result{}, nil
}

// RunBaseReconciliationStages runs the base reconciliation stages which are to setup the logger, get the reconciliation
// subject, get CAPI and CloudStackClusters, and call either r.Reconcile or r.ReconcileDelete.
func (r *ReconciliationRunner) RunBaseReconciliationStages() (res ctrl.Result, retErr error) {
	defer func() {
		if err := r.Patcher.Patch(r.RequestCtx, r.ReconciliationSubject); err != nil {
			if !strings.Contains(err.Error(), "is invalid: status.ready") {
				err = errors.Wrapf(err, "error patching reconciliation subject")
				retErr = multierror.Append(retErr, err)
			}
		}
	}()
	return r.RunReconciliationStages(
		r.SetupLogger,
		r.GetReconciliationSubject,
		r.GetCAPICluster,
		r.GetCSCluster,
		r.IfDeletionTimestampIsZero(r.Reconcile),
		r.Else(r.ReconcileDelete),
	)
}

// SetReturnEarly sets the runner to return early. This causes the runner to break from running further
// reconciliation stages and return whatever result the current method returns.
func (r *ReconciliationRunner) SetReturnEarly() {
	r.returnEarly = true
}

// CheckIfPaused returns with reque later set if paused.
func (r *ReconciliationRunner) CheckIfPaused() (ctrl.Result, error) {
	r.Log.V(1).Info("Checking if paused.")
	if annotations.IsPaused(r.CAPICluster, r.ReconciliationSubject) {
		return r.RequeueWithMessage("Cluster is paused. Refusing to reconcile.")
	}
	return reconcile.Result{}, nil
}

// GetReconcilationSubject gets the reconciliation subject of type defined by the concrete reconciler. It also sets up
// a patch helper at this point.
func (r *ReconciliationRunner) GetReconciliationSubject() (res ctrl.Result, reterr error) {
	r.Log.V(1).Info("Getting reconciliation subject.")
	err := client.IgnoreNotFound(r.Client.Get(r.RequestCtx, r.Request.NamespacedName, r.ReconciliationSubject))
	if r.ReconciliationSubject.GetName() == "" {
		r.SetReturnEarly()
	}
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "error encountered while fetching reconciliation subject.")
	}
	return r.SetupPatcher()
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

// GetOwnerOfKind uses the ReconciliationSubject's owner references to get the owner object of kind passed.
func (r *ReconciliationRunner) GetOwnerOfKind(owner client.Object) CloudStackReconcilerMethod {
	return func() (ctrl.Result, error) {
		err := GetOwnerOfKind(r.RequestCtx, r.Client, r.ReconciliationSubject, owner)
		return ctrl.Result{}, err
	}
}

// NewChildObjectMeta creates a meta object with ownership reference and labels matching the current cluster.
func (r *ReconciliationRunner) NewChildObjectMeta(name string) metav1.ObjectMeta {
	ownerGVK := r.ReconciliationSubject.GetObjectKind().GroupVersionKind()
	return metav1.ObjectMeta{
		Name:      strings.ToLower(name),
		Namespace: r.Request.Namespace,
		Labels: map[string]string{capiv1.ClusterLabelName: r.CAPICluster.Name,
			infrav1.CloudStackClusterLabelName: r.CSCluster.ClusterName},
		OwnerReferences: []metav1.OwnerReference{
			*metav1.NewControllerRef(r.ReconciliationSubject, ownerGVK),
		},
	}
}

// RequeueIfMissingBaseCRDs checks that the ReconciliationSubject, CAPI Cluster, and CloudStackCluster objects were
// actually fetched and reques if not. The base reconciliation stages will continue even if not so as to allow deletion.
func (r *ReconciliationRunner) RequeueIfMissingBaseCRDs() (ctrl.Result, error) {
	if r.ReconciliationSubject.GetName() == "" {
		return r.RequeueWithMessage("Reconciliation subject wasn't found. Requeueing.")
	} else if r.CSCluster.GetName() == "" {
		return r.RequeueWithMessage("CloudStackCluster wasn't found. Requeueing.")
	} else if r.CAPICluster.GetName() == "" {
		return r.RequeueWithMessage("CAPI Cluster wasn't found. Requeueing.")
	}
	return ctrl.Result{}, nil
}

// GetObjectByName gets an object by name and type of object. The namespace is assumed to be the same
// as the ReconciliationSubject. Not found is not considered an error. Check the object after.
func (r *ReconciliationRunner) GetObjectByName(name string, target client.Object, nameGetter ...func() string) CloudStackReconcilerMethod {
	return func() (ctrl.Result, error) {
		if len(nameGetter) == 1 {
			name = nameGetter[0]()
		}
		name = strings.ToLower(name)
		objectKey := client.ObjectKey{Name: strings.ToLower(name), Namespace: r.Request.Namespace}
		return r.ReturnWrappedError(
			client.IgnoreNotFound(r.Client.Get(r.RequestCtx, objectKey, target)), "failed to get object")
	}
}
