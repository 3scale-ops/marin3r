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

package reconciler

import (
	"bytes"
	"fmt"

	envoyapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/roivaz/marin3r/pkg/cache"
	"github.com/roivaz/marin3r/pkg/envoy"
	"github.com/roivaz/marin3r/pkg/util"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	configMapKey = "config.yaml"
)

type ConfigMapReconcileJob struct {
	eventType EventType
	nodeID    string
	configMap *corev1.ConfigMap
}

func NewConfigMapReconcileJob(nodeID string, eventType EventType, configMap *corev1.ConfigMap) *ConfigMapReconcileJob {
	return &ConfigMapReconcileJob{
		eventType: eventType,
		nodeID:    nodeID,
		configMap: configMap,
	}
}

func (job ConfigMapReconcileJob) Push(queue chan ReconcileJob) {
	queue <- job
}

type resFromFile struct {
	Clusters        []*envoyapi.Cluster               `protobuf:"bytes,1,rep,name=clusters,json=clusters" json:"clusters"`
	Listeners       []*envoyapi.Listener              `protobuf:"bytes,2,rep,name=listeners,json=listeners" json:"listeners"`
	LoadAssignments []*envoyapi.ClusterLoadAssignment `protobuf:"bytes,3,rep,name=load_assignments,json=loadAssignments" json:"load_assignments"`
}

// This is so resFromFile implements the protbuf api an the resFromFile
// struct can be directly unmarshalled into envoyapi structs
func (m *resFromFile) Reset()         { *m = resFromFile{} }
func (m *resFromFile) String() string { return proto.CompactTextString(m) }
func (*resFromFile) ProtoMessage()    {}

func (job ConfigMapReconcileJob) process(c cache.Cache, client *util.K8s, namespace string, logger *zap.SugaredLogger) ([]string, error) {

	logger.Debugf("Processing ConfigMap job for node-id %s", job.nodeID)
	// Check if it's the first time we see this
	// nodeID, in which case we need to bootstrap
	// its cache
	if _, ok := c[job.nodeID]; !ok {
		logger.Infof("Boostraping cache for node-id %s", job.nodeID)
		c.NewNodeCache(job.nodeID)
		// We need to trigger a reconcile for the secrets
		// so this new cache gets populated with them
		err := syncNodeSecrets(client, namespace, job.nodeID, c)
		if err != nil {
			logger.Errorf("Error populating secrets cache for node-id %s: '%s'", job.nodeID, err)
			// Delete the node cache so in the
			// next job it will try to rebuild the
			// secrets cache again
			c.DeleteNodeCache(job.nodeID)
			return []string{}, err
		}
	}

	switch job.eventType {

	case Add, Update:

		// Clear current cached clusters and listeners, we don't care about
		// previous values because the yaml in the ConfigMap provider is
		// expected to be complete
		c.ClearResources(job.nodeID, cache.Cluster)
		c.ClearResources(job.nodeID, cache.Listener)

		j, err := yaml.YAMLToJSON([]byte(job.configMap.Data["config.yaml"]))
		if err != nil {
			logger.Errorf("Error converting yaml to json: '%s'", err)
			return []string{}, fmt.Errorf("Error converting yaml to json: '%s'", err)
		}

		rff := resFromFile{}
		if err := jsonpb.Unmarshal(bytes.NewReader(j), &rff); err != nil {
			logger.Errorf("Error unmarshalling config for node-id %s: '%s'", job.nodeID, err)
			return []string{}, fmt.Errorf("Error unmarshalling config for node-id %s: '%s'", job.nodeID, err)
		}

		for _, cluster := range rff.Clusters {
			c.SetResource(job.nodeID, cluster.Name, cache.Cluster, cluster)
		}

		for _, lis := range rff.Listeners {
			c.SetResource(job.nodeID, lis.Name, cache.Listener, lis)
		}

	case Delete:
		// Just warn the user about the deletion of the config
		logger.Warnf("The config for node-id '%s' is about to be deleted", job.nodeID)
	}

	return []string{job.nodeID}, nil
}

// SyncNodeSecrets synchronously builds/rebuilds the whole secrets cache
func syncNodeSecrets(client *util.K8s, namespace, nodeID string, c cache.Cache) error {

	list, err := client.Clientset.CoreV1().Secrets(namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, s := range list.Items {
		if cn, ok := s.GetAnnotations()[certificateAnnotation]; ok {
			c.SetResource(nodeID, cn, cache.Secret, envoy.NewSecret(cn, string(s.Data["tls.key"]), string(s.Data["tls.crt"])))
		}
	}
	return nil
}
