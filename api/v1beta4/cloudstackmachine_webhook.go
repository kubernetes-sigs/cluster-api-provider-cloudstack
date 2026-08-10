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

	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/webhookutil"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var cloudstackmachinelog = logf.Log.WithName("cloudstackmachine-resource")

func (r *CloudStackMachine) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta4-cloudstackmachine,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachines,verbs=create;update,versions=v1beta4,name=mcloudstackmachine.kb.io,admissionReviewVersions=v1;v1beta1

var _ admission.Defaulter[*CloudStackMachine] = &CloudStackMachine{}

// Default implements admission.Defaulter so a webhook will be registered for the type
func (r *CloudStackMachine) Default(_ context.Context, obj *CloudStackMachine) error {
	cloudstackmachinelog.V(1).Info("entered api default setting webhook, no defaults to set", "api resource name", obj.Name)
	return nil
}

//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta4-cloudstackmachine,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachines,verbs=create;update,versions=v1beta4,name=vcloudstackmachine.kb.io,admissionReviewVersions=v1;v1beta1

var _ admission.Validator[*CloudStackMachine] = &CloudStackMachine{}

// ValidateCreate implements admission.Validator so a webhook will be registered for the type
func (r *CloudStackMachine) ValidateCreate(_ context.Context, obj *CloudStackMachine) (admission.Warnings, error) {
	cloudstackmachinelog.V(1).Info("entered validate create webhook", "api resource name", obj.Name)

	var errorList field.ErrorList

	errorList = webhookutil.EnsureAtLeastOneFieldExists(obj.Spec.Offering.ID, obj.Spec.Offering.Name, "Offering", errorList)
	errorList = webhookutil.EnsureAtLeastOneFieldExists(obj.Spec.Template.ID, obj.Spec.Template.Name, "Template", errorList)
	if len(obj.Spec.DiskOffering.ID) > 0 || len(obj.Spec.DiskOffering.Name) > 0 {
		errorList = webhookutil.EnsureIntFieldsAreNotNegative(obj.Spec.DiskOffering.CustomSize, "customSizeInGB", errorList)
	}

	return nil, webhookutil.AggregateObjErrors(obj.GroupVersionKind().GroupKind(), obj.Name, errorList)
}

// ValidateUpdate implements admission.Validator so a webhook will be registered for the type
func (r *CloudStackMachine) ValidateUpdate(_ context.Context, oldObj, newObj *CloudStackMachine) (admission.Warnings, error) {
	cloudstackmachinelog.V(1).Info("entered validate update webhook", "api resource name", newObj.Name)

	var errorList field.ErrorList

	oldSpec := oldObj.Spec

	errorList = webhookutil.EnsureEqualStrings(newObj.Spec.Offering.ID, oldSpec.Offering.ID, "offering", errorList)
	errorList = webhookutil.EnsureEqualStrings(newObj.Spec.Offering.Name, oldSpec.Offering.Name, "offering", errorList)
	errorList = webhookutil.EnsureEqualStrings(newObj.Spec.DiskOffering.ID, oldSpec.DiskOffering.ID, "diskOffering", errorList)
	errorList = webhookutil.EnsureEqualStrings(newObj.Spec.DiskOffering.Name, oldSpec.DiskOffering.Name, "diskOffering", errorList)
	errorList = webhookutil.EnsureIntFieldsAreNotNegative(newObj.Spec.DiskOffering.CustomSize, "customSizeInGB", errorList)
	errorList = webhookutil.EnsureEqualStrings(newObj.Spec.DiskOffering.MountPath, oldSpec.DiskOffering.MountPath, "mountPath", errorList)
	errorList = webhookutil.EnsureEqualStrings(newObj.Spec.DiskOffering.Device, oldSpec.DiskOffering.Device, "device", errorList)
	errorList = webhookutil.EnsureEqualStrings(newObj.Spec.DiskOffering.Filesystem, oldSpec.DiskOffering.Filesystem, "filesystem", errorList)
	errorList = webhookutil.EnsureEqualStrings(newObj.Spec.DiskOffering.Label, oldSpec.DiskOffering.Label, "label", errorList)
	errorList = webhookutil.EnsureEqualStrings(newObj.Spec.SSHKey, oldSpec.SSHKey, "sshkey", errorList)
	errorList = webhookutil.EnsureEqualStrings(newObj.Spec.Template.ID, oldSpec.Template.ID, "template", errorList)
	errorList = webhookutil.EnsureEqualStrings(newObj.Spec.Template.Name, oldSpec.Template.Name, "template", errorList)
	errorList = webhookutil.EnsureEqualMapStringString(&newObj.Spec.Details, &oldSpec.Details, "details", errorList)
	errorList = webhookutil.EnsureEqualStrings(newObj.Spec.Affinity, oldSpec.Affinity, "affinity", errorList)

	if !reflect.DeepEqual(newObj.Spec.AffinityGroupIDs, oldSpec.AffinityGroupIDs) {
		errorList = append(errorList, field.Forbidden(field.NewPath("spec", "AffinityGroupIDs"), "AffinityGroupIDs"))
	}

	return nil, webhookutil.AggregateObjErrors(newObj.GroupVersionKind().GroupKind(), newObj.Name, errorList)
}

// ValidateDelete implements admission.Validator so a webhook will be registered for the type
func (r *CloudStackMachine) ValidateDelete(_ context.Context, obj *CloudStackMachine) (admission.Warnings, error) {
	cloudstackmachinelog.V(1).Info("entered validate delete webhook", "api resource name", obj.Name)
	return nil, nil
}
