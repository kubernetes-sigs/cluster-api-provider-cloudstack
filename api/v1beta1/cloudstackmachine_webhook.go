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

package v1beta1

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/aws/cluster-api-provider-cloudstack/pkg/webhook_utilities"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var cloudstackmachinelog = logf.Log.WithName("cloudstackmachine-resource")

func (r *CloudStackMachine) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta1-cloudstackmachine,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachines,verbs=create;update,versions=v1beta1,name=mcloudstackmachine.kb.io,admissionReviewVersions=v1beta1

var _ webhook.Defaulter = &CloudStackMachine{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *CloudStackMachine) Default() {
	cloudstackmachinelog.Info("default", "name", r.Name)
	// No defaulted values supported yet.
}

//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-cloudstackmachine,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachines,verbs=create;update,versions=v1beta1,name=vcloudstackmachine.kb.io,admissionReviewVersions=v1beta1

var _ webhook.Validator = &CloudStackMachine{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *CloudStackMachine) ValidateCreate() error {
	cloudstackmachinelog.Info("validate create", "name", r.Name)

	var errorList field.ErrorList

	// IdentityRefs must be Secrets.
	if r.Spec.IdentityRef != nil && r.Spec.IdentityRef.Kind != defaultIdentityRefKind {
		errorList = append(errorList, field.Forbidden(field.NewPath("spec", "identityRef", "kind"), "must be a Secret"))
	}

	affinity := strings.ToLower(r.Spec.Affinity)
	if !(affinity == "" || affinity == "no" || affinity == "pro" || affinity == "anti") {
		errorList = append(errorList, field.Invalid(field.NewPath("spec", "Affinity"), r.Spec.Affinity,
			`Affinity must be "no", "pro", "anti", or unspecified.`))
	}
	if (affinity != "no") && affinity != "" && (len(r.Spec.AffinityGroupIds) > 0) {
		errorList = append(errorList, field.Forbidden(field.NewPath("spec", "AffinityGroupIds"),
			"AffinityGroupIds cannot be specified when Affinity is specified as anything but `no`"))
	}

	errorList = webhook_utilities.EnsureFieldExists(r.Spec.Offering, "Offering", errorList)
	errorList = webhook_utilities.EnsureFieldExists(r.Spec.Template, "Template", errorList)

	return webhook_utilities.AggregateObjErrors(r.GroupVersionKind().GroupKind(), r.Name, errorList)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *CloudStackMachine) ValidateUpdate(old runtime.Object) error {
	cloudstackmachinelog.Info("validate update", "name", r.Name)

	var errorList field.ErrorList

	oldMachine, ok := old.(*CloudStackMachine)
	if !ok {
		return errors.NewBadRequest(fmt.Sprintf("expected a CloudStackMachine but got a %T", old))
	}
	oldSpec := oldMachine.Spec

	errorList = webhook_utilities.EnsureStringFieldsAreEqual(r.Spec.Offering, oldSpec.Offering, "offering", errorList)
	errorList = webhook_utilities.EnsureStringFieldsAreEqual(r.Spec.SSHKey, oldSpec.SSHKey, "sshkey", errorList)
	errorList = webhook_utilities.EnsureStringFieldsAreEqual(r.Spec.Template, oldSpec.Template, "template", errorList)
	errorList = webhook_utilities.EnsureStringStringMapFieldsAreEqual(&r.Spec.Details, &oldSpec.Details, "details", errorList)
	errorList = webhook_utilities.EnsureStringFieldsAreEqual(r.Spec.Affinity, oldSpec.Affinity, "template", errorList)
	if !reflect.DeepEqual(r.Spec.AffinityGroupIds, oldSpec.AffinityGroupIds) { // Equivalent to other Ensure funcs.
		errorList = append(errorList, field.Forbidden(field.NewPath("spec", "AffinityGroupIds"), "AffinityGroupIds"))
	}
	if r.Spec.IdentityRef != nil && oldSpec.IdentityRef != nil {
		errorList = webhook_utilities.EnsureStringFieldsAreEqual(
			r.Spec.IdentityRef.Kind, oldSpec.IdentityRef.Kind, "identityRef.Kind", errorList)
		errorList = webhook_utilities.EnsureStringFieldsAreEqual(
			r.Spec.IdentityRef.Name, oldSpec.IdentityRef.Name, "identityRef.Name", errorList)
	}

	return webhook_utilities.AggregateObjErrors(r.GroupVersionKind().GroupKind(), r.Name, errorList)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *CloudStackMachine) ValidateDelete() error {
	cloudstackmachinelog.Info("validate delete", "name", r.Name)
	// No deletion validations.  Deletion webhook not enabled.
	return nil
}
