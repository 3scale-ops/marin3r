package podv1mutator

import (
	"fmt"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
)

func Test_envoySidecarConfig_PopulateFromAnnotations(t *testing.T) {
	type args struct {
		annotations map[string]string
	}
	tests := []struct {
		name    string
		esc     *envoySidecarConfig
		args    args
		want    *envoySidecarConfig
		wantErr bool
	}{
		{
			"Populate '*envoySidecarConfig' from annotations",
			&envoySidecarConfig{},
			args{map[string]string{
				"marin3r.3scale.net/node-id":                                                "node-id",
				"marin3r.3scale.net/ports":                                                  "xxxx:1111",
				"marin3r.3scale.net/host-port-mappings":                                     "xxxx:3000",
				"marin3r.3scale.net/container-name":                                         "container",
				"marin3r.3scale.net/envoy-image":                                            "image",
				"marin3r.3scale.net/ads-configmap":                                          "cm",
				"marin3r.3scale.net/cluster-id":                                             "cluster-id",
				"marin3r.3scale.net/config-volume":                                          "config-volume",
				"marin3r.3scale.net/tls-volume":                                             "tls-volume",
				"marin3r.3scale.net/client-certificate":                                     "client-cert",
				"marin3r.3scale.net/envoy-extra-args":                                       "--log-level debug",
				fmt.Sprintf("%s/%s", marin3rAnnotationsDomain, paramResourceRequestsCPU):    "500m",
				fmt.Sprintf("%s/%s", marin3rAnnotationsDomain, paramResourceRequestsMemory): "700Mi",
				fmt.Sprintf("%s/%s", marin3rAnnotationsDomain, paramResourceLimitsCPU):      "1000m",
				fmt.Sprintf("%s/%s", marin3rAnnotationsDomain, paramResourceLimitsMemory):   "900Mi",
			}},
			&envoySidecarConfig{
				name:  "container",
				image: "image",
				ports: []corev1.ContainerPort{
					{Name: "xxxx", ContainerPort: 1111, HostPort: 3000},
				},
				bootstrapConfigMap: "cm",
				nodeID:             "node-id",
				clusterID:          "cluster-id",
				tlsVolume:          "tls-volume",
				configVolume:       "config-volume",
				clientCertSecret:   "client-cert",
				extraArgs:          "--log-level debug",
				resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("700Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1000m"),
						corev1.ResourceMemory: resource.MustParse("900Mi"),
					},
				},
			},
			false,
		}, {
			"Populate '*envoySidecarConfig' from annotations, leave all by default",
			&envoySidecarConfig{},
			args{map[string]string{
				"marin3r.3scale.net/node-id": "node-id",
			}},
			&envoySidecarConfig{
				name:               DefaultContainerName,
				image:              DefaultImage,
				ports:              []corev1.ContainerPort{},
				bootstrapConfigMap: DefaultBootstrapConfigMapV2,
				nodeID:             "node-id",
				clusterID:          "node-id",
				tlsVolume:          DefaultTLSVolume,
				configVolume:       DefaultConfigVolume,
				clientCertSecret:   DefaultClientCertificate,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.esc.PopulateFromAnnotations(tt.args.annotations); (err != nil) != tt.wantErr {
				t.Errorf("envoySidecarConfig.PopulateFromAnnotations() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.esc, tt.want) {
				t.Errorf("envoySidecarConfig.PopulateFromAnnotations() = '%v', want '%v'", tt.esc, tt.want)

			}
		})
	}
}

func Test_getContainerPorts(t *testing.T) {
	type args struct {
		annotations map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    []corev1.ContainerPort
		wantErr bool
	}{
		{
			"Slice of ContainerPorts from annotation, one port",
			args{map[string]string{
				"marin3r.3scale.net/ports": "xxxx:1111",
			}},
			[]corev1.ContainerPort{
				{Name: "xxxx", ContainerPort: 1111},
			},
			false,
		}, {
			"Slice of ContainerPorts from annotation, multiple ports",
			args{map[string]string{
				"marin3r.3scale.net/ports": "xxxx:1111,yyyy:2222,zzzz:3333",
			}},
			[]corev1.ContainerPort{
				{Name: "xxxx", ContainerPort: 1111},
				{Name: "yyyy", ContainerPort: 2222},
				{Name: "zzzz", ContainerPort: 3333},
			},
			false,
		}, {
			"Wrong annotations produces empty slice",
			args{map[string]string{
				"marin3r.3scale.net/xxxx": "xxxx:1111,yyyy:2222,zzzz:3333",
			}},
			[]corev1.ContainerPort{},
			false,
		}, {
			"No annotation produces empty slice",
			args{map[string]string{}},
			[]corev1.ContainerPort{},
			false,
		}, {
			"Mix spec with proto and spec without proto",
			args{map[string]string{
				"marin3r.3scale.net/ports": "xxxx:1111:UDP,yyyy:2222",
			}},
			[]corev1.ContainerPort{
				{Name: "xxxx", ContainerPort: 1111, Protocol: "UDP"},
				{Name: "yyyy", ContainerPort: 2222},
			},
			false,
		}, {
			"With host-port-mapping annotation",
			args{map[string]string{
				"marin3r.3scale.net/ports":              "xxxx:1111:UDP,yyyy:2222,zzzz:3333",
				"marin3r.3scale.net/host-port-mappings": "xxxx:5000,yyyy:6000",
			}},
			[]corev1.ContainerPort{
				{Name: "xxxx", ContainerPort: 1111, Protocol: "UDP", HostPort: 5000},
				{Name: "yyyy", ContainerPort: 2222, HostPort: 6000},
				{Name: "zzzz", ContainerPort: 3333},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getContainerPorts(tt.args.annotations)
			if (err != nil) != tt.wantErr {
				t.Errorf("getContainerPorts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getContainerPorts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parsePortSpec(t *testing.T) {
	type args struct {
		spec string
	}
	tests := []struct {
		name    string
		args    args
		want    *corev1.ContainerPort
		wantErr bool
	}{
		{
			"ContainerPort struct from annotation string, without protocol",
			args{"http:3000"},
			&corev1.ContainerPort{
				Name:          "http",
				ContainerPort: 3000,
			},
			false,
		}, {
			"ContainerPort struct from annotation string, with TCP protocol",
			args{"http:8080:TCP"},
			&corev1.ContainerPort{
				Name:          "http",
				ContainerPort: 8080,
				Protocol:      corev1.Protocol("TCP"),
			},
			false,
		}, {
			"ContainerPort struct from annotation string, with UDP protocol",
			args{"http:8080:UDP"},
			&corev1.ContainerPort{
				Name:          "http",
				ContainerPort: 8080,
				Protocol:      corev1.Protocol("UDP"),
			},
			false,
		}, {
			"ContainerPort struct from annotation string, with SCTP protocol",
			args{"http:8080:SCTP"},
			&corev1.ContainerPort{
				Name:          "http",
				ContainerPort: 8080,
				Protocol:      corev1.Protocol("SCTP"),
			},
			false,
		}, {
			"Error, wrong protocol",
			args{"http:8080:XXX"},
			nil,
			true,
		}, {
			"Error, privileged port",
			args{"http:80:TCP"},
			nil,
			true,
		},
		{
			"Error, missing params",
			args{"http"},
			nil,
			true,
		}, {
			"Error, not a port",
			args{"http:xxxx"},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePortSpec(tt.args.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePortSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parsePortSpec() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_hostPortMapping(t *testing.T) {
	type args struct {
		portName    string
		annotations map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    int32
		wantErr bool
	}{
		{
			"Parse a single 'host-port-mapping' spec",
			args{"http", map[string]string{"marin3r.3scale.net/host-port-mappings": "http:3000"}},
			3000,
			false,
		}, {
			"Parse several 'host-port-mapping' specs",
			args{"admin", map[string]string{"marin3r.3scale.net/host-port-mappings": "http:3000,admin:6000"}},
			6000,
			false,
		}, {
			"Incorrect 'host-port-mapping' spec 1",
			args{"http", map[string]string{"marin3r.3scale.net/host-port-mappings": "admin,6000"}},
			0,
			true,
		}, {
			"Not a port",
			args{"admin", map[string]string{"marin3r.3scale.net/host-port-mappings": "admin:4000i"}},
			0,
			true,
		}, {
			"Privileged port not allowed",
			args{"admin", map[string]string{"marin3r.3scale.net/host-port-mappings": "admin:80"}},
			0,
			true,
		}, {
			"Port not found in spec",
			args{"http", map[string]string{"marin3r.3scale.net/host-port-mappings": "admin:80"}},
			0,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hostPortMapping(tt.args.portName, tt.args.annotations)
			if (err != nil) != tt.wantErr {
				t.Errorf("hostPortMapping() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("hostPortMapping() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_portNumber(t *testing.T) {
	type args struct {
		sport string
	}
	tests := []struct {
		name    string
		args    args
		want    int32
		wantErr bool
	}{
		{"Parse port string", args{"1111"}, 1111, false},
		{"Error, not a number", args{"xxxx"}, 0, true},
		{"Port 1024 is allowed", args{"1023"}, 0, true},
		{"Error, privileged port 1024", args{"1023"}, 0, true},
		{"Error, privileged port 0", args{"0"}, 0, true},
		{"Error, privileged port 100", args{"100"}, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := portNumber(tt.args.sport)
			if (err != nil) != tt.wantErr {
				t.Errorf("portNumber() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("portNumber() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_envoySidecarConfig_container(t *testing.T) {
	tests := []struct {
		name string
		esc  *envoySidecarConfig
		want corev1.Container
	}{
		{
			"Returns resolved container",
			&envoySidecarConfig{
				name:  "sidecar",
				image: "sidecar:latest",
				ports: []corev1.ContainerPort{
					{
						Name:          "udp",
						ContainerPort: 6000,
						Protocol:      corev1.Protocol("UDP"),
					},
					{
						Name:          "https",
						ContainerPort: 8443,
					},
				},
				bootstrapConfigMap: "ads-configmap",
				nodeID:             "test-id",
				clusterID:          "cluster-id",
				tlsVolume:          "tls-volume",
				configVolume:       "config-volume",
				clientCertSecret:   "secret",
			},
			corev1.Container{
				Name:    "sidecar",
				Image:   "sidecar:latest",
				Command: []string{"envoy"},
				Args: []string{
					"-c",
					"/etc/envoy/bootstrap/config.json",
					"--service-node",
					"test-id",
					"--service-cluster",
					"cluster-id",
				},
				Ports: []corev1.ContainerPort{
					{
						Name:          "udp",
						ContainerPort: 6000,
						Protocol:      corev1.Protocol("UDP"),
					},
					{
						Name:          "https",
						ContainerPort: 8443,
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "tls-volume",
						ReadOnly:  true,
						MountPath: "/etc/envoy/tls/client",
					},
					{
						Name:      "config-volume",
						ReadOnly:  true,
						MountPath: "/etc/envoy/bootstrap",
					},
				},
			},
		},
		{
			"Returns resolved container, with extra-args",
			&envoySidecarConfig{
				name:  "sidecar",
				image: "sidecar:latest",
				ports: []corev1.ContainerPort{
					{
						Name:          "udp",
						ContainerPort: 6000,
						Protocol:      corev1.Protocol("UDP"),
					},
					{
						Name:          "https",
						ContainerPort: 8443,
					},
				},
				bootstrapConfigMap: "ads-configmap",
				nodeID:             "test-id",
				clusterID:          "cluster-id",
				tlsVolume:          "tls-volume",
				configVolume:       "config-volume",
				clientCertSecret:   "secret",
				extraArgs:          "-x xxxx -z zzzz",
			},
			corev1.Container{
				Name:    "sidecar",
				Image:   "sidecar:latest",
				Command: []string{"envoy"},
				Args: []string{
					"-c",
					"/etc/envoy/bootstrap/config.json",
					"--service-node",
					"test-id",
					"--service-cluster",
					"cluster-id",
					"-x",
					"xxxx",
					"-z",
					"zzzz",
				},
				Ports: []corev1.ContainerPort{
					{
						Name:          "udp",
						ContainerPort: 6000,
						Protocol:      corev1.Protocol("UDP"),
					},
					{
						Name:          "https",
						ContainerPort: 8443,
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "tls-volume",
						ReadOnly:  true,
						MountPath: "/etc/envoy/tls/client",
					},
					{
						Name:      "config-volume",
						ReadOnly:  true,
						MountPath: "/etc/envoy/bootstrap",
					},
				},
			},
		},
		{
			"Returns resolved container with resource requirements",
			&envoySidecarConfig{
				name:  "sidecar",
				image: "sidecar:latest",
				ports: []corev1.ContainerPort{
					{
						Name:          "udp",
						ContainerPort: 6000,
						Protocol:      corev1.Protocol("UDP"),
					},
					{
						Name:          "https",
						ContainerPort: 8443,
					},
				},
				bootstrapConfigMap: "ads-configmap",
				nodeID:             "test-id",
				clusterID:          "cluster-id",
				tlsVolume:          "tls-volume",
				configVolume:       "config-volume",
				clientCertSecret:   "secret",
				resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("700Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1000m"),
						corev1.ResourceMemory: resource.MustParse("900Mi"),
					},
				},
			},
			corev1.Container{
				Name:    "sidecar",
				Image:   "sidecar:latest",
				Command: []string{"envoy"},
				Args: []string{
					"-c",
					"/etc/envoy/bootstrap/config.json",
					"--service-node",
					"test-id",
					"--service-cluster",
					"cluster-id",
				},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("700Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1000m"),
						corev1.ResourceMemory: resource.MustParse("900Mi"),
					},
				},
				Ports: []corev1.ContainerPort{
					{
						Name:          "udp",
						ContainerPort: 6000,
						Protocol:      corev1.Protocol("UDP"),
					},
					{
						Name:          "https",
						ContainerPort: 8443,
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "tls-volume",
						ReadOnly:  true,
						MountPath: "/etc/envoy/tls/client",
					},
					{
						Name:      "config-volume",
						ReadOnly:  true,
						MountPath: "/etc/envoy/bootstrap",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.esc.container(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("envoySidecarConfig.container() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getStringParam(t *testing.T) {
	type args struct {
		key         string
		annotations map[string]string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"Return string value from annotation",
			args{"envoy-image", map[string]string{"marin3r.3scale.net/envoy-image": "image"}},
			"image",
		}, {
			"Return string value from default",
			args{"envoy-image", map[string]string{}},
			DefaultImage,
		}, {
			"Return cluster-id from annotation",
			args{"cluster-id", map[string]string{"marin3r.3scale.net/cluster-id": "cluster-id"}},
			"cluster-id",
		}, {
			"Return cluster-id from default (defaults to node-id)",
			args{"cluster-id", map[string]string{"marin3r.3scale.net/node-id": "test"}},
			"test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getStringParam(tt.args.key, tt.args.annotations); got != tt.want {
				t.Errorf("getStringParam() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getNodeID(t *testing.T) {
	type args struct {
		annotations map[string]string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"Return node-id from annotation",
			args{map[string]string{"marin3r.3scale.net/node-id": "test-id"}},
			"test-id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getNodeID(tt.args.annotations); got != tt.want {
				t.Errorf("getNodeID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_envoySidecarConfig_volumes(t *testing.T) {
	tests := []struct {
		name string
		esc  *envoySidecarConfig
		want []corev1.Volume
	}{
		{
			"Returns resolved pod volumes",
			&envoySidecarConfig{
				bootstrapConfigMap: "ads-configmap",
				tlsVolume:          "tls-volume",
				configVolume:       "config-volume",
				clientCertSecret:   "secret",
			},
			[]corev1.Volume{
				{
					Name: "tls-volume",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "secret",
						},
					},
				},
				{
					Name: "config-volume",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "ads-configmap",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.esc.volumes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("envoySidecarConfig.volumes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_lookupMarin3rAnnotation(t *testing.T) {
	type args struct {
		annotations map[string]string
		key         string
	}

	tests := []struct {
		name   string
		args   args
		want   string
		wantOk bool
	}{
		{
			"Marin3r annotation exists",
			args{
				annotations: map[string]string{
					fmt.Sprintf("%s/%s", marin3rAnnotationsDomain, "existingkey"): "examplevalue",
				},
				key: "existingkey",
			},
			"examplevalue",
			true,
		},
		{
			"Marin3r annotation exists and is empty",
			args{
				annotations: map[string]string{
					fmt.Sprintf("%s/%s", marin3rAnnotationsDomain, "existingkey"): "",
				},
				key: "existingkey",
			},
			"",
			true,
		},
		{
			"Marin3r annotation does not exist",
			args{
				annotations: map[string]string{
					fmt.Sprintf("%s/%s", marin3rAnnotationsDomain, "existingkey"): "myval",
				},
				key: "unexistingkey",
			},
			"",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := lookupMarin3rAnnotation(tt.args.key, tt.args.annotations)
			if ok != tt.wantOk {
				t.Errorf("lookupMarin3rAnnotation() ok = %v, wantOk %v", got, tt.wantOk)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("lookupMarin3rAnnotation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getContainerResourceRequirements(t *testing.T) {
	type args struct {
		annotations map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    corev1.ResourceRequirements
		wantErr bool
	}{
		{
			"No resource requirement annotations",
			args{map[string]string{}},
			corev1.ResourceRequirements{},
			false,
		},
		{
			"invalid resource requirement annotation value",
			args{map[string]string{
				fmt.Sprintf("%s/%s", marin3rAnnotationsDomain, paramResourceLimitsMemory): "invalidMemoryValue",
			}},
			corev1.ResourceRequirements{},
			true,
		},
		{
			"resource requirement annotation set but invalid empty value",
			args{map[string]string{
				fmt.Sprintf("%s/%s", marin3rAnnotationsDomain, paramResourceRequestsCPU): "",
			}},
			corev1.ResourceRequirements{},
			true,
		},
		{
			"resource cpu request set",
			args{map[string]string{
				fmt.Sprintf("%s/%s", marin3rAnnotationsDomain, paramResourceRequestsCPU): "100m",
			}},
			corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("100m"),
				},
			},
			false,
		},
		{
			"resource cpu limit set",
			args{map[string]string{
				fmt.Sprintf("%s/%s", marin3rAnnotationsDomain, paramResourceLimitsCPU): "200m",
			}},
			corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("200m"),
				},
			},
			false,
		},
		{
			"resource memory request set",
			args{map[string]string{
				fmt.Sprintf("%s/%s", marin3rAnnotationsDomain, paramResourceRequestsMemory): "100Mi",
			}},
			corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("100Mi"),
				},
			},
			false,
		},
		{
			"resource memory limit set",
			args{map[string]string{
				fmt.Sprintf("%s/%s", marin3rAnnotationsDomain, paramResourceLimitsMemory): "200Mi",
			}},
			corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
			},
			false,
		},
		{
			"resource requests and limits set",
			args{map[string]string{
				fmt.Sprintf("%s/%s", marin3rAnnotationsDomain, paramResourceRequestsCPU):    "500m",
				fmt.Sprintf("%s/%s", marin3rAnnotationsDomain, paramResourceRequestsMemory): "700Mi",
				fmt.Sprintf("%s/%s", marin3rAnnotationsDomain, paramResourceLimitsCPU):      "1000m",
				fmt.Sprintf("%s/%s", marin3rAnnotationsDomain, paramResourceLimitsMemory):   "900Mi",
			}},
			corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("700Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1000m"),
					corev1.ResourceMemory: resource.MustParse("900Mi"),
				},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getContainerResourceRequirements(tt.args.annotations)
			if (err != nil) != tt.wantErr {
				t.Errorf("getContainerResourceRequirements() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !equality.Semantic.DeepEqual(got, tt.want) {
				t.Errorf("getContainerResourceRequirements() = %v, want %v", got, tt.want)
			}
		})
	}

}

func Test_getBootstrapConfigMap(t *testing.T) {
	type args struct {
		annotations map[string]string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"Returns user defined ConfigMap",
			args{map[string]string{
				fmt.Sprintf("%s/%s", marin3rAnnotationsDomain, paramBootstrapConfigMap): "custom-cm",
			}},
			"custom-cm",
		},
		{
			"Returns v3 ConfigMap when v3 version specified",
			args{map[string]string{
				fmt.Sprintf("%s/%s", marin3rAnnotationsDomain, paramEnvoyAPIVersion): "v3",
			}},
			DefaultBootstrapConfigMapV3,
		},
		{
			"Returns v2 ConfigMap when v2 version specified",
			args{map[string]string{
				fmt.Sprintf("%s/%s", marin3rAnnotationsDomain, paramEnvoyAPIVersion): "v2",
			}},
			DefaultBootstrapConfigMapV2,
		},
		{
			"Returns v2 ConfigMap when the version annotation does not parse correctly",
			args{map[string]string{
				fmt.Sprintf("%s/%s", marin3rAnnotationsDomain, paramEnvoyAPIVersion): "xx",
			}},
			DefaultBootstrapConfigMapV2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getBootstrapConfigMap(tt.args.annotations); got != tt.want {
				t.Errorf("getBootstrapConfigMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
