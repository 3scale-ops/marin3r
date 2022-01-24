package podv1mutator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/3scale-ops/marin3r/pkg/envoy/container/defaults"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	// try to load admissionv1 into scheme
	_ "k8s.io/api/admission/v1"
)

const (
	// MutatePath is the path where the webhook server listens
	// for admission requests
	MutatePath string = "/pod-v1-mutate"
)

// PodMutator injects envoy containers into Pods
type PodMutator struct {
	Client  client.Client
	decoder *admission.Decoder
}

//+kubebuilder:webhook:path=/pod-v1-mutate,mutating=true,failurePolicy=fail,sideEffects=None,groups=core,resources=pods,verbs=create,versions=v1,name=sidecar-injector.marin3r.3scale.net,admissionReviewVersions=v1

// Handle injects an envoy container in every incoming Pod
func (a *PodMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	err := a.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if _, ok := lookupMarin3rAnnotation(paramNodeID, pod.GetAnnotations()); !ok {
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("missing '%s/%s' annotation", marin3rAnnotationsDomain, paramNodeID))
	}

	// Get the patches for the envoy sidecar container
	config := envoySidecarConfig{}
	err = config.PopulateFromAnnotations(context.Background(), a.Client, req.Namespace, pod.GetAnnotations())
	if err != nil {
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("error trying to build envoy container config: '%s'", err))
	}

	pod.Spec.InitContainers = append(pod.Spec.InitContainers, config.initContainers()...)
	pod.Spec.Containers = append(pod.Spec.Containers, config.containers()...)
	pod.Spec.Volumes = append(pod.Spec.Volumes, config.volumes()...)

	if isShtdnMgrEnabled(pod.GetAnnotations()) {
		// Increase the TerminationGracePeriodSeconds parameter if shutdown
		// manager is enabled
		pod.Spec.TerminationGracePeriodSeconds = pointer.Int64Ptr(defaults.GracefulShutdownTimeoutSeconds)
		// Add extra container lifecycle hooks
		containers, err := config.addExtraLifecycleHooks(pod.Spec.Containers, pod.GetAnnotations())
		if err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		pod.Spec.Containers = containers
	}

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// podMutator implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (a *PodMutator) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}
