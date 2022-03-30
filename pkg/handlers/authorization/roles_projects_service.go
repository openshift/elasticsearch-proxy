package authorization

import (
	"time"

	"github.com/openshift/elasticsearch-proxy/pkg/apis"

	"github.com/bluele/gcache"
	"github.com/openshift/elasticsearch-proxy/pkg/clients"
	"github.com/openshift/elasticsearch-proxy/pkg/config"
	"github.com/openshift/elasticsearch-proxy/pkg/handlers"
	log "github.com/sirupsen/logrus"
)

var (
	exists = struct{}{}
)

type rolesService struct {
	cache gcache.Cache
}

func NewRolesProjectsService(size int, expiry time.Duration, roleConfig map[string]config.BackendRoleConfig, client clients.OpenShiftClient) *rolesService {
	return &rolesService{
		cache: gcache.New(size).
			LRU().
			Expiration(expiry).
			LoaderFunc(loadFromOpenshift(roleConfig, client)).
			Build(),
	}
}

type rolesProjects struct {
	review   *clients.TokenReview
	roles    map[string]struct{}
	projects []apis.Project
}

func (s *rolesService) getRolesAndProjects(token string) (*rolesProjects, error) {
	v, err := s.cache.Get(token)
	if err != nil {
		return nil, err
	}
	cacheVal := v.(*rolesProjects)
	return cacheVal, nil
}

func loadFromOpenshift(roleConfig map[string]config.BackendRoleConfig, client clients.OpenShiftClient) func(key interface{}) (interface{}, error) {
	return func(key interface{}) (interface{}, error) {
		token := key.(string)
		tokenReview, err := client.TokenReview(token)
		log.Debugf("TokenReview: %v", tokenReview)
		if err != nil {
			log.Errorf("Error fetching user info %v", err)
			return nil, err
		}
		if !tokenReview.Status.Authenticated {
			return nil, handlers.NewError("401", tokenReview.Status.Error)
		}

		username := tokenReview.UserName()
		groups := tokenReview.Groups()
		log.Debugf("User is %q in Groups: %v", username, groups)

		roles := evaluateRoles(client, username, groups, roleConfig)
		projects, err := listProjects(client, token)
		if err != nil {
			return nil, err
		}
		return &rolesProjects{review: tokenReview, roles: roles, projects: projects}, nil
	}
}

func evaluateRoles(client clients.OpenShiftClient, userName string, groups []string, roleConfig map[string]config.BackendRoleConfig) map[string]struct{} {
	roles := map[string]struct{}{}
	for name, sar := range roleConfig {
		if allowed, err := client.SubjectAccessReview(groups, userName, sar.Namespace, sar.Verb, sar.Resource, sar.ResourceAPIGroup); err == nil {
			log.Debugf("%q for %q SAR: %v", userName, name, allowed)
			if allowed {
				roles[name] = exists
			}
		} else {
			log.Warnf("Unable to evaluate %s SAR for user %s", name, userName)
		}
	}
	return roles
}

func listProjects(client clients.OpenShiftClient, token string) ([]apis.Project, error) {
	var namespaces []clients.Namespace
	namespaces, err := client.ListNamespaces(token)
	if err != nil {
		log.Errorf("There was an error fetching projects: %v", err)
		return nil, err
	}
	projects := make([]apis.Project, len(namespaces))
	for i, ns := range namespaces {
		projects[i] = apis.Project{Name: ns.Name()}
	}
	return projects, nil
}
