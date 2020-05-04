package clusterlogging

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/openshift/elasticsearch-proxy/pkg/clients"
	"github.com/openshift/elasticsearch-proxy/pkg/config"
	"github.com/openshift/elasticsearch-proxy/pkg/handlers"
	log "github.com/sirupsen/logrus"
)

const (
	clusterLogging   = "clusterlogging"
	kibanaPathPrefix = "/.kibana"
	appPattern       = "app"
	infraPattern     = "infra"
	auditPattern     = "audit"
	adminRole        = "admin_reader"
)

var requiredHeaders = []string{"Securitytenant", "X-Forwarded-For", "X-Forwarded-Roles", "X-Forwarded-User", "X-Ocp-Ns"}

type setString map[string]interface{}

type handler struct {
	config *config.Options

	esClient clients.ElasticsearchClient
}

//NewHandlers is the initializer for clusterlogging handlers
func NewHandlers(opts *config.Options) []handlers.RequestHandler {
	es, err := clients.NewESClient(*opts)
	if err != nil {
		log.Fatalf("Unable to initialize the cluster logging proxy handler %v", err)
	}
	return []handlers.RequestHandler{
		&handler{
			config:   opts,
			esClient: es,
		},
	}
}

func (ext *handler) Process(req *http.Request, context *handlers.RequestContext) (*http.Request, error) {

	// if we are getting a .kibana/config url...
	if ext.isKibanaIndex(*req.URL) {
		kibanaVersion := ext.getKibanaVersion(*req.URL)
		if kibanaVersion != "" {
			log.Tracef("Received request: kibana ver: %v", kibanaVersion)
		}

		// use ES client and set headers for Index request (WithHeaders) from req...
		// we let the security plugin correctly choose the tenant -- so we only seed as required
		// only seed infra and audit if they are an admin role though...

		headers := make(map[string]string)
		// pass along headers: Securitytenant, X-Forwarded-For, X-Forwarded-Roles, X-Forwarded-User, X-Ocp-Ns
		for _, header := range requiredHeaders {
			if val, ok := req.Header[header]; ok {
				// conform the array to a single string
				headers[header] = strings.Join(val, " ")
			}
		}

		ext.ensureAppIndexPattern(headers)

		// if they are an admin user also add Infra and Audit (based on context role)
		if ext.isAdminUser(context.Roles) {
			ext.ensureInfraIndexPattern(headers)
			ext.ensureAuditIndexPattern(headers)
		}

		if kibanaVersion != "" {
			ext.ensureDefaultIndexPattern(kibanaVersion, appPattern, headers)
		}
	}

	return req, nil
}

func (ext *handler) ensureAppIndexPattern(headers map[string]string) error {
	return ext.ensureIndexPattern(appPattern, headers)
}

func (ext *handler) ensureInfraIndexPattern(headers map[string]string) error {
	return ext.ensureIndexPattern(infraPattern, headers)
}

func (ext *handler) ensureAuditIndexPattern(headers map[string]string) error {
	return ext.ensureIndexPattern(auditPattern, headers)
}

// to create index pattern:
func (ext *handler) ensureIndexPattern(pattern string, headers map[string]string) error {
	resp, err := ext.esClient.GetWithHeader(".kibana", "doc", fmt.Sprintf("index-pattern:%s", pattern), headers)
	log.Infof("Get index with header received resp: '%s' \n and err: %v", resp, err)

	if err != nil {
		return err
	}

	if resp == "" {
		resp, err = ext.esClient.IndexWithHeader(".kibana", "doc", fmt.Sprintf("index-pattern:%s", pattern), ext.getIndexPatternBody(pattern), headers)
		log.Infof("Create index with header received resp: '%s' \n and err: %v", resp, err)

		if err != nil {
			return err
		}
	}

	return nil
}

// set default index pattern only if it doesn't yet exist...
func (ext *handler) ensureDefaultIndexPattern(kibanaVersion, pattern string, headers map[string]string) error {

	resp, err := ext.esClient.GetWithHeader(".kibana", "doc", fmt.Sprintf("config:%s", kibanaVersion), headers)
	log.Infof("Get index with header received resp: '%s' \n and err: %v", resp, err)

	if err != nil {
		return err
	}

	if resp == "" {
		// we don't already have a config id for our tenant
		resp, err = ext.esClient.IndexWithHeader(".kibana", "doc", fmt.Sprintf("config:%s", kibanaVersion), ext.getDefaultIndexPatternBody(pattern), headers)
		log.Infof("Create defaultindexpattern with header received resp: '%s' \n and err: %v", resp, err)

		if err != nil {
			return err
		}
	} else {
		// check to make sure we don't already have a "defaultIndex" setting...
		// guard against -> "defaultIndex" : null
		if getDefaultIndex(resp) == "" {
			resp, err = ext.esClient.IndexWithHeader(".kibana", "doc", fmt.Sprintf("config:%s", kibanaVersion), ext.getDefaultIndexPatternBody(pattern), headers)
			log.Infof("Create defaultindexpattern with header received resp: '%s' \n and err: %v", resp, err)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getDefaultIndex(resp string) string {

	defaultIndex := ""

	var results map[string]interface{}
	err := json.Unmarshal([]byte(resp), &results)
	if err != nil {
		log.Error("Unable to unmarshal response")
	}

	if results["_source"] != nil {
		sourceMap := results["_source"].(map[string]interface{})
		if sourceMap["config"] != nil {
			configMap := sourceMap["config"].(map[string]interface{})
			if index, ok := configMap["defaultIndex"].(string); ok {
				defaultIndex = index
			}
		}
	}

	return defaultIndex
}

func (ext *handler) isKibanaIndex(uri url.URL) bool {
	log.Infof("Evaluating: %v", uri.Path)
	return strings.HasPrefix(uri.Path, kibanaPathPrefix)
}

func (ext *handler) getKibanaVersion(uri url.URL) string {

	version := ""

	// protects against "[/.kibana/_search]" case
	for _, val := range strings.SplitAfter(uri.Path, "config:") {
		if strings.HasPrefix(val, kibanaPathPrefix) {
			continue
		}

		// covers "/.kibana/doc/config:6.8.1/_create" case
		versionSplit := strings.Split(val, "/")
		if len(versionSplit) > 0 {
			version = versionSplit[0]
			break
		}
	}

	return version
}

func (ext *handler) getIndexPatternBody(patternName string) string {
	return fmt.Sprintf("{\"type\":\"index-pattern\",\"index-pattern\":{\"title\":\"%s\",\"timeFieldName\":\"@timestamp\"}}", patternName)
}

func (ext *handler) getDefaultIndexPatternBody(patternName string) string {
	return fmt.Sprintf("{\"type\":\"config\",\"config\":{\"defaultIndex\":\"%s\"}}", patternName)
}

func (ext *handler) isAdminUser(roles []string) bool {

	for _, role := range roles {
		if role == adminRole {
			return true
		}
	}

	return false
}

func (ext *handler) Name() string {
	return clusterLogging
}
