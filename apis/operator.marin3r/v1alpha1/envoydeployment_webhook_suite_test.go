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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("EnvoyDeployment webhook", func() {
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

	Context("resource validation", func() {
		It("fails for an EnvoyDeployment with both static and dynamic replicas configuration", func() {
			ec := &EnvoyDeployment{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: namespace},
				Spec: EnvoyDeploymentSpec{
					EnvoyConfigRef:      "test",
					DiscoveryServiceRef: "test",
					Replicas: &ReplicasSpec{
						Static: pointer.Int32Ptr(5),
						Dynamic: &DynamicReplicasSpec{
							MinReplicas: pointer.Int32Ptr(2),
							MaxReplicas: 10,
						},
					},
				},
			}
			err := k8sClient.Create(ctx, ec)
			Expect(err).To(HaveOccurred())
		})
		It("fails for an EnvoyDeployment with both minAvailable and maxUnavailable fields configured for the PDB", func() {
			ec := &EnvoyDeployment{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: namespace},
				Spec: EnvoyDeploymentSpec{
					EnvoyConfigRef:      "test",
					DiscoveryServiceRef: "test",
					PodDisruptionBudget: &PodDisruptionBudgetSpec{
						MinAvailable:   &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
						MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
					},
				},
			}
			err := k8sClient.Create(ctx, ec)
			Expect(err).To(HaveOccurred())
		})
	})
})
