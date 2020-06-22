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
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestAdmitFuncHandler(t *testing.T) {

	tests := []struct {
		name    string
		request *http.Request
		want    admissionv1.AdmissionReview
		wantErr bool
	}{
		{
			name: "Returns a wellformed AdmissionReview",
			request: func() *http.Request {
				admissionReview := admissionv1.AdmissionReview{
					TypeMeta: metav1.TypeMeta{Kind: "admission", APIVersion: "v1"},
					Request: &admissionv1.AdmissionRequest{
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
				}

				body, err := json.Marshal(admissionReview)
				req, err := http.NewRequest("POST", "/mutate", bytes.NewReader(body))
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("Content-Type", jsonContentType)
				return req
			}(),
			want: admissionv1.AdmissionReview{
				TypeMeta: metav1.TypeMeta{
					Kind:       "AdmissionReview",
					APIVersion: "admission.k8s.io/v1",
				},
				Response: &admissionv1.AdmissionResponse{
					UID:       "xxxx",
					Allowed:   true,
					PatchType: func() *admissionv1.PatchType { s := admissionv1.PatchTypeJSONPatch; return &s }(),
					Patch: func() []byte {
						patches := []PatchOperation{
							{
								Op:   "add",
								Path: "/spec/containers/-",
								Value: corev1.Container{
									Name:    "envoy-sidecar",
									Image:   "envoyproxy/envoy:v1.14.1",
									Command: []string{"envoy"},
									Args:    []string{"-c", "/etc/envoy/bootstrap/config.json", "--service-node", "test", "--service-cluster", "test"},
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
						}
						patchBytes, _ := json.Marshal(patches)
						return patchBytes

					}(),
				},
			},
			wantErr: false,
		},
		{
			name: "Error, only POST allowed",
			request: func() *http.Request {
				req, err := http.NewRequest("GET", "/mutate", nil)
				if err != nil {
					t.Fatal(err)
				}
				return req
			}(),
			want:    admissionv1.AdmissionReview{},
			wantErr: true,
		},
		{
			name: "Error, only wrong content type",
			request: func() *http.Request {
				req, err := http.NewRequest("POST", "/mutate", nil)
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("Content-Type", "text")
				return req
			}(),
			want:    admissionv1.AdmissionReview{},
			wantErr: true,
		},
		{
			name: "Error, malformed admission review",
			request: func() *http.Request {
				req, err := http.NewRequest("POST", "/mutate", bytes.NewReader([]byte(`{"wrong": "wrong"}`)))
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("Content-Type", jsonContentType)
				return req
			}(),

			want:    admissionv1.AdmissionReview{},
			wantErr: true,
		},
		{
			name: "MutatePod returns an error and the admission request is denied",
			request: func() *http.Request {
				admissionReview := admissionv1.AdmissionReview{
					TypeMeta: metav1.TypeMeta{Kind: "admission", APIVersion: "v1"},
					Request: &admissionv1.AdmissionRequest{
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
										"marin3r.3scale.net/ports": "wrong-port-syntax"
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
				}

				body, err := json.Marshal(admissionReview)
				req, err := http.NewRequest("POST", "/mutate", bytes.NewReader(body))
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("Content-Type", jsonContentType)
				return req
			}(),
			want: admissionv1.AdmissionReview{
				TypeMeta: metav1.TypeMeta{
					Kind:       "AdmissionReview",
					APIVersion: "admission.k8s.io/v1",
				},
				Response: &admissionv1.AdmissionResponse{
					UID:     "xxxx",
					Allowed: false,
					Result: &metav1.Status{
						Message: "Incorrect format, the por specification format for the envoy sidecar container is 'name:port[:protocol]'",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handler := AdmitFuncHandler(MutatePod)
			handler.ServeHTTP(rr, tt.request)

			if (rr.Result().StatusCode != 200) != tt.wantErr {
				t.Errorf("AdmitFuncHandler() got HTTP status code %v", rr.Result().StatusCode)
				return
			}

			body, _ := ioutil.ReadAll(rr.Body)
			got := admissionv1.AdmissionReview{}
			_, _, _ = universalDeserializer.Decode(body, nil, &got)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MutatePod() = %v, want %v", got, tt.want)
			}
		})
	}
}
