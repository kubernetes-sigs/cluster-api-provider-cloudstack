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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/webhookutil"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var cloudstackclusterlog = logf.Log.WithName("cloudstackcluster-resource")

// cloudStackClusterMutatePath must match the path in the mutating webhook marker below.
const cloudStackClusterMutatePath = "/mutate-infrastructure-cluster-x-k8s-io-v1beta4-cloudstackcluster"

func (r *CloudStackCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	// The defaulting webhook is registered by hand so its response can carry explicit removals for an
	// empty controlPlaneEndpoint. The builder would register a plain defaulter at the same path, so it is
	// only given the validator here.
	mgr.GetWebhookServer().Register(cloudStackClusterMutatePath, &admission.Webhook{
		Handler: &controlPlaneEndpointPruner{inner: admission.WithDefaulter(mgr.GetScheme(), r).Handler},
	})
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithValidator(r).
		Complete()
}

// controlPlaneEndpointPruner keeps manifests that leave spec.controlPlaneEndpoint.host empty working on
// v1beta4. Before v1beta4, host: "" was accepted and CAPC filled in a free public IP during reconcile.
// The v1beta4 hub reuses CAPI's core/v1beta2 APIEndpoint, whose schema needs host to be non-empty,
// port to be at least 1, and the endpoint to have at least one property. The stock cluster templates
// send host: "" when CLUSTER_ENDPOINT_IP is set to an empty string, so they would now be rejected.
//
// Mutating webhooks run before the apiserver checks the CRD schema, so removing the empty fields here
// restores the old behaviour. This cannot live in Default(): host and port are omitempty, so an empty
// value decodes the same as a missing one. Default() then either changes nothing, so no patch is sent,
// or has to rewrite the whole endpoint, which drops a port set next to an empty host. controller-runtime
// also discards "remove" operations that only mirror the Go round trip unless
// DefaulterRemoveUnknownOrOmitableFields is set, and CAPI v1.11 stopped using that option.
//
// The pruner therefore works on the raw request. It removes the empty fields first and then runs the
// defaulter on the pruned request. JSON patch operations apply in order, so the removals followed by the
// defaulter's patch (computed against the pruned request) take the original request to the defaulted
// one. A future Default() that touches the endpoint, such as defaulting the port, composes with this.
type controlPlaneEndpointPruner struct {
	inner admission.Handler
}

func (h *controlPlaneEndpointPruner) Handle(ctx context.Context, req admission.Request) admission.Response {
	if req.Operation == admissionv1.Delete {
		return h.inner.Handle(ctx, req)
	}

	pruned, changed, err := pruneEmptyControlPlaneEndpoint(req.Object.Raw)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	if !changed {
		return h.inner.Handle(ctx, req)
	}

	removals := admission.PatchResponseFromRaw(req.Object.Raw, pruned)
	if !removals.Allowed {
		return removals
	}

	prunedReq := req
	prunedReq.Object = runtime.RawExtension{Raw: pruned}
	resp := h.inner.Handle(ctx, prunedReq)
	if !resp.Allowed {
		return resp
	}
	resp.Patches = slices.Concat(removals.Patches, resp.Patches)
	patchType := admissionv1.PatchTypeJSONPatch
	resp.PatchType = &patchType
	return resp
}

// pruneEmptyControlPlaneEndpoint removes host: "" and port: 0 from spec.controlPlaneEndpoint, and the
// endpoint itself when nothing is left. A port set next to an empty host is kept, because the
// controller uses it as the public port when it picks the IP.
func pruneEmptyControlPlaneEndpoint(raw []byte) ([]byte, bool, error) {
	var obj map[string]any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber() // Avoid rounding large integers elsewhere in the object through float64.
	if err := dec.Decode(&obj); err != nil {
		return nil, false, err
	}

	spec, _ := obj["spec"].(map[string]any)
	endpoint, ok := spec["controlPlaneEndpoint"].(map[string]any)
	if !ok {
		return raw, false, nil
	}

	changed := false
	if host, ok := endpoint["host"].(string); ok && host == "" {
		delete(endpoint, "host")
		changed = true
	}
	if port, ok := endpoint["port"].(json.Number); ok && port.String() == "0" {
		delete(endpoint, "port")
		changed = true
	}
	if len(endpoint) == 0 {
		delete(spec, "controlPlaneEndpoint")
		changed = true
	}
	if !changed {
		return raw, false, nil
	}

	pruned, err := json.Marshal(obj)
	return pruned, true, err
}

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta4-cloudstackcluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=cloudstackclusters,verbs=create;update,versions=v1beta4,name=mcloudstackcluster.kb.io,admissionReviewVersions=v1;v1beta1

var _ admission.Defaulter[*CloudStackCluster] = &CloudStackCluster{}

// Default implements admission.Defaulter so a webhook will be registered for the type
func (r *CloudStackCluster) Default(_ context.Context, obj *CloudStackCluster) error {
	cloudstackclusterlog.V(1).Info("entered api default setting webhook", "api resource name", obj.Name)
	return nil
}

// +kubebuilder:webhook:name=vcloudstackcluster.kb.io,groups=infrastructure.cluster.x-k8s.io,resources=cloudstackclusters,versions=v1beta4,verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1beta4-cloudstackcluster,mutating=false,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1;v1beta1

var _ admission.Validator[*CloudStackCluster] = &CloudStackCluster{}

// ValidateCreate implements admission.Validator so a webhook will be registered for the type
func (r *CloudStackCluster) ValidateCreate(_ context.Context, obj *CloudStackCluster) (admission.Warnings, error) {
	cloudstackclusterlog.V(1).Info("entered validate create webhook", "api resource name", obj.Name)

	var errorList field.ErrorList

	if len(obj.Spec.FailureDomains) == 0 {
		errorList = append(errorList, field.Required(field.NewPath("spec", "FailureDomains"), "FailureDomains"))
	} else {
		for _, fdSpec := range obj.Spec.FailureDomains {
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

	return nil, webhookutil.AggregateObjErrors(obj.GroupVersionKind().GroupKind(), obj.Name, errorList)
}

// ValidateUpdate implements admission.Validator so a webhook will be registered for the type
func (r *CloudStackCluster) ValidateUpdate(_ context.Context, oldObj, newObj *CloudStackCluster) (admission.Warnings, error) {
	cloudstackclusterlog.V(1).Info("entered validate update webhook", "api resource name", newObj.Name)

	spec := newObj.Spec
	oldSpec := oldObj.Spec

	errorList := field.ErrorList(nil)

	if err := ValidateFailureDomainUpdates(oldSpec.FailureDomains, spec.FailureDomains); err != nil {
		errorList = append(errorList, err)
	}

	if oldSpec.ControlPlaneEndpoint.Host != "" {
		errorList = webhookutil.EnsureEqualStrings(
			spec.ControlPlaneEndpoint.Host, oldSpec.ControlPlaneEndpoint.Host, "controlplaneendpoint.host", errorList)
		errorList = webhookutil.EnsureEqualStrings(
			string(spec.ControlPlaneEndpoint.Port), string(oldSpec.ControlPlaneEndpoint.Port),
			"controlplaneendpoint.port", errorList)
	}

	return nil, webhookutil.AggregateObjErrors(newObj.GroupVersionKind().GroupKind(), newObj.Name, errorList)
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

// ValidateDelete implements admission.Validator so a webhook will be registered for the type
func (r *CloudStackCluster) ValidateDelete(_ context.Context, obj *CloudStackCluster) (admission.Warnings, error) {
	cloudstackclusterlog.V(1).Info("entered validate delete webhook", "api resource name", obj.Name)
	return nil, nil
}
