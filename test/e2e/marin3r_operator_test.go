package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/3scale/marin3r/pkg/apis"
	operatorv1alpha1 "github.com/3scale/marin3r/pkg/apis/operator/v1alpha1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	retryInterval           = time.Second * 5
	timeout                 = time.Second * 75
	cleanupRetryInterval    = time.Second * 1
	cleanupTimeout          = time.Second * 5
	deploymentRetryInterval = time.Second * 30
	deploymentTimeout       = time.Minute * 25
)

func TestMarin3rOperator(t *testing.T) {
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
	err = e2eutil.WaitForDeployment(t, f.KubeClient, "default", "marin3r-instance", 1, time.Second*5, time.Second*30)
	if err != nil {
		t.Fatal(err)
	}
}
