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
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	infrastructurev1beta1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	csCtrlrUtils "github.com/aws/cluster-api-provider-cloudstack/controllers/utils"
	"github.com/pkg/errors"
)

// CloudStackAffinityGroupReconciler reconciles a CloudStackAffinityGroup object
type CloudStackAffinityGroupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackaffinitygroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackaffinitygroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackaffinitygroups/finalizers,verbs=update
// Need to watch machine templates for creation of an affinity group.
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinetemplate,verbs=get;list;watch;

func (r *CloudStackAffinityGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	fmt.Println("blah blah blah")
	fmt.Println("blah blah blah")
	fmt.Println("blah blah blah")
	fmt.Printf("%+v", req)
	fmt.Printf("%+v", req)
	fmt.Println("blah blah blah")
	fmt.Println("blah blah blah")
	fmt.Println("blah blah blah")
	fmt.Println("blah blah blah")

	return ctrl.Result{RequeueAfter: csCtrlrUtils.RequeueTimeout}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloudStackAffinityGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	controller, err := ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1beta1.CloudStackAffinityGroup{}).
		WithEventFilter(
			predicate.Funcs{
				UpdateFunc: func(e event.UpdateEvent) bool {
					oldObj := e.ObjectOld.(*infrav1.CloudStackAffinityGroup).DeepCopy()
					newObj := e.ObjectNew.(*infrav1.CloudStackAffinityGroup).DeepCopy()
					oldObj.Status = infrav1.CloudStackAffinityGroupStatus{}
					newObj.Status = infrav1.CloudStackAffinityGroupStatus{}

					return !reflect.DeepEqual(oldObj, newObj)
				},
			},
		).Build(r)
	if err != nil {
		return errors.Wrap(err, "error encountered while building CloudStackAffinityGroup controller")
	}

	// Add a watch on CAPI Cluster objects for unpause and ready events.
	_ = controller.Watch(
		&source.Kind{Type: &infrav1.CloudStackMachineTemplate{}},
		handler.EnqueueRequestsFromMapFunc(MachineTemplatetoAGMapFunc(infrav1.GroupVersion.WithKind("CloudStackAffinityGroup"))))
	// predicate.Funcs{
	// UpdateFunc: func(e event.UpdateEvent) bool {
	// 	oldCluster := e.ObjectOld.(*capiv1.CloudStackMachineTemplate)
	// 	newCluster := e.ObjectNew.(*capiv1.CloudStackMachineTemplate)
	// 	return oldCluster.Spec.Spec.Spec.Af && !newCluster.Spec.Paused
	// },
	// DeleteFunc: func(e event.DeleteEvent) bool { return false },
	// CreateFunc: func(e event.CreateEvent) bool { return true }})
	// Add a watch on CAPI Cluster objects for unpause and ready events.
	return controller.Watch(
		&source.Kind{Type: &capiv1.Cluster{}},
		handler.EnqueueRequestsFromMapFunc(MachineTemplatetoAGMapFunc(infrav1.GroupVersion.WithKind("CloudStackAffinityGroup"))),
		predicate.Funcs{
			// 	UpdateFunc: func(e event.UpdateEvent) bool {
			// 		oldCluster := e.ObjectOld.(*capiv1.Cluster)
			// 		newCluster := e.ObjectNew.(*capiv1.Cluster)
			// 		return oldCluster.Spec.Paused && !newCluster.Spec.Paused
			// 	},
			CreateFunc: func(e event.CreateEvent) bool {
				// _, ok := e.Object.GetAnnotations()[capiv1.PausedAnnotation]
				return true
			},
		})
	//return errors.Wrap(err, "error encountered while building CloudStackCluster controller")
}

// MachineTemplatetoAGMapFunc returns a handler.ToRequestsFunc that watches for
// CloudStackMachineTemplate events and returns reconciliation requests for an infrastructure provider object.
func MachineTemplatetoAGMapFunc(gvk schema.GroupVersionKind) handler.MapFunc {
	return func(o client.Object) []reconcile.Request {
		fmt.Println("Here")
		fmt.Println("Here")
		fmt.Println("Here")
		fmt.Println("Here")
		fmt.Println("Here")
		fmt.Println("Here")
		fmt.Println("Here")
		c, ok := o.(*infrav1.CloudStackMachineTemplate)
		if !ok {
			return nil
		}

		// Return early if the Affinity group setting is none.
		affinityType := c.Spec.Spec.Spec.Affinity
		if affinityType == infrav1.NoAffinity {
			return nil
		}

		return []reconcile.Request{
			{NamespacedName: client.ObjectKey{
				Namespace: c.Namespace,
				Name:      c.Spec.Spec.Spec.Template.Name + "=" + affinityType,
			}},
		}
	}
}
