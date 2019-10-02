package security_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/elasticsearch-proxy/pkg/apis/security"
	cl "github.com/openshift/elasticsearch-proxy/pkg/handlers/clusterlogging/types"
	test "github.com/openshift/elasticsearch-proxy/test"
)

func newProjects(projects ...string) []cl.Project {
	result := []cl.Project{}
	for _, project := range projects {
		result = append(result, cl.Project{project, "123abc"})
	}
	return result
}

var _ = Describe("Generating Security roles", func() {
	var (
		docs   security.ACLDocuments
		users  []cl.UserInfo
		result string
		err    error
	)
	BeforeEach(func() {
		docs = security.ACLDocuments{}
		docs.Set(security.NewRoles())
		docs.Set(security.NewRolesMapping())
		users = []cl.UserInfo{
			{
				Username: "user2.bar@email.com",
				Groups:   []string{},
				Projects: newProjects("xyz", "foo.bar"),
			},
			{
				Username: "CN=jdoe,OU=DL IT,OU=User Accounts,DC=example,DC=com",
				Groups:   []string{"myspecialgroup"},
				Projects: newProjects("distinguishedproj"),
			},
		}
		for _, user := range users {
			docs.AddUser(&user, 15)
		}
	})
	Describe("for shared_ops kibana index mode", func() {

		It("should produce a well formed roles.yaml", func() {
			result, err = docs.Roles().ToYaml()
			Expect(err).To(BeNil())
			test.Expect(result).ToMatchYaml(`
			gen_user_4c54bf89fe913f39fc22d76309f80cdc6192928f:
				cluster: [CLUSTER_MONITOR_KIBANA, USER_CLUSTER_OPERATIONS]
				expires: 15
				indices:
					?kibana?4c54bf89fe913f39fc22d76309f80cdc6192928f:
						'*': [INDEX_KIBANA]
					project?distinguishedproj?123abc?*:
						'*': [INDEX_PROJECT]
			gen_user_994a33f6a157ba4a286395f81a4333db1e6cefb6:
				cluster: [CLUSTER_MONITOR_KIBANA, USER_CLUSTER_OPERATIONS]
				expires: 15
				indices:
					?kibana?994a33f6a157ba4a286395f81a4333db1e6cefb6:
						'*': [INDEX_KIBANA]
					project?foo?bar?123abc?*:
						'*': [INDEX_PROJECT]
					project?xyz?123abc?*:
						'*': [INDEX_PROJECT]
			`)
		})

		It("should produce a well formed rolesmapping.yaml", func() {
			result, err = docs.RolesMapping().ToYaml()
			Expect(err).To(BeNil())
			test.Expect(result).ToMatchYaml(`
			gen_user_4c54bf89fe913f39fc22d76309f80cdc6192928f:
				expires: 15
				users: ['CN=jdoe,OU=DL IT,OU=User Accounts,DC=example,DC=com']
			gen_user_994a33f6a157ba4a286395f81a4333db1e6cefb6:
				expires: 15
				users: [user2.bar@email.com]
			`)
		})
	})
})
