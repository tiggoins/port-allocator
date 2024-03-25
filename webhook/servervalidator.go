package webhook

import (
	v1 "k8s.io/api/admission/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"

	"ingress.validator.io/scheme"
	"ingress.validator.io/sets"
)

const (
	EmptyHost    = "Host is empty"
	ConflictHost = "Host is already exist in the cluster"
)

type Validator struct {
	ghosts *sets.IngressHost
}

func NewValidator(ghosts *sets.IngressHost) *Validator {
	return &Validator{
		ghosts: ghosts,
	}
}

func (vr *Validator) ValidateIngress(ar *v1.AdmissionReview) *v1.AdmissionResponse {
	networkingIngress := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	klog.V(2).Infof("Port-allocator starts verifying the creation of %s/%s by %s", ar.Request.Namespace, ar.Request.Name, ar.Request.UserInfo.Username)

	var raw runtime.RawExtension
	switch ar.Request.Operation {
	case v1.Create, v1.Update:
		raw = ar.Request.Object
	default:
		klog.Warning("Allow the request only for Create and Update operations,and won't handle other requests.")
		return &v1.AdmissionResponse{Allowed: true}
	}

	reviewResponse := &v1.AdmissionResponse{}
	deserializer := scheme.Codecs.UniversalDeserializer()

	switch ar.Request.Resource {
	case networkingIngress:
		nIngress := networkingv1.Ingress{}
		if _, _, err := deserializer.Decode(raw.Raw, nil, &nIngress); err != nil {
			klog.Error(err)
			reviewResponse.Allowed = false
			reviewResponse.Result = &metav1.Status{
				Reason:  metav1.StatusReasonInternalError,
				Message: "Cannot decode raw.Raw into networking/v1.Ingress",
			}
			klog.V(2).Infof("Ingress-validator admit finished,Result=Internalerror,Reason=`Error happened in decode raw.Raw into Ingress`")
			return reviewResponse
		}
		for _, rule := range nIngress.Spec.Rules {
			if rule.Host == "" {
				reviewResponse.Allowed = false
				reviewResponse.Result = &metav1.Status{
					Reason:  metav1.StatusReasonForbidden,
					Message: "The Host filed in Ingress resource is empty.Please recreate it",
				}
				klog.V(2).Infof("Ingress-validator admit finished,Result=Forbidden,Reason=%s", EmptyHost)
				return reviewResponse
			}
			if vr.ghosts.Has(rule.Host) {
				reviewResponse.Allowed = false
				reviewResponse.Result = &metav1.Status{
					Reason:  metav1.StatusReasonForbidden,
					Message: "The Host field in Ingress resource already exists in the cluster.Please recreate it.",
				}
				klog.V(2).Infof("Ingress-validator admit finished,Result=Forbidden,Reason=%s", ConflictHost)
				return reviewResponse
			}
			klog.V(2).Infof("Ingress-validator admit finished,Result=Approved")
			vr.ghosts.Add(rule.Host)
		}
		return reviewResponse
	default:
		klog.Errorf("Expect resource to be %v", networkingIngress)
	}
	return reviewResponse
}
