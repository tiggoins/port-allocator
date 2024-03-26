package webhook

import (
	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

type Mutator struct{}

func NewMutator() *Mutator {
	return &Mutator{}
}

func (mu *Mutator) mutateService(ar *v1.AdmissionReview) *v1.AdmissionResponse {
	klog.V(2).Infof("Port-allocator starts verifying the creation of %s/%s by %s", ar.Request.Namespace, ar.Request.Name, ar.Request.UserInfo.Username)

	reviewResponse := &v1.AdmissionResponse{Allowed: true}
	// skip if operation is not create and update
	if ar.Request.Operation != v1.Create && ar.Request.Operation != v1.Update {
		return reviewResponse
	}

	service := corev1.Service{}
	deserializer := Codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(ar.Request.Object.Raw, nil, &service); err != nil {
		klog.Error(err)
		reviewResponse.Allowed = false
		reviewResponse.Result = &metav1.Status{
			Reason:  metav1.StatusReasonInternalError,
			Message: "Cannot decode object into v1.Service",
		}
		klog.V(2).Infof("Error happened in decode object into Service")
		return reviewResponse
	}

	// permit if serive.type is not nodePort
	if service.Spec.Type != corev1.ServiceTypeNodePort {
		klog.V(2).Infof("Service %s/%s is not nodeport type,will allow the request.", ar.Request.Namespace, ar.Request.Name)
		return reviewResponse
	}

	for i := 0; i < len(service.Spec.Ports); i++ {
		// 添加的nodeport是否在范围内的逻辑，不在则强制修改到范围内
		service.Spec.Ports[i].NodePort = 30000
	}

	return reviewResponse
}
