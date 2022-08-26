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
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/webhookutil"
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

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta2-cloudstackmachinetemplate,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinetemplates,verbs=create;update,versions=v1beta2,name=mcloudstackmachinetemplate.kb.io,admissionReviewVersions=v1beta1
var _ webhook.Defaulter = &CloudStackMachineTemplate{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *CloudStackMachineTemplate) Default() {
	cloudstackmachinetemplatelog.V(1).Info("entered default setting webhook", "api resource name", r.Name)
	// No defaulted values supported yet.
}

//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta2-cloudstackmachinetemplate,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachinetemplates,verbs=create;update,versions=v1beta2,name=vcloudstackmachinetemplate.kb.io,admissionReviewVersions=v1beta1

var _ webhook.Validator = &CloudStackMachineTemplate{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *CloudStackMachineTemplate) ValidateCreate() error {
	cloudstackmachinetemplatelog.V(1).Info("entered validate create webhook", "api resource name", r.Name)

	var (
		errorList field.ErrorList
		spec      = r.Spec.Spec.Spec // CloudStackMachineTemplateSpec.CloudStackMachineTemplateResource.CloudStackMachineSpec
	)

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

	return webhookutil.AggregateObjErrors(r.GroupVersionKind().GroupKind(), r.Name, errorList)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *CloudStackMachineTemplate) ValidateUpdate(old runtime.Object) error {
	return r.ValidateCreate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *CloudStackMachineTemplate) ValidateDelete() error {
	cloudstackmachinetemplatelog.V(1).Info("entered validate delete webhook", "api resource name", r.Name)
	// No deletion validations.  Deletion webhook not enabled.
	return nil
}
