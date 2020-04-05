// Copyright 2020 rvazquez@redhat.com
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package webhook

import (
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const (
	// annotations
	// node-id annotation the only required one, and the one
	// that determines if pod must be mutated or not
	nodeIDAnnotation = "marin3r.3scale.net/node-id"
	// format "name:port:protocol,name:port:protcol"
	portsAnnotation            = "marin3r.3scale.net/ports"
	containerNameAnnotation    = "marin3r.3scale.net/container-name"
	imageAnnotation            = "marin3r.3scale.net/image"
	adsConfigMapAnnotation     = "marin3r.3scale.net/ads-configmap"
	clusterIDAnnotation        = "marin3r.3scale.net/cluster-id"
	configVolumeAnnotation     = "marin3r.3scale.net/config-volume"
	tlsVolumeAnnotation        = "marin3r.3scale.net/tls-volume"
	clientCertSecretAnnotation = "marin3r.3scale.net/client-certificate"

	// default values
	defaultContainerName     = "envoy-sidecar"
	defaultContainerImage    = "envoyproxy/envoy:v1.13.1"
	defaultADSConfigMap      = "envoy-sidecar-bootstrap"
	defaultEnvoyConfigVolume = "envoy-sidecar-bootstrap"
	defaultEnvoyTLSVolume    = "envoy-sidecar-tls"
	defaultPortProtocol      = "TCP"
	defaultClientCertSecret  = "envoy-sidecar-client-cert"
)

type envoySidecarConfig struct {
	name             string
	image            string
	ports            []corev1.ContainerPort
	adsConfigMap     string
	nodeID           string
	clusterID        string
	tlsVolume        string
	configVolume     string
	clientCertSecret string
}

func (esc *envoySidecarConfig) PopulateFromAnnotations(annotations map[string]string) error {

	esc.name = getContainerName(annotations)
	esc.image = getContainerImage(annotations)

	ports, err := getContainerPorts(annotations)
	if err != nil {
		return err
	}
	esc.ports = ports
	esc.adsConfigMap = getADSConfigMap(annotations)
	esc.nodeID = getEnvoyNodeID(annotations)
	esc.clusterID = getEnvoyClusterID(annotations)
	esc.configVolume = getConfigVolumeName(annotations)
	esc.tlsVolume = getTLSVolumeName(annotations)
	esc.clientCertSecret = getClientCertSecret(annotations)

	return nil
}

func getContainerName(annotations map[string]string) string {
	if name, ok := annotations[containerNameAnnotation]; ok {
		return name
	} else {
		return defaultContainerName
	}
}

func getContainerImage(annotations map[string]string) string {
	if image, ok := annotations[imageAnnotation]; ok {
		return image
	} else {
		return defaultContainerImage
	}
}

func getContainerPorts(annotations map[string]string) ([]corev1.ContainerPort, error) {
	plist := []corev1.ContainerPort{}
	if ports, ok := annotations[portsAnnotation]; ok {
		for _, containerPort := range strings.Split(ports, ",") {
			if p, err := parsePortSpec(containerPort); err != nil {
				return []corev1.ContainerPort{}, err
			} else {
				plist = append(plist, *p)
			}
		}

	} else {
		// no ports defined for envoy sidecat
		return []corev1.ContainerPort{}, nil
	}

	return plist, nil
}

func parsePortSpec(spec string) (*corev1.ContainerPort, error) {
	params := strings.Split(spec, ":")
	// Each port spec should at least contain name and port number
	if len(params) < 2 {
		return nil, fmt.Errorf("Incorrect format, the por specification format for the envoy sidecar container is 'name:port[:protocol]'")
	}
	// Parse and validate the port number
	portNumber, err := strconv.Atoi(params[1])
	if err != nil {
		return nil, fmt.Errorf("%s doesn't look a number, the por specification format for the envoy sidecar container is 'name:port[:protocol]'", params[1])
	}
	// Check if port is within allowed ranges. Privileged ports are not allowed
	if portNumber < 1024 || portNumber > 65535 {
		return nil, fmt.Errorf("Port number %s is not in the range 1024-65535", params[1])
	}

	// Check that protocol matches one of the allowed protocols
	if len(params) == 3 {
		if params[2] == "TCP" || params[2] == "UDP" || params[2] == "SCTP" {
			return &corev1.ContainerPort{
				Name:          params[0],
				ContainerPort: int32(portNumber),
				Protocol:      corev1.Protocol(params[2]),
			}, nil
		}
	}

	// No protocol specified
	return &corev1.ContainerPort{
		Name:          params[0],
		ContainerPort: int32(portNumber),
	}, nil

}

func getADSConfigMap(annotations map[string]string) string {
	if cm, ok := annotations[adsConfigMapAnnotation]; ok {
		return cm
	} else {
		return defaultADSConfigMap
	}
}
func getEnvoyNodeID(annotations map[string]string) string {
	// the node-id annotation is always present, otherwise
	// mutation won't even be triggered
	return annotations[nodeIDAnnotation]
}

func getEnvoyClusterID(annotations map[string]string) string {
	if clusterID, ok := annotations[clusterIDAnnotation]; ok {
		return clusterID
	} else {
		return annotations[nodeIDAnnotation]
	}
}

func getTLSVolumeName(annotations map[string]string) string {
	if name, ok := annotations[tlsVolumeAnnotation]; ok {
		return name
	} else {
		return defaultEnvoyTLSVolume
	}
}

func getConfigVolumeName(annotations map[string]string) string {
	if clusterID, ok := annotations[configVolumeAnnotation]; ok {
		return clusterID
	} else {
		return defaultEnvoyConfigVolume
	}
}

func getClientCertSecret(annotations map[string]string) string {
	if name, ok := annotations[clientCertSecretAnnotation]; ok {
		return name
	} else {
		return defaultClientCertSecret
	}
}

func (esc *envoySidecarConfig) container() corev1.Container {

	return corev1.Container{
		Name:    esc.name,
		Image:   esc.image,
		Command: []string{"envoy"},
		Args: []string{
			"-c",
			"/etc/envoy/bootstrap/config.yaml",
			"--service-node",
			esc.nodeID,
			"--service-cluster",
			esc.clusterID,
		},
		Ports: esc.ports,
		VolumeMounts: []corev1.VolumeMount{
			corev1.VolumeMount{
				Name:      esc.tlsVolume,
				ReadOnly:  true,
				MountPath: "/etc/envoy/tls/client",
			},
			corev1.VolumeMount{
				Name:      esc.configVolume,
				ReadOnly:  true,
				MountPath: "/etc/envoy/bootstrap",
			},
		},
	}
}

func (esc *envoySidecarConfig) volumes() []corev1.Volume {

	volumes := []corev1.Volume{
		corev1.Volume{
			Name: esc.tlsVolume,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: esc.clientCertSecret,
				},
			},
		},
		corev1.Volume{
			Name: esc.configVolume,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: esc.adsConfigMap,
					},
				},
			},
		},
	}

	return volumes
}
