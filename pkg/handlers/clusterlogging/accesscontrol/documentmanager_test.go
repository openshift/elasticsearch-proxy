package accesscontrol

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/elasticsearch-proxy/pkg/apis/security"
	"github.com/openshift/elasticsearch-proxy/pkg/config"
	cl "github.com/openshift/elasticsearch-proxy/pkg/handlers/clusterlogging/types"
)

type securityClientFake struct {
	docs     *security.ACLDocuments
	loadErr  error
	writeErr error
}

func (sc *securityClientFake) FetchACLs() (*security.ACLDocuments, error) {
	if sc.loadErr != nil {
		return nil, sc.loadErr
	}
	return sc.docs, nil
}

func (sc *securityClientFake) FlushACL(doc security.ACLDocuments) error {
	if sc.writeErr != nil {
		return sc.writeErr
	}
	return nil
}

var _ = Describe("DocumentManager", func() {

	var (
		dm                 *DocumentManager
		user               *cl.UserInfo
		fakeSecurityClient *securityClientFake
	)
	BeforeEach(func() {
		dm = &DocumentManager{}
		user = &cl.UserInfo{Groups: []string{"foo"}}
		fakeSecurityClient = &securityClientFake{}
		dm.securityClient = fakeSecurityClient
	})

	Describe("when the infra group name is ''", func() {
		It("a user should not evaluate as an infra group member", func() {
			Expect(dm.isInfraGroupMember(user)).Should(BeFalse())
		})
	})

	Describe("when the infra group name is a value", func() {
		It("a user should evaluate as an infra group member if they are in the group", func() {
			dm.Options = config.Options{}
			dm.Options.InfraRoleName = "foo"
			Expect(dm.isInfraGroupMember(user)).Should(BeTrue())
		})
	})

	Describe("when trySyncACL", func() {

		BeforeEach(func() {
			fakeSecurityClient.docs = &security.ACLDocuments{
				security.DocTypeRoles:        security.NewRoles(),
				security.DocTypeRolesmapping: security.NewRolesMapping(),
			}
		})

		It("should back-off when there is a load failure", func() {
			fakeSecurityClient.loadErr = fmt.Errorf("an error %s", "")
			Expect(dm.trySyncACL(user)).Should(BeFalse())
		})
		It("should back-off when there is a write failure", func() {
			fakeSecurityClient.writeErr = fmt.Errorf("an error %s", "")
			Expect(dm.trySyncACL(user)).Should(BeFalse())
		})

		It("should succeed when there are no errors", func() {
			Expect(dm.trySyncACL(user)).Should(BeTrue())
			Expect(fakeSecurityClient.docs.Roles().Size()).To(Equal(1))
			Expect(fakeSecurityClient.docs.RolesMapping().Size()).To(Equal(1))
		})
	})

	Describe("when SyncACL", func() {

		Describe("and user is an infra group member", func() {

			It("should skip additional processing", func() {
				dm.Options = config.Options{}
				dm.Options.InfraRoleName = "foo"
				fakeSecurityClient.loadErr = fmt.Errorf("an error %s", "")
				sleeper := func(t time.Duration) {
					//This function should not be calledc
					Expect(true).Should(BeFalse())
				}
				dm.fnSleeper = sleeper
				dm.SyncACL(user)
			})

		})
	})

})
