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

package v1beta1

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/aws/cluster-api-provider-cloudstack/pkg/webhookutil"
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

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta1-cloudstackmachinetemplate,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinetemplates,verbs=create;update,versions=v1beta1,name=mcloudstackmachinetemplate.kb.io,admissionReviewVersions=v1beta1
var _ webhook.Defaulter = &CloudStackMachineTemplate{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *CloudStackMachineTemplate) Default() {
	cloudstackmachinetemplatelog.Info("default", "name", r.Name)
	// No defaulted values supported yet.
}

//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-cloudstackmachinetemplate,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinetemplates,verbs=create;update,versions=v1beta1,name=vcloudstackmachinetemplate.kb.io,admissionReviewVersions=v1beta1

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

	affinity := strings.ToLower(spec.Affinity)
	if !(affinity == "" || affinity == "no" || affinity == "pro" || affinity == "anti") {
		errorList = append(errorList, field.Invalid(field.NewPath("spec", "Affinity"), spec.Affinity,
			`Affinity must be "no", "pro", "anti", or unspecified.`))
	}
	if affinity != "no" && affinity != "" && len(spec.AffinityGroupIds) > 0 {
		errorList = append(errorList, field.Forbidden(field.NewPath("spec", "AffinityGroupIds"),
			"AffinityGroupIds cannot be specified when Affinity is specified as anything but `no`"))
	}

	errorList = webhookutil.EnsureFieldExists(spec.Offering, "Offering", errorList)
	errorList = webhookutil.EnsureFieldExists(spec.Template, "Template", errorList)

	return webhookutil.AggregateObjErrors(r.GroupVersionKind().GroupKind(), r.Name, errorList)
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

	errorList = webhookutil.EnsureStringFieldsAreEqual(spec.Offering, oldSpec.Offering, "offering", errorList)
	errorList = webhookutil.EnsureStringFieldsAreEqual(spec.SSHKey, oldSpec.SSHKey, "sshkey", errorList)
	errorList = webhookutil.EnsureStringFieldsAreEqual(spec.Template, oldSpec.Template, "template", errorList)
	errorList = webhookutil.EnsureStringStringMapFieldsAreEqual(&spec.Details, &oldSpec.Details, "details", errorList)

	errorList = webhookutil.EnsureStringFieldsAreEqual(spec.Affinity, oldSpec.Affinity, "affinity", errorList)

	if !reflect.DeepEqual(spec.AffinityGroupIds, oldSpec.AffinityGroupIds) { // Equivalent to other Ensure funcs.
		errorList = append(errorList, field.Forbidden(field.NewPath("spec", "AffinityGroupIds"), "AffinityGroupIds"))
	}

	if spec.IdentityRef != nil && oldSpec.IdentityRef != nil {
		errorList = webhookutil.EnsureStringFieldsAreEqual(
			spec.IdentityRef.Kind, oldSpec.IdentityRef.Kind, "identityRef.Kind", errorList)
		errorList = webhookutil.EnsureStringFieldsAreEqual(
			spec.IdentityRef.Name, oldSpec.IdentityRef.Name, "identityRef.Name", errorList)
	}

	return webhookutil.AggregateObjErrors(r.GroupVersionKind().GroupKind(), r.Name, errorList)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *CloudStackMachineTemplate) ValidateDelete() error {
	cloudstackmachinetemplatelog.Info("validate delete", "name", r.Name)
	// No deletion validations.  Deletion webhook not enabled.
	return nil
}
