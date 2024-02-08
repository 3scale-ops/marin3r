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

package controllers

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/3scale-ops/basereconciler/reconciler"
	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss/stats"
	xdss_v3 "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss/v3"
	envoy "github.com/3scale-ops/marin3r/pkg/envoy"
	"github.com/goombaio/namegenerator"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ecrV3Reconciler *EnvoyConfigRevisionReconciler
var nameGenerator namegenerator.Generator
var ctx = context.Background()
var cancel context.CancelFunc

func TestAPIs(t *testing.T) {
	if os.Getenv("RUN_ENVTEST") == "0" {
		t.Skip("Skipping envtest tests")
	}
	RegisterFailHandler(Fail)

	RunSpecs(t, "Marin3r API group Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	seed := time.Now().UTC().UnixNano()
	nameGenerator = namegenerator.NewNameGenerator(seed)

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = marin3rv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = operatorv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		// Disable the metrics port to allow running the
		// test suite in parallel
		MetricsBindAddress: "0",
	})
	Expect(err).ToNot(HaveOccurred())

	// Add the EnvoyConfig controller
	err = (&EnvoyConfigReconciler{
		Reconciler: &reconciler.Reconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("envoyconfig"),
			Scheme: mgr.GetScheme(),
		},
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	// Add the EnvoyConfigRevision v3 controller
	ecrV3Reconciler = &EnvoyConfigRevisionReconciler{
		Reconciler: &reconciler.Reconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("envoyconfigrevision_v3"),
			Scheme: mgr.GetScheme(),
		},
		XdsCache:       xdss_v3.NewCache(),
		APIVersion:     envoy.APIv3,
		DiscoveryStats: stats.New(),
	}
	err = ecrV3Reconciler.SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
	}()

	k8sClient = mgr.GetClient()
	Expect(k8sClient).ToNot(BeNil())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})
