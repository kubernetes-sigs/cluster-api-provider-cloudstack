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
var cloudstackmachinetemplatelog = logf.Log.WithName("cloudstackmachinetemplate-resource")

func (r *CloudStackMachineTemplate) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1alpha4-cloudstackmachinetemplate,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinetemplates,verbs=create;update,versions=v1alpha4,name=mcloudstackmachinetemplate.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &CloudStackMachineTemplate{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *CloudStackMachineTemplate) Default() {
	cloudstackmachinetemplatelog.Info("default", "name", r.Name)
	// No defaulted values supported yet.
}

//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1alpha4-cloudstackmachinetemplate,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinetemplates,verbs=create;update,versions=v1alpha4,name=vcloudstackmachinetemplate.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &CloudStackMachineTemplate{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *CloudStackMachineTemplate) ValidateCreate() error {
	cloudstackmachinetemplatelog.Info("validate create", "name", r.Name)

	var (
		errorList field.ErrorList
		spec      = r.Spec.Spec.Spec // CloudStackMachineTemplateSpec.CloudStackMachineTemplateResource.CloudStackMachineSpec
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
func (r *CloudStackMachineTemplate) ValidateUpdate(old runtime.Object) error {
	cloudstackmachinetemplatelog.Info("validate update", "name", r.Name)

	var (
		errorList field.ErrorList
		spec      = r.Spec.Spec.Spec // CloudStackMachineTemplateSpec.CloudStackMachineTemplateResource.CloudStackMachineSpec
	)

	oldMachineTemplate, ok := old.(*CloudStackMachineTemplate)
	if !ok {
		return errors.NewBadRequest(fmt.Sprintf("expected a CloudStackMachineTemplate but got a %T", old))
	}
	oldSpec := oldMachineTemplate.Spec.Spec.Spec

	errorList = webhook_utilities.EnsureFieldsAreEqual(spec.Offering, oldSpec.Offering, "offering", errorList)
	errorList = webhook_utilities.EnsureFieldsAreEqual(spec.SSHKey, oldSpec.SSHKey, "sshkey", errorList)
	errorList = webhook_utilities.EnsureFieldsAreEqual(spec.Template, oldSpec.Template, "template", errorList)
	errorList = webhook_utilities.EnsureFieldsAreEqual(spec.IdentityRef.Kind, oldSpec.IdentityRef.Kind, "identityRef.Kind", errorList)
	errorList = webhook_utilities.EnsureFieldsAreEqual(spec.IdentityRef.Name, oldSpec.IdentityRef.Name, "identityRef.Name", errorList)

	return webhook_utilities.AggregateObjErrors(r.GroupVersionKind().GroupKind(), r.Name, errorList)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *CloudStackMachineTemplate) ValidateDelete() error {
	cloudstackmachinetemplatelog.Info("validate delete", "name", r.Name)
	// No deletion validations.  Deletion webhook not enabled.
	return nil
}
