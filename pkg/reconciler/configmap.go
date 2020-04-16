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

	"github.com/3scale/marin3r/pkg/cache"
	"github.com/3scale/marin3r/pkg/envoy"
	"github.com/3scale/marin3r/pkg/util"
	envoyapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	_ "github.com/cncf/udpa/go/udpa/annotations"
	_ "github.com/envoyproxy/go-control-plane/envoy/annotations"

	// the list of config imports have been generated executing the following command
	// in the go-control-plane repo:
	//
	// for pkg in $(dirname $(grep -R -l "package envoy_config") | egrep -v "v2alpha|v3|v1"); \
	//   do echo "_ \"github.com/envoyproxy/go-control-plane/${pkg}\""; done
	_ "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/cluster/redis"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/accesslog/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/fault/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/buffer/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/compressor/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/cors/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/csrf/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/dynamo/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/ext_authz/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/fault/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/grpc_http1_bridge/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/grpc_web/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/gzip/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/header_to_metadata/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/health_check/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/ip_tagging/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/lua/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/on_demand/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/rate_limit/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/rbac/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/router/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/squash/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/transcoder/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/listener/http_inspector/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/listener/original_dst/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/listener/proxy_protocol/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/listener/tls_inspector/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/client_ssl_auth/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/direct_response/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/echo/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/ext_authz/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/mongo_proxy/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/rate_limit/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/rbac/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/redis_proxy/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/sni_cluster/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/tcp_proxy/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/health_checker/redis/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/listener/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/metrics/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/ratelimit/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/retry/omit_canary_hosts/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/retry/omit_host_metadata/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/retry/previous_hosts/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/retry/previous_priorities"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/trace/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/transport_socket/raw_buffer/v2"
)

const (
	configMapKey = "config.yaml"
)

// ConfigMapReconcileJob is a struct that holds the
// required information for ConfigMap type jobs
type ConfigMapReconcileJob struct {
	eventType EventType
	nodeID    string
	configMap corev1.ConfigMap
}

// NewConfigMapReconcileJob creates a new ConfigMapReconcileJob
// from provided parameters
func NewConfigMapReconcileJob(nodeID string, eventType EventType, configMap corev1.ConfigMap) *ConfigMapReconcileJob {
	return &ConfigMapReconcileJob{
		eventType: eventType,
		nodeID:    nodeID,
		configMap: configMap,
	}
}

// Push pushes the ConfigMapReconcileJob to the queue
func (job ConfigMapReconcileJob) Push(queue chan ReconcileJob) {
	queue <- job
}

type resFromFile struct {
	Clusters        []*envoyapi.Cluster               `protobuf:"bytes,1,rep,name=clusters,json=clusters" json:"clusters"`
	Listeners       []*envoyapi.Listener              `protobuf:"bytes,2,rep,name=listeners,json=listeners" json:"listeners"`
	LoadAssignments []*envoyapi.ClusterLoadAssignment `protobuf:"bytes,3,rep,name=load_assignments,json=loadAssignments" json:"load_assignments"`
}

// This is so resFromFile implements the protbuf api and the resFromFile
// struct can be directly unmarshalled into envoyapi structs

// Reset is noop function for resFromFile to implement protobuf interface
func (m *resFromFile) Reset() { *m = resFromFile{} }

// String is noop function for resFromFile to implement protobuf interface
func (m *resFromFile) String() string { return proto.CompactTextString(m) }

// ProtoMessage is noop function for resFromFile to implement protobuf interface
func (*resFromFile) ProtoMessage() {}

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
