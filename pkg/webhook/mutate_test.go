// Copyright 2020 rvazquez@redhat.com
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package webhook

import (
	"reflect"
	"testing"

	"go.uber.org/zap"
	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestMutatePod(t *testing.T) {
	type args struct {
		req    *admissionv1.AdmissionRequest
		logger *zap.SugaredLogger
	}
	tests := []struct {
		name    string
		args    args
		want    []PatchOperation
		wantErr bool
	}{
		{
			name: "Mutates the pod given the admission request",
			args: args{
				req: &admissionv1.AdmissionRequest{
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
								  "annotations": {
									"marin3r.3scale.net/node-id": "test"
								  }
								},
								"spec": {
								  "containers": [
									{
									  "name": "myapp",
									  "image": "myapp"
									}
								  ]
								}
							}
							`,
						),
						Object: nil,
					},
				},
				logger: func() *zap.SugaredLogger { lg, _ := zap.NewDevelopment(); return lg.Sugar() }(),
			},
			want: []PatchOperation{
				{
					Op:   "add",
					Path: "/spec/containers/-",
					Value: corev1.Container{
						Name:    "envoy-sidecar",
						Image:   "envoyproxy/envoy:v1.14.1",
						Command: []string{"envoy"},
						Args:    []string{"-c", "/etc/envoy/bootstrap/config.yaml", "--service-node", "test", "--service-cluster", "test"},
						Ports:   []corev1.ContainerPort{},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "envoy-sidecar-tls",
								ReadOnly:  true,
								MountPath: "/etc/envoy/tls/client",
							},
							{
								Name:      "envoy-sidecar-bootstrap",
								ReadOnly:  true,
								MountPath: "/etc/envoy/bootstrap",
							},
						}},
				},
				{
					Op:   "add",
					Path: "/spec/volumes/-",
					Value: corev1.Volume{
						Name: "envoy-sidecar-tls",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "envoy-sidecar-client-cert",
							}}},
				},
				{
					Op:   "add",
					Path: "/spec/volumes/-",
					Value: corev1.Volume{
						Name: "envoy-sidecar-bootstrap",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "envoy-sidecar-bootstrap",
								}}}},
				},
			},
			wantErr: false,
		},
		{
			name: "Does not mutate pod if marin3r annotation not present",
			args: args{
				req: &admissionv1.AdmissionRequest{
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
								  "namespace": "default"
								},
								"spec": {
								  "containers": [
									{
									  "name": "myapp",
									  "image": "myapp"
									}
								  ]
								}
							}
							`,
						),
						Object: nil,
					},
				},
				logger: func() *zap.SugaredLogger { lg, _ := zap.NewDevelopment(); return lg.Sugar() }(),
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Empty patch for non pod resource",
			args: args{
				req: &admissionv1.AdmissionRequest{
					UID:       "xxxx",
					Kind:      metav1.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
					Resource:  metav1.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"},
					Namespace: "default",
					Operation: admissionv1.Create,
					UserInfo:  authenticationv1.UserInfo{Username: "xxxx", UID: "xxxx"},
					Object:    runtime.RawExtension{},
				},
				logger: func() *zap.SugaredLogger { lg, _ := zap.NewDevelopment(); return lg.Sugar() }(),
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Resource unmarshall error",
			args: args{
				req: &admissionv1.AdmissionRequest{
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
								"wrong": "wrong",
							}
							`,
						),
						Object: nil,
					},
				},
				logger: func() *zap.SugaredLogger { lg, _ := zap.NewDevelopment(); return lg.Sugar() }(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Bad annotation syntax",
			args: args{
				req: &admissionv1.AdmissionRequest{
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
							  "annotations": {
								"marin3r.3scale.net/node-id": "test",
								"marin3r.3scale.net/ports": "wrong-syntax"
							  }
							},
							"spec": {
							  "containers": [
								{
								  "name": "myapp",
								  "image": "myapp"
								}
							  ]
							}
						}
						`,
						),
						Object: nil,
					},
				},
				logger: func() *zap.SugaredLogger { lg, _ := zap.NewDevelopment(); return lg.Sugar() }(),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MutatePod(tt.args.req, tt.args.logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("MutatePod() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MutatePod() = %v, want %v", got, tt.want)
			}
		})
	}
}
