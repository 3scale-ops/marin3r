package podv1mutator

import (
	"context"
	"encoding/json"
	"sort"
	"testing"

	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func init() {
	operatorv1alpha1.AddToScheme(scheme.Scheme)
}

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
				Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
					&operatorv1alpha1.DiscoveryService{ObjectMeta: metav1.ObjectMeta{Name: "instance", Namespace: "default"}},
				).Build(),
				decoder: func() *admission.Decoder {
					d, _ := admission.NewDecoder(scheme.Scheme)
					return d
				}(),
			},
			args: args{
				ctx: context.TODO(),
				req: admission.Request{
					AdmissionRequest: admissionv1.AdmissionRequest{
						UID:       "xxxx",
						Kind:      metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
						Resource:  metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
						Namespace: "default",
						Operation: admissionv1.Create,
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
			want: []byte(`[{"op":"add","path":"/spec/containers/1","value":{"args":["-c","/etc/envoy/bootstrap/config.json","--service-node","test","--service-cluster","test"],"command":["envoy"],"image":"envoyproxy/envoy:v1.20.1","imagePullPolicy":"IfNotPresent","livenessProbe":{"failureThreshold":10,"httpGet":{"path":"/ready","port":9901,"scheme":"HTTP"},"initialDelaySeconds":30,"periodSeconds":10,"successThreshold":1,"timeoutSeconds":1},"name":"envoy-sidecar","ports":[{"containerPort":9901,"name":"admin","protocol":"TCP"}],"readinessProbe":{"failureThreshold":1,"httpGet":{"path":"/ready","port":9901,"scheme":"HTTP"},"initialDelaySeconds":15,"periodSeconds":5,"successThreshold":1,"timeoutSeconds":1},"resources":{},"terminationMessagePath":"/dev/termination-log","terminationMessagePolicy":"File","volumeMounts":[{"mountPath":"/etc/envoy/tls/client","name":"envoy-sidecar-tls","readOnly":true},{"mountPath":"/etc/envoy/bootstrap","name":"envoy-sidecar-bootstrap","readOnly":true}]}},{"op":"add","path":"/spec/initContainers","value":[{"args":["init-manager","--admin-access-log-path","/dev/null","--admin-bind-address","0.0.0.0:9901","--api-version","v3","--client-certificate-path","/etc/envoy/tls/client","--config-file","/etc/envoy/bootstrap/config.json","--resources-path","/etc/envoy/bootstrap","--rtds-resource-name","runtime","--xdss-host","marin3r-instance.default.svc","--xdss-port","18000","--envoy-image","envoyproxy/envoy:v1.20.1"],"env":[{"name":"POD_NAME","valueFrom":{"fieldRef":{"apiVersion":"v1","fieldPath":"metadata.name"}}},{"name":"POD_NAMESPACE","valueFrom":{"fieldRef":{"apiVersion":"v1","fieldPath":"metadata.namespace"}}},{"name":"HOST_NAME","valueFrom":{"fieldRef":{"apiVersion":"v1","fieldPath":"spec.nodeName"}}}],"image":"quay.io/3scale/marin3r:v0.9.2-alpha.1","imagePullPolicy":"IfNotPresent","name":"envoy-init-mgr","resources":{},"terminationMessagePath":"/dev/termination-log","terminationMessagePolicy":"File","volumeMounts":[{"mountPath":"/etc/envoy/bootstrap","name":"envoy-sidecar-bootstrap"}]}]},{"op":"add","path":"/spec/volumes","value":[{"name":"envoy-sidecar-tls","secret":{"defaultMode":420,"secretName":"envoy-sidecar-client-cert"}},{"emptyDir":{},"name":"envoy-sidecar-bootstrap"}]}]`),
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
