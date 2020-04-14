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

	"github.com/3scale/marin3r/pkg/cache"
	"github.com/3scale/marin3r/pkg/util"
	envoyauth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache"
	"go.uber.org/zap"
)

func testReconciler(rcaches cache.Cache) Reconciler {
	lg, _ := zap.NewDevelopment()
	logger := lg.Sugar()
	stopper := make(chan struct{})
	scache := xds_cache.NewSnapshotCache(true, xds_cache.IDHash{}, nil)

	var rc cache.Cache
	if rcaches != nil {
		rc = rcaches
	} else {
		rc = cache.NewCache()
	}

	return Reconciler{
		client:        &util.K8s{},
		namespace:     "namespace",
		cache:         rc,
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
func (job testJob) process(c cache.Cache, client *util.K8s, namespace string, logger *zap.SugaredLogger) ([]string, error) {

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
		c.NewNodeCache(nodeID)
		c.SetResource(
			nodeID,
			fmt.Sprintf("%s-secret", nodeID),
			cache.Secret,
			&envoyauth.Secret{Name: fmt.Sprintf("%s-secret", nodeID)},
		)
	}

	return job.nodeIDs, nil
}

func TestReconciler_RunReconciler(t *testing.T) {
	type args struct {
		jobs []ReconcileJob
	}
	type want struct {
		nodeID    string
		resources map[string]xds_cache.Resource
	}
	tests := []struct {
		name string
		rec  Reconciler
		args args
		want []want
	}{
		{
			"Processes a job for a single nodeID and generates the expected snapshot",
			testReconciler(nil),
			args{[]ReconcileJob{
				testJob{nodeIDs: []string{"node"}, fail: false, panic: false},
			},
			},
			[]want{
				{"node", map[string]xds_cache.Resource{"node-secret": &envoyauth.Secret{Name: "node-secret"}}},
			},
		}, {
			"Processes a job for a several nodeIDs and generates the expected snapshots",
			testReconciler(nil),
			args{[]ReconcileJob{
				testJob{nodeIDs: []string{"node1", "node2"}, fail: false, panic: false},
			}},
			[]want{
				{"node1", map[string]xds_cache.Resource{"node1-secret": &envoyauth.Secret{Name: "node1-secret"}}},
				{"node2", map[string]xds_cache.Resource{"node2-secret": &envoyauth.Secret{Name: "node2-secret"}}},
			},
		}, {
			"Processes a job that returns error without altering the snapshotCache",
			testReconciler(nil),
			args{[]ReconcileJob{
				testJob{nodeIDs: []string{"node"}, fail: false, panic: false},
				testJob{nodeIDs: []string{"node"}, fail: true, panic: false},
			}},
			[]want{
				{"node", map[string]xds_cache.Resource{"node-secret": &envoyauth.Secret{Name: "node-secret"}}},
			},
		}, {
			"Keeps processing jobs after one job panics",
			testReconciler(nil),
			args{[]ReconcileJob{
				testJob{nodeIDs: []string{"node"}, fail: false, panic: true},
				testJob{nodeIDs: []string{"node"}, fail: false, panic: false},
			}},
			[]want{
				{"node", map[string]xds_cache.Resource{"node-secret": &envoyauth.Secret{Name: "node-secret"}}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			go func(reconciler Reconciler) {
				reconciler.RunReconciler()
			}(tt.rec)

			// Push jobs to the queue
			for _, job := range tt.args.jobs {
				job.Push(tt.rec.Queue)
			}

			// Wait for jobs to be processes
			time.Sleep(100 * time.Millisecond)

			for _, w := range tt.want {
				snap, err := (*tt.rec.snapshotCache).GetSnapshot(w.nodeID)
				if err != nil {
					t.Fatalf("error recovering processed cache for node %s: '%s'", w.nodeID, err)
				}
				got := snap.Resources[cache.Secret].Items
				if !reflect.DeepEqual(got, w.resources) {
					t.Errorf("RunReconciler() = '%v', want '%v'", got, w.resources)
				}
			}

		})
	}
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
