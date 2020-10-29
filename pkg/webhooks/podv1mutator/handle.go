package podv1mutator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
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

// Handle injects an envoy container in every incoming Pod
func (a *PodMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	err := a.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if _, ok := pod.GetAnnotations()[fmt.Sprintf("%s/%s", marin3rAnnotationsDomain, paramNodeID)]; !ok {
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("Missing '%s/%s' annotation", marin3rAnnotationsDomain, paramNodeID))
	}

	// Get the patches for the envoy sidecar container
	config := envoySidecarConfig{}
	err = config.PopulateFromAnnotations(pod.GetAnnotations())
	if err != nil {
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("Error trying to load envoy container config from annotations: '%s'", err))
	}

	pod.Spec.Containers = append(pod.Spec.Containers, config.container())
	pod.Spec.Volumes = append(pod.Spec.Volumes, config.volumes()...)

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
