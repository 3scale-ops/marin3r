package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestPortNumber(t *testing.T) {

	cases := map[string]int32{
		"8080": 8080,
		"3000": 3000,
	}

	for input, expected := range cases {
		port, _ := portNumber(input)
		assert.Equal(t, expected, port, "Expected port received")
	}
}

func TestHostPortMapping(t *testing.T) {
	cases := []struct {
		annotations map[string]string
		pname       string
		expected    int32
	}{
		{
			annotations: map[string]string{
				"marin3r.3scale.net/host-port-mappings": "http:3000",
			},
			pname:    "http",
			expected: 3000,
		},
		{
			annotations: map[string]string{
				"marin3r.3scale.net/host-port-mappings": "https:8443",
			},
			pname:    "https",
			expected: 8443,
		},
	}

	for _, c := range cases {
		iport, err := hostPortMapping(c.pname, c.annotations)
		if err != nil {
			panic(err)
		}
		assert.Equal(t, c.expected, iport, "Expected port received")
	}
}

func TestParsePortSpec(t *testing.T) {

	specs := map[string]*corev1.ContainerPort{
		"http:3000": &corev1.ContainerPort{
			Name:          "http",
			ContainerPort: 3000,
		},
		"http:8080:TCP": &corev1.ContainerPort{
			Name:          "http",
			ContainerPort: 8080,
			Protocol:      corev1.Protocol("TCP"),
		},
		"http:22000:SCTP": &corev1.ContainerPort{
			Name:          "http",
			ContainerPort: 22000,
			Protocol:      corev1.Protocol("SCTP"),
		},
		"udp:5555:UDP": &corev1.ContainerPort{
			Name:          "udp",
			ContainerPort: 5555,
			Protocol:      corev1.Protocol("UDP"),
		},
	}

	for k, v := range specs {
		r, err := parsePortSpec(k)
		if err != nil {
			panic(err)
		}
		assert.Equal(t, v, r, "ContainerPort objects are equal")
	}
}

func TestGetContainerPorts(t *testing.T) {

	specs := []struct {
		annotations map[string]string
		expected    []corev1.ContainerPort
	}{
		{
			annotations: map[string]string{
				"marin3r.3scale.net/ports": "http:3000",
			},
			expected: []corev1.ContainerPort{
				corev1.ContainerPort{
					Name:          "http",
					ContainerPort: 3000,
				},
			},
		}, {
			annotations: map[string]string{
				"marin3r.3scale.net/ports": "http:8080,https:8443",
			},
			expected: []corev1.ContainerPort{
				corev1.ContainerPort{
					Name:          "http",
					ContainerPort: 8080,
				},
				corev1.ContainerPort{
					Name:          "https",
					ContainerPort: 8443,
				},
			},
		}, {
			annotations: map[string]string{
				"marin3r.3scale.net/ports": "udp:6000:UDP,https:8443",
			},
			expected: []corev1.ContainerPort{
				corev1.ContainerPort{
					Name:          "udp",
					ContainerPort: 6000,
					Protocol:      corev1.Protocol("UDP"),
				},
				corev1.ContainerPort{
					Name:          "https",
					ContainerPort: 8443,
				},
			},
		}, {
			annotations: map[string]string{
				"marin3r.3scale.net/ports":              "http:3000:UDP,https:4000",
				"marin3r.3scale.net/host-port-mappings": "http:8080,https:8443",
			},
			expected: []corev1.ContainerPort{
				corev1.ContainerPort{
					Name:          "http",
					ContainerPort: 3000,
					HostPort:      8080,
					Protocol:      corev1.Protocol("UDP"),
				},
				corev1.ContainerPort{
					Name:          "https",
					ContainerPort: 4000,
					HostPort:      8443,
				},
			},
		},
	}

	for _, v := range specs {
		r, err := getContainerPorts(v.annotations)
		if err != nil {
			panic(err)
		}
		assert.Equal(t, v.expected, r, "ContainerPort slices are equal")
	}
}

func TestContainer(t *testing.T) {

	cases := []struct {
		config   envoySidecarConfig
		expected corev1.Container
	}{
		{
			config: envoySidecarConfig{
				name:  "sidecar",
				image: "sidecar:latest",
				ports: []corev1.ContainerPort{
					corev1.ContainerPort{
						Name:          "udp",
						ContainerPort: 6000,
						Protocol:      corev1.Protocol("UDP"),
					},
					corev1.ContainerPort{
						Name:          "https",
						ContainerPort: 8443,
					},
				},
				adsConfigMap:     "ads-configmap",
				nodeID:           "test-id",
				clusterID:        "cluster-id",
				tlsVolume:        "tls-volume",
				configVolume:     "config-volume",
				clientCertSecret: "secret",
			},
			expected: corev1.Container{
				Name:    "sidecar",
				Image:   "sidecar:latest",
				Command: []string{"envoy"},
				Args: []string{
					"-c",
					"/etc/envoy/bootstrap/config.yaml",
					"--service-node",
					"test-id",
					"--service-cluster",
					"cluster-id",
				},
				Ports: []corev1.ContainerPort{
					corev1.ContainerPort{
						Name:          "udp",
						ContainerPort: 6000,
						Protocol:      corev1.Protocol("UDP"),
					},
					corev1.ContainerPort{
						Name:          "https",
						ContainerPort: 8443,
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					corev1.VolumeMount{
						Name:      "tls-volume",
						ReadOnly:  true,
						MountPath: "/etc/envoy/tls/client",
					},
					corev1.VolumeMount{
						Name:      "config-volume",
						ReadOnly:  true,
						MountPath: "/etc/envoy/bootstrap",
					},
				},
			},
		},
	}

	for _, c := range cases {
		r := c.config.container()
		assert.Equal(t, c.expected, r, "ContainerPort slices are equal")
	}
}
