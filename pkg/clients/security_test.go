package clients

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/elasticsearch-proxy/pkg/apis/security"
)

const (
	rolesVersion        = 174
	rolesmappingVersion = 175
)

var (
	encodedRoles        = map[string]interface{}{"roles": "ewogICJzZ19yb2xlX3JzeXNsb2ciIDogewogICAgImNsdXN0ZXIiIDogWwogICAgICAiQ0xVU1RFUl9NT05JVE9SIiwKICAgICAgImluZGljZXM6ZGF0YS93cml0ZS9idWxrIgogICAgXSwKICAgICJpbmRpY2VzIiA6IHsKICAgICAgIioiIDogewogICAgICAgICIqIiA6IFsKICAgICAgICAgICJDUlVEIiwKICAgICAgICAgICJDUkVBVEVfSU5ERVgiCiAgICAgICAgXQogICAgICB9CiAgICB9CiAgfSwKICAic2dfcm9sZV9qYWVnZXIiIDogewogICAgImNsdXN0ZXIiIDogWwogICAgICAiU0VBUkNIIiwKICAgICAgIkNMVVNURVJfTU9OSVRPUiIsCiAgICAgICJpbmRpY2VzOmRhdGEvd3JpdGUvYnVsayIKICAgIF0sCiAgICAiaW5kaWNlcyIgOiB7CiAgICAgICIqamFlZ2VyLXNlcnZpY2UtKiIgOiB7CiAgICAgICAgIioiIDogWwogICAgICAgICAgIlJFQUQiLAogICAgICAgICAgIkNSVUQiLAogICAgICAgICAgIlNFQVJDSCIsCiAgICAgICAgICAiTUFOQUdFIiwKICAgICAgICAgICJDUkVBVEVfSU5ERVgiCiAgICAgICAgXQogICAgICB9LAogICAgICAiKmphZWdlci1zcGFuLSoiIDogewogICAgICAgICIqIiA6IFsKICAgICAgICAgICJSRUFEIiwKICAgICAgICAgICJDUlVEIiwKICAgICAgICAgICJTRUFSQ0giLAogICAgICAgICAgIk1BTkFHRSIsCiAgICAgICAgICAiQ1JFQVRFX0lOREVYIgogICAgICAgIF0KICAgICAgfSwKICAgICAgIipqYWVnZXItc3Bhbi1hcmNoaXZlIiA6IHsKICAgICAgICAiKiIgOiBbCiAgICAgICAgICAiUkVBRCIsCiAgICAgICAgICAiQ1JVRCIsCiAgICAgICAgICAiU0VBUkNIIiwKICAgICAgICAgICJNQU5BR0UiLAogICAgICAgICAgIkNSRUFURV9JTkRFWCIKICAgICAgICBdCiAgICAgIH0sCiAgICAgICIqamFlZ2VyLXNwYW4tYXJjaGl2ZS0qIiA6IHsKICAgICAgICAiKiIgOiBbCiAgICAgICAgICAiUkVBRCIsCiAgICAgICAgICAiQ1JVRCIsCiAgICAgICAgICAiU0VBUkNIIiwKICAgICAgICAgICJNQU5BR0UiLAogICAgICAgICAgIkNSRUFURV9JTkRFWCIKICAgICAgICBdCiAgICAgIH0KICAgIH0KICB9LAogICJzZ19yb2xlX3Byb21ldGhldXMiIDogewogICAgImNsdXN0ZXIiIDogWwogICAgICAiTUVUUklDUyIKICAgIF0sCiAgICAiaW5kaWNlcyIgOiB7CiAgICAgICIqIiA6IHsKICAgICAgICAiKiIgOiBbCiAgICAgICAgICAiaW5kaWNlczptb25pdG9yKiIKICAgICAgICBdCiAgICAgIH0KICAgIH0KICB9LAogICJzZ19yb2xlX2N1cmF0b3IiIDogewogICAgImNsdXN0ZXIiIDogWwogICAgICAiTUFOQUdFIiwKICAgICAgIkNMVVNURVJfTU9OSVRPUiIKICAgIF0sCiAgICAiaW5kaWNlcyIgOiB7CiAgICAgICIqIiA6IHsKICAgICAgICAiKiIgOiBbCiAgICAgICAgICAiUkVBRCIsCiAgICAgICAgICAiTUFOQUdFIgogICAgICAgIF0KICAgICAgfQogICAgfQogIH0sCiAgInNnX3JvbGVfa2liYW5hIiA6IHsKICAgICJjbHVzdGVyIiA6IFsKICAgICAgIkNMVVNURVJfQ09NUE9TSVRFX09QUyIsCiAgICAgICJDTFVTVEVSX01PTklUT1IiCiAgICBdLAogICAgImluZGljZXMiIDogewogICAgICAiP2tpYmFuYSIgOiB7CiAgICAgICAgIioiIDogWwogICAgICAgICAgIklORElDRVNfQUxMIgogICAgICAgIF0KICAgICAgfQogICAgfQogIH0sCiAgInNnX3JvbGVfZmx1ZW50ZCIgOiB7CiAgICAiY2x1c3RlciIgOiBbCiAgICAgICJDTFVTVEVSX01PTklUT1IiLAogICAgICAiaW5kaWNlczpkYXRhL3dyaXRlL2J1bGsiCiAgICBdLAogICAgImluZGljZXMiIDogewogICAgICAiKiIgOiB7CiAgICAgICAgIioiIDogWwogICAgICAgICAgIkNSVUQiLAogICAgICAgICAgIkNSRUFURV9JTkRFWCIKICAgICAgICBdCiAgICAgIH0KICAgIH0KICB9LAogICJzZ19wcm9qZWN0X29wZXJhdGlvbnMiIDogewogICAgImluZGljZXMiIDogewogICAgICAiKj8qPyoiIDogewogICAgICAgICIqIiA6IFsKICAgICAgICAgICJSRUFEIiwKICAgICAgICAgICJpbmRpY2VzOmFkbWluL3ZhbGlkYXRlL3F1ZXJ5KiIsCiAgICAgICAgICAiaW5kaWNlczphZG1pbi9nZXQqIiwKICAgICAgICAgICJpbmRpY2VzOmFkbWluL21hcHBpbmdzL2ZpZWxkcy9nZXQqIgogICAgICAgIF0KICAgICAgfSwKICAgICAgIj9vcGVyYXRpb25zPyoiIDogewogICAgICAgICIqIiA6IFsKICAgICAgICAgICJSRUFEIiwKICAgICAgICAgICJpbmRpY2VzOmFkbWluL3ZhbGlkYXRlL3F1ZXJ5KiIsCiAgICAgICAgICAiaW5kaWNlczphZG1pbi9nZXQqIiwKICAgICAgICAgICJpbmRpY2VzOmFkbWluL21hcHBpbmdzL2ZpZWxkcy9nZXQqIgogICAgICAgIF0KICAgICAgfQogICAgfQogIH0sCiAgInNnX3JvbGVfYWRtaW4iIDogewogICAgImNsdXN0ZXIiIDogWwogICAgICAiQ0xVU1RFUl9BTEwiCiAgICBdLAogICAgImluZGljZXMiIDogewogICAgICAiKiIgOiB7CiAgICAgICAgIioiIDogWwogICAgICAgICAgIkFMTCIKICAgICAgICBdCiAgICAgIH0KICAgIH0KICB9Cn0="}
	encodedRolesmapping = map[string]interface{}{"rolesmapping": "ewogICJzZ19yb2xlX3JzeXNsb2ciIDogewogICAgInVzZXJzIiA6IFsKICAgICAgIkNOPXN5c3RlbS5sb2dnaW5nLnJzeXNsb2csT1U9T3BlblNoaWZ0LE89TG9nZ2luZyIKICAgIF0KICB9LAogICJzZ19yb2xlX2phZWdlciIgOiB7CiAgICAidXNlcnMiIDogWwogICAgICAiQ049dXNlci5qYWVnZXIsT1U9T3BlblNoaWZ0LE89TG9nZ2luZyIKICAgIF0sCiAgICAiYmFja2VuZHJvbGVzIiA6IFsKICAgICAgImphZWdlciIKICAgIF0KICB9LAogICJzZ19yb2xlX3Byb21ldGhldXMiIDogewogICAgInVzZXJzIiA6IFsgXSwKICAgICJiYWNrZW5kcm9sZXMiIDogWwogICAgICAicHJvbWV0aGV1cyIKICAgIF0KICB9LAogICJzZ19yb2xlX2N1cmF0b3IiIDogewogICAgInVzZXJzIiA6IFsKICAgICAgIkNOPXN5c3RlbS5sb2dnaW5nLmN1cmF0b3IsT1U9T3BlblNoaWZ0LE89TG9nZ2luZyIKICAgIF0KICB9LAogICJzZ19yb2xlX2tpYmFuYSIgOiB7CiAgICAidXNlcnMiIDogWwogICAgICAiQ049c3lzdGVtLmxvZ2dpbmcua2liYW5hLE9VPU9wZW5TaGlmdCxPPUxvZ2dpbmciCiAgICBdCiAgfSwKICAic2dfcm9sZV9mbHVlbnRkIiA6IHsKICAgICJ1c2VycyIgOiBbCiAgICAgICJDTj1zeXN0ZW0ubG9nZ2luZy5mbHVlbnRkLE9VPU9wZW5TaGlmdCxPPUxvZ2dpbmciCiAgICBdCiAgfSwKICAic2dfcm9sZV9hZG1pbiIgOiB7CiAgICAidXNlcnMiIDogWwogICAgICAiQ049c3lzdGVtLmFkbWluLE9VPU9wZW5TaGlmdCxPPUxvZ2dpbmciCiAgICBdCiAgfQp9"}
)

type elasticsearchClientFake struct {
	err          error
	deleteErr    error
	response     string
	mgetResponse *MGetResponse
	payload      string
	requestPath  string
}

func (es *elasticsearchClientFake) Get(index, docType, id string) (string, error) {
	if es.err != nil {
		return "", es.err
	}
	return es.response, nil
}
func (es *elasticsearchClientFake) MGet(index string, items MGetRequest) (*MGetResponse, error) {
	if es.err != nil {
		return nil, es.err
	}
	return es.mgetResponse, nil
}
func (es *elasticsearchClientFake) Delete(index, docType, id string) (string, error) {
	if es.deleteErr != nil {
		return "", es.deleteErr
	}
	return es.response, nil
}
func (es *elasticsearchClientFake) Index(index, docType, id, body string, version int) (string, error) {
	if es.err != nil {
		return "", es.err
	}
	es.payload = body
	es.requestPath = fmt.Sprintf("%s/%s/%s?version=%d", index, docType, id, version)
	return es.response, nil
}

var _ = Describe("SecurityClient", func() {

	var (
		sc         *DefaultESSecurityClient
		fakeClient *elasticsearchClientFake
		err        error
		docs       *security.ACLDocuments
	)
	BeforeEach(func() {
		fakeClient = &elasticsearchClientFake{}
		sc = &DefaultESSecurityClient{fakeClient}
	})

	Describe("#FetchACLs", func() {

		It("Should return an error when the client errors", func() {

			fakeClient.err = fmt.Errorf("fake error%s", "")

			_, err := sc.FetchACLs()
			Expect(err).To(Not(BeNil()))
		})

		Describe("when no client errors", func() {
			BeforeEach(func() {
				fakeClient.mgetResponse = &MGetResponse{
					Docs: []MGetResponseItem{
						{
							Index:   ".searchguard",
							Version: rolesVersion,
							Found:   true,
							Source:  encodedRoles,
							MGetItem: MGetItem{
								Type: DocType,
								Id:   string(security.DocTypeRoles),
							},
						},
						{
							Index:   ".searchguard",
							Version: rolesmappingVersion,
							Found:   true,
							Source:  encodedRolesmapping,
							MGetItem: MGetItem{
								Type: DocType,
								Id:   string(security.DocTypeRolesmapping),
							},
						},
					},
				}
				docs, err = sc.FetchACLs()
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("should return rolemappings", func() {
				Expect(docs.RolesMapping()).To(Not(BeNil()))
				Expect(docs.Roles().Version()).To(Equal(rolesVersion))
			})
			It("should return roles", func() {
				Expect(docs.Roles()).To(Not(BeNil()))
				Expect(docs.RolesMapping().Version()).To(Equal(rolesmappingVersion))
			})
		})
	})

	Describe("#FlushACL", func() {
		BeforeEach(func() {
			roles := security.NewRoles()
			roles.DocVersion = rolesVersion
			rolesmapping := security.NewRolesMapping()
			rolesmapping.DocVersion = rolesmappingVersion
			docs = &security.ACLDocuments{
				security.DocTypeRoles:        roles,
				security.DocTypeRolesmapping: rolesmapping,
			}
		})
		Describe("when error writing documents", func() {
			It("should return the error", func() {
				fakeClient.err = fmt.Errorf("fake error%s", "")
				err = sc.FlushACL(*docs)
				Expect(err).To(HaveOccurred())
			})
		})
		Describe("when error invalidating cache", func() {
			It("should return the error", func() {
				fakeClient.deleteErr = fmt.Errorf("fake error%s", "")
				err = sc.FlushACL(*docs)
				Expect(err).To(HaveOccurred())
			})
		})
		Describe("when successful", func() {
			BeforeEach(func() {
				delete(map[security.DocType]security.ACLDocument(*docs), security.DocTypeRoles)
				err = sc.FlushACL(*docs)
			})
			It("should not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("should include the document versions", func() {
				act := &map[string]interface{}{}
				err = json.Unmarshal([]byte(fakeClient.payload), act)
				Expect(err).NotTo(HaveOccurred())
				Expect(*act).Should(HaveKey("rolesmapping"))
				Expect(fakeClient.requestPath).Should(HaveSuffix(fmt.Sprintf("?version=%v", rolesmappingVersion)))
			})
		})
	})
})
