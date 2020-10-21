package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/3scale/marin3r/pkg/apis"
	operatorv1alpha1 "github.com/3scale/marin3r/pkg/apis/operator/v1alpha1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	retryInterval           = time.Second * 5
	timeout                 = time.Second * 60
	cleanupRetryInterval    = time.Second * 1
	cleanupTimeout          = time.Second * 5
	deploymentRetryInterval = time.Second * 10
	deploymentTimeout       = time.Minute * 60
)

func TestMarin3rOperator(t *testing.T) {

	t.Run("Creates a new DiscoveryService", func(t *testing.T) {

		err := framework.AddToFrameworkScheme(apis.AddToOperatorScheme, &operatorv1alpha1.DiscoveryServiceList{})
		if err != nil {
			t.Fatalf("failed to add custom resource scheme to framework: %v", err)
		}

		ctx := framework.NewTestCtx(t)
		defer ctx.Cleanup()

		err = ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
		if err != nil {
			t.Fatalf("failed to initialize cluster resources: %v", err)
		}

		// Create a DiscoveryService resource
		ds := &operatorv1alpha1.DiscoveryService{
			ObjectMeta: metav1.ObjectMeta{Name: "instance"},
			Spec: operatorv1alpha1.DiscoveryServiceSpec{
				Image:                     "localhost:5000/marin3r:test",
				DiscoveryServiceNamespace: "default",
				EnabledNamespaces:         []string{"default"},
			},
		}

		f := framework.Global
		err = f.Client.Create(context.TODO(), ds, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
		if err != nil {
			t.Fatal(err)
		}

		// wait for the discovery service Deployment to reach 1 replica
		err = e2eutil.WaitForDeployment(t, f.KubeClient, "default", "marin3r-instance", 1, deploymentRetryInterval, deploymentTimeout)
		if err != nil {
			t.Fatal(err)
		}

		// TODO: validate certificates
	})

	t.Run("Rollout triggered on server certificate change", func(t *testing.T) {

		err := framework.AddToFrameworkScheme(apis.AddToOperatorScheme, &operatorv1alpha1.DiscoveryServiceList{})
		if err != nil {
			t.Fatalf("failed to add custom resource scheme to framework: %v", err)
		}

		ctx := framework.NewTestCtx(t)
		defer ctx.Cleanup()

		err = ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
		if err != nil {
			t.Fatalf("failed to initialize cluster resources: %v", err)
		}

		f := framework.Global

		// Create a DiscoveryService resource
		ds := &operatorv1alpha1.DiscoveryService{
			ObjectMeta: metav1.ObjectMeta{Name: "rolloutoncertchange"},
			Spec: operatorv1alpha1.DiscoveryServiceSpec{
				Image:                     "localhost:5000/marin3r:test",
				DiscoveryServiceNamespace: "default",
				EnabledNamespaces:         []string{"default"},
			},
		}

		err = f.Client.Create(context.TODO(), ds, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
		if err != nil {
			t.Fatal(err)
		}

		err = e2eutil.WaitForDeployment(t, f.KubeClient, "default", "marin3r-rolloutoncertchange", 1, deploymentRetryInterval, deploymentTimeout)
		if err != nil {
			t.Fatal(err)
		}

		// Get current replicaset name
		selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{"app.kubernetes.io/instance": "rolloutoncertchange"},
		})
		rsList := &appsv1.ReplicaSetList{}
		err = f.Client.List(context.TODO(), rsList, &client.ListOptions{LabelSelector: selector})
		rs := rsList.Items[0].ObjectMeta.Name

		// Delete the secret that holds the certificate so the DiscoveryServiceCertificate controller triggers the creation
		// of a new one
		secret := &corev1.Secret{}
		err = f.Client.Get(context.TODO(), types.NamespacedName{Name: "marin3r-server-cert-rolloutoncertchange", Namespace: "default"}, secret)
		err = f.Client.Delete(context.TODO(), secret, &client.DeleteOptions{})
		if err != nil {
			t.Fatal(err)
		}

		err = waitForSecret(t, f.KubeClient, "default", "marin3r-server-cert-rolloutoncertchange", retryInterval, timeout)
		if err != nil {
			t.Fatal(err)
		}

		// Wait for the original replicaset to have 0 available replicas
		err = waitForReplicaSetDrain(t, f.KubeClient, "default", rs, retryInterval, timeout)
		if err != nil {
			t.Fatal(err)
		}

		// The deployment should still have the proper number of available replicas
		err = e2eutil.WaitForDeployment(t, f.KubeClient, "default", "marin3r-rolloutoncertchange", 1, deploymentRetryInterval, deploymentTimeout)
		if err != nil {
			t.Fatal(err)
			return
		}
	})
}
