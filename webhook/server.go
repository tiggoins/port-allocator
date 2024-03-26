package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/spf13/pflag"
	v1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
)

// admitv1beta1Func handles a v1 admission
type admitv1Func func(*v1.AdmissionReview) *v1.AdmissionResponse

type Server struct {
	certfile string
	keyfile  string
	port     int
	ctx      context.Context
	admit    admitv1Func
	server   *http.Server
}

func NewServerFlagSet() *pflag.FlagSet {
	serverFlags := pflag.NewFlagSet("server", pflag.ExitOnError)
	serverFlags.String("tls-cert-file", "", "Path to the certificate file (MUST specify)")
	serverFlags.String("tls-key-file", "", "Path to the key file (MUST Specify)")
	serverFlags.IntP("port", "p", 443, "Port to listen on (default to 443)")

	return serverFlags
}

func NewServer(ctx context.Context) *Server {
	var errorList []error
	s := new(Server)
	// pflag.CommandLine.StringVar(&s.certfile, "tls-cert-file", "", "Path to the certificate file (MUST specify)")
	certfile, err := pflag.CommandLine.GetString("tls-cert-file")
	if err != nil {
		errorList = append(errorList, err)
	}
	// pflag.CommandLine.StringVar(&s.keyfile, "tls-key-file", "", "Path to the key file (MUST Specify)")
	keyfile, err := pflag.CommandLine.GetString("tls-key-file")
	if err != nil {
		errorList = append(errorList, err)
	}
	// pflag.CommandLine.IntVarP(&s.port, "port", "p", 443, "Port to listen on (default to 443)")
	port, err := pflag.CommandLine.GetInt("port")
	if err != nil {
		errorList = append(errorList, err)
	}

	if len(errorList) != 0 {
		klog.Fatalln(errorList)
	}

	s.certfile = certfile
	s.keyfile = keyfile
	s.port = port
	s.ctx = ctx
	s.admit = NewMutator().mutateService

	return s
}

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if data, err := io.ReadAll(r.Body); err == nil {
		body = data
	} else {
		klog.V(2).ErrorS(err, "Error happened when reading request body")
		return
	}

	if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
		klog.Errorf("contentType=%s, expect application/json", contentType)
		return
	}

	klog.V(5).Info(fmt.Sprintf("handling request: %s", body))

	deserializer := Codecs.UniversalDeserializer()
	obj, gvk, err := deserializer.Decode(body, nil, nil)
	if err != nil {
		msg := fmt.Sprintf("Request decode error: %v", err)
		klog.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	var responseObj runtime.Object
	switch *gvk {
	case v1.SchemeGroupVersion.WithKind("AdmissionReview"):
		requestedAdmissionReview, ok := obj.(*v1.AdmissionReview)
		if !ok {
			klog.Errorf("Expected v1.AdmissionReview but got: %T", obj)
			return
		}
		responseAdmissionReview := &v1.AdmissionReview{}
		responseAdmissionReview.SetGroupVersionKind(*gvk)
		responseAdmissionReview.Response = s.admit(requestedAdmissionReview)
		responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
		responseObj = responseAdmissionReview
	default:
		msg := fmt.Sprintf("Unsupported group version kind: %v", gvk)
		klog.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	klog.V(5).Info(fmt.Sprintf("sending response: %v", responseObj))
	respBytes, err := json.Marshal(responseObj)
	if err != nil {
		klog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(respBytes); err != nil {
		klog.Error(err)
	}
}

func (s *Server) Start() {
	http.HandleFunc("/port-allocator", s.serve)
	http.HandleFunc("/readyz", func(w http.ResponseWriter, req *http.Request) { w.Write([]byte("ok")) })

	logger := log.New(new(httpLogger), "", 0)
	server := &http.Server{
		Addr:      fmt.Sprintf(":%d", s.port),
		TLSConfig: s.configTLS(),
		ErrorLog:  logger,
	}
	s.server = server

	klog.V(2).Infof("Staring namespaced-based nodeport allocator，listening on port %d", s.port)
	if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
		klog.Fatalln(err)
	}
}

func (s *Server) Shutdown() error {
	klog.Info("Received interrupt signal, shutting down server gracefully...")
	if err := s.server.Shutdown(s.ctx); err != nil {
		return err
	}
	return nil
}

type httpLogger struct{}

func (*httpLogger) Write(b []byte) (n int, err error) {
	m := string(b)
	if strings.HasPrefix(m, "http: TLS handshake error") && strings.HasSuffix(m, ": EOF\n") {
		// decrease the log level of TLS error
		klog.V(10).Info(m)
	}
	return len(b), nil
}
