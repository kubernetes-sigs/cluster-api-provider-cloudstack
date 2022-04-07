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

package controllers

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
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

func (r *CloudStackBaseReconciler) runWith(
	ctx context.Context, req ctrl.Request, fns ...CloudStackReconcilerMethod) (ctrl.Result, error) {
	for _, fn := range fns {
		if rslt, err := fn(ctx, req); err != nil || rslt.Requeue == true || rslt.RequeueAfter != 0 {
			return rslt, err
		}
	}
	return ctrl.Result{}, nil
}

// CheckIfPaused returns with reque later set if paused.
func (r *CloudStackBaseReconciler) CheckIfPaused(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if annotations.IsPaused(r.CAPICluster, r.CSCluster) {
		r.Log.Info("Cluster is paused. Refusing to reconcile.")
		return reconcile.Result{RequeueAfter: requeueTimeout}, nil
	}
	return reconcile.Result{}, nil
}

// patchChangesBackToAPI writes the changes made to the Reconciler's local copy of the reconcilation subject back
// to the API.
func (r *CloudStackBaseReconciler) patchChangesBackToAPI(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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

type CloudStackZoneUser struct {
	CloudStackBaseReconciler
	Zones *infrav1.CloudStackZoneList
}

// FetchZones fetches CloudStackZones owned by a CloudStackCluster via an ownership label.
func (r *CloudStackZoneUser) FetchZones(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	labels := map[string]string{"OwnedBy": r.CSCluster.Name}
	if err := r.Client.List(
		ctx,
		r.Zones,
		client.InNamespace(r.CSCluster.Namespace),
		client.MatchingLabels(labels),
	); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to list zones")
	}
	return ctrl.Result{}, nil
}

// GenerateZones generates a CloudStackClusterZone CRD for each of the CloudStackCluster's spec Zones.
func (r *CloudStackZoneUser) GenerateZones(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	for _, zone := range r.CSCluster.Spec.Zones {
		if err := r.GenerateZone(ctx, zone); err != nil {
			if !strings.Contains(strings.ToLower(err.Error()), "already exists") {
				return reconcile.Result{}, errors.Wrap(err, "error encountered when creating CloudStackZone")
			}
		}
	}
	return ctrl.Result{}, nil
}

// generateZone generates a specified CloudStackZone CRD owned by the CloudStackCluster.
func (r *CloudStackZoneUser) GenerateZone(ctx context.Context, zoneSpec infrav1.Zone) error {
	csZone := &infrav1.CloudStackZone{
		ObjectMeta: metav1.ObjectMeta{
			Name:        strings.ToLower(zoneSpec.Name),
			Namespace:   r.CSCluster.Namespace,
			Labels:      map[string]string{"OwnedBy": r.CSCluster.Name},
			Annotations: map[string]string{},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(r.CSCluster, controlplanev1.GroupVersion.WithKind("CloudStackCluster")),
			},
		},
		Spec: infrav1.CloudStackZoneSpec{Name: zoneSpec.Name},
	}

	if err := r.Client.Create(ctx, csZone); err != nil {
		return errors.Wrap(err, "failed to create zone")
	}
	return nil
}
