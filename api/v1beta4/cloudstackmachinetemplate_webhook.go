/*
Copyright 2024 The Kubernetes Authors.

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

package v1beta4

import (
	"context"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/webhookutil"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var cloudstackmachinetemplatelog = logf.Log.WithName("cloudstackmachinetemplate-resource")

func (r *CloudStackMachineTemplate) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta4-cloudstackmachinetemplate,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinetemplates,verbs=create;update,versions=v1beta4,name=mcloudstackmachinetemplate.kb.io,admissionReviewVersions=v1;v1beta1

var _ admission.Defaulter[*CloudStackMachineTemplate] = &CloudStackMachineTemplate{}

// Default implements admission.Defaulter so a webhook will be registered for the type
func (r *CloudStackMachineTemplate) Default(_ context.Context, obj *CloudStackMachineTemplate) error {
	cloudstackmachinetemplatelog.V(1).Info("entered default setting webhook", "api resource name", obj.Name)
	return nil
}

// +kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta4-cloudstackmachinetemplate,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinetemplates,verbs=create;update,versions=v1beta4,name=vcloudstackmachinetemplate.kb.io,admissionReviewVersions=v1;v1beta1

var _ admission.Validator[*CloudStackMachineTemplate] = &CloudStackMachineTemplate{}

// ValidateCreate implements admission.Validator so a webhook will be registered for the type
func (r *CloudStackMachineTemplate) ValidateCreate(_ context.Context, obj *CloudStackMachineTemplate) (admission.Warnings, error) {
	cloudstackmachinetemplatelog.V(1).Info("entered validate create webhook", "api resource name", obj.Name)

	var errorList field.ErrorList

	spec := obj.Spec.Template.Spec

	affinity := strings.ToLower(spec.Affinity)
	if !(affinity == "" || affinity == "no" || affinity == "pro" || affinity == "anti") {
		errorList = append(errorList, field.Invalid(field.NewPath("spec", "Affinity"), spec.Affinity,
			`Affinity must be "no", "pro", "anti", or unspecified.`))
	}
	if affinity != "no" && affinity != "" && len(spec.AffinityGroupIDs) > 0 {
		errorList = append(errorList, field.Forbidden(field.NewPath("spec", "AffinityGroupIDs"),
			"AffinityGroupIDs cannot be specified when Affinity is specified as anything but `no`"))
	}

	errorList = webhookutil.EnsureAtLeastOneFieldExists(spec.Offering.ID, spec.Offering.Name, "Offering", errorList)
	errorList = webhookutil.EnsureAtLeastOneFieldExists(spec.Template.ID, spec.Template.Name, "Template", errorList)

	return nil, webhookutil.AggregateObjErrors(obj.GroupVersionKind().GroupKind(), obj.Name, errorList)
}

// ValidateUpdate implements admission.Validator so a webhook will be registered for the type
func (r *CloudStackMachineTemplate) ValidateUpdate(_ context.Context, oldObj, newObj *CloudStackMachineTemplate) (admission.Warnings, error) {
	cloudstackmachinetemplatelog.V(1).Info("entered validate update webhook", "api resource name", newObj.Name)

	spec := newObj.Spec.Template.Spec
	oldSpec := oldObj.Spec.Template.Spec

	errorList := field.ErrorList(nil)
	errorList = webhookutil.EnsureEqualStrings(spec.Offering.ID, oldSpec.Offering.ID, "offering", errorList)
	errorList = webhookutil.EnsureEqualStrings(spec.Offering.Name, oldSpec.Offering.Name, "offering", errorList)
	errorList = webhookutil.EnsureEqualStrings(spec.DiskOffering.ID, oldSpec.DiskOffering.ID, "diskOffering", errorList)
	errorList = webhookutil.EnsureEqualStrings(spec.DiskOffering.Name, oldSpec.DiskOffering.Name, "diskOffering", errorList)
	errorList = webhookutil.EnsureEqualStrings(spec.SSHKey, oldSpec.SSHKey, "sshkey", errorList)
	errorList = webhookutil.EnsureEqualStrings(spec.Template.ID, oldSpec.Template.ID, "template", errorList)
	errorList = webhookutil.EnsureEqualStrings(spec.Template.Name, oldSpec.Template.Name, "template", errorList)
	errorList = webhookutil.EnsureEqualMapStringString(&spec.Details, &oldSpec.Details, "details", errorList)
	errorList = webhookutil.EnsureEqualStrings(spec.Affinity, oldSpec.Affinity, "affinity", errorList)

	if !reflect.DeepEqual(spec.AffinityGroupIDs, oldSpec.AffinityGroupIDs) {
		errorList = append(errorList, field.Forbidden(field.NewPath("spec", "AffinityGroupIDs"), "AffinityGroupIDs"))
	}

	return nil, webhookutil.AggregateObjErrors(newObj.GroupVersionKind().GroupKind(), newObj.Name, errorList)
}

// ValidateDelete implements admission.Validator so a webhook will be registered for the type
func (r *CloudStackMachineTemplate) ValidateDelete(_ context.Context, obj *CloudStackMachineTemplate) (admission.Warnings, error) {
	cloudstackmachinetemplatelog.V(1).Info("entered validate delete webhook", "api resource name", obj.Name)
	return nil, nil
}
