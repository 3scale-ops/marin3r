package stats

import (
	"context"
	"reflect"
	"testing"
	"time"

	kv "github.com/patrickmn/go-cache"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestStats_RunGC(t *testing.T) {
	type args struct {
		client    kubernetes.Interface
		namespace string
		stopCh    <-chan struct{}
	}
	tests := []struct {
		name       string
		cacheItems map[string]kv.Item
		args       args
		execute    func(kubernetes.Interface, string)
		wantItems  map[string]kv.Item
	}{
		{
			name: "deletes stats for the deleted pod",
			cacheItems: map[string]kv.Item{
				"node:" + "endpoint" + ":*:pod-xxxx:request_counter": {Object: int64(5), Expiration: int64(0)},
				"node:" + "endpoint" + ":xxxx:pod-xxxx:ack_counter":  {Object: int64(1), Expiration: int64(0)},
				"node:" + "endpoint" + ":xxxx:pod-xxxx:nack_counter": {Object: int64(13), Expiration: int64(0)},
			},
			args: args{
				client:    fake.NewSimpleClientset(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-xxxx", Namespace: "ns"}}),
				namespace: "ns",
				stopCh:    make(<-chan struct{}),
			},
			execute: func(c kubernetes.Interface, ns string) {
				time.Sleep(time.Millisecond * 10)
				c.CoreV1().Pods(ns).Delete(context.TODO(), "pod-xxxx", *metav1.NewDeleteOptions(0))
				time.Sleep(time.Millisecond * 100)
			},
			wantItems: map[string]kv.Item{},
		},
		{
			name: "deletes stats for the deleted pod",
			cacheItems: map[string]kv.Item{
				"node:" + "endpoint" + ":*:pod-xxxx:request_counter": {Object: int64(5), Expiration: int64(0)},
				"node:" + "endpoint" + ":xxxx:pod-aaaa:nack_counter": {Object: int64(13), Expiration: int64(0)},
			},
			args: args{
				client:    fake.NewSimpleClientset(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-aaaa", Namespace: "ns"}}),
				namespace: "ns",
				stopCh:    make(<-chan struct{}),
			},
			execute: func(c kubernetes.Interface, ns string) {
				time.Sleep(time.Millisecond * 10)
				c.CoreV1().Pods(ns).Delete(context.TODO(), "pod-aaaa", *metav1.NewDeleteOptions(0))
				time.Sleep(time.Millisecond * 100)
			},
			wantItems: map[string]kv.Item{
				"node:" + "endpoint" + ":*:pod-xxxx:request_counter": {Object: int64(5), Expiration: int64(0)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewWithItems(tt.cacheItems, time.Now())
			s.RunGC(tt.args.client, tt.args.namespace, tt.args.stopCh)
			tt.execute(tt.args.client, tt.args.namespace)
			if got := s.DumpAll(); !reflect.DeepEqual(got, tt.wantItems) {
				t.Errorf("Stats.RunGC() got diff: %v, want %v", got, tt.wantItems)
			}
		})
	}
}
