package common

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"time"

	certmanagerv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const pollingPeriod time.Duration = 10

// Background represents a procedure that runs in the background, periodically auto-detecting features
type Background struct {
	dc                  discovery.DiscoveryInterface
	ticker              *time.Ticker
	SubscriptionChannel chan schema.GroupVersionKind
}

// NewAutoDetect creates a new auto-detect runner
func NewAutoDetect(mgr manager.Manager) (*Background, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		return nil, err
	}

	// Create a new channel that GVK type will be sent down
	subChan := make(chan schema.GroupVersionKind, 1)

	return &Background{dc: dc, SubscriptionChannel: subChan}, nil
}

// Start initializes the auto-detection process that runs in the background
func (b *Background) Start() {
	// periodically attempts to auto detect all the capabilities for this operator
	b.ticker = time.NewTicker(pollingPeriod * time.Second)

	go func() {
		b.autoDetectCapabilities()

		for range b.ticker.C {
			b.autoDetectCapabilities()
		}
	}()
}

// Stop causes the background process to stop auto detecting capabilities
func (b *Background) Stop() {
	b.ticker.Stop()
	close(b.SubscriptionChannel)
}

func (b *Background) autoDetectCapabilities() {
	b.detectCertManagerResources()
}

func (b *Background) detectCertManagerResources() {
	// detect the Certificate resource type exist on the cluster
	resourceExists, _ := k8sutil.ResourceExists(b.dc, certmanagerv1alpha2.SchemeGroupVersion.String(), certmanagerv1alpha2.CertificateKind)
	if resourceExists {
		b.SubscriptionChannel <- certmanagerv1alpha2.SchemeGroupVersion.WithKind(certmanagerv1alpha2.CertificateKind)
	}
}
