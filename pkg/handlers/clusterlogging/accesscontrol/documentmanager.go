package accesscontrol

import (
	"time"

	"github.com/openshift/elasticsearch-proxy/pkg/apis/security"
	"github.com/openshift/elasticsearch-proxy/pkg/clients"
	"github.com/openshift/elasticsearch-proxy/pkg/config"
	cl "github.com/openshift/elasticsearch-proxy/pkg/handlers/clusterlogging/types"
	log "github.com/sirupsen/logrus"
)

//DocumentManager understands how to load and sync ACL documents
type DocumentManager struct {
	config.Options
	securityClient clients.SecurityClient
	fnSleeper      func(time.Duration)
}

//NewDocumentManager creates an instance or returns error
func NewDocumentManager(config config.Options) (*DocumentManager, error) {
	log.Tracef("Instantiating a new document manager using: %+v", config)
	sgClient, err := clients.NewESSecurityClient(config)
	if err != nil {
		return nil, err
	}
	sleeper := func(t time.Duration) {
		time.Sleep(t)
	}
	return &DocumentManager{
		config,
		sgClient,
		sleeper,
	}, nil
}

//SyncACL to include the given UserInfo
func (dm *DocumentManager) SyncACL(userInfo *cl.UserInfo) {
	log.Debugf("SyncACL for %+v", userInfo)
	if dm.isInfraGroupMember(userInfo) {
		log.Debugf("Skipping sync of ACLs for infragroup member %s. Permissions are assumed to be static", userInfo.Username)
		return
	}

	//TODO: there must be a better way to do this.  Is Channels
	//from small call location above better?
	for delay := range []int{1, 1, 2, 3, 5, 8} {
		if !dm.trySyncACL(userInfo) {
			log.Debugf("Unable to sync ACLs, sleeping for %q seconds...", delay)
			dm.fnSleeper(time.Duration(delay) * time.Second)
		} else {
			break
		}
	}
}

func (dm *DocumentManager) trySyncACL(userInfo *cl.UserInfo) bool {
	log.Debugf("trySyncACL for %+v", userInfo)
	docs, err := dm.loadACL()
	if err != nil {
		log.Warnf("Unable to load ACL docs: %v", err)
		return false
	}
	docs.ExpirePermissions()
	docs.AddUser(userInfo, nextExpireTime(dm.ExtConfig.PermissionExpirationSeconds))
	if err = dm.writeACL(*docs); err != nil {
		log.Debugf("Error writing ACL doc: %v", err)
		return false
	}
	return true
}

func (dm *DocumentManager) writeACL(docs security.ACLDocuments) error {
	log.Debug("Writing ACLs...")
	if err := dm.securityClient.FlushACL(docs); err != nil {
		return err
	}
	return nil
}

func (dm *DocumentManager) loadACL() (*security.ACLDocuments, error) {
	log.Debug("Loading ACLs...")
	//TODO work on mget of roles/mappings
	docs, err := dm.securityClient.FetchACLs()
	if err != nil {
		return nil, err
	}
	log.Debugf("Loaded ACLs: %v", docs)
	return docs, nil
}

func (dm *DocumentManager) isInfraGroupMember(user *cl.UserInfo) bool {
	for _, group := range user.Groups {
		if group == dm.ExtConfig.InfraRoleName {
			log.Tracef("%s is a member of the InfraGroup (%s)", user.Username, dm.ExtConfig.InfraRoleName)
			return true
		}
	}
	return false
}

func nextExpireTime(expire int64) int64 {
	return time.Now().Unix() + expire
}
