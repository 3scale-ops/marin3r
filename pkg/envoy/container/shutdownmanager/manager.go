package shutdownmanager

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/common/expfmt"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	ShutdownEndpoint string = "/shutdown"
	DrainEndpoint    string = "/drain"
	HealthEndpoint   string = "/healthz"
)

const (
	envoyPrometheusEndpoint string = "/stats/prometheus"
	envoyShutdownEndpoint   string = "/healthcheck/fail"
	prometheusStat          string = "envoy_http_downstream_cx_active"
	adminListenerPrefix     string = "admin"
)

var (
	logger = ctrl.Log.WithName("shutdownmanager")
)

// Manager holds configuration to run the shutdown manager
type Manager struct {
	// HTTPServePort defines what port the shutdown-manager listens on
	HTTPServePort int
	// ShutdownReadyFile is the default file path used in the /shutdown endpoint
	ShutdownReadyFile string
	// ShutdownReadyCheckInterval is the polling interval for the file used in the /shutdown endpoint
	ShutdownReadyCheckInterval time.Duration
	// CheckDrainInterval defines time delay between polling Envoy for open connections
	CheckDrainInterval time.Duration
	// CheckDrainDelay defines time to wait before polling Envoy for open connections
	CheckDrainDelay time.Duration
	// StartDrainDelay defines time to wait before draining Envoy connections
	StartDrainDelay time.Duration
	// EnvoyAdminAddress specifies envoy's admin url
	EnvoyAdminAddress string
	// MinOpenConnections is the number of open connections allowed before performing envoy shutdown
	MinOpenConnections int
}

func (mgr *Manager) envoyPrometheusURL() string {
	return strings.Join([]string{mgr.EnvoyAdminAddress, envoyPrometheusEndpoint}, "")
}

func (mgr *Manager) envoyShutdownURL() string {
	return strings.Join([]string{mgr.EnvoyAdminAddress, envoyShutdownEndpoint}, "")
}

// Starts the shutdown manager
func (mgr *Manager) Start(ctx context.Context) error {

	logger.Info("started envoy shutdown manager")
	defer logger.Info("stopped")

	mux := http.NewServeMux()
	srv := http.Server{Addr: fmt.Sprintf(":%d", mgr.HTTPServePort), Handler: mux}
	errCh := make(chan error)

	mux.HandleFunc(HealthEndpoint, mgr.healthzHandler)
	mux.HandleFunc(ShutdownEndpoint, mgr.shutdownHandler)
	mux.HandleFunc(DrainEndpoint, mgr.drainHandler)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	<-ctx.Done()
	srv.Shutdown(ctx)

	select {
	case <-ctx.Done():
		// Shutdown the server when the context is canceled
		srv.Shutdown(ctx)
	case err := <-errCh:
		return err
	}

	return nil
}

// healthzHandler handles the /healthz endpoint which is used for the shutdown-manager's liveness probe.
func (s *Manager) healthzHandler(w http.ResponseWriter, r *http.Request) {
	l := logger.WithValues("context", "healthzHandler")
	if _, err := w.Write([]byte("OK")); err != nil {
		l.Error(err, "healthcheck failed")
	}
}

// shutdownHandler handles the /shutdown endpoint which is used by Envoy to determine if it can
// terminate safely. The endpoint blocks until the expected file exists in the filesystem.
// This endpoint is called from Envoy's container preStop hook in order to delay shutdown of the server
// until it is safe to do so or the timeout is reached.
func (mgr *Manager) shutdownHandler(w http.ResponseWriter, r *http.Request) {
	l := logger.WithValues("context", "waitForDrainHandler")
	ctx := r.Context()
	for {
		_, err := os.Stat(mgr.ShutdownReadyFile)

		if err == nil {
			l.Info(fmt.Sprintf("file %s exists, sending HTTP response", mgr.ShutdownReadyFile))
			if _, err := w.Write([]byte("OK")); err != nil {
				l.Error(err, "error sending HTTP response")
			}
			return

		} else if os.IsNotExist(err) {
			l.Info(fmt.Sprintf("file %s does not exist, recheck in %v", mgr.ShutdownReadyFile, mgr.ShutdownReadyCheckInterval))
		} else {
			l.Error(err, "error checking for file")
		}

		select {
		case <-time.After(mgr.ShutdownReadyCheckInterval):
		case <-ctx.Done():
			l.Info("client request cancelled")
			return
		}
	}
}

// drainHandler is called from the shutdown-manager container preStop hook and will:
// * Call Envoy container admin api to start graceful connection draining of the listeners
// * Block until the prometheus metrics returned by Envoy container admin api return the
//   desired min number of connection (usually 0). The admin listener connections are not
//   computed towards this value.
func (mgr *Manager) drainHandler(w http.ResponseWriter, r *http.Request) {
	l := logger.WithValues("context", "DrainListeners")
	ctx := r.Context()

	l.Info(fmt.Sprintf("waiting %s before draining connections", mgr.StartDrainDelay))
	time.Sleep(mgr.StartDrainDelay)

	// Send shutdown signal to Envoy to start draining connections
	l.Info("start draining envoy listeners")

	// Retry any failures to shutdownEnvoy() in a Backoff time window
	err := retry.OnError(
		wait.Backoff{Steps: 4, Duration: 200 * time.Millisecond, Factor: 5.0, Jitter: 0.1},
		func(err error) bool { return true },
		func() error { l.Info("signaling start drain"); return mgr.shutdownEnvoy() },
	)

	if err != nil {
		l.Error(err, "error signaling envoy to start draining listeners after 4 attempts")
	}

	l.Info(fmt.Sprintf("waiting %s before polling for draining connections", mgr.CheckDrainDelay))
	time.Sleep(mgr.CheckDrainDelay)

	for {
		openConnections, err := mgr.getOpenConnections()
		if err == nil {
			if openConnections <= mgr.MinOpenConnections {
				l.Info("min number of open connections found, shutting down", "open_connections", openConnections, "min_connections", mgr.MinOpenConnections)
				file, err := os.Create(mgr.ShutdownReadyFile)
				if err != nil {
					l.Error(err, "")
				}
				defer file.Close()
				if _, err := w.Write([]byte("OK")); err != nil {
					l.Error(err, "error sending HTTP response")
				}
				return
			}
			l.Info("polled open connections", "open_connections", openConnections, "min_connections", mgr.MinOpenConnections)

		} else {
			l.Error(err, "")
		}

		select {
		case <-time.After(mgr.CheckDrainInterval):
		case <-ctx.Done():
			l.Info("request cancelled")
			return
		}
	}
}

// shutdownEnvoy sends a POST request to /healthcheck/fail to start draining listeners
func (mgr *Manager) shutdownEnvoy() error {
	resp, err := http.Post(mgr.envoyShutdownURL(), "", nil)
	if err != nil {
		return fmt.Errorf("creating POST request to %s failed: %s", mgr.envoyShutdownURL(), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("POST for %q returned HTTP status %s", mgr.envoyShutdownURL(), resp.Status)
	}
	return nil
}

// getOpenConnections parses a http request to a prometheus endpoint returning the sum of values found
func (mgr *Manager) getOpenConnections() (int, error) {
	// Make request to Envoy Prometheus endpoint
	resp, err := http.Get(mgr.envoyPrometheusURL())
	if err != nil {
		return -1, fmt.Errorf("creating GET request to %s failed: %s", mgr.envoyPrometheusURL(), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return -1, fmt.Errorf("GET for %q returned HTTP status %s", mgr.envoyPrometheusURL(), resp.Status)
	}

	// Parse Prometheus listener stats for open connections
	return parseOpenConnections(resp.Body)
}

// parseOpenConnections returns the sum of open connections from a Prometheus HTTP request
func parseOpenConnections(stats io.Reader) (int, error) {
	var parser expfmt.TextParser
	openConnections := 0

	if stats == nil {
		return -1, fmt.Errorf("stats input was nil")
	}

	// Parse Prometheus http response
	metricFamilies, err := parser.TextToMetricFamilies(stats)
	if err != nil {
		return -1, fmt.Errorf("parsing Prometheus text format failed: %v", err)
	}

	// Validate stat exists in output
	if _, ok := metricFamilies[prometheusStat]; !ok {
		return -1, fmt.Errorf("error finding Prometheus stat %q in the request result", prometheusStat)
	}

	// Look up open connections value
	for _, metrics := range metricFamilies[prometheusStat].Metric {
		for _, lp := range metrics.Label {
			if lp.GetValue() != adminListenerPrefix {
				openConnections += int(metrics.Gauge.GetValue())
			}
		}
	}
	return openConnections, nil
}
