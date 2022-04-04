/*
Copyright 2022.

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

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	csCtrlrUtils "github.com/aws/cluster-api-provider-cloudstack/controllers/utils"
	"github.com/aws/cluster-api-provider-cloudstack/pkg/cloud"
	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

// CloudStackIsolatedNetworkReconciler reconciles a CloudStackIsolatedNetwork object
type CloudStackIsolatedNetworkReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	CS     cloud.Client
}

type IsoNetContext struct {
	IsoNet       *infrav1.CloudStackIsolatedNetwork
	OwnerCluster *infrav1.CloudStackCluster
	OwnerZone    *infrav1.CloudStackZone
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackisolatednetworks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackisolatednetworks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackisolatednetworks/finalizers,verbs=update

func (r *CloudStackIsolatedNetworkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (retRes ctrl.Result, retErr error) {
	log := r.Log.WithValues("IsolatedNetwork", req.Name, "namespace", req.Namespace)
	log.V(1).Info("Reconcile CloudStackIsolatedNetwork")

	crds, err := r.FetchRelatedResources(ctx, req)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Setup patcher. This ensures modifications to the csIsoNet copy fetched above are patched into the origin.
	patchHelper, err := patch.NewHelper(crds.IsoNet, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}
	defer func() { // If there was no error on return, but the patch fails, set the error accordingly.
		if err = patchHelper.Patch(ctx, crds.IsoNet); err != nil {
			msg := "error patching CloudStackIsolatedNetwork %s/%s"
			err = errors.Wrapf(err, msg, crds.IsoNet.Namespace, crds.IsoNet.Name)
			retErr = multierror.Append(retErr, err)
		}
	}()

	if err := r.CS.FetchIsolatedNetwork(crds.OwnerZone, crds.IsoNet); err != nil { // If err, then network doesn't exist.
		return ctrl.Result{}, r.CS.CreateIsolatedNetwork(crds.OwnerZone, crds.OwnerCluster)
	}
	// if err := r.CS.INetAssociatePublicIPAddress(crds.IsoNet); err != nil {
	// 	return ctrl.Result{}, errors.Wrapf(err, "error encountered when associating public IP address to csCluster")
	// }
	// if err := r.CS.GetOrCreateLoadBalancerRule(crds.IsoNet); err != nil {
	// 	return ctrl.Result{}, err
	// }
	crds.IsoNet.Status.Ready = true

	return ctrl.Result{}, nil
}

// FetchRelatedResources fetches kubernetes resources required to reconcile an isolated network.
func (r *CloudStackIsolatedNetworkReconciler) FetchRelatedResources(
	ctx context.Context, req ctrl.Request,
) (*IsoNetContext, error) {
	var err error

	crds := &IsoNetContext{}
	// Fetch the CloudStackIsolatedNetwork.
	if err := r.Client.Get(ctx, req.NamespacedName, crds.IsoNet); err != nil {
		return nil, err
	}
	meta := crds.IsoNet.ObjectMeta
	if crds.OwnerCluster, err = csCtrlrUtils.GetOwnerCloudStackCluster(ctx, r.Client, meta); err != nil {
		return nil, err
	}
	if crds.OwnerZone, err = csCtrlrUtils.GetOwnerZone(ctx, r.Client, meta); err != nil {
		return nil, err
	}

	return crds, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloudStackIsolatedNetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.CloudStackIsolatedNetwork{}).
		Complete(r)
}
