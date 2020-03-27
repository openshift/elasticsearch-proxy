package proxy

import (
	"crypto/tls"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"strings"

	"github.com/openshift/elasticsearch-proxy/pkg/config"
)

type MetricsServer struct {
	Handler http.Handler
	Opts    *config.Options
}

func (s *MetricsServer) ListenAndServe() {
	s.ServeHTTPS()
}

func (s *MetricsServer) ServeHTTPS() {
	addr := s.Opts.MetricsListeningAddress
	config := &tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS12,
	}
	if config.NextProtos == nil {
		config.NextProtos = []string{"http/1.1"}
	}

	var err error
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.LoadX509KeyPair(s.Opts.MetricsTLSCertFile, s.Opts.MetricsTLSKeyFile)
	if err != nil {
		log.Fatalf("Loading metrics tls config (%s, %s) failed - %s", s.Opts.MetricsTLSCertFile, s.Opts.MetricsTLSKeyFile, err)
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Listen (%s) failed - %s", addr, err)
	}
	log.Printf("HTTPS: listening on %s", ln.Addr())

	tlsListener := tls.NewListener(tcpKeepAliveListener{ln.(*net.TCPListener)}, config)
	srv := &http.Server{Handler: s.Handler}
	err = srv.Serve(tlsListener)

	if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
		log.Errorf("https.Serve() - %s", err)
	}

	log.Printf("HTTPS: closing %s", tlsListener.Addr())
}
