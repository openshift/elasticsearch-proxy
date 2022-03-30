package authorization

import (
	"fmt"
	"net/http"

	"github.com/bluele/gcache"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	authenticationapi "k8s.io/api/authentication/v1"

	"github.com/openshift/elasticsearch-proxy/pkg/apis"
	"github.com/openshift/elasticsearch-proxy/pkg/clients"
	"github.com/openshift/elasticsearch-proxy/pkg/config"
	"github.com/openshift/elasticsearch-proxy/pkg/handlers"
)

var _ = Describe("Process", func() {

	var (
		err        error
		req        *http.Request
		handler    *authorizationHandler
		cacheEntry *rolesProjects
	)

	BeforeEach(func() {
		req, _ = http.NewRequest("post", "https://someplace", nil)
		req.Header.Set("X-OCP-NS", "deleteme")
		req.Header.Set("X-Forwarded-Roles", "deleteme")
		handler = &authorizationHandler{
			config: &config.Options{
				AuthBackEndRoles: map[string]config.BackendRoleConfig{
					"roleA": config.BackendRoleConfig{},
					"roleB": config.BackendRoleConfig{},
				},
			},
			fnSubjectExtractor: func(req *http.Request) string {
				return "CN=foo,OU=org-unit,O=org"
			},
		}
	})

	Context("when certs are provided", func() {
		Context("without access token and does not error", func() {
			BeforeEach(func() {
				req, err = handler.Process(req)
				Expect(err).To(BeNil())
			})
			It("should pass the subject as the user", func() {
				Expect(req.Header.Get("X-Forwarded-User")).To(Equal("CN=foo,OU=org-unit,O=org"))
			})
			It("should sanitize the headers", func() {
				Expect(req.Header.Get("Authorization")).To(BeEmpty())
				Expect(req.Header.Get("X-Forwarded-Roles")).To(BeEmpty())
				Expect(req.Header.Get("X-OCP-NS")).To(BeEmpty())
			})
			It("should store subject into the request context", func() {
				Expect(req.Context().Value(handlers.SubjectKey)).To(Equal("CN=foo,OU=org-unit,O=org"))
			})
		})
		Context("with empty bearer token and does not error", func() {
			It("should pass the subject as the user", func() {
				req.Header.Set("Authorization", "Bearer  ")
				_, err = handler.Process(req)
				Expect(err).To(BeNil())
				Expect(req.Header.Get("X-Forwarded-User")).To(Equal("CN=foo,OU=org-unit,O=org"))
			})
		})
		Context("and it returns an empty subject", func() {
			It("should error", func() {
				handler.fnSubjectExtractor = func(req *http.Request) string {
					return "  "
				}
				req, err = handler.Process(req)
				Expect(err).To(Not(BeNil()))
			})
		})
	})

	Context("when certs are not provided and it does not error", func() {
		var otherCacheEntry *rolesProjects
		BeforeEach(func() {
			req.Header.Set("Authorization", "Bearer somebearertoken")
			cacheEntry = &rolesProjects{
				review: &clients.TokenReview{
					&authenticationapi.TokenReview{
						Status: authenticationapi.TokenReviewStatus{
							User: authenticationapi.UserInfo{
								Username: "myname",
							},
						},
					},
				},
				roles: map[string]struct{}{
					"roleA":        struct{}{},
					"roleB":        struct{}{},
					"admin_reader": struct{}{},
				},
				projects: []apis.Project{
					apis.Project{
						Name: "projecta",
					},
					apis.Project{
						Name: "projectb",
					},
				},
			}
			otherCacheEntry = &rolesProjects{
				review: &clients.TokenReview{
					&authenticationapi.TokenReview{
						Status: authenticationapi.TokenReviewStatus{
							User: authenticationapi.UserInfo{
								Username: "other",
							},
						},
					},
				},
			}
			handler.cache = &rolesService{
				cache: gcache.New(2).
					LRU().
					LoaderFunc(func(key interface{}) (interface{}, error) {
						if key == "1234" {
							return otherCacheEntry, nil
						}
						return cacheEntry, nil
					}).
					Build(),
			}
			req, err = handler.Process(req)
			Expect(err).To(BeNil())
		})
		Context("and has a forwarded access and bearer token", func() {
			It("should pass the subject as the user", func() {
				req, err = handler.Process(req)
				Expect(err).To(BeNil())

				req.Header.Set("Authorization", "Bearer  abc123")
				req.Header.Set("X-Forwarded-Access-Token", "1234")
				req, err = handler.Process(req)
				Expect(err).To(BeNil())
				Expect(req.Header.Get("Authorization")).To(BeEmpty())
				Expect(req.Header.Get("X-Forwarded-Access-Token")).To(BeEmpty())
				Expect(req.Header.Get("X-Forwarded-User")).To(Equal("other"))
				Expect(req.Context().Value(handlers.UsernameKey)).To(Equal("other"))
				Expect(req.Context().Value(handlers.ProjectsKey)).To(BeEmpty())
				Expect(req.Context().Value(handlers.RolesKey)).To(BeEmpty())
			})
		})
		Context("and has a bearer token only", func() {
			It("should add forward user to the request", func() {
				Expect(req.Header.Get("X-Forwarded-User")).To(Equal("myname"))
			})
			It("should add forwarded for header to the request", func() {
				Expect(req.Header.Get("X-Forwarded-For")).To(Equal("localhost"))
			})
			It("should add role headers to the request", func() {
				entries, ok := req.Header["X-Forwarded-Roles"]
				Expect(ok).To(BeTrue(), fmt.Sprintf("Expected a user's roles to be added to be proxy headers: %v", req.Header))
				Expect(entries).To(Equal([]string{"roleA,roleB"}))
			})
			It("should add a user's projects to the request", func() {
				entries, ok := req.Header["X-Ocp-Ns"]
				Expect(ok).To(BeTrue(), fmt.Sprintf("Expected a user's projects to be added to be proxy headers: %v", req.Header))
				Expect(entries).To(Equal([]string{"\"projecta\",\"projectb\""}))
			})
			It("should store username, roles and project in request context", func() {
				wantProjects := []apis.Project{
					{Name: "projecta"},
					{Name: "projectb"},
				}
				wantRoles := []string{"roleA", "roleB"}
				Expect(req.Context().Value(handlers.UsernameKey)).To(Equal("myname"))
				Expect(req.Context().Value(handlers.ProjectsKey)).To(ConsistOf(wantProjects))
				Expect(req.Context().Value(handlers.RolesKey)).To(ConsistOf(wantRoles))
			})

			Context("and has the spec'd default role with predefined roles", func() {

				BeforeEach(func() {
					req.Header.Set("Authorization", "Bearer somebearertoken")
					handler.config.AuthAdminRole = ""
					handler.config.AuthDefaultRole = "project_reader"
					req, err = handler.Process(req)
					Expect(err).To(BeNil())
				})

				It("should not update the request to include the default role", func() {
					entries, ok := req.Header["X-Forwarded-Roles"]
					Expect(ok).To(BeTrue(), fmt.Sprintf("Expected 'X-Forwarded-Roles' in the headers: %v", req.Header))
					Expect(entries).To(Equal([]string{"roleA,roleB"}), "Exp. to not apply the default role")
				})

				It("should not store default role in request context", func() {
					wantRoles := []string{"roleA", "roleB"}
					Expect(req.Context().Value(handlers.RolesKey)).To(ConsistOf(wantRoles))
				})
			})

			Context("and has the spec'd default role without predefined roles", func() {

				BeforeEach(func() {
					cacheEntry.roles = map[string]struct{}{}
					req.Header.Set("Authorization", "Bearer somebearertoken")

					handler.config.AuthAdminRole = ""
					handler.config.AuthDefaultRole = "project_reader"
					req, err = handler.Process(req)
					Expect(err).To(BeNil())
				})

				It("should update the request to only include the default role", func() {
					entries, ok := req.Header["X-Forwarded-Roles"]
					Expect(ok).To(BeTrue(), fmt.Sprintf("Expected 'X-Forwarded-Roles' in the headers: %v", req.Header))
					Expect(entries).To(Equal([]string{"project_reader"}), "Exp. to the default role to apply")
				})

				It("should store default role in request context", func() {
					wantRoles := []string{"project_reader"}
					Expect(req.Context().Value(handlers.RolesKey)).To(ConsistOf(wantRoles))
				})
			})

			Context("and has the spec'd admin role", func() {

				BeforeEach(func() {
					req.Header.Set("Authorization", "Bearer somebearertoken")
					handler.config.AuthAdminRole = "admin_reader"
					handler.config.AuthBackEndRoles["admin_reader"] = config.BackendRoleConfig{}
					req, err = handler.Process(req)
					Expect(err).To(BeNil())
				})

				It("should update the request to only include the admin role", func() {
					entries, ok := req.Header["X-Forwarded-Roles"]
					Expect(ok).To(BeTrue(), fmt.Sprintf("Expected 'X-Forwarded-Roles' in the headers: %v", req.Header))
					Expect(entries).To(Equal([]string{"admin_reader"}), "Exp. to find 'admin_reader' in the roles")
				})

				It("should store admin role in request context", func() {
					wantRoles := []string{"admin_reader"}
					Expect(req.Context().Value(handlers.RolesKey)).To(ConsistOf(wantRoles))
				})
			})
		})
	})

})
