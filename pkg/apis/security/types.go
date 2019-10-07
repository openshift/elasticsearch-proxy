package security

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	cl "github.com/openshift/elasticsearch-proxy/pkg/handlers/clusterlogging/types"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

//DocType is the type of security document
type DocType string

const (
	//DocTypeRoles are the security roles
	DocTypeRoles DocType = "roles"

	//DocTypeRolesmapping are the security mappings of users to roles
	DocTypeRolesmapping DocType = "rolesmapping"
)

var epoch = time.Unix(0, 0)

//ACLDocuments are the security documents
type ACLDocuments map[DocType]ACLDocument

//ACLDocument is a specific security document
type ACLDocument interface {
	//Map of entries that are expirable
	Map() map[string]Expirable

	//Remove a permission with the given name
	Remove(name string)

	//ToJson to covert the document to a JSON string
	ToJson() (string, error)

	//Type of the ACLDocument
	Type() DocType

	//Size provides the number of entries
	Size() int

	//Version of the document
	Version() int
}

func (docs *ACLDocuments) List() []ACLDocument {
	entries := []ACLDocument{}
	for _, doc := range *docs {
		entries = append(entries, doc)
	}
	return entries
}

func (docs *ACLDocuments) Set(aclDoc ACLDocument) {
	(*docs)[aclDoc.Type()] = aclDoc
}

func (docs *ACLDocuments) Roles() *Roles {
	if val, exists := (*docs)[DocTypeRoles]; exists {
		r := val.(*Roles)
		if r.roleNames == nil {
			r.roleNames = map[string]Role{}
		}
		return r
	}
	return nil
}

func (docs *ACLDocuments) RolesMapping() *RolesMapping {
	if val, exists := (*docs)[DocTypeRolesmapping]; exists {
		r := val.(*RolesMapping)
		if r.roleNames == nil {
			r.roleNames = map[string]RoleMapping{}
		}
		return r
	}
	return nil
}

//AddUser permissions to the ACL documents
func (docs *ACLDocuments) AddUser(user *cl.UserInfo, expires int64) {
	log.Tracef("Adding permissions for %s to expire at %v", user.Username, expires)
	roleName := roleName(user)
	role := Role{
		ClusterPermissions: Permissions{"CLUSTER_MONITOR_KIBANA", "USER_CLUSTER_OPERATIONS"},
		IndicesPermissions: newSecurityDocumentPermissions(user),
	}
	role.ExpiresInMillis = expires
	docs.Roles().Set(roleName, role)
	rolemapping := RoleMapping{
		Users: []string{user.Username},
	}
	rolemapping.ExpiresInMillis = expires
	docs.RolesMapping().Set(roleName, rolemapping)
}

//ExpirePermissions which are older then now
func (docs *ACLDocuments) ExpirePermissions() {
	log.Debug("Expiring permissions...")
	now := time.Now()
	for docType, aclDoc := range *docs {
		for name, entry := range aclDoc.Map() {
			expire := time.Unix(0, entry.GetExpiresInMillis())
			if expire.After(epoch) && expire.Before(now) {
				log.Tracef("Expiring %s: %s", docType, name)
				aclDoc.Remove(name)
			}
		}
	}
}

type Expirable interface {
	GetExpiresInMillis() int64
}

//Roles are the roles for the ES Cluster
// root
//   roleName:
//     cluster:
//     expires:
//     indices:
//       indexName:
//         docType: [permissions]
type Roles struct {
	roleNames  map[string]Role
	DocVersion int
}

func NewRoles() *Roles {
	return &Roles{roleNames: map[string]Role{}}
}

func (role *Role) GetExpiresInMillis() int64 {
	return role.ExpiresInMillis
}

func (roles *Roles) Version() int {
	return roles.DocVersion
}

func (roles *Roles) Size() int {
	return len(roles.roleNames)
}
func (roles *Roles) Map() map[string]Expirable {
	entries := map[string]Expirable{}
	for name, entry := range roles.roleNames {
		copy := entry
		entries[name] = &copy
	}
	return entries
}

func (roles *Roles) Set(name string, role Role) {
	(roles.roleNames)[name] = role
}

func (roles *Roles) Remove(name string) {
	delete(roles.roleNames, name)
}

func (roles *Roles) Type() DocType {
	return DocTypeRoles
}
func (roles *Roles) ToYaml() (string, error) {
	return toYaml(roles.roleNames)
}

func (roles *Roles) ToJson() (string, error) {
	return ToJson(roles.roleNames)
}

type Role struct {
	ExpiresInMillis    int64            `yaml:"expires,omitempty" json:"expires,omitempty"`
	ClusterPermissions Permissions      `yaml:"cluster,omitempty" json:"cluster,omitempty"`
	IndicesPermissions IndexPermissions `yaml:"indices,omitempty" json:"indices,omitempty"`
}
type Permissions []string

type IndexPermissions map[string]DocumentPermissions

type DocumentPermissions map[string]Permissions

func toYaml(acl interface{}) (string, error) {
	var out []byte
	var err error
	if out, err = yaml.Marshal(acl); err != nil {
		return "", err
	}
	return string(out), nil
}

func ToJson(acl interface{}) (string, error) {
	log.Tracef("Converting acl to json: %+v", acl)
	var out []byte
	var err error
	if out, err = json.Marshal(acl); err != nil {
		return "", err
	}
	resp := string(out)
	log.Tracef("Converted: %s", resp)
	return resp, nil
}

func (roles *Roles) FromJson(acl string) error {
	if err := json.Unmarshal([]byte(acl), roles); err != nil {
		return err
	}
	return nil
}

func (rolesmapping *RolesMapping) FromJson(acl string) error {
	if err := json.Unmarshal([]byte(acl), rolesmapping); err != nil {
		return err
	}
	return nil
}

//Rolesmapping are the mapping of username/groups to roles
// root
//  roleName
//    expires:
//    users:
//    groups:
type RolesMapping struct {
	roleNames  map[string]RoleMapping
	DocVersion int
}

func NewRolesMapping() *RolesMapping {
	return &RolesMapping{roleNames: map[string]RoleMapping{}}
}

func (rolesMapping *RolesMapping) Map() map[string]Expirable {
	entries := map[string]Expirable{}
	for name, entry := range rolesMapping.roleNames {
		copy := entry
		entries[name] = &copy
	}
	return entries
}

func (rolesMapping *RolesMapping) Size() int {
	return len(rolesMapping.roleNames)
}
func (rolesMapping *RolesMapping) Version() int {
	return rolesMapping.DocVersion
}
func (roleMapping *RoleMapping) GetExpiresInMillis() int64 {
	return roleMapping.ExpiresInMillis
}

func (rolesMapping *RolesMapping) Set(name string, rolemapping RoleMapping) {
	(rolesMapping.roleNames)[name] = rolemapping
}
func (rolesMapping *RolesMapping) Remove(name string) {
	delete(rolesMapping.roleNames, name)
}

type RoleMapping struct {
	ExpiresInMillis int64    `yaml:"expires,omitempty" json:"expires,omitempty"`
	Users           []string `yaml:"users,omitempty" json:"users,omitempty"`
}

func (rolesmapping *RolesMapping) Type() DocType {
	return DocTypeRolesmapping
}

func (rolesmapping *RolesMapping) ToYaml() (string, error) {
	return toYaml(rolesmapping.roleNames)
}

func (rolesmapping *RolesMapping) ToJson() (string, error) {
	return ToJson(rolesmapping.roleNames)
}

func newSecurityDocumentPermissions(user *cl.UserInfo) IndexPermissions {
	permissions := IndexPermissions{}
	permissions[fix(kibanaIndexName(user))] = DocumentPermissions{
		"*": Permissions{
			"INDEX_KIBANA",
		},
	}
	for _, project := range user.Projects {
		permissions[fix(projectIndexName(project))] = DocumentPermissions{
			"*": Permissions{
				"INDEX_PROJECT",
			},
		}
	}
	return permissions
}

func fix(indexName string) string {
	return strings.Replace(indexName, ".", "?", -1)
}

func projectIndexName(p cl.Project) string {
	return fmt.Sprintf("project.%s.%s.*", p.Name, p.UUID)
}

func kibanaIndexName(user *cl.UserInfo) string {
	return fmt.Sprintf(".kibana.%s", usernameHash(user))
}

func roleName(user *cl.UserInfo) string {
	return fmt.Sprintf("gen_user_%s", usernameHash(user))
}

func usernameHash(user *cl.UserInfo) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(user.Username)))
}
