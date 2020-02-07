package clusterlogging

import (
	"net/http"

	"github.com/openshift/elasticsearch-proxy/pkg/config"
	"github.com/openshift/elasticsearch-proxy/pkg/handlers"
	ac "github.com/openshift/elasticsearch-proxy/pkg/handlers/clusterlogging/accesscontrol"
	"github.com/openshift/elasticsearch-proxy/pkg/handlers/clusterlogging/types"
	log "github.com/sirupsen/logrus"
)

const (
	clusterLogging = "clusterlogging"
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
	name := context.UserName
	if context.IsWhiteListed(name) || ext.hasInfraRole(context) {
		log.Debugf("Skipping additional processing, %s is whitelisted or has the infra role", name)
		return req, nil
	}
	modRequest := req
	userInfo := newUserInfo(context)
	// modify kibana request
	// seed kibana dashboards
	ext.documentManager.SyncACL(userInfo)

	return modRequest, nil
}

func (ext *handler) hasInfraRole(context *handlers.RequestContext) bool {
	for _, role := range context.Roles {
		if role == ext.config.InfraRoleName {
			log.Tracef("%s has the the Infra Role (%s)", context.UserName, ext.config.InfraRoleName)
			return true
		}
	}
	return false
}

func newUserInfo(context *handlers.RequestContext) *types.UserInfo {
	info := &types.UserInfo{
		Username: context.UserName,
		Projects: context.Projects,
		Groups:   context.Groups,
	}
	log.Tracef("Created userInfo: %+v", info)
	return info
}

func (ext *handler) Name() string {
	return clusterLogging
}
