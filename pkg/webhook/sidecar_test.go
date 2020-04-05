package webhook

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
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
				"marin3r.3scale.net/node-id":            "node-id",
				"marin3r.3scale.net/ports":              "xxxx:1111",
				"marin3r.3scale.net/host-port-mappings": "xxxx:3000",
				"marin3r.3scale.net/container-name":     "container",
				"marin3r.3scale.net/image":              "image",
				"marin3r.3scale.net/ads-configmap":      "cm",
				"marin3r.3scale.net/cluster-id":         "cluster-id",
				"marin3r.3scale.net/config-volume":      "config-volume",
				"marin3r.3scale.net/tls-volume":         "tls-volume",
				"marin3r.3scale.net/client-certificate": "client-cert",
			}},
			&envoySidecarConfig{
				name:  "container",
				image: "image",
				ports: []corev1.ContainerPort{
					corev1.ContainerPort{Name: "xxxx", ContainerPort: 1111, HostPort: 3000},
				},
				adsConfigMap:     "cm",
				nodeID:           "node-id",
				clusterID:        "cluster-id",
				tlsVolume:        "tls-volume",
				configVolume:     "config-volume",
				clientCertSecret: "client-cert",
			},
			false,
		}, {
			"Populate '*envoySidecarConfig' from annotations, leave all by default",
			&envoySidecarConfig{},
			args{map[string]string{
				"marin3r.3scale.net/node-id": "node-id",
			}},
			&envoySidecarConfig{
				name:             defaultContainerName,
				image:            defaultImage,
				ports:            []corev1.ContainerPort{},
				adsConfigMap:     defaultADSConfigMap,
				nodeID:           "node-id",
				clusterID:        "node-id",
				tlsVolume:        defaultTLSVolume,
				configVolume:     defaultConfigVolume,
				clientCertSecret: defaultClientCertificate,
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
				corev1.ContainerPort{Name: "xxxx", ContainerPort: 1111},
			},
			false,
		}, {
			"Slice of ContainerPorts from annotation, multiple ports",
			args{map[string]string{
				"marin3r.3scale.net/ports": "xxxx:1111,yyyy:2222,zzzz:3333",
			}},
			[]corev1.ContainerPort{
				corev1.ContainerPort{Name: "xxxx", ContainerPort: 1111},
				corev1.ContainerPort{Name: "yyyy", ContainerPort: 2222},
				corev1.ContainerPort{Name: "zzzz", ContainerPort: 3333},
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
				corev1.ContainerPort{Name: "xxxx", ContainerPort: 1111, Protocol: "UDP"},
				corev1.ContainerPort{Name: "yyyy", ContainerPort: 2222},
			},
			false,
		}, {
			"With host-port-mapping annotation",
			args{map[string]string{
				"marin3r.3scale.net/ports":              "xxxx:1111:UDP,yyyy:2222,zzzz:3333",
				"marin3r.3scale.net/host-port-mappings": "xxxx:5000,yyyy:6000",
			}},
			[]corev1.ContainerPort{
				corev1.ContainerPort{Name: "xxxx", ContainerPort: 1111, Protocol: "UDP", HostPort: 5000},
				corev1.ContainerPort{Name: "yyyy", ContainerPort: 2222, HostPort: 6000},
				corev1.ContainerPort{Name: "zzzz", ContainerPort: 3333},
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
			corev1.Container{
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
			args{"image", map[string]string{"marin3r.3scale.net/image": "image"}},
			"image",
		}, {
			"Return string value from default",
			args{"image", map[string]string{}},
			defaultImage,
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
