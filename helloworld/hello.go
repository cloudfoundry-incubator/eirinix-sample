package hello

import (
	"context"
	"errors"
	"net/http"

	eirinix "github.com/SUSE/eirinix"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Extension changes pod definitions
type Extension struct{ Logger *zap.SugaredLogger }

// New returns the persi extension
func New() eirinix.Extension {
	return &Extension{}
}

// Handle manages volume claims for ExtendedStatefulSet pods
func (ext *Extension) Handle(ctx context.Context, eiriniManager eirinix.Manager, pod *corev1.Pod, req admission.Request) admission.Response {

	if pod == nil {
		return admission.Errored(http.StatusBadRequest, errors.New("No pod could be decoded from the request"))
	}

	log := eiriniManager.GetLogger().Named("Hello world!")
	ext.Logger = log

	podCopy := pod.DeepCopy()
	log.Infof("Hello from my Eirini extension! Eirini application POD: %s (%s)", podCopy.Name, podCopy.Namespace)
	for i := range podCopy.Spec.Containers {
		c := &podCopy.Spec.Containers[i]
		c.Env = append(c.Env, corev1.EnvVar{Name: "STICKY_MESSAGE", Value: "Eirinix is awesome!"})
	}
	return eiriniManager.PatchFromPod(req, podCopy)
}
