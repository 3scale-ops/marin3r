package podv1mutator

import (
	"context"
	"encoding/json"
	"sort"
	"testing"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/deprecated/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestPodMutator_Handle(t *testing.T) {
	type fields struct {
		Client  client.Client
		decoder *admission.Decoder
	}
	type args struct {
		ctx context.Context
		req admission.Request
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []byte
	}{
		{
			name: "Mutates pod",
			fields: fields{
				Client: fake.NewFakeClient(),
				decoder: func() *admission.Decoder {
					d, _ := admission.NewDecoder(scheme.Scheme)
					return d
				}(),
			},
			args: args{
				ctx: context.TODO(),
				req: admission.Request{
					AdmissionRequest: admissionv1beta1.AdmissionRequest{
						UID:       "xxxx",
						Kind:      metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
						Resource:  metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
						Namespace: "default",
						Operation: admissionv1beta1.Create,
						UserInfo:  authenticationv1.UserInfo{Username: "xxxx", UID: "xxxx"},
						Object: runtime.RawExtension{
							Raw: []byte(`
								{
									"apiVersion": "v1",
									"kind": "Pod",
									"metadata": {
									  "name": "myapp-pod",
									  "namespace": "default",
									  "creationTimestamp": null,
									  "annotations": {
										"marin3r.3scale.net/node-id": "test"
									  }
									},
									"spec": {
									  "containers": [
										{
										  "name": "myapp",
										  "image": "myapp",
										  "resources": {}
										}
									  ]
									},
									"status": {}
								}
								`,
							),
							Object: nil,
						},
					},
				},
			},
			want: []byte(`[{"op":"add","path":"/spec/containers/1","value":{"args":["-c","/etc/envoy/bootstrap/config.json","--service-node","test","--service-cluster","test"],"command":["envoy"],"image":"envoyproxy/envoy:v1.14.1","name":"envoy-sidecar","resources":{},"volumeMounts":[{"mountPath":"/etc/envoy/tls/client","name":"envoy-sidecar-tls","readOnly":true},{"mountPath":"/etc/envoy/bootstrap","name":"envoy-sidecar-bootstrap","readOnly":true}]}},{"op":"add","path":"/spec/volumes","value":[{"name":"envoy-sidecar-tls","secret":{"secretName":"envoy-sidecar-client-cert"}},{"configMap":{"name":"envoy-sidecar-bootstrap"},"name":"envoy-sidecar-bootstrap"}]}]`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &PodMutator{
				Client:  tt.fields.Client,
				decoder: tt.fields.decoder,
			}
			got := a.Handle(tt.args.ctx, tt.args.req)
			sort.SliceStable(got.Patches, func(i, j int) bool {
				return got.Patches[i].Path < got.Patches[j].Path
			})

			gotPatches, err := json.Marshal(got.Patches)
			if err != nil {
				t.Errorf("Could not serialize got.Patches")
			}
			if string(gotPatches) != string(tt.want) {
				t.Errorf("PodMutator.Handle() = %v, want %v", string(gotPatches), string(tt.want))
			}
		})
	}
}

func TestPodMutator_InjectDecoder(t *testing.T) {
	type fields struct {
		Client  client.Client
		decoder *admission.Decoder
	}
	type args struct {
		d *admission.Decoder
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Injects decoder",
			fields: fields{
				Client:  fake.NewFakeClient(),
				decoder: nil,
			},
			args: args{
				d: func() *admission.Decoder {
					d, _ := admission.NewDecoder(scheme.Scheme)
					return d
				}(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &PodMutator{
				Client:  tt.fields.Client,
				decoder: tt.fields.decoder,
			}
			if err := a.InjectDecoder(tt.args.d); (err != nil) != tt.wantErr {
				t.Errorf("PodMutator.InjectDecoder() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
