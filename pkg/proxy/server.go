package proxy

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/http/pprof"
	"net/url"
	"strings"

	configOptions "github.com/openshift/elasticsearch-proxy/pkg/config"
	handlers "github.com/openshift/elasticsearch-proxy/pkg/handlers"
	"github.com/openshift/elasticsearch-proxy/pkg/handlers/instrumentation"
	"github.com/openshift/elasticsearch-proxy/pkg/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/yhat/wsutil"
)

type ProxyServer struct {
	serveMux http.Handler

	//handlers
	requestHandlers []handlers.RequestHandler
}

//RegisterRequestHandlers adds request handlers to the
func (p *ProxyServer) RegisterRequestHandlers(reqHandlers []handlers.RequestHandler) {
	p.requestHandlers = append(p.requestHandlers, reqHandlers...)
}

type UpstreamProxy struct {
	upstream  string
	handler   http.Handler
	wsHandler http.Handler
}

func (u *UpstreamProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("GAP-Upstream-Address", u.upstream)
	if u.wsHandler != nil && r.Header.Get("Connection") == "Upgrade" && r.Header.Get("Upgrade") == "websocket" {
		u.wsHandler.ServeHTTP(w, r)
	} else {
		u.handler.ServeHTTP(w, r)
	}
}

func NewReverseProxy(target *url.URL, opts *configOptions.Options) (*httputil.ReverseProxy, error) {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.FlushInterval = opts.UpstreamFlush

	transport := &http.Transport{
		MaxConnsPerHost:       opts.HTTPMaxConnsPerHost,
		MaxIdleConns:          opts.HTTPMaxIdleConns,
		MaxIdleConnsPerHost:   opts.HTTPMaxIdleConnsPerHost,
		IdleConnTimeout:       opts.HTTPIdleConnTimeout,
		TLSHandshakeTimeout:   opts.HTTPTLSHandshakeTimeout,
		ExpectContinueTimeout: opts.HTTPExpectContinueTimeout,
	}
	if len(opts.UpstreamCAs) > 0 {
		pool, err := util.GetCertPool(opts.UpstreamCAs, false)
		if err != nil {
			return nil, err
		}
		transport.TLSClientConfig = &tls.Config{
			RootCAs: pool,
		}
	}
	proxy.Transport = transport

	return proxy, nil
}

func setProxyUpstreamHostHeader(proxy *httputil.ReverseProxy, target *url.URL) {
	director := proxy.Director
	proxy.Director = func(req *http.Request) {
		director(req)
		// use RequestURI so that we aren't unescaping encoded slashes in the request path
		req.Host = target.Host
		req.URL.Opaque = req.RequestURI
		req.URL.RawQuery = ""
	}
}

func NewWebSocketOrRestReverseProxy(u *url.URL, opts *configOptions.Options) (restProxy http.Handler) {
	u.Path = ""
	proxy, err := NewReverseProxy(u, opts)
	if err != nil {
		log.Fatal("Failed to initialize Reverse Proxy: ", err)
	}
	setProxyUpstreamHostHeader(proxy, u)

	// this should give us a wss:// scheme if the url is https:// based.
	var wsProxy *wsutil.ReverseProxy = nil
	if opts.ProxyWebSockets {
		wsScheme := "ws" + strings.TrimPrefix(u.Scheme, "http")
		wsURL := &url.URL{Scheme: wsScheme, Host: u.Host}
		wsProxy = wsutil.NewSingleHostReverseProxy(wsURL)

		if wsScheme == "wss" && len(opts.UpstreamCAs) > 0 {
			pool, err := util.GetCertPool(opts.UpstreamCAs, false)
			if err != nil {
				log.Fatal("Failed to fetch CertPool: ", err)
			}
			wsProxy.TLSClientConfig = &tls.Config{
				RootCAs: pool,
			}
		}

	}
	return &UpstreamProxy{u.Host, proxy, wsProxy}
}

func NewProxyServer(opts *configOptions.Options) *ProxyServer {
	serveMux := http.NewServeMux()

	serveMux.HandleFunc("/metrics", func(rw http.ResponseWriter, r *http.Request) {
		promhttp.Handler().ServeHTTP(rw, r)
	})
	serveMux.HandleFunc("/debug/pprof/", pprof.Index)
	serveMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	serveMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	serveMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	serveMux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	u := opts.ElasticsearchURL
	path := u.Path
	switch u.Scheme {
	case "http", "https":
		log.Infof("mapping path %q => upstream %q", path, u)
		ins := instrumentation.NewHandler(prometheus.DefaultRegisterer)
		proxy := NewWebSocketOrRestReverseProxy(u, opts)
		serveMux.Handle(path, ins.WithHandler("proxy", proxy))

	default:
		panic(fmt.Sprintf("unknown upstream protocol %s", u.Scheme))
	}

	return &ProxyServer{
		serveMux: serveMux,
	}
}

func (p *ProxyServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	log.Debugf("Serving request: %s", req.URL.Path)
	log.Tracef("Content-Length: %v", req.ContentLength)
	log.Tracef("Headers: %v", req.Header)

	for _, reqhandler := range p.requestHandlers {
		log.Debugf("Handling request %q", reqhandler.Name())

		var err error
		req, err = reqhandler.Process(req)
		if err != nil {
			log.Errorf("Error processing request in handler %s: %v", reqhandler.Name(), err)
			p.StructuredError(rw, err)
			break
		}
	}

	p.serveMux.ServeHTTP(rw, req)
}

func (p *ProxyServer) StructuredError(rw http.ResponseWriter, err error) {
	structuredError := handlers.NewStructuredError(err)
	log.Debugf("Error %d %s %s", structuredError.Code, structuredError.Message, structuredError.Error)
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(structuredError.Code)

	b, err := json.Marshal(structuredError)
	if err != nil {
		log.Errorf("failed marshalling structured error: %s", err)
		return
	}
	_, _ = rw.Write(b)
}
