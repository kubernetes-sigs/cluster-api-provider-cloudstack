/*
Copyright 2026 The Kubernetes Authors.

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
	"encoding/json"
	"net/http"
	"testing"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const clusterHeader = `"apiVersion":"infrastructure.cluster.x-k8s.io/v1beta4","kind":"CloudStackCluster","metadata":{"name":"c"}`

func TestPruneEmptyControlPlaneEndpoint(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		wantChanged bool
		want        string // Compared as JSON; ignored when wantChanged is false.
		wantErr     bool
	}{
		{name: "no spec", raw: `{"kind":"CloudStackCluster"}`},
		{name: "no endpoint", raw: `{"spec":{"failureDomains":[]}}`},
		{name: "endpoint is not an object", raw: `{"spec":{"controlPlaneEndpoint":"10.0.0.1"}}`},
		{name: "endpoint is null", raw: `{"spec":{"controlPlaneEndpoint":null}}`},
		{name: "spec is not an object", raw: `{"spec":"oops"}`},
		{name: "valid endpoint is left alone", raw: `{"spec":{"controlPlaneEndpoint":{"host":"10.0.0.1","port":6443}}}`},
		{
			name:        "empty host keeps the port",
			raw:         `{"spec":{"controlPlaneEndpoint":{"host":"","port":6444}}}`,
			wantChanged: true,
			want:        `{"spec":{"controlPlaneEndpoint":{"port":6444}}}`,
		},
		{
			name:        "zero port keeps the host",
			raw:         `{"spec":{"controlPlaneEndpoint":{"host":"10.0.0.1","port":0}}}`,
			wantChanged: true,
			want:        `{"spec":{"controlPlaneEndpoint":{"host":"10.0.0.1"}}}`,
		},
		{
			name:        "empty host and zero port remove the endpoint",
			raw:         `{"spec":{"controlPlaneEndpoint":{"host":"","port":0}}}`,
			wantChanged: true,
			want:        `{"spec":{}}`,
		},
		{
			name:        "empty endpoint object is removed",
			raw:         `{"spec":{"controlPlaneEndpoint":{}}}`,
			wantChanged: true,
			want:        `{"spec":{}}`,
		},
		{
			name:        "other fields and large integers survive",
			raw:         `{"metadata":{"generation":9007199254740993},"spec":{"syncWithACS":true,"controlPlaneEndpoint":{"host":""}}}`,
			wantChanged: true,
			want:        `{"metadata":{"generation":9007199254740993},"spec":{"syncWithACS":true}}`,
		},
		{name: "malformed JSON", raw: `{"spec":`, wantErr: true},
		{name: "not a JSON object", raw: `[1,2]`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			got, changed, err := pruneEmptyControlPlaneEndpoint([]byte(tt.raw))
			if tt.wantErr {
				g.Expect(err).To(gomega.HaveOccurred())
				return
			}
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(changed).To(gomega.Equal(tt.wantChanged))
			if !tt.wantChanged {
				g.Expect(string(got)).To(gomega.Equal(tt.raw))
				return
			}
			g.Expect(string(got)).To(gomega.MatchJSON(tt.want))
		})
	}
}

// defaulterFunc stands in for a future Default() that touches the endpoint itself.
type defaulterFunc func(*CloudStackCluster)

func (f defaulterFunc) Default(_ context.Context, obj *CloudStackCluster) error {
	f(obj)
	return nil
}

var (
	// defaultPort sets the port only when it is missing.
	defaultPort = defaulterFunc(func(c *CloudStackCluster) {
		if c.Spec.ControlPlaneEndpoint.Port == 0 {
			c.Spec.ControlPlaneEndpoint.Port = 6443
		}
	})
	// overridePort rewrites a port that is already set. controller-runtime then drops the "remove host"
	// op from the defaulter's own patch, because it matches the removal from the plain Go round trip, so
	// only the pruner's removal keeps host: "" from reaching schema validation.
	overridePort = defaulterFunc(func(c *CloudStackCluster) {
		c.Spec.ControlPlaneEndpoint.Port = 6443
	})
)

func TestControlPlaneEndpointPruner(t *testing.T) {
	scheme := runtime.NewScheme()
	gomega.NewWithT(t).Expect(AddToScheme(scheme)).To(gomega.Succeed())

	tests := []struct {
		name      string
		defaulter admission.Defaulter[*CloudStackCluster]
		endpoint  string
		want      string // Expected spec.controlPlaneEndpoint after the patch; "" means absent.
	}{
		{
			name:      "stock Default with empty host keeps the port",
			defaulter: &CloudStackCluster{},
			endpoint:  `{"host":"","port":6444}`,
			want:      `{"port":6444}`,
		},
		{
			name:      "stock Default with empty host alone removes the endpoint",
			defaulter: &CloudStackCluster{},
			endpoint:  `{"host":""}`,
			want:      "",
		},
		{
			name:      "stock Default leaves a valid endpoint alone",
			defaulter: &CloudStackCluster{},
			endpoint:  `{"host":"10.0.0.1","port":6443}`,
			want:      `{"host":"10.0.0.1","port":6443}`,
		},
		{
			name:      "Default that sets a missing port still drops the empty host",
			defaulter: defaultPort,
			endpoint:  `{"host":""}`,
			want:      `{"port":6443}`,
		},
		{
			name:      "Default that sets a missing port replaces a zero port and drops the empty host",
			defaulter: defaultPort,
			endpoint:  `{"host":"","port":0}`,
			want:      `{"port":6443}`,
		},
		{
			name:      "Default that sets a missing port keeps a custom port",
			defaulter: defaultPort,
			endpoint:  `{"host":"","port":6444}`,
			want:      `{"port":6444}`,
		},
		{
			name:      "Default that rewrites a set port still drops the empty host",
			defaulter: overridePort,
			endpoint:  `{"host":"","port":6444}`,
			want:      `{"port":6443}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			raw := []byte(`{` + clusterHeader + `,"spec":{"failureDomains":[],"controlPlaneEndpoint":` + tt.endpoint + `}}`)
			pruner := &controlPlaneEndpointPruner{inner: admission.WithDefaulter(scheme, tt.defaulter).Handler}

			resp := pruner.Handle(context.Background(), newRequest(admissionv1.Create, raw))
			g.Expect(resp.Allowed).To(gomega.BeTrue(), "response: %+v", resp.Result)

			patched := applyPatches(g, raw, resp)
			var obj map[string]any
			g.Expect(json.Unmarshal(patched, &obj)).To(gomega.Succeed())
			endpoint, found := obj["spec"].(map[string]any)["controlPlaneEndpoint"]
			if tt.want == "" {
				g.Expect(found).To(gomega.BeFalse(), "endpoint: %v", endpoint)
				return
			}
			got, err := json.Marshal(endpoint)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(string(got)).To(gomega.MatchJSON(tt.want))
		})
	}

	t.Run("malformed request is rejected with 400", func(t *testing.T) {
		g := gomega.NewWithT(t)
		pruner := &controlPlaneEndpointPruner{inner: admission.WithDefaulter(scheme, &CloudStackCluster{}).Handler}
		resp := pruner.Handle(context.Background(), newRequest(admissionv1.Create, []byte(`{"spec":`)))
		g.Expect(resp.Allowed).To(gomega.BeFalse())
		g.Expect(resp.Result.Code).To(gomega.BeEquivalentTo(http.StatusBadRequest))
	})

	t.Run("delete is passed through without reading the object", func(t *testing.T) {
		g := gomega.NewWithT(t)
		pruner := &controlPlaneEndpointPruner{inner: admission.WithDefaulter(scheme, &CloudStackCluster{}).Handler}
		resp := pruner.Handle(context.Background(), newRequest(admissionv1.Delete, nil))
		g.Expect(resp.Allowed).To(gomega.BeTrue())
		g.Expect(resp.Patches).To(gomega.BeEmpty())
	})
}

func newRequest(op admissionv1.Operation, raw []byte) admission.Request {
	return admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{
		UID:       "test",
		Operation: op,
		Object:    runtime.RawExtension{Raw: raw},
	}}
}

// applyPatches applies the response's JSON patch operations in order, as the apiserver does.
func applyPatches(g *gomega.WithT, raw []byte, resp admission.Response) []byte {
	if len(resp.Patches) == 0 {
		return raw
	}
	ops, err := json.Marshal(resp.Patches)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	patch, err := jsonpatch.DecodePatch(ops)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	patched, err := patch.Apply(raw)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	return patched
}
