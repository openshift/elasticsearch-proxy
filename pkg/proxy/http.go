package proxy

import (
	"crypto/tls"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"strings"

	"github.com/openshift/elasticsearch-proxy/pkg/config"
	"github.com/openshift/elasticsearch-proxy/pkg/util"
)

type Server struct {
	Handler http.Handler
	Opts    *config.Options
}

func (s *Server) ListenAndServe() {
	if s.Opts.ListeningAddress == "" {
		log.Fatalf("Must specify https-addres")
	}
	s.ServeHTTPS()
}

func (s *Server) ServeHTTPS() {
	addr := s.Opts.ListeningAddress
	config := &tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS12,
	}
	if config.NextProtos == nil {
		config.NextProtos = []string{"http/1.1"}
	}

	var err error
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.LoadX509KeyPair(s.Opts.TLSCertFile, s.Opts.TLSKeyFile)
	if err != nil {
		log.Fatalf("Loading tls config (%s, %s) failed - %s", s.Opts.TLSCertFile, s.Opts.TLSKeyFile, err)
	}

	if len(s.Opts.TLSClientCAFile) > 0 {
		config.ClientAuth = tls.VerifyClientCertIfGiven
		config.ClientCAs, err = util.GetCertPool([]string{s.Opts.TLSClientCAFile}, false)
		if err != nil {
			log.Fatalf("Unable to get cert pool: %v", err)
		}
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen (%s) failed - %s", addr, err)
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
