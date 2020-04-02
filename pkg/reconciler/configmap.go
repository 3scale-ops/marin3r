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

	envoyapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
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

func (srj ConfigMapReconcileJob) Push(queue chan ReconcileJob) {
	queue <- srj
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

func (job ConfigMapReconcileJob) process(c *Caches, logger *zap.SugaredLogger) {

	switch job.eventType {

	case Add, Update:

		// Clear current cached clusters and listeners, we don't care about
		// previous values because the yaml in the ConfigMap provider is
		// expected to be complete
		c.clusters = map[string]*envoyapi.Cluster{}
		c.listeners = map[string]*envoyapi.Listener{}
		// c.endpoints = map[string]*envoyapi.Listener{}

		j, err := yaml.YAMLToJSON([]byte(job.configMap.Data["config.yaml"]))
		if err != nil {
			logger.Errorf("Error converting yaml to json: '%s'", err)
			return
		}

		rff := resFromFile{}
		if err := jsonpb.Unmarshal(bytes.NewReader(j), &rff); err != nil {
			logger.Errorf("Error unmarshalling config for node-id %s: '%s'", nodeID, err)
			return
		}

		for _, cluster := range rff.Clusters {
			c.clusters[cluster.Name] = cluster
		}

		for _, lis := range rff.Listeners {
			c.listeners[lis.Name] = lis
		}

	case Delete:
		// Just warn the user about the deletion of the config
		logger.Warnf("The config for node-id '%s' is about to be deleted", nodeID)
	}
}
