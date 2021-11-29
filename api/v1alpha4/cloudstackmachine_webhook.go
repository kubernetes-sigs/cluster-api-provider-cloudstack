/*
Copyright 2021.

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

package v1alpha4

import (
	"fmt"
	"gitlab.aws.dev/ce-pike/merida/cluster-api-provider-capc/pkg/webhook_utilities"
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

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1alpha4-cloudstackmachine,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachines,verbs=create;update,versions=v1alpha4,name=mcloudstackmachine.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &CloudStackMachine{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *CloudStackMachine) Default() {
	cloudstackmachinelog.Info("default", "name", r.Name)
	// No defaulted values supported yet.
}

//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1alpha4-cloudstackmachine,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachines,verbs=create;update,versions=v1alpha4,name=vcloudstackmachine.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &CloudStackMachine{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *CloudStackMachine) ValidateCreate() error {
	cloudstackmachinelog.Info("validate create", "name", r.Name)

	var (
		errorList field.ErrorList
		spec      = r.Spec
	)

	// IdentityRefs must be Secrets.
	if spec.IdentityRef != nil && spec.IdentityRef.Kind != defaultIdentityRefKind {
		errorList = append(errorList, field.Forbidden(field.NewPath("spec", "identityRef", "kind"), "must be a Secret"))
	}

	errorList = webhook_utilities.EnsureFieldExists(spec.Offering, "Offering", errorList)
	errorList = webhook_utilities.EnsureFieldExists(spec.Template, "Template", errorList)

	return webhook_utilities.AggregateObjErrors(r.GroupVersionKind().GroupKind(), r.Name, errorList)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *CloudStackMachine) ValidateUpdate(old runtime.Object) error {
	cloudstackmachinelog.Info("validate update", "name", r.Name)

	var (
		errorList field.ErrorList
		spec      = r.Spec
	)

	oldMachine, ok := old.(*CloudStackMachine)
	if !ok {
		return errors.NewBadRequest(fmt.Sprintf("expected a CloudStackMachine but got a %T", old))
	}
	oldSpec := oldMachine.Spec

	errorList = webhook_utilities.EnsureStringFieldsAreEqual(spec.Offering, oldSpec.Offering, "offering", errorList)
	errorList = webhook_utilities.EnsureStringFieldsAreEqual(spec.SSHKey, oldSpec.SSHKey, "sshkey", errorList)
	errorList = webhook_utilities.EnsureStringFieldsAreEqual(spec.Template, oldSpec.Template, "template", errorList)
	errorList = webhook_utilities.EnsureStringStringMapFieldsAreEqual(&spec.Details, &oldSpec.Details, "details", errorList)
	if spec.IdentityRef != nil && oldSpec.IdentityRef != nil {
		errorList = webhook_utilities.EnsureStringFieldsAreEqual(spec.IdentityRef.Kind, oldSpec.IdentityRef.Kind, "identityRef.Kind", errorList)
		errorList = webhook_utilities.EnsureStringFieldsAreEqual(spec.IdentityRef.Name, oldSpec.IdentityRef.Name, "identityRef.Name", errorList)
	}

	return webhook_utilities.AggregateObjErrors(r.GroupVersionKind().GroupKind(), r.Name, errorList)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *CloudStackMachine) ValidateDelete() error {
	cloudstackmachinelog.Info("validate delete", "name", r.Name)
	// No deletion validations.  Deletion webhook not enabled.
	return nil
}
