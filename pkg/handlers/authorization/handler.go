package authorization

import (
	"errors"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/openshift/elasticsearch-proxy/pkg/clients"
	"github.com/openshift/elasticsearch-proxy/pkg/config"
	"github.com/openshift/elasticsearch-proxy/pkg/handlers"
)

const (
	headerAuthorization         = "Authorization"
	headerForwardedFor          = "X-Forwarded-For"
	headerForwardedUser         = "X-Forwarded-User"
	headerForwardedRoles        = "X-Forwarded-Roles"
	headerForwardedNamespace    = "X-OCP-NS"
	headerForwardedNamespaceUid = "X-OCP-NSUID"
	headerXForwardedAccessToken = "X-Forwarded-Access-Token"
)

type authorizationHandler struct {
	config             *config.Options
	osClient           clients.OpenShiftClient
	cache              *rolesService
	fnSubjectExtractor certSubjectExtractor
}

//NewHandlers is the initializer for this handler
func NewHandlers(opts *config.Options) []handlers.RequestHandler {
	osClient, err := clients.NewOpenShiftClient()
	if err != nil {
		log.Fatalf("Error constructing OpenShiftClient %v", err)
	}
	return []handlers.RequestHandler{
		&authorizationHandler{
			config:             opts,
			osClient:           osClient,
			cache:              NewRolesProjectsService(1000, opts.CacheExpiry, opts.AuthBackEndRoles, osClient),
			fnSubjectExtractor: defaultCertSubjectExtractor,
		},
	}
}
func (auth *authorizationHandler) Name() string {
	return "authorization"
}

//Process the request for authorization. The handler first attempts to get userinfo using bearer token
//and falls back to the certificate subject or fails
func (auth *authorizationHandler) Process(req *http.Request, context *handlers.RequestContext) (*http.Request, error) {
	log.Tracef("Processing request in handler %q", auth.Name())
	log.Tracef("ContentLength: %v ", req.ContentLength)
	log.Tracef("Headers: %v ", req.Header)
	context.Token = getBearerTokenFrom(req)
	sanitizeHeaders(req)
	if context.Token != "" {
		log.Trace("Handling a request with token...")
		rolesProjects, err := auth.cache.getRolesAndProjects(context.Token)
		if err != nil {
			return req, err
		}
		context.UserName = rolesProjects.review.UserName()
		if context.UserName == "" {
			log.Trace("Unable to determine a user's identify from bearer token")
			return req, errors.New("Unable to determine username")
		}
		req.Header.Set(headerForwardedUser, context.UserName)
		context.Projects = rolesProjects.projects
		projectNames := []string{}
		projectUIDs := []string{}
		for _, project := range context.Projects {
			projectNames = append(projectNames, project.Name)
			projectUIDs = append(projectUIDs, project.UUID)
		}
		req.Header.Add(headerForwardedNamespace, strings.Join(projectNames, ","))
		req.Header.Add(headerForwardedNamespaceUid, strings.Join(projectUIDs, ","))
		if auth.config.AuthDefaultRole != "" {
			context.Roles = append(context.Roles, auth.config.AuthDefaultRole)
		}
		for name := range auth.config.AuthBackEndRoles {
			if _, ok := rolesProjects.roles[name]; ok {
				context.Roles = append(context.Roles, name)
			}
		}
		if context.RoleSet().Has(auth.config.AuthAdminRole) {
			log.Debugf("User has the configurated admin role %v. Removing all other roles.", auth.config.AuthAdminRole)
			context.Roles = []string{auth.config.AuthAdminRole}
		}
		req.Header.Add(headerForwardedRoles, strings.Join(context.RoleSet().List(), ","))
	} else {
		log.Trace("Handling a request without token...")
		subject := auth.fnSubjectExtractor(req)
		if strings.TrimSpace(subject) == "" {
			log.Trace("Unable to determine a user's identify from certificate subject")
			return req, errors.New("Unable to determine username")
		}
		req.Header.Set(headerForwardedUser, subject)
	}
	req.Header.Add(headerForwardedFor, "localhost")
	log.Tracef("Authenticated user %q", req.Header.Get(headerForwardedUser))
	return req, nil
}

func sanitizeHeaders(req *http.Request) {
	req.Header.Del(headerAuthorization)
	req.Header.Del(headerForwardedRoles)
	req.Header.Del(headerForwardedUser)
	req.Header.Del(headerForwardedNamespace)
	req.Header.Del(headerForwardedNamespaceUid)
	req.Header.Del(headerXForwardedAccessToken)
}

func getBearerTokenFrom(req *http.Request) string {
	if token := req.Header.Get(headerXForwardedAccessToken); strings.TrimSpace(token) != "" {
		return token
	}
	parts := strings.SplitN(req.Header.Get(headerAuthorization), " ", 2)
	if len(parts) > 1 && parts[0] == "Bearer" {
		return strings.TrimSpace(parts[1])
	}
	return ""
}
