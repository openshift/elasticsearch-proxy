package authorization

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gotest.tools/assert"

	"errors"
	"time"

	osprojectv1 "github.com/openshift/api/project/v1"
	"github.com/openshift/elasticsearch-proxy/pkg/apis"
	"github.com/openshift/elasticsearch-proxy/pkg/clients"
	"github.com/openshift/elasticsearch-proxy/pkg/config"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	token = "ignored"
)

var _ = Describe("#evaluateRoles", func() {

	It("should only return allowed roles", func() {
		client := &mockOpenShiftClient{sarResponses: map[string]bool{
			"allowed":    true,
			"notallowed": false,
		}}
		backendRoles := map[string]config.BackendRoleConfig{
			"allowed": config.BackendRoleConfig{
				Namespace:        "anamespace",
				Verb:             "allowed",
				Resource:         "someresource",
				ResourceAPIGroup: "",
			},
			"notallowed": config.BackendRoleConfig{
				Namespace:        "anamespace",
				Verb:             "notallowed",
				Resource:         "someresource",
				ResourceAPIGroup: "",
			},
		}
		groups := []string{}
		roles := evaluateRoles(client, "auser", groups, backendRoles)
		Expect(roles).To(Equal(map[string]struct{}{"allowed": struct{}{}}))
	})

})
var _ = Describe("RolesProjectsService", func() {
	Context("#getRolesAndProjects", func() {

		var (
			err              error
			service          *rolesService
			rolesAndProjects *rolesProjects

			newService = func(client clients.OpenShiftClient) *rolesService {
				return NewRolesProjectsService(120, time.Nanosecond, map[string]config.BackendRoleConfig{"key": {}}, client)
			}

			expectValidRolesProjects = func(rolesAndProjects *rolesProjects, err error, expRoles map[string]struct{}) {
				Expect(err).To(BeNil(), "Exp. role failures to not geneate an error")
				Expect(rolesAndProjects.review.UserName()).To(Equal("jdoe"))
				Expect(rolesAndProjects.review.Groups()).To(Equal([]string{"foo", "bar"}))
				Expect(rolesAndProjects.roles).To(Equal(expRoles), "Exp. a set or valid roles")
				Expect(rolesAndProjects.projects).To(Equal([]apis.Project{{Name: "myproject"}}))
			}
		)

		It("should return the error when unable to do a tokenreview", func() {
			service = newService(&mockOpenShiftClient{tokenReviewErr: errors.New("failed to get token")})
			_, err = service.getRolesAndProjects(token)
			Expect(err).To(BeEquivalentTo(errors.New("failed to get token")))
		})
		It("should return a 401 error when token is expired", func() {
			service = newService(&mockOpenShiftClient{tokenReviewStatusErr: "token expired"})
			_, err = service.getRolesAndProjects(token)
			Expect(err).To(BeEquivalentTo(errors.New("got 401 token expired")))
		})
		It("should return an empty role set when subjectaccessreviews fail", func() {
			service = newService(&mockOpenShiftClient{subjectAccessErr: errors.New("review failed")})
			rolesAndProjects, err = service.getRolesAndProjects(token)
			expectValidRolesProjects(rolesAndProjects, err, map[string]struct{}{})
		})
		It("should return the error when unable to retrieve a project list", func() {
			service = newService(&mockOpenShiftClient{projectsErr: errors.New("projects failed")})
			_, err = service.getRolesAndProjects(token)
			Expect(err).To(BeEquivalentTo(errors.New("projects failed")))
		})
		It("should return roles and projects when successful", func() {
			service = newService(&mockOpenShiftClient{})
			rolesAndProjects, err = service.getRolesAndProjects(token)
			expectValidRolesProjects(rolesAndProjects, err, map[string]struct{}{"key": exists})
		})
	})
})

func TestCacheExpiry(t *testing.T) {
	client := &mockOpenShiftClient{}
	duration := time.Millisecond * 50
	s := NewRolesProjectsService(120, duration, map[string]config.BackendRoleConfig{"key": {}}, client)
	s.getRolesAndProjects(token)
	assert.Equal(t, 1, client.tokenReviewCounter)
	s.getRolesAndProjects(token)
	assert.Equal(t, 1, client.tokenReviewCounter)
	time.Sleep(duration)
	s.getRolesAndProjects(token)
	assert.Equal(t, 2, client.tokenReviewCounter)
}

type mockOpenShiftClient struct {
	tokenReviewStatusErr string
	tokenReviewErr       error
	subjectAccessErr     error
	projectsErr          error
	tokenReviewCounter   int
	sarResponses         map[string]bool
}

func (c *mockOpenShiftClient) TokenReview(token string) (*clients.TokenReview, error) {
	c.tokenReviewCounter++
	authenticated := true
	if c.tokenReviewStatusErr != "" {
		authenticated = false
	}
	return &clients.TokenReview{&authenticationv1.TokenReview{
		Status: authenticationv1.TokenReviewStatus{
			Authenticated: authenticated,
			User:          authenticationv1.UserInfo{Username: "jdoe", Groups: []string{"foo", "bar"}},
			Error:         c.tokenReviewStatusErr,
		}},
	}, c.tokenReviewErr
}

func (c *mockOpenShiftClient) SubjectAccessReview(groups []string, user, namespace, verb, resource, apiGroup string) (bool, error) {
	if c.sarResponses != nil {
		if value, ok := c.sarResponses[verb]; ok {
			return value, nil
		}
	}
	return true, c.subjectAccessErr
}

func (c *mockOpenShiftClient) ListNamespaces(token string) ([]clients.Namespace, error) {
	return []clients.Namespace{{Ns: osprojectv1.Project{ObjectMeta: metav1.ObjectMeta{Name: "myproject"}}}}, c.projectsErr
}
