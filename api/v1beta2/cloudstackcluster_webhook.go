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

package v1beta2

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/webhookutil"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var cloudstackclusterlog = logf.Log.WithName("cloudstackcluster-resource")

func (r *CloudStackCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

var _ webhook.Defaulter = &CloudStackCluster{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *CloudStackCluster) Default() {
	cloudstackclusterlog.V(1).Info("entered api default setting webhook", "api resource name", r.Name)
	// No defaulted values supported yet.
}

// +kubebuilder:webhook:name=vcloudstackcluster.kb.io,groups=infrastructure.cluster.x-k8s.io,resources=cloudstackclusters,versions=v1beta2,verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1beta2-cloudstackcluster,mutating=false,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1beta1

var _ webhook.Validator = &CloudStackCluster{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *CloudStackCluster) ValidateCreate() error {
	cloudstackclusterlog.V(1).Info("entered validate create webhook", "api resource name", r.Name)

	var errorList field.ErrorList

	if (r.Spec.Account != "") && (r.Spec.Domain == "") {
		errorList = append(errorList, field.Required(
			field.NewPath("spec", "account"), "specifying account requires additionally specifying domain"))
	}

	// Require FailureDomains and their respective sub-fields.
	if len(r.Spec.FailureDomains) <= 0 {
		errorList = append(errorList, field.Required(field.NewPath("spec", "FailureDomains"), "FailureDomains"))
	} else {
		for _, fdSpec := range r.Spec.FailureDomains {
			if fdSpec.Zone.Network.Name == "" && fdSpec.Zone.Network.ID == "" {
				errorList = append(errorList, field.Required(
					field.NewPath("spec", "FailureDomains", "Zone", "Network"),
					"each Zone requires a Network specification"))
			}
			if fdSpec.ACSEndpoint.Name == "" || fdSpec.ACSEndpoint.Namespace == "" {
				errorList = append(errorList, field.Required(
					field.NewPath("spec", "FailureDomains", "ACSEndpoint"),
					"Name and Namespace are required"))
			}
		}
	}

	return webhookutil.AggregateObjErrors(r.GroupVersionKind().GroupKind(), r.Name, errorList)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *CloudStackCluster) ValidateUpdate(old runtime.Object) error {
	cloudstackclusterlog.V(1).Info("entered validate update webhook", "api resource name", r.Name)

	spec := r.Spec

	oldCluster, ok := old.(*CloudStackCluster)
	if !ok {
		return errors.NewBadRequest(fmt.Sprintf("expected a CloudStackCluster but got a %T", old))
	}
	oldSpec := oldCluster.Spec

	// No spec fields may be updated.
	errorList := field.ErrorList(nil)
	if !reflect.DeepEqual(oldSpec.FailureDomains, spec.FailureDomains) {
		errorList = append(errorList, field.Forbidden(
			field.NewPath("spec", "FailureDomains"), "FailureDomains and sub-attributes may not be modified after creation"))
	}
	if oldSpec.ControlPlaneEndpoint.Host != "" { // Need to allow one time endpoint setting via CAPC cluster controller.
		errorList = webhookutil.EnsureStringFieldsAreEqual(
			spec.ControlPlaneEndpoint.Host, oldSpec.ControlPlaneEndpoint.Host, "controlplaneendpoint.host", errorList)
		errorList = webhookutil.EnsureStringFieldsAreEqual(
			string(spec.ControlPlaneEndpoint.Port), string(oldSpec.ControlPlaneEndpoint.Port),
			"controlplaneendpoint.port", errorList)
	}

	return webhookutil.AggregateObjErrors(r.GroupVersionKind().GroupKind(), r.Name, errorList)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *CloudStackCluster) ValidateDelete() error {
	cloudstackclusterlog.V(1).Info("entered validate delete webhook", "api resource name", r.Name)
	// No deletion validations.  Deletion webhook not enabled.
	return nil
}
