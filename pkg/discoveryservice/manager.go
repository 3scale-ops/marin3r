package discoveryservice

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	marin3rcontroller "github.com/3scale/marin3r/controllers/marin3r"
	envoy "github.com/3scale/marin3r/pkg/envoy"
	"github.com/3scale/marin3r/pkg/webhooks/podv1mutator"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	util_runtime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	tlsCertificateFile    = "tls.crt"
	tlsCertificateKeyFile = "tls.key"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("discoveryservice")
)

func init() {
	util_runtime.Must(clientgoscheme.AddToScheme(scheme))
	util_runtime.Must(marin3rv1alpha1.AddToScheme(scheme))
}

// Manager holds configuration to
// run a marin3r discovery service
type Manager struct {
	// The xDS server port
	XdsServerPort int
	// The mutating webhook server port
	WebhookPort int
	// Bind address for the metrics server
	MetricsAddr string
	// The directory where server certificate and key are located
	ServerCertificatePath string
	// The directory where the CA used to authenticate clients with the xDS server is
	CACertificatePath string
	// Cfg is the config to connect to the k8s API server
	Cfg *rest.Config
}

// Start runs the DiscoveryServiceManager, which runs the EnvoyConfig and
// EnvoyConfigRevision controlles, the xDS server and the mutating webhook server
func (dsm *Manager) Start(ctx context.Context) {

	mgr, err := ctrl.NewManager(dsm.Cfg, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: dsm.MetricsAddr,
		Port:               dsm.WebhookPort,
		LeaderElection:     false,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// watch for syscalls
	stopCh := signals.SetupSignalHandler()

	var wait sync.WaitGroup

	// Start envoy's aggregated discovery service
	xdss := NewDualXdsServer(
		ctx,
		uint(dsm.XdsServerPort),
		&tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				// Sadly, these 2 non 256 are required to use http2 in go
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			},
			Certificates: []tls.Certificate{loadCertificate(dsm.ServerCertificatePath, setupLog)},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    loadCA(dsm.CACertificatePath, setupLog),
		},
		marin3rcontroller.OnError(dsm.Cfg),
		setupLog,
	)

	wait.Add(1)
	go func() {
		defer wait.Done()
		if err := xdss.Start(stopCh); err != nil {
			setupLog.Error(err, "xDS server returned an unrecoverable error, shutting down")
			os.Exit(1)
		}
	}()

	// Start controllers
	if err := (&marin3rcontroller.EnvoyConfigReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("envoyconfig"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "envoyconfig")
		os.Exit(1)
	}

	if err := (&marin3rcontroller.EnvoyConfigRevisionReconciler{
		Client:     mgr.GetClient(),
		Log:        ctrl.Log.WithName("controllers").WithName(fmt.Sprintf("envoyconfigrevision_%s", string(envoy.APIv2))),
		Scheme:     mgr.GetScheme(),
		XdsCache:   xdss.GetCache(envoy.APIv2),
		APIVersion: envoy.APIv2,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", fmt.Sprintf("envoyconfigrevision_%s", string(envoy.APIv2)))
		os.Exit(1)
	}

	if err := (&marin3rcontroller.EnvoyConfigRevisionReconciler{
		Client:     mgr.GetClient(),
		Log:        ctrl.Log.WithName("controllers").WithName(fmt.Sprintf("envoyconfigrevision_%s", string(envoy.APIv3))),
		Scheme:     mgr.GetScheme(),
		XdsCache:   xdss.GetCache(envoy.APIv3),
		APIVersion: envoy.APIv3,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", fmt.Sprintf("envoyconfigrevision_%s", string(envoy.APIv3)))
		os.Exit(1)
	}

	if err := (&marin3rcontroller.SecretReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("secret"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "secret")
		os.Exit(1)
	}

	// Setup webhooks
	hookServer := mgr.GetWebhookServer()
	hookServer.CertDir = dsm.ServerCertificatePath
	ctrl.Log.Info("registering webhooks to the webhook server")
	hookServer.Register(podv1mutator.MutatePath, &webhook.Admission{Handler: &podv1mutator.PodMutator{Client: mgr.GetClient()}})

	// Start the controllers
	wait.Add(1)
	go func() {
		defer wait.Done()
		if err := mgr.Start(stopCh); err != nil {
			setupLog.Error(err, "Controller manager exited non-zero")
			os.Exit(1)
		}
	}()

	// Wait for shutdown
	wait.Wait()
	setupLog.Info("Controller has shut down")
}

func loadCertificate(directory string, logger logr.Logger) tls.Certificate {
	certificate, err := tls.LoadX509KeyPair(
		fmt.Sprintf("%s/%s", directory, tlsCertificateFile),
		fmt.Sprintf("%s/%s", directory, tlsCertificateKeyFile),
	)
	if err != nil {
		logger.Error(err, "Could not load server certificate")
		os.Exit(1)
	}
	return certificate
}

func loadCA(directory string, logger logr.Logger) *x509.CertPool {
	certPool := x509.NewCertPool()
	if bs, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", directory, tlsCertificateFile)); err != nil {
		logger.Error(err, "Failed to read client ca cert")
		os.Exit(1)
	} else {
		ok := certPool.AppendCertsFromPEM(bs)
		if !ok {
			logger.Error(err, "Failed to append client certs")
			os.Exit(1)
		}
	}
	return certPool
}
