package webhook

import (
	"crypto/tls"
	"k8s.io/klog/v2"
)

func (s *Server) configTLS() *tls.Config {
	cert, err := tls.LoadX509KeyPair(s.certfile, s.keyfile)
	if err != nil {
		klog.Fatal(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
}
