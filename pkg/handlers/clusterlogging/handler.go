package clusterlogging

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/openshift/elasticsearch-proxy/pkg/config"
	"github.com/openshift/elasticsearch-proxy/pkg/handlers"
	ac "github.com/openshift/elasticsearch-proxy/pkg/handlers/clusterlogging/accesscontrol"
	log "github.com/sirupsen/logrus"
)

const (
	clusterLogging = "clusterlogging"
	kibanaPrefix   = "/.kibana"
)

type setString map[string]interface{}

type handler struct {
	config *config.Options

	documentManager *ac.DocumentManager
}

//NewHandlers is the initializer for clusterlogging handlers
func NewHandlers(opts *config.Options) []handlers.RequestHandler {
	dm, err := ac.NewDocumentManager(*opts)
	if err != nil {
		log.Fatalf("Unable to initialize the cluster logging proxy handler %v", err)
	}
	return []handlers.RequestHandler{
		&handler{
			config:          opts,
			documentManager: dm,
		},
	}
}

func (ext *handler) Process(req *http.Request, context *handlers.RequestContext) (*http.Request, error) {

	log.Infof("Received context: UserName: %v", context.UserName)
	log.Infof("Received context: Roles: %v", context.Roles)

	// if we are getting a .kibana/config url...
	if ext.isKibanaIndex(*req.URL) {
		kibanaVersion := ext.getKibanaVersion(*req.URL)
		if kibanaVersion != "" {
			log.Infof("Received request: kibana ver: %v", kibanaVersion)
		}

		log.Infof("Received request: Header: %v", req.Header)

		// use ES client and set headers for Index request (WithHeaders) from req...
		// we let the security plugin correctly choose the tenant -- so we only seed as required
		// only seed infra and audit if they are an admin role though...

		// to create index pattern:
		//IndexWithHeader(".kibana", "doc", "index-pattern:app", ext.getIndexPatternBody("app"), req.Header)

		// to set default index pattern:
		// note: only do this if it doesn't yet exist...
		//IndexWithHeader(".kibana", "doc", fmt.Sprintf("config:%s", kibanaVersion), ext.getIndexPatternBody("app"), req.Header)
	}

	return req, nil
}

func (ext *handler) isKibanaIndex(uri url.URL) bool {
	log.Infof("Evaluating: %v", uri.Path)
	return strings.HasPrefix(uri.Path, kibanaPrefix)
}

func (ext *handler) getKibanaVersion(uri url.URL) string {

	version := ""

	// protects against "[/.kibana/_search]" case
	for _, val := range strings.SplitAfter(uri.Path, "config:") {
		if strings.HasPrefix(val, kibanaPrefix) {
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

func (ext *handler) Name() string {
	return clusterLogging
}
