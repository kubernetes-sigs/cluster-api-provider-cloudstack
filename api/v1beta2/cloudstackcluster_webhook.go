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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
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

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta2-cloudstackcluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=cloudstackclusters,verbs=create;update,versions=v1beta2,name=mcloudstackcluster.kb.io,admissionReviewVersions=v1beta1

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

	// Require FailureDomains and their respective sub-fields.
	if len(r.Spec.FailureDomains) <= 0 {
		errorList = append(errorList, field.Required(field.NewPath("spec", "FailureDomains"), "FailureDomains"))
	} else {
		for _, fdSpec := range r.Spec.FailureDomains { // Require failureDomain names meet k8s qualified name spec.
			for _, errMsg := range validation.IsDNS1123Subdomain(fdSpec.Name) {
				errorList = append(errorList, field.Invalid(
					field.NewPath("spec", "failureDomains", "name"), fdSpec.Name, errMsg))
			}
			if fdSpec.Zone.Network.Name == "" && fdSpec.Zone.Network.ID == "" {
				errorList = append(errorList, field.Required(
					field.NewPath("spec", "failureDomains", "Zone", "Network"),
					"each Zone requires a Network specification"))
			}
			if fdSpec.ACSEndpoint.Name == "" || fdSpec.ACSEndpoint.Namespace == "" {
				errorList = append(errorList, field.Required(
					field.NewPath("spec", "failureDomains", "ACSEndpoint"),
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

	errorList := field.ErrorList(nil)

	if err := ValidateFailureDomainUpdates(oldSpec.FailureDomains, spec.FailureDomains); err != nil {
		errorList = append(errorList, err)
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

// ValidateFailureDomainUpdates verifies that at least one failure domain has not been deleted, and
// failure domains that are held over have not been modified.
func ValidateFailureDomainUpdates(oldFDs, newFDs []CloudStackFailureDomainSpec) *field.Error {
	newFDsByName := map[string]CloudStackFailureDomainSpec{}
	for _, newFD := range newFDs {
		newFDsByName[newFD.Name] = newFD
	}

	atLeastOneRemains := false
	for _, oldFD := range oldFDs {
		if newFD, present := newFDsByName[oldFD.Name]; present {
			atLeastOneRemains = true
			if !FailureDomainsEqual(newFD, oldFD) {
				return field.Forbidden(field.NewPath("spec", "FailureDomains"),
					fmt.Sprintf("Cannot change FailureDomain %s", oldFD.Name))
			}
		}
	}
	if !atLeastOneRemains {
		return field.Forbidden(field.NewPath("spec", "FailureDomains"), "At least one FailureDomain must be unchanged on update.")
	}
	return nil
}

// FailureDomainsEqual is a manual deep equal on failure domains.
func FailureDomainsEqual(fd1, fd2 CloudStackFailureDomainSpec) bool {
	return fd1.Name == fd2.Name &&
		fd1.ACSEndpoint == fd2.ACSEndpoint &&
		fd1.Account == fd2.Account &&
		fd1.Domain == fd2.Domain &&
		fd1.Zone.Name == fd2.Zone.Name &&
		fd1.Zone.ID == fd2.Zone.ID &&
		fd1.Zone.Network.Name == fd2.Zone.Network.Name &&
		fd1.Zone.Network.ID == fd2.Zone.Network.ID &&
		fd1.Zone.Network.Type == fd2.Zone.Network.Type
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *CloudStackCluster) ValidateDelete() error {
	cloudstackclusterlog.V(1).Info("entered validate delete webhook", "api resource name", r.Name)
	// No deletion validations.  Deletion webhook not enabled.
	return nil
}
