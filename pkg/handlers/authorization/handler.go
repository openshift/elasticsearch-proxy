package authorization

import (
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	clients "github.com/openshift/elasticsearch-proxy/pkg/clients"
	"github.com/openshift/elasticsearch-proxy/pkg/config"
	handlers "github.com/openshift/elasticsearch-proxy/pkg/handlers"
)

const (
	headerAuthorization  = "Authorization"
	headerForwardedUser  = "X-Forwarded-User"
	headerForwardedRoles = "X-Forwarded-Roles"
)

type authorizationHandler struct {
	config   *config.Options
	osClient clients.OpenShiftClient
}

//NewHandlers is the initializer for this handler
func NewHandlers(opts *config.Options) (_ []handlers.RequestHandler) {
	osClient, err := clients.NewOpenShiftClient()
	if err != nil {
		log.Fatalf("Error constructing OpenShiftClient %v", err)
	}
	return []handlers.RequestHandler{
		&authorizationHandler{
			opts,
			osClient,
			// defaultbackendRoleConfig,
		},
	}
}
func (auth *authorizationHandler) Name() string {
	return "authorization"
}

func (auth *authorizationHandler) Process(req *http.Request, context *handlers.RequestContext) (*http.Request, error) {
	log.Tracef("Processing request in handler %q", auth.Name())
	context.Token = getBearerTokenFrom(req)
	if context.Token == "" {
		log.Debugf("Skipping %s as there is no bearer token present", auth.Name())
		return req, nil
	}
	sanitizeHeaders(req)
	json, err := auth.osClient.TokenReview(context.Token)
	if err != nil {
		log.Errorf("Error fetching user info %v", err)
		return req, err
	}
	context.UserName = json.UserName()
	log.Debugf("User is %q", json.UserName())
	if context.UserName != "" {
		req.Header.Set(headerForwardedUser, context.UserName)
	}
	auth.fetchRoles(req, context)
	return req, nil
}

func (auth *authorizationHandler) fetchRoles(req *http.Request, context *handlers.RequestContext) {
	log.Debug("Determining roles...")
	for name, sar := range auth.config.AuthBackEndRoles {
		if allowed, err := auth.osClient.SubjectAccessReview(context.UserName, sar.Namespace, sar.Verb, sar.Resource, sar.ResourceAPIGroup); err == nil {
			log.Debugf("%q for %q SAR: %v", context.UserName, name, allowed)
			if allowed {
				context.Roles = append(context.Roles, name)
				req.Header.Add(headerForwardedRoles, name)
			}
		} else {
			log.Warnf("Unable to evaluate %s SAR for user %s", name, context.UserName)
		}
	}
}

func sanitizeHeaders(req *http.Request) {
	req.Header.Del(headerAuthorization)
}

func getBearerTokenFrom(req *http.Request) string {
	parts := strings.SplitN(req.Header.Get(headerAuthorization), " ", 2)
	if len(parts) > 0 && parts[0] == "Bearer" {
		return parts[1]
	}
	log.Trace("No bearer token found on request. Returning ''")
	return ""
}
