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
	"fmt"

	"github.com/3scale-ops/basereconciler/util"
	"github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_resources "github.com/3scale-ops/marin3r/pkg/envoy/resources"
	envoy_serializer "github.com/3scale-ops/marin3r/pkg/envoy/serializer"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var validationlog = logf.Log.WithName("v1alpha1 validation")

func (r *EnvoyConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-marin3r-3scale-net-v1alpha1-envoyconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=marin3r.3scale.net,resources=envoyconfigs,verbs=create;update,versions=v1alpha1,name=envoyconfig.marin3r.3scale.net-v1alpha1,admissionReviewVersions=v1

var _ webhook.Validator = &EnvoyConfig{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *EnvoyConfig) ValidateCreate() (admission.Warnings, error) {
	validationlog.Info("ValidateCreate", "type", "EnvoyConfig", "resource", util.ObjectKey(r).String())
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *EnvoyConfig) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	validationlog.Info("validateUpdate", "type", "EnvoyConfig", "resource", util.ObjectKey(r).String())
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *EnvoyConfig) ValidateDelete() (admission.Warnings, error) { return nil, nil }

// Validates the EnvoyConfig resource
func (r *EnvoyConfig) Validate() error {
	if (r.Spec.EnvoyResources == nil && r.Spec.Resources == nil) || (r.Spec.EnvoyResources != nil && r.Spec.Resources != nil) {
		return fmt.Errorf("one and only one of 'spec.EnvoyResources', 'spec.Resources' must be set")
	}

	if r.Spec.EnvoyResources != nil {
		if err := r.ValidateEnvoyResources(); err != nil {
			return err
		}

	} else {
		if err := r.ValidateResources(); err != nil {
			return err
		}
	}

	return nil
}

// Validate Envoy Resources against schema
func (r *EnvoyConfig) ValidateResources() error {
	errList := []error{}

	for _, res := range r.Spec.Resources {

		switch res.Type {

		case envoy.Secret:
			if res.GenerateFromTlsSecret == nil && res.GenerateFromOpaqueSecret == nil {
				errList = append(errList, fmt.Errorf("one of 'generateFromTlsSecret', 'generateFromOpaqueSecret' must be set for type '%s'", envoy.Secret))
			}
			if res.Value != nil {
				errList = append(errList, fmt.Errorf("'value' cannot be used for type '%s'", envoy.Secret))
			}
			if res.GenerateFromEndpointSlices != nil {
				errList = append(errList, fmt.Errorf("'generateFromEndpointSlice' can only be used type '%s'", envoy.Endpoint))
			}

		case envoy.Endpoint:
			if res.GenerateFromEndpointSlices != nil && res.Value != nil {
				errList = append(errList, fmt.Errorf("only one of 'generateFromEndpointSlice', 'value' allowed for type '%s'", envoy.Secret))
			}
			if res.GenerateFromEndpointSlices == nil && res.Value == nil {
				errList = append(errList, fmt.Errorf("one of 'generateFromEndpointSlice', 'value' must be set for type '%s'", envoy.Secret))
			}
			if res.Value != nil {
				if err := envoy_resources.Validate(string(res.Value.Raw), envoy_serializer.JSON, r.GetEnvoyAPIVersion(), envoy.Type(res.Type)); err != nil {
					errList = append(errList, err)
				}
			}
			if res.GenerateFromTlsSecret != nil {
				errList = append(errList, fmt.Errorf("'generateFromTlsSecret' can only be used type '%s'", envoy.Secret))
			}
			if res.Blueprint != nil {
				errList = append(errList, fmt.Errorf("'blueprint' can only be used type '%s'", envoy.Secret))
			}

		default:
			if res.GenerateFromEndpointSlices != nil {
				errList = append(errList, fmt.Errorf("'generateFromEndpointSlice' can only be used type '%s'", envoy.Endpoint))
			}
			if res.GenerateFromTlsSecret != nil {
				errList = append(errList, fmt.Errorf("'generateFromTlsSecret' can only be used type '%s'", envoy.Secret))
			}
			if res.Blueprint != nil {
				errList = append(errList, fmt.Errorf("'blueprint' cannot be empty for type '%s'", envoy.Secret))
			}
			if res.Value != nil {
				if err := envoy_resources.Validate(string(res.Value.Raw), envoy_serializer.JSON, r.GetEnvoyAPIVersion(), envoy.Type(res.Type)); err != nil {
					errList = append(errList, err)
				}
			} else {
				errList = append(errList, fmt.Errorf("'value' cannot be empty for type '%s'", res.Type))
			}
		}

	}

	if len(errList) > 0 {
		return NewMultiError(errList)
	}
	return nil
}

// Validate EnvoyResources against schema
func (r *EnvoyConfig) ValidateEnvoyResources() error {
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

	for _, route := range r.Spec.EnvoyResources.ScopedRoutes {
		if err := envoy_resources.Validate(route.Value, r.GetSerialization(), r.GetEnvoyAPIVersion(), envoy.ScopedRoute); err != nil {
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
		return NewMultiError(errList)
	}
	return nil
}
