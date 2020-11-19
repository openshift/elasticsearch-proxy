package proxy

import (
	"crypto/tls"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/openshift/elasticsearch-proxy/pkg/config"
)

type MetricsServer struct {
	Handler http.Handler
	Opts    *config.Options
}

func (s *MetricsServer) ListenAndServe() {
	addr := s.Opts.MetricsListeningAddress
	if addr == "" {
		log.Fatal("missing metrics listening address")
	}

	certFile := s.Opts.MetricsTLSCertFile
	if certFile == "" {
		log.Fatal("missing TLS metrics server certificate file")
	}

	keyFile := s.Opts.MetricsTLSKeyFile
	if keyFile == "" {
		log.Fatal("missing TLS metrics server key file")
	}

	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS12,
		NextProtos: []string{"http/1.1"},
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
		log.Errorf("failed metrics to listen and serve TLS: %s", err)
	}

	log.Printf("metrics closing: %s", addr)
}
