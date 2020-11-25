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

package e2e

import (
	"context"
	"testing"
	"time"

	testutil "github.com/3scale/marin3r/test/e2e/util"
	"github.com/go-logr/logr"
	"github.com/goombaio/namegenerator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator.marin3r/v1alpha1"
)

const (
	image             string = "quay.io/3scale/marin3r:test"
	targetNamespace   string = "default"
	envoyListenerPort uint32 = 8080
)

var (
	cfg           *rest.Config
	k8sClient     client.Client
	testEnv       *envtest.Environment
	nameGenerator namegenerator.Generator
	logger        logr.Logger
	testNamespace string
	ds            *operatorv1alpha1.DiscoveryService
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"e2e test suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = SynchronizedBeforeSuite(

	// Only runs in node 1
	func(done Done) []byte {

		logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))
		logger = ctrl.Log.WithName("e2e")

		seed := time.Now().UTC().UnixNano()
		nameGenerator = namegenerator.NewNameGenerator(seed)

		By("bootstrapping test environment")
		testEnv = &envtest.Environment{
			CRDDirectoryPaths:  []string{"../../../config/crd/bases"},
			UseExistingCluster: pointer.BoolPtr(true),
		}

		var err error
		cfg, err = testEnv.Start()
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg).ToNot(BeNil())

		err = marin3rv1alpha1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())
		err = operatorv1alpha1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(err).ToNot(HaveOccurred())
		Expect(k8sClient).ToNot(BeNil())

		// Use the same DiscoveryService instance for the whole suite
		By("creating a DiscoveryService instance")
		ds = &operatorv1alpha1.DiscoveryService{
			ObjectMeta: metav1.ObjectMeta{
				Name: "instance",
			},
			Spec: operatorv1alpha1.DiscoveryServiceSpec{
				Image:                     image,
				DiscoveryServiceNamespace: targetNamespace,
				EnabledNamespaces:         []string{},
				Debug:                     true,
			},
		}
		err = k8sClient.Create(context.Background(), ds)
		Expect(err).ToNot(HaveOccurred())

		close(done)
		return []byte{}
	},

	// Runs in all nodes but 1
	func(data []byte, done Done) {
		logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))
		logger = ctrl.Log.WithName("e2e")

		seed := time.Now().UTC().UnixNano()
		nameGenerator = namegenerator.NewNameGenerator(seed)

		testEnv = &envtest.Environment{
			UseExistingCluster: pointer.BoolPtr(true),
		}

		var err error
		cfg, err = testEnv.Start()

		err = marin3rv1alpha1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())
		err = operatorv1alpha1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(err).ToNot(HaveOccurred())
		Expect(k8sClient).ToNot(BeNil())

		ds = &operatorv1alpha1.DiscoveryService{}
		key := types.NamespacedName{Name: "instance"}
		err = k8sClient.Get(context.Background(), key, ds)
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() int {
			return testutil.ReadyReplicas(
				k8sClient,
				targetNamespace,
				client.MatchingLabels{
					"app.kubernetes.io/name":       "marin3r",
					"app.kubernetes.io/managed-by": "marin3r-operator",
					"app.kubernetes.io/component":  "discovery-service",
					"app.kubernetes.io/instance":   ds.GetName(),
				},
			)
		}, 60*time.Second, 5*time.Second).Should(Equal(1))
		close(done)
	},
	60,
)

var _ = SynchronizedAfterSuite(

	// Runs un all other nodes
	func() {},
	// Runs in node 1
	func() {
		By("tearing down the test environment")

		// Delete DiscoveryService instance, but do not wait
		logger.Info("Cleanup", "DiscoveryService", ds.GetName())
		err := k8sClient.Delete(context.Background(), ds)
		Expect(err).ToNot(HaveOccurred())

		err = testEnv.Stop()
		Expect(err).ToNot(HaveOccurred())
	},
)
