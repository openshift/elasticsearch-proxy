package authorization

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"

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
func (auth *authorizationHandler) Process(req *http.Request) (*http.Request, error) {
	log.Tracef("Processing request in handler %q", auth.Name())
	log.Tracef("ContentLength: %v ", req.ContentLength)
	log.Tracef("Headers: %v ", req.Header)

	ctx := req.Context()
	token := getBearerTokenFrom(req)
	sanitizeHeaders(req)

	if token != "" {
		log.Trace("Handling a request with token...")

		rolesProjects, err := auth.cache.getRolesAndProjects(token)
		if err != nil {
			return req, err
		}

		username := rolesProjects.review.UserName()
		if username == "" {
			log.Trace("Unable to determine a user's identify from bearer token")
			return req, errors.New("Unable to determine username")
		}

		req.Header.Set(headerForwardedUser, username)
		ctx = context.WithValue(ctx, handlers.UsernameKey, username)

		projects := rolesProjects.projects

		projectNames := []string{}
		projectUIDs := []string{}
		for _, project := range projects {
			projectNames = append(projectNames, fmt.Sprintf("%q", project.Name))
			projectUIDs = append(projectUIDs, project.UUID)
		}

		req.Header.Add(headerForwardedNamespace, strings.Join(projectNames, ","))
		req.Header.Add(headerForwardedNamespaceUid, strings.Join(projectUIDs, ","))
		ctx = context.WithValue(ctx, handlers.ProjectsKey, projects)

		var roles []string
		if auth.config.AuthDefaultRole != "" {
			roles = append(roles, auth.config.AuthDefaultRole)
		}

		for name := range auth.config.AuthBackEndRoles {
			if _, ok := rolesProjects.roles[name]; ok {
				roles = append(roles, name)
			}
		}

		rs := sets.NewString(roles...)
		if rs.Has(auth.config.AuthAdminRole) {
			log.Debugf("User has the configurated admin role %v. Removing all other roles.", auth.config.AuthAdminRole)
			roles = []string{auth.config.AuthAdminRole}
			rs = sets.NewString(roles...)
		}

		req.Header.Add(headerForwardedRoles, strings.Join(rs.List(), ","))
		ctx = context.WithValue(ctx, handlers.RolesKey, roles)

	} else {
		log.Trace("Handling a request without token...")

		subject := auth.fnSubjectExtractor(req)
		if strings.TrimSpace(subject) == "" {
			log.Trace("Unable to determine a user's identify from certificate subject")
			return req, errors.New("Unable to determine username")
		}

		req.Header.Set(headerForwardedUser, subject)
		ctx = context.WithValue(ctx, handlers.SubjectKey, subject)
	}

	req.Header.Add(headerForwardedFor, "localhost")
	log.Tracef("Authenticated user %q", req.Header.Get(headerForwardedUser))

	return req.WithContext(ctx), nil
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
