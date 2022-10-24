/*


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

package v1alpha1

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("EnvoyConfig webhook", func() {
	var namespace string

	BeforeEach(func() {
		// Create a namespace for each block
		namespace = "test-ns-" + nameGenerator.Generate()
		// Add any setup steps that needs to be executed before each test
		testNamespace := &v1.Namespace{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
			ObjectMeta: metav1.ObjectMeta{Name: namespace},
		}

		err := k8sClient.Create(context.Background(), testNamespace)
		Expect(err).ToNot(HaveOccurred())

		n := &v1.Namespace{}
		Eventually(func() bool {
			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: namespace}, n)
			return err == nil
		}, 60*time.Second, 5*time.Second).Should(BeTrue())

	})

	AfterEach(func() {

		// Delete the namespace
		testNamespace := &v1.Namespace{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
			ObjectMeta: metav1.ObjectMeta{Name: namespace},
		}
		// Add any teardown steps that needs to be executed after each test
		err := k8sClient.Delete(ctx, testNamespace, client.PropagationPolicy(metav1.DeletePropagationForeground))
		Expect(err).ToNot(HaveOccurred())

		n := &v1.Namespace{}
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: namespace}, n)
			if err != nil && errors.IsNotFound(err) {
				return false
			}
			return true
		}, 60*time.Second, 5*time.Second).Should(BeTrue())
	})

	Context("envoy resource validation", func() {
		It("fails for an EnvoyConfig with a syntax error in one of the envoy resources (from json)", func() {
			ec := &EnvoyConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "test-envoyconfig", Namespace: namespace},
				Spec: EnvoyConfigSpec{
					NodeID:        "test",
					Serialization: pointer.StringPtr("json"),
					EnvoyAPI:      pointer.StringPtr("v3"),
					EnvoyResources: &EnvoyResources{
						Clusters: []EnvoyResource{{
							Name: pointer.String("cluster"),
							// the connect_timeout value unit is wrong
							Value: `{"name":"cluster1","type":"STRICT_DNS","connect_timeout":"2xs","load_assignment":{"cluster_name":"cluster1"}}`,
						}},
					},
				},
			}
			err := k8sClient.Create(ctx, ec)
			Expect(err).To(HaveOccurred())
		})
		It("fails for an EnvoyConfig with a syntax error in one of the envoy resources (from yaml)", func() {
			ec := &EnvoyConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "test-envoyconfig", Namespace: namespace},
				Spec: EnvoyConfigSpec{
					NodeID:        "test",
					Serialization: pointer.StringPtr("yaml"),
					EnvoyAPI:      pointer.StringPtr("v3"),
					EnvoyResources: &EnvoyResources{
						Listeners: []EnvoyResource{{
							Name: pointer.String("test"),
							// the "port" property should be "port_value"
							Value: `
                              name: listener1
                              address:
                                socket_address:
                                  address: 0.0.0.0
                                  port: 8443
                            `,
						}},
					},
				},
			}
			err := k8sClient.Create(ctx, ec)
			Expect(err).To(HaveOccurred())
		})

		It("fails for an EnvoyConfig that points to a Secret in a different namespace", func() {
			ec := &EnvoyConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "test-envoyconfig", Namespace: namespace},
				Spec: EnvoyConfigSpec{
					NodeID:        "test",
					Serialization: pointer.StringPtr("yaml"),
					EnvoyAPI:      pointer.StringPtr("v3"),
					EnvoyResources: &EnvoyResources{
						Secrets: []EnvoySecretResource{{
							Name: "secret",
							Ref: &v1.SecretReference{
								Name:      "secret",
								Namespace: "other-ns",
							},
						}},
					},
				},
			}
			err := k8sClient.Create(ctx, ec)
			Expect(err).To(HaveOccurred())
		})
	})
})
