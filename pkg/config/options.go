package config

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	options "github.com/mreiferson/go-options"
	log "github.com/sirupsen/logrus"
)

//Options that can be set by Command Line Flag, or Config File
type Options struct {
	ProxyWebSockets  bool     `flag:"proxy-websockets"`
	ListeningAddress string   `flag:"listening-address"`
	TLSCertFile      string   `flag:"tls-cert"`
	TLSKeyFile       string   `flag:"tls-key"`
	TLSClientCAFile  string   `flag:"tls-client-ca"`
	OpenShiftCAs     []string `flag:"openshift-ca"`

	MetricsListeningAddress string `flag:"metrics-listening-address"`
	MetricsTLSCertFile      string `flag:"metrics-tls-cert"`
	MetricsTLSKeyFile       string `flag:"metrics-tls-key"`

	Elasticsearch    string `flag:"elasticsearch-url"`
	ElasticsearchURL *url.URL
	UpstreamFlush    time.Duration `flag:"upstream-flush"`
	UpstreamCAs      []string      `flag:"upstream-ca"`

	SSLInsecureSkipVerify bool `flag:"ssl-insecure-skip-verify"`
	RequestLogging        bool `flag:"request-logging"`

	//Auth Handler Configs

	//RawAuthBackEndRole is a map of rolename to SubjectAccessReviews to check to apply a given role to a user
	//these are parsed and added to AuthBackEndRoles
	RawAuthBackEndRole []string `flag:"auth-backend-role"`

	//AuthBackEndRoles is a map of rolename to SubjectAccessReviews to check to apply a given role to a user
	AuthBackEndRoles map[string]BackendRoleConfig
	CacheExpiry      time.Duration `flag:"cache-expiry"`
	//AuthWhiteListedNames  is the list of names compared against cert CN for which a request will be passed through
	//with no additional processing
	AuthWhiteListedNames []string `flag:"auth-whitelisted-name"`

	//AuthAdminRole is the name of the only role that will be
	//passed on the request if it is found in the list of roles
	AuthAdminRole string `flag:"auth-admin-role"`

	//AuthDefaultRole is the role added when no other roles are provided
	AuthDefaultRole string `flag:"auth-default-role"`

	//net/http.Server timeouts for the server side of the proxy
	HTTPReadTimeout  time.Duration `flag:"http-read-timeout"`
	HTTPWriteTimeout time.Duration `flag:"http-write-timeout"`
	HTTPIdleTimeout  time.Duration `flag:"http-idle-timeout"`

	//net/http.Transport limits and timeouts
	HTTPMaxConnsPerHost       int           `flag:"http-max-conns-per-host"`
	HTTPMaxIdleConns          int           `flag:"http-max-idle-conns"`
	HTTPMaxIdleConnsPerHost   int           `flag:"http-max-idle-conns-per-host"`
	HTTPIdleConnTimeout       time.Duration `flag:"http-idle-conn-timeout"`
	HTTPTLSHandshakeTimeout   time.Duration `flag:"http-tls-handshake-timeout"`
	HTTPExpectContinueTimeout time.Duration `flag:"http-expect-continue-timeout"`
}

//Init the configuration options based on the values passed via the CLI
func Init(args []string) (*Options, error) {
	opts := newOptions()
	flagSet := newFlagSet()

	flagSet.Parse(args)

	options.Resolve(opts, flagSet, nil)

	if opts.SSLInsecureSkipVerify {
		insecureTransport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		http.DefaultClient = &http.Client{Transport: insecureTransport}
	}
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	return opts, nil
}

func newOptions() *Options {
	return &Options{
		ProxyWebSockets:           true,
		ListeningAddress:          ":443",
		Elasticsearch:             "https://localhost:9200",
		UpstreamFlush:             time.Duration(5) * time.Millisecond,
		RequestLogging:            false,
		AuthBackEndRoles:          map[string]BackendRoleConfig{},
		AuthWhiteListedNames:      []string{},
		AuthAdminRole:             "",
		HTTPReadTimeout:           time.Duration(1) * time.Minute,
		HTTPWriteTimeout:          time.Duration(1) * time.Minute,
		HTTPIdleTimeout:           time.Duration(1) * time.Minute,
		HTTPMaxConnsPerHost:       25,
		HTTPMaxIdleConns:          25,
		HTTPMaxIdleConnsPerHost:   25,
		HTTPIdleConnTimeout:       time.Duration(1) * time.Minute,
		HTTPTLSHandshakeTimeout:   time.Duration(10) * time.Second,
		HTTPExpectContinueTimeout: time.Duration(1) * time.Second,
	}
}

//Validate the configuration options and return errors
func (o *Options) Validate() error {
	log.Tracef("Validating options: %v", o)
	msgs := make([]string, 0)

	if len(o.Elasticsearch) < 1 {
		msgs = append(msgs, "missing setting: upstream")
	} else {
		log.Tracef("Validating ElasticsearchURL: %q", o.Elasticsearch)
		elasticsearchURL, err := url.Parse(o.Elasticsearch)
		if err != nil {
			msgs = append(msgs, fmt.Sprintf(
				"error parsing ElasticsearchURL=%q %s",
				o.Elasticsearch, err))
		}
		if elasticsearchURL.Path == "" {
			elasticsearchURL.Path = "/"
		}
		o.ElasticsearchURL = elasticsearchURL
	}

	if len(o.TLSClientCAFile) > 0 && (len(o.TLSKeyFile) == 0 || len(o.TLSCertFile) == 0) {
		msgs = append(msgs, "tls-client-ca requires tls-key-file or tls-cert-file to be set to listen on tls")
	}

	if o.MetricsListeningAddress != "" && (o.MetricsTLSCertFile == "" || o.MetricsTLSKeyFile == "") {
		msgs = append(msgs, "metrics-listening-address requires metrics-tls-cert and metrics-tls-key to be set")
	}

	//Auth Handler validations
	if len(o.RawAuthBackEndRole) > 0 {
		for _, raw := range o.RawAuthBackEndRole {
			parts := strings.Split(raw, "=")
			if len(parts) != 2 {
				msgs = append(msgs, fmt.Sprintf("auth-backend-role %q should be name=SAR", raw))
				continue
			}
			name := parts[0]
			sar := parts[1]
			roleConfig, err := parseBackendRoleConfig(sar)
			if err != nil {
				msgs = append(msgs, fmt.Sprintf("Unable to parse backend roleConfig %q: %v", raw, err))
				continue
			}
			if _, exists := o.AuthBackEndRoles[name]; exists {
				msgs = append(msgs, fmt.Sprintf("Backend role with that name %q already exists", raw))
				continue
			}
			o.AuthBackEndRoles[name] = *roleConfig
		}
	}

	if o.HTTPReadTimeout < 0 {
		msgs = append(msgs, "http-read-timeout can not be negative")
	}
	if o.HTTPWriteTimeout < 0 {
		msgs = append(msgs, "http-write-timeout can not be negative")
	}
	if o.HTTPIdleTimeout < 0 {
		msgs = append(msgs, "http-idle-timeout can not be negative")
	}
	if o.HTTPMaxConnsPerHost < 0 {
		msgs = append(msgs, "http-max-conns-per-host can not be negative")
	}
	if o.HTTPMaxIdleConns < 0 {
		msgs = append(msgs, "http-max-idle-conns can not be negative")
	}
	if o.HTTPMaxIdleConnsPerHost < 0 {
		msgs = append(msgs, "http-max-idle-conns-per-host can not be negative")
	}
	if o.HTTPIdleConnTimeout < 0 {
		msgs = append(msgs, "http-idle-conn-timeout can not be negative")
	}
	if o.HTTPTLSHandshakeTimeout < 0 {
		msgs = append(msgs, "http-tls-handshake-timeout can not be negative")
	}
	if o.HTTPExpectContinueTimeout < 0 {
		msgs = append(msgs, "http-expect-continue-timeout can not be negative")
	}

	if len(msgs) != 0 {
		return fmt.Errorf("Invalid configuration:\n  %s",
			strings.Join(msgs, "\n  "))
	}

	return nil
}
