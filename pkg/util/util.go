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

package util

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type K8s struct {
	Clientset kubernetes.Interface
}

func FakeClusterClient(objects ...runtime.Object) *K8s {
	client := K8s{}
	client.Clientset = fake.NewSimpleClientset(objects...)
	return &client
}

func FakeErrorClusterClient(objects ...runtime.Object) *K8s {
	client := K8s{}
	client.Clientset = &fake.Clientset{}
	client.Clientset.(*fake.Clientset).AddReactor("*", "*",
		func(action testing.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, fmt.Errorf("Faked error")
		},
	)
	return &client
}

func InClusterClient() (*K8s, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	client := K8s{}
	client.Clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func OutOfClusterClient() (*K8s, error) {
	var kubeconfig string

	if env := os.Getenv("KUBECONFIG"); env != "" {
		kubeconfig = env
	} else if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	} else {
		return nil, fmt.Errorf("kubeconfig not in default path and env var not set")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	client := K8s{}
	client.Clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &client, err
}
