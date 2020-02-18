package main

import (
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/openshift/elasticsearch-proxy/pkg/config"
	auth "github.com/openshift/elasticsearch-proxy/pkg/handlers/authorization"

	"github.com/openshift/elasticsearch-proxy/pkg/proxy"
	log "github.com/sirupsen/logrus"
)

const (
	serviceAccountTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

func main() {
	initLogging()

	opts, err := config.Init(os.Args[1:])
	if err != nil {
		log.Printf("%s", err)
		os.Exit(1)
	}

	proxyServer := proxy.NewProxyServer(opts)

	log.Printf("Registering Handlers....")
	proxyServer.RegisterRequestHandlers(auth.NewHandlers(opts))

	var h http.Handler = proxyServer
	if opts.RequestLogging {
		h = proxy.LoggingHandler(os.Stdout, h, true)
	}
	s := &proxy.Server{
		Handler: h,
		Opts:    opts,
	}
	s.ListenAndServe()
}

func initLogging() {
	logLevel := os.Getenv("LOGLEVEL")
	if logLevel == "" {
		logLevel = "warn"
	}
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		level = log.WarnLevel
		log.Infof("Setting loglevel to 'warn' as unable to parse %s", logLevel)
	}
	log.SetLevel(level)
}
