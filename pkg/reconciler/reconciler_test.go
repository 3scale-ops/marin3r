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
	"reflect"
	"testing"
	"time"

	envoyauth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/roivaz/marin3r/pkg/envoy"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

func testReconciler() *Reconciler {
	lg, _ := zap.NewDevelopment()
	logger := lg.Sugar()
	stopper := make(chan struct{})
	scache := cache.NewSnapshotCache(true, cache.IDHash{}, nil)
	return NewReconciler(&kubernetes.Clientset{}, "xxxx", &scache, stopper, logger)
}

type testJob struct {
	nodeID string
}

func (job testJob) Push(queue chan ReconcileJob) { queue <- job }
func (job testJob) process(c caches, clientset *kubernetes.Clientset, namespace string, logger *zap.SugaredLogger) ([]string, error) {
	c[job.nodeID] = NewNodeCaches()
	c[job.nodeID].secrets = map[string]*envoyauth.Secret{"test-secret": envoy.NewSecret("test-secret", "xxxx", "xxxx")}
	return []string{job.nodeID}, nil
}

func TestReconciler_RunReconciler(t *testing.T) {

	t.Run("Stops goroutine on stopper channel closed", func(t *testing.T) {
		rec := testReconciler()
		close(rec.stopper)
		rec.RunReconciler()
	})

	t.Run("Processes a test job and generates snapshot", func(t *testing.T) {
		rec := testReconciler()

		go func() {
			rec.RunReconciler()
		}()

		job := testJob{nodeID: "test-node"}
		job.Push(rec.Queue)
		time.Sleep(50 * time.Millisecond)

		// Validate that the job was processed by checking that the wanted secret
		// exists in the snapshotCache
		snap, err := (*rec.snapshotCache).GetSnapshot("test-node")
		want := envoy.NewSecret("test-secret", "xxxx", "xxxx")
		if err != nil {
			t.Fatal("error recovering processed cache")
		}
		if got := snap.Resources[cache.Secret].Items["test-secret"]; !reflect.DeepEqual(got, want) {
			t.Errorf("RunReconciler() = '%v', want '%v'", snap.Resources[cache.Secret].Items["test-secret"], want)
		}
	})

	// TODO cases
	t.Run("Generates snapshot for each nodeID in the list returned by 'process'", func(t *testing.T) {})
	t.Run("Does not generate snapshot on 'process' error", func(t *testing.T) {})

}

func TestReconciler_runStopWatcher(t *testing.T) {
	tests := []struct {
		name string
		r    *Reconciler
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.runStopWatcher()
		})
	}
}

func TestReconciler_makeSnapshot(t *testing.T) {
	type args struct {
		nodeID string
	}
	tests := []struct {
		name string
		r    *Reconciler
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.makeSnapshot(tt.args.nodeID)
		})
	}
}
