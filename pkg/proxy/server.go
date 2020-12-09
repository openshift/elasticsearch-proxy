package proxy

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/http2"

	configOptions "github.com/openshift/elasticsearch-proxy/pkg/config"
	handlers "github.com/openshift/elasticsearch-proxy/pkg/handlers"
	"github.com/openshift/elasticsearch-proxy/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/yhat/wsutil"
)

type ProxyServer struct {
	serveMux http.Handler

	// handlers
	requestHandlers []handlers.RequestHandler
}

// RegisterRequestHandlers adds request handlers to the
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

func NewReverseProxy(target *url.URL, upstreamFlush time.Duration, rootCAs []string) (*httputil.ReverseProxy, error) {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.FlushInterval = upstreamFlush

	transport := &http.Transport{
		MaxConnsPerHost:       25,
		MaxIdleConns:          25,
		MaxIdleConnsPerHost:   25,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	if len(rootCAs) > 0 {
		pool, err := util.GetCertPool(rootCAs, false)
		if err != nil {
			return nil, err
		}
		transport.TLSClientConfig = &tls.Config{
			RootCAs: pool,
		}
	}
	if err := http2.ConfigureTransport(transport); err != nil {
		if len(rootCAs) > 0 {
			return nil, err
		}
		log.Warnf("Failed to configure http2 transport: %v", err)
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
	proxy, err := NewReverseProxy(u, opts.UpstreamFlush, opts.UpstreamCAs)
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
	u := opts.ElasticsearchURL
	path := u.Path
	switch u.Scheme {
	case "http", "https":
		log.Printf("mapping path %q => upstream %q", path, u)
		proxy := NewWebSocketOrRestReverseProxy(u, opts)
		serveMux.Handle(path, proxy)

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
	var err error
	alteredReq := req
	responseLogger := &responseLogger{rw}
	context := handlers.RequestContext{}
	for _, reqhandler := range p.requestHandlers {
		alteredReq, err = reqhandler.Process(alteredReq, &context)
		log.Printf("Handling request %q", reqhandler.Name())
		if err != nil {
			log.Printf("Error processing request in handler %s: %v", reqhandler.Name(), err)
			p.StructuredError(responseLogger, err)
			return
		}
	}
	log.Debugf("Request: %v", alteredReq)
	p.serveMux.ServeHTTP(responseLogger, alteredReq)
}

func (p *ProxyServer) StructuredError(rw http.ResponseWriter, err error) {
	structuredError := handlers.NewStructuredError(err)
	log.Debugf("Error %d %s %s", structuredError.Code, structuredError.Message, structuredError.Error)
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(structuredError.Code)
	buffer := new(bytes.Buffer)
	encodingError := json.NewEncoder(buffer).Encode(structuredError)
	if encodingError != nil {
		log.Errorf("Error writing response body for Error %v", err)
		return
	}
	rw.Write(buffer.Bytes())
}
