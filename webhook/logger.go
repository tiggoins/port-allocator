package webhook

import (
	"k8s.io/klog/v2"
	"strings"
)

type httpLogger struct{}

func (*httpLogger) Write(b []byte) (n int, err error) {
	m := string(b)
	if strings.HasPrefix(m, "http: TLS handshake error") && strings.HasSuffix(m, ": EOF\n") {
		// decrease the log level of TLS error
		klog.V(10).Info(m)
	}
	return len(b), nil
}
