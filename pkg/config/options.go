package config

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	options "github.com/mreiferson/go-options"
	cltypes "github.com/openshift/elasticsearch-proxy/pkg/handlers/clusterlogging/types"
	log "github.com/sirupsen/logrus"
)

//Options are that can be set by Command Line Flag, or Config File
type Options struct {
	ProxyWebSockets  bool     `flag:"proxy-websockets"`
	ListeningAddress string   `flag:"listening-address"`
	TLSCertFile      string   `flag:"tls-cert"`
	TLSKeyFile       string   `flag:"tls-key"`
	TLSClientCAFile  string   `flag:"tls-client-ca"`
	OpenShiftCAs     []string `flag:"openshift-ca"`

	Elasticsearch    string `flag:"elasticsearch-url"`
	ElasticsearchURL *url.URL
	UpstreamFlush    time.Duration `flag:"upstream-flush"`
	UpstreamCAs      []string      `flag:"upstream-ca"`

	SSLInsecureSkipVerify bool `flag:"ssl-insecure-skip-verify"`
	RequestLogging        bool `flag:"request-logging"`

	//Auth Handler Configs
	RawAuthBackEndRole []string `flag:"auth-backend-role"`
	AuthBackEndRoles   map[string]BackendRoleConfig
	CacheExpiry        time.Duration `flag:"cache-expiry"`

	//OCP Cluster Logging configs
	cltypes.ExtConfig
}

//Init the configuration options based on the values passed via the CLI
func Init(args []string) (*Options, error) {
	opts := newOptions()
	flagSet := newFlagSet()

	cltypes.RegisterFlagSets(flagSet)

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
		ProxyWebSockets:  true,
		ListeningAddress: ":443",
		Elasticsearch:    "https://127.0.0.1:9200",
		UpstreamFlush:    time.Duration(5) * time.Millisecond,
		RequestLogging:   false,
		AuthBackEndRoles: map[string]BackendRoleConfig{},
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

	//Cluster Logging Handler Validations
	if len(o.RawKibanaIndexMode) > 0 {
		mode, err := cltypes.ParseKibanaIndexMode(o.RawKibanaIndexMode)
		if err != nil {
			msgs = append(msgs, err.Error())
		} else {
			o.KibanaIndexMode = mode
		}
	}

	if len(msgs) != 0 {
		return fmt.Errorf("Invalid configuration:\n  %s",
			strings.Join(msgs, "\n  "))
	}

	return nil
}
