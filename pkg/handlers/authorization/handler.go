package authorization

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/openshift/elasticsearch-proxy/pkg/clients"
	"github.com/openshift/elasticsearch-proxy/pkg/config"
	"github.com/openshift/elasticsearch-proxy/pkg/handlers"
)

const (
	headerAuthorization         = "Authorization"
	headerForwardedUser         = "X-Forwarded-User"
	headerForwardedRoles        = "X-Forwarded-Roles"
	headerForwardedNamespace    = "X-OCP-NS"
	headerForwardedNamespaceUid = "X-OCP-NSUID"
)

type authorizationHandler struct {
	config        *config.Options
	osClient      clients.OpenShiftClient
	cache         *rolesService
	fnCNExtractor certCNExtractor
}

//NewHandlers is the initializer for this handler
func NewHandlers(opts *config.Options) (_ []handlers.RequestHandler) {
	osClient, err := clients.NewOpenShiftClient()
	if err != nil {
		log.Fatalf("Error constructing OpenShiftClient %v", err)
	}
	return []handlers.RequestHandler{
		&authorizationHandler{
			config:        opts,
			osClient:      osClient,
			cache:         NewRolesProjectsService(1000, opts.CacheExpiry, opts.AuthBackEndRoles, osClient),
			fnCNExtractor: defaultCertCNExtractor,
		},
	}
}
func (auth *authorizationHandler) Name() string {
	return "authorization"
}

func (auth *authorizationHandler) Process(req *http.Request, context *handlers.RequestContext) (*http.Request, error) {
	log.Tracef("Processing request in handler %q", auth.Name())
	context.WhiteListedNames = auth.config.AuthWhiteListedNames
	if auth.isWhiteListed(req) {
		log.Trace("Skipping processing in request because CN is whitelisted")
		return req, nil
	}
	context.Token = getBearerTokenFrom(req)
	if context.Token == "" {
		log.Debugf("Skipping %s as there is no bearer token present", auth.Name())
		return req, errors.New("missing bearer token")
	}
	sanitizeHeaders(req)
	rolesProjects, err := auth.cache.getRolesAndProjects(context.Token)
	if err != nil {
		return req, fmt.Errorf("could not run SAR or fech projects: %v", err)
	}
	context.UserName = rolesProjects.review.UserName()
	if rolesProjects.review.UserName() != "" {
		req.Header.Set(headerForwardedUser, context.UserName)
	}
	context.Projects = rolesProjects.projects
	projectNames := []string{}
	projectUIDs := []string{}
	for _, project := range context.Projects {
		projectNames = append(projectNames, project.Name)
		projectUIDs = append(projectUIDs, project.UUID)
	}
	req.Header.Add(headerForwardedNamespace, strings.Join(projectNames, ","))
	req.Header.Add(headerForwardedNamespaceUid, strings.Join(projectUIDs, ","))
	for name := range auth.config.AuthBackEndRoles {
		if _, ok := rolesProjects.roles[name]; ok {
			context.Roles = append(context.Roles, name)
		}
	}
	roleSet := context.RoleSet().List()
	if auth.config.AuthDefaultRoleName != "" {
		roleSet = append(roleSet, auth.config.AuthDefaultRoleName)
	}
	req.Header.Add(headerForwardedRoles, strings.Join(roleSet, ","))
	return req, nil
}

func sanitizeHeaders(req *http.Request) {
	req.Header.Del(headerAuthorization)
}

func getBearerTokenFrom(req *http.Request) string {
	parts := strings.SplitN(req.Header.Get(headerAuthorization), " ", 2)
	if len(parts) > 1 && parts[0] == "Bearer" {
		return parts[1]
	}
	return ""
}

func (auth *authorizationHandler) isWhiteListed(req *http.Request) bool {
	if auth.fnCNExtractor == nil {
		return false
	}
	name := auth.fnCNExtractor(req)
	if name == "" {
		return false
	}
	for _, whitelisted := range auth.config.AuthWhiteListedNames {
		if name == whitelisted {
			log.Tracef("CN %s is whitelisted", name)
			return true
		}
	}
	return false
}
