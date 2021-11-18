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
var cloudstackclusterlog = logf.Log.WithName("cloudstackcluster-resource")

func (r *CloudStackCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1alpha4-cloudstackcluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=cloudstackclusters,verbs=create;update,versions=v1alpha4,name=mcloudstackcluster.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &CloudStackCluster{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *CloudStackCluster) Default() {
	cloudstackclusterlog.Info("default", "name", r.Name)
	// No mutations defined yet.
}

//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1alpha4-cloudstackcluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=cloudstackclusters,verbs=create;update,versions=v1alpha4,name=vcloudstackcluster.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &CloudStackCluster{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *CloudStackCluster) ValidateCreate() error {
	cloudstackclusterlog.Info("validate create", "name", r.Name)

	var (
		errorList field.ErrorList
		spec      = r.Spec
	)

	// IdentityRefs must be Secrets.
	if spec.IdentityRef != nil && spec.IdentityRef.Kind != defaultIdentityRefKind {
		errorList = append(errorList, field.Forbidden(field.NewPath("spec", "identityRef", "kind"), "must be a Secret"))
	}

	// Zone and Network are required fields
	errorList = webhook_utilities.EnsureFieldExists(spec.Zone, "Zone", errorList)
	errorList = webhook_utilities.EnsureFieldExists(spec.Network, "Network", errorList)

	return webhook_utilities.AggregateObjErrors(r.GroupVersionKind().GroupKind(), r.Name, errorList)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *CloudStackCluster) ValidateUpdate(old runtime.Object) error {
	cloudstackclusterlog.Info("validate update", "name", r.Name)

	var (
		errorList field.ErrorList
		spec      = r.Spec
	)

	oldCluster, ok := old.(*CloudStackCluster)
	if !ok {
		return errors.NewBadRequest(fmt.Sprintf("expected an CloudStackCluster but got a %T", old))
	}
	oldSpec := oldCluster.Spec

	// IdentityRefs must be Secrets.
	if spec.IdentityRef != nil && spec.IdentityRef.Kind != defaultIdentityRefKind {
		errorList = append(errorList, field.Forbidden(field.NewPath("spec", "identityRef", "kind"), "must be a Secret"))
	}

	// No spec fields may be changed
	errorList = webhook_utilities.EnsureFieldsAreEqual(spec.Zone, oldSpec.Zone, "zone", errorList)
	errorList = webhook_utilities.EnsureFieldsAreEqual(spec.Network, oldSpec.Network, "network", errorList)
	errorList = webhook_utilities.EnsureFieldsAreEqual(spec.IdentityRef.Kind, oldSpec.IdentityRef.Kind, "identityRef.Kind", errorList)
	errorList = webhook_utilities.EnsureFieldsAreEqual(spec.IdentityRef.Name, oldSpec.IdentityRef.Name, "identityRef.Name", errorList)

	return webhook_utilities.AggregateObjErrors(r.GroupVersionKind().GroupKind(), r.Name, errorList)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *CloudStackCluster) ValidateDelete() error {
	cloudstackclusterlog.Info("validate delete", "name", r.Name)
	// No deletion validations.  Deletion webhook not enabled.
	return nil
}
