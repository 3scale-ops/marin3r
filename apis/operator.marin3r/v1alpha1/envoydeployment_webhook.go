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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var envoydeploymentlog = logf.Log.WithName("envoydeployment-resource")

func (r *EnvoyDeployment) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-operator-marin3r-3scale-net-v1alpha1-envoydeployment,mutating=false,failurePolicy=fail,sideEffects=None,groups=operator.marin3r.3scale.net,resources=envoydeployments,verbs=create;update,versions=v1alpha1,name=envoydeployment.operator.marin3r.3scale.net,admissionReviewVersions=v1

var _ webhook.Validator = &EnvoyDeployment{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *EnvoyDeployment) ValidateCreate() (admission.Warnings, error) {
	envoydeploymentlog.V(1).Info("validate create", "name", r.Name)

	return nil, r.Validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *EnvoyDeployment) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	envoydeploymentlog.V(1).Info("validate update", "name", r.Name)

	return nil, r.Validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *EnvoyDeployment) ValidateDelete() (admission.Warnings, error) { return nil, nil }

// Validate checks that the spec of the EnvoyDeployment resource is correct
func (r *EnvoyDeployment) Validate() error {

	if r.Spec.Replicas != nil {
		if err := r.Spec.Replicas.Validate(); err != nil {
			return err
		}
	}

	if r.Spec.PodDisruptionBudget != nil {
		if err := r.Spec.PodDisruptionBudget.Validate(); err != nil {
			return err
		}
	}

	return nil
}
