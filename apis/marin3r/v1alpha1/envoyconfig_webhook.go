/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_resources "github.com/3scale-ops/marin3r/pkg/envoy/resources"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var envoyconfiglog = logf.Log.WithName("envoyconfig-resource")

func (r *EnvoyConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-marin3r-3scale-net-v1alpha1-envoyconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=marin3r.3scale.net,resources=envoyconfigs,verbs=create;update,versions=v1alpha1,name=envoyconfig.marin3r.3scale.net,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &EnvoyConfig{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *EnvoyConfig) ValidateCreate() error {
	envoyconfiglog.Info("validate create", "name", r.Name)
	if err := r.ValidateResources(); err != nil {
		return err
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *EnvoyConfig) ValidateUpdate(old runtime.Object) error {
	envoyconfiglog.Info("validate update", "name", r.Name)
	if err := r.ValidateResources(); err != nil {
		return err
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *EnvoyConfig) ValidateDelete() error { return nil }

// Validate Envoy resources against schema
func (r *EnvoyConfig) ValidateResources() error {
	errList := []error{}

	for _, endpoint := range r.Spec.EnvoyResources.Endpoints {
		if err := envoy_resources.Validate(endpoint.Value, r.GetSerialization(), r.GetEnvoyAPIVersion(), envoy.Endpoint); err != nil {
			errList = append(errList, err)
		}
	}

	for _, cluster := range r.Spec.EnvoyResources.Clusters {
		if err := envoy_resources.Validate(cluster.Value, r.GetSerialization(), r.GetEnvoyAPIVersion(), envoy.Cluster); err != nil {
			errList = append(errList, err)
		}
	}

	for _, route := range r.Spec.EnvoyResources.Routes {
		if err := envoy_resources.Validate(route.Value, r.GetSerialization(), r.GetEnvoyAPIVersion(), envoy.Route); err != nil {
			errList = append(errList, err)
		}
	}

	for _, listener := range r.Spec.EnvoyResources.Listeners {
		if err := envoy_resources.Validate(listener.Value, r.GetSerialization(), r.GetEnvoyAPIVersion(), envoy.Listener); err != nil {
			errList = append(errList, err)
		}
	}

	for _, runtime := range r.Spec.EnvoyResources.Runtimes {
		if err := envoy_resources.Validate(runtime.Value, r.GetSerialization(), r.GetEnvoyAPIVersion(), envoy.Runtime); err != nil {
			errList = append(errList, err)
		}
	}

	for _, secret := range r.Spec.EnvoyResources.Secrets {
		if err := secret.Validate(r.GetNamespace()); err != nil {
			errList = append(errList, err)
		}
	}

	if len(errList) > 0 {
		return NewValidationError(errList)
	}
	return nil
}
