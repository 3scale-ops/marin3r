package reconcilers

import (
	"context"
	"reflect"
	"testing"

	"github.com/3scale/marin3r/pkg/common"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestNewDeploymentReconciler(t *testing.T) {
	type args struct {
		ctx    context.Context
		logger logr.Logger
		client client.Client
		scheme *runtime.Scheme
		owner  metav1.Object
	}
	tests := []struct {
		name string
		args args
		want DeploymentReconciler
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewDeploymentReconciler(tt.args.ctx, tt.args.logger, tt.args.client, tt.args.scheme, tt.args.owner); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDeploymentReconciler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeploymentReconciler_Reconcile(t *testing.T) {

	var generatorFn DeploymentGeneratorFn = func() *appsv1.Deployment {
		return &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "aaaa",
				Namespace: "aaaa",
				Labels: map[string]string{
					"key1": "value1",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: pointer.Int32Ptr(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"key1": "value1"},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						CreationTimestamp: metav1.Time{},
						Labels:            map[string]string{"key1": "value1"},
					},
					Spec: corev1.PodSpec{},
				},
			},
		}
	}

	t.Run("Creates a new Deployment", func(t *testing.T) {
		r := DeploymentReconciler{
			ctx:    context.TODO(),
			logger: logf.Log.WithName("test"),
			client: fake.NewFakeClient(&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "aaaa",
					Namespace: "aaaa",
				},
				Spec: appsv1.DeploymentSpec{},
			}),
			scheme: scheme.Scheme,
			owner:  &appsv1.Deployment{}, // this is irrelevant for this tests
		}

		result, err := r.Reconcile(
			types.NamespacedName{Name: "aaaa", Namespace: "aaaa"},
			generatorFn,
		)
		if err != nil {
			t.Errorf("DeploymentReconciler.Reconcile() error = %v", err)
			return
		}
		wantResult := reconcile.Result{}
		if result != wantResult {
			t.Errorf("DeploymentReconciler.Reconcile() bad result. Got %v, wanted %v", result, wantResult)
			return
		}
		gotDep := &appsv1.Deployment{}
		if err := r.client.Get(context.TODO(), types.NamespacedName{Name: "aaaa", Namespace: "aaaa"}, gotDep); err != nil {
			t.Errorf("DeploymentReconciler.Reconcile() Deployment does not exist")
			return
		}

		if *gotDep.Spec.Replicas != *generatorFn().Spec.Replicas {
			t.Errorf("DeploymentReconciler.Reconcile() spec does not match desired one")
		}
	})

	t.Run("Updates a Deployment", func(t *testing.T) {
		r := DeploymentReconciler{
			ctx:    context.TODO(),
			logger: logf.Log.WithName("test"),
			client: fake.NewFakeClient(),
			scheme: scheme.Scheme,
			owner:  &appsv1.Deployment{}, // this is irrelevant for this tests
		}

		result, err := r.Reconcile(
			types.NamespacedName{Name: "aaaa", Namespace: "aaaa"},
			generatorFn,
		)
		if err != nil {
			t.Errorf("DeploymentReconciler.Reconcile() error = %v", err)
			return
		}
		wantResult := reconcile.Result{}
		if result != wantResult {
			t.Errorf("DeploymentReconciler.Reconcile() bad rsult. Got %v, wanted %v", result, wantResult)
			return
		}
		gotDep := &appsv1.Deployment{}
		if err := r.client.Get(context.TODO(), types.NamespacedName{Name: "aaaa", Namespace: "aaaa"}, gotDep); err != nil {
			t.Errorf("DeploymentReconciler.Reconcile() Deployment was not created")
		}
	})

}

func TestDeploymentReconciler_reconcileDeployment(t *testing.T) {
	type args struct {
		existentObj common.KubernetesObject
		desiredObj  common.KubernetesObject
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "Labels match desired labels after reconcile",
			args: args{
				existentObj: &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aaaa",
						Namespace: "aaaa",
						Labels: map[string]string{
							"key1": "value1",
						},
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: pointer.Int32Ptr(1),
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"key1": "value1"},
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								CreationTimestamp: metav1.Time{},
								Labels:            map[string]string{"key1": "value1"},
							},
							Spec: corev1.PodSpec{},
						},
					},
				},
				desiredObj: &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aaaa",
						Namespace: "aaaa",
						Labels: map[string]string{
							"key1": "value1",
							"key2": "value2",
						},
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: pointer.Int32Ptr(1),
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"key1": "value1"},
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								CreationTimestamp: metav1.Time{},
								Labels:            map[string]string{"key1": "value1"},
							},
							Spec: corev1.PodSpec{},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Spec matches desired spec after reconcile",
			args: args{
				existentObj: &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aaaa",
						Namespace: "aaaa",
						Labels: map[string]string{
							"key1": "value1",
						},
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: pointer.Int32Ptr(1),
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"key1": "value1"},
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								CreationTimestamp: metav1.Time{},
								Labels:            map[string]string{"key1": "value1"},
							},
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									corev1.Container{
										Name: "container1",
										Resources: corev1.ResourceRequirements{
											Requests: corev1.ResourceList{
												v1.ResourceCPU:    resource.MustParse("1000m"),
												v1.ResourceMemory: resource.MustParse("200Mi"),
											},
											Limits: corev1.ResourceList{
												v1.ResourceCPU:    resource.MustParse("2000m"),
												v1.ResourceMemory: resource.MustParse("400Mi"),
											},
										},
									},
								},
							},
						},
					},
				},
				desiredObj: &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aaaa",
						Namespace: "aaaa",
						Labels: map[string]string{
							"key1": "value1",
						},
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: pointer.Int32Ptr(2),
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"key1": "value1"},
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								CreationTimestamp: metav1.Time{},
								Labels:            map[string]string{"key1": "value1"},
							},
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									corev1.Container{
										Name: "container2",
										Resources: corev1.ResourceRequirements{
											Requests: corev1.ResourceList{
												v1.ResourceCPU:    resource.MustParse("2000m"),
												v1.ResourceMemory: resource.MustParse("600Mi"),
											},
											Limits: corev1.ResourceList{
												v1.ResourceCPU:    resource.MustParse("4000m"),
												v1.ResourceMemory: resource.MustParse("800Mi"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "No reconciliation performed when Resources are semantically equal",
			args: args{
				existentObj: &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aaaa",
						Namespace: "aaaa",
					},
					Spec: appsv1.DeploymentSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									corev1.Container{
										Name: "container1",
										Resources: corev1.ResourceRequirements{
											Requests: corev1.ResourceList{
												v1.ResourceCPU:    resource.MustParse("1000m"),
												v1.ResourceMemory: resource.MustParse("2048Mi"),
											},
											Limits: corev1.ResourceList{
												v1.ResourceCPU:    resource.MustParse("2000m"),
												v1.ResourceMemory: resource.MustParse("4096Mi"),
											},
										},
									},
								},
							},
						},
					},
				},
				desiredObj: &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aaaa",
						Namespace: "aaaa",
					},
					Spec: appsv1.DeploymentSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									corev1.Container{
										Name: "container1",
										Resources: corev1.ResourceRequirements{
											Requests: corev1.ResourceList{
												v1.ResourceCPU:    resource.MustParse("1"),
												v1.ResourceMemory: resource.MustParse("2Gi"),
											},
											Limits: corev1.ResourceList{
												v1.ResourceCPU:    resource.MustParse("2"),
												v1.ResourceMemory: resource.MustParse("4Gi"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "False is returned when no changes required",
			args: args{
				existentObj: &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aaaa",
						Namespace: "aaaa",
						Labels: map[string]string{
							"key1": "value1",
						},
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: pointer.Int32Ptr(1),
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"key1": "value1"},
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								CreationTimestamp: metav1.Time{},
								Labels:            map[string]string{"key1": "value1"},
							},
							Spec: corev1.PodSpec{},
						},
					},
				},
				desiredObj: &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aaaa",
						Namespace: "aaaa",
						Labels: map[string]string{
							"key1": "value1",
						},
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: pointer.Int32Ptr(1),
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"key1": "value1"},
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								CreationTimestamp: metav1.Time{},
								Labels:            map[string]string{"key1": "value1"},
							},
							Spec: corev1.PodSpec{},
						},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			existent := tt.args.existentObj.(*appsv1.Deployment)
			desired := tt.args.desiredObj.(*appsv1.Deployment)

			r := DeploymentReconciler{
				ctx:    context.TODO(),
				logger: logf.Log.WithName("test"),
				client: fake.NewFakeClient(existent),
				scheme: scheme.Scheme,
				owner:  &appsv1.Deployment{}, // this is irrelevant for this tests
			}

			got, err := r.reconcileDeployment(tt.args.existentObj, tt.args.desiredObj)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeploymentReconciler.reconcileDeployment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DeploymentReconciler.reconcileDeployment() = %v, want %v", got, tt.want)
				return
			}

			// Check labels match
			if !equality.Semantic.DeepEqual(existent.GetLabels(), desired.GetLabels()) {
				t.Errorf("DeploymentReconciler.reconcileDeployment() ObjectMeta.Labels don't match. Got %v, want %v", existent.GetLabels(), desired.GetLabels())
			}
			// Check Spec matches
			if !equality.Semantic.DeepEqual(existent.Spec, desired.Spec) {
				t.Errorf("DeploymentReconciler.reconcileDeployment() Spec don't match. Got %v, want %v", existent.Spec, desired.Spec)
			}
		})
	}
}
