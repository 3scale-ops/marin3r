package e2e

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

func NewTestPortForwarder(cfg *rest.Config, pod corev1.Pod, localPort, podPort uint32,
	out io.Writer, stopCh chan struct{}, readyCh chan struct{}) (*portforward.PortForwarder, error) {

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward",
		pod.Namespace, pod.Name)
	hostIP := strings.TrimLeft(cfg.Host, "https:/")

	transport, upgrader, err := spdy.RoundTripperFor(cfg)
	if err != nil {
		return nil, err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, &url.URL{Scheme: "https", Path: path, Host: hostIP})
	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", localPort, podPort)}, stopCh, readyCh, out, out)
	if err != nil {
		return nil, err
	}
	return fw, nil
}
