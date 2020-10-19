package proxy

import (
	"crypto/tls"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/openshift/elasticsearch-proxy/pkg/config"
	"github.com/openshift/elasticsearch-proxy/pkg/util"
)

type Server struct {
	Handler http.Handler
	Opts    *config.Options
}

func (s *Server) ListenAndServe() {
	addr := s.Opts.ListeningAddress
	if addr == "" {
		log.Fatal("missing proxy listening address")
	}

	certFile := s.Opts.TLSCertFile
	if certFile == "" {
		log.Fatal("missing TLS proxy server certificate file")
	}

	keyFile := s.Opts.TLSKeyFile
	if keyFile == "" {
		log.Fatal("missing TLS proxy server key file")
	}

	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS12,
		NextProtos: []string{"http/1.1"},
	}

	if s.Opts.TLSClientCAFile != "" {
		cas, err := util.GetCertPool([]string{s.Opts.TLSClientCAFile}, false)
		if err != nil {
			log.Fatalf("failed to load certificates pool: %v", err)
		}

		cfg.ClientAuth = tls.VerifyClientCertIfGiven
		cfg.ClientCAs = cas
	}

	srv := &http.Server{
		Addr:         addr,
		Handler:      s.Handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  60 * time.Second,
		TLSConfig:    cfg,
	}
	srv.SetKeepAlivesEnabled(true)

	err := srv.ListenAndServeTLS(certFile, keyFile)
	if err != nil && err != http.ErrServerClosed {
		log.Errorf("failed proxy to listen and serve TLS: %s", err)
	}

	log.Printf("proxy closing: %s", addr)
}
