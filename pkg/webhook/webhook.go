package webhook

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"go.uber.org/zap"
	admissionv1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

const (
	// TODO: make the annotation to look for configurable
	annotation = "marin3r.3scale.net/node-id"
	// adsEndpoint = "marin3r.3scale.net/ads"
	envoyContainerName = "envoy-sidecar"
	envoyConfigVolume  = "envoy-sidecar-bootstrap"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()

	// (https://github.com/kubernetes/kubernetes/issues/57982)
	defaulter = runtime.ObjectDefaulter(runtimeScheme)
)

func init() {
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionregistrationv1.AddToScheme(runtimeScheme)
	_ = admissionv1.AddToScheme(runtimeScheme)
	_ = v1.AddToScheme(runtimeScheme)
}

type envoySidecar struct {
	containerTemplate string
	envoyConfig       string
}

type EnvoyInjector struct {
	logger *zap.SugaredLogger
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func NewEnvoyInjector(logger *zap.SugaredLogger) *EnvoyInjector {
	return &EnvoyInjector{
		logger: logger,
	}
}

func (ei *EnvoyInjector) Mutate(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		msg := fmt.Sprintf("Error reading AdmissionReview body: %v", err)
		ei.logger.Errorf(msg)
		http.Error(w, msg, http.StatusBadRequest)
	}

	var admissionResponse *admissionv1.AdmissionResponse
	ar := admissionv1.AdmissionReview{}
	if _, _, err = deserializer.Decode(body, nil, &ar); err != nil {
		admissionResponse = ei.admissionResponseError("An error ocurred deserializing AdmissionReview request: %v", err)
	} else {
		admissionResponse = ei.mutatePod(&ar)
	}

	if admissionResponse != nil {
		ar.Response = admissionResponse
		if ar.Request != nil {
			ar.Response.UID = ar.Request.UID
		}
	}

	resp, err := json.Marshal(ar)
	if err != nil {
		ei.logger.Errorf("Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}

	ei.logger.Debugf("Ready to write response ...")
	if _, err := w.Write(resp); err != nil {
		ei.logger.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}

// mutatePod applies transformations on the pod. Returns:
// - The resulting container
// - A bool value indicating if transformations were required
// - The error
func (ei *EnvoyInjector) mutatePod(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {

	req := ar.Request
	pod := corev1.Pod{}
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		return ei.admissionResponseError("Could not unmarshal raw object: %v", err)
	}

	ei.logger.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, pod.Name, req.UID, req.Operation, req.UserInfo)

	// Get the patch for the envoy sidecar container
	cPatch, err := ei.patchContainer(pod.Spec.Containers)
	if err != nil {
		return ei.admissionResponseError("Error injecting envoy container: %v", err)
	}

	// Get the patch for the envoy sidecar volumes
	// TODO

	var patch []patchOperation
	patch = append(patch, cPatch)

	// Marshal the list of patches and return it
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return ei.admissionResponseError("Error marshalling patches: %v", err)
	}
	ei.logger.Debugf("AdmissionResponse: patch=%v\n", string(patchBytes))
	return &admissionv1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *admissionv1.PatchType {
			pt := admissionv1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

func (ei *EnvoyInjector) patchContainer(podContainers []corev1.Container) (patchOperation, error) {
	// Get the envoy sidecar container object
	envoyContainer, err := ei.getEnvoyContainer()
	if err != nil {
		return patchOperation{}, err
	}

	path, op := ei.getPatchPath(podContainers, envoyContainer)
	if path == "" {
		return patchOperation{}, nil
	}

	return patchOperation{
			Op:    op,
			Path:  path,
			Value: envoyContainer},
		nil
}

// containerNeedsUpdate checks if the envoy sidecar container is already present in the pod or if it needs patching.
// The function returns the path where the container is (or will be) located if patch is required, or the empty string
// if it is not.
func (ei *EnvoyInjector) getPatchPath(podContainers []corev1.Container, envoyContainer corev1.Container) (string, string) {
	// Check if the pod already has the envoy sidecar container
	for index, container := range podContainers {
		if container.Name == envoyContainerName {
			// Check if the envoy sidecar container requires changes
			if apiequality.Semantic.DeepEqual(container, envoyContainer) {
				ei.logger.Debugf("Envoy sidecer container founr at position %v requires update", index)
				return fmt.Sprintf("/spec/containers/%v", index), "replace"

			} else {
				return "", ""
			}
		}
	}
	// The envoy container was not found, so we will append it at
	// the end of the containers array
	// '/-' refers to the end of an array in jsonPatch
	return "/spec/containers/-", "add"
}

func (ei *EnvoyInjector) getEnvoyContainer() (corev1.Container, error) {

	envoyContainer := corev1.Container{
		Name:  "envoy-sidecar",
		Image: "envoyproxy/envoy:v1.13.1",
		// Command: []string{"envoy"},
		// Args: []string{
		// 	"-c",
		// 	"/etc/envoy/bootstrap/config.yaml",
		// 	"--service-cluster",
		// 	"envoy1",
		// 	"--service-node",
		// 	"envoy1",
		// },
		Ports: []corev1.ContainerPort{
			corev1.ContainerPort{
				Name:          "http",
				ContainerPort: 8080,
			},
			corev1.ContainerPort{
				Name:          "https",
				ContainerPort: 8443,
			},
		},
	}
	return envoyContainer, nil
}

func (ei *EnvoyInjector) admissionResponseError(template string, err error) *admissionv1.AdmissionResponse {
	msg := fmt.Sprintf(template, err)
	ei.logger.Errorf(msg)
	return &admissionv1.AdmissionResponse{
		Result: &metav1.Status{
			Message: msg,
		},
	}
}
