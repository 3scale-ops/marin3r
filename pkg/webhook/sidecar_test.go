package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

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
		r, _ := parsePortSpec(k)
		assert.Equal(t, r, v, "ContainerPort objects are equal")
	}
}

func TestGetContainerPorts(t *testing.T) {

	specs := []struct {
		annotations map[string]string
		result      []corev1.ContainerPort
	}{
		{
			annotations: map[string]string{
				"marin3r.3scale.net/ports": "http:3000",
			},
			result: []corev1.ContainerPort{
				corev1.ContainerPort{
					Name:          "http",
					ContainerPort: 3000,
				},
			},
		}, {
			annotations: map[string]string{
				"marin3r.3scale.net/ports": "http:8080,https:8443",
			},
			result: []corev1.ContainerPort{
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
			result: []corev1.ContainerPort{
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
		},
	}

	for _, v := range specs {
		r, _ := getContainerPorts(v.annotations)
		assert.Equal(t, r, v.result, "ContainerPort slices are equal")
	}
}

func TestContainer(t *testing.T) {

	cases := []struct {
		config envoySidecarConfig
		result corev1.Container
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
			result: corev1.Container{
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
		assert.Equal(t, r, c.result, "ContainerPort slices are equal")
	}
}
