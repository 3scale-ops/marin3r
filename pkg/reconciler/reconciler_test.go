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
	"fmt"
	"reflect"
	"testing"
	"time"

	envoyapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoyauth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoysd "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

func testReconciler(rcaches caches) *Reconciler {
	lg, _ := zap.NewDevelopment()
	logger := lg.Sugar()
	stopper := make(chan struct{})
	scache := cache.NewSnapshotCache(true, cache.IDHash{}, nil)
	var rc caches
	// return NewReconciler(&kubernetes.Clientset{}, "xxxx", &scache, stopper, logger)
	if rcaches != nil {
		rc = rcaches
	} else {
		rc = make(caches)
	}

	return &Reconciler{
		clientset:     &kubernetes.Clientset{},
		namespace:     "namespace",
		caches:        rc,
		snapshotCache: &scache,
		Queue:         make(chan ReconcileJob),
		logger:        logger,
		stopper:       stopper,
	}
}

type testJob struct {
	nodeIDs []string
	fail    bool
	panic   bool
}

func (job testJob) Push(queue chan ReconcileJob) { queue <- job }
func (job testJob) process(c caches, clientset *kubernetes.Clientset, namespace string, logger *zap.SugaredLogger) ([]string, error) {

	// Simulate a job panic
	if job.fail {
		panic("job failed")
	}
	// Simulate a job failure
	if job.fail {
		return []string{}, fmt.Errorf("job failed")
	}

	// Simulate a successful job
	for _, nodeID := range job.nodeIDs {
		c[nodeID] = NewNodeCaches()
		c[nodeID].secrets[fmt.Sprintf("%s-secret", nodeID)] = &envoyauth.Secret{Name: fmt.Sprintf("%s-secret", nodeID)}
	}

	return job.nodeIDs, nil
}

func TestReconciler_RunReconciler(t *testing.T) {

	type args struct {
		jobs []testJob
	}

	type want struct {
		nodeID string
		value  map[string]cache.Resource
	}

	reconcileLoopTestFn := func(jobs []testJob, want []want) {
		rec := testReconciler(nil)

		go func() {
			rec.RunReconciler()
		}()

		// Push all jobs to the queue
		for _, job := range jobs {
			job.Push(rec.Queue)
		}

		// Wait for jobs to be processes
		time.Sleep(1 * time.Second)

		// Check if the final status of the snapshotCache is the wanted
		for _, w := range want {

			snap, err := (*rec.snapshotCache).GetSnapshot(w.nodeID)
			if err != nil {
				t.Fatalf("error recovering processed cache for node %s: '%s'", w.nodeID, err)
			}
			got := snap.Resources[cache.Secret].Items
			if !reflect.DeepEqual(got, w.value) {
				t.Errorf("RunReconciler() = '%v', want '%v'", got, w.value)
			}
		}
	}

	tests := []struct {
		name string
		args args
		want []want
	}{
		{
			"Processes a job for a single nodeID and generates the expected snapshot",
			args{[]testJob{
				testJob{nodeIDs: []string{"node"}, fail: false, panic: false}},
			},
			[]want{
				want{"node", map[string]cache.Resource{"node-secret": &envoyauth.Secret{Name: "node-secret"}}},
			},
		},
		{
			"Processes a job for a several nodeIDs and generates the expected snapshots",
			args{[]testJob{
				testJob{nodeIDs: []string{"node1", "node2"}, fail: false, panic: false},
			}},
			[]want{
				want{"node1", map[string]cache.Resource{"node1-secret": &envoyauth.Secret{Name: "node1-secret"}}},
				want{"node2", map[string]cache.Resource{"node2-secret": &envoyauth.Secret{Name: "node2-secret"}}},
			},
		}, {
			"Processes a job that returns error without altering the snapshotCache",
			args{[]testJob{
				testJob{nodeIDs: []string{"node"}, fail: false, panic: false},
				testJob{nodeIDs: []string{"node"}, fail: true, panic: false},
			}},
			[]want{
				want{"node", map[string]cache.Resource{"node-secret": &envoyauth.Secret{Name: "node-secret"}}},
			},
		}, {
			"Keeps processing jobs after one job panics",
			args{[]testJob{
				testJob{nodeIDs: []string{"node"}, fail: false, panic: true},
				testJob{nodeIDs: []string{"node"}, fail: false, panic: false},
			}},
			[]want{
				want{"node", map[string]cache.Resource{"node-secret": &envoyauth.Secret{Name: "node-secret"}}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { reconcileLoopTestFn(tt.args.jobs, tt.want) })
	}

	t.Run("Stops goroutine on stopper channel closed", func(t *testing.T) {
		rec := testReconciler(nil)
		close(rec.stopper)
		rec.RunReconciler()
	})

}

func TestReconciler_runStopWatcher(t *testing.T) {

	t.Run("Closes Reconciler.Queue channel when receives the stopper signal", func(t *testing.T) {
		rec := testReconciler(nil)
		close(rec.stopper)
		go func() {
			rec.runStopWatcher()
		}()
		<-rec.Queue
	})
}

func TestReconciler_makeSnapshot(t *testing.T) {
	type args struct {
		nodeID string
	}
	tests := []struct {
		name string
		r    *Reconciler
		args args
		want cache.Snapshot
	}{
		{
			name: "Generate new snapshot with listeners",
			r: testReconciler(map[string]*nodeCaches{
				"node1": &nodeCaches{
					secrets: map[string]*envoyauth.Secret{},
					listeners: map[string]*envoyapi.Listener{
						"listener1": &envoyapi.Listener{Name: "listener1"},
					},
					clusters: map[string]*envoyapi.Cluster{},
					endpoint: map[string]*envoyapi.ClusterLoadAssignment{},
					runtime:  map[string]*envoysd.Runtime{},
				},
			}),
			args: args{nodeID: "node1"},
			want: cache.Snapshot{
				Resources: [6]cache.Resources{
					cache.Resources{Version: "1", Items: map[string]cache.Resource{}}, // cache.Enspoint
					cache.Resources{Version: "1", Items: map[string]cache.Resource{}}, // cache.Cluster
					cache.Resources{Version: "1", Items: map[string]cache.Resource{}}, // cache.Route
					cache.Resources{Version: "1", Items: map[string]cache.Resource{
						"listener1": &envoyapi.Listener{Name: "listener1"},
					}},
					cache.Resources{Version: "1", Items: map[string]cache.Resource{}}, // cache.Secret
					cache.Resources{Version: "1", Items: map[string]cache.Resource{}}, // cache.Runtime
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.makeSnapshot(tt.args.nodeID)
			got, err := (*tt.r.snapshotCache).GetSnapshot(tt.args.nodeID)
			if err != nil {
				t.Fatalf("error recovering processed cache for node %s: '%s'", tt.args.nodeID, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RunReconciler() = '%v', want '%v'", got, tt.want)
			}
		})
	}
}

// func testReconciler() *Reconciler {
// 	lg, _ := zap.NewDevelopment()
// 	logger := lg.Sugar()
// 	stopper := make(chan struct{})
// 	scache := cache.NewSnapshotCache(true, cache.IDHash{}, nil)
// 	return NewReconciler(&kubernetes.Clientset{}, "xxxx", &scache, stopper, logger)
// }
