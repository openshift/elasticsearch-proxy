package types

import (
	"flag"
	"fmt"
)

//ExtConfig defines configuration the proxy may use to make
//decisions (e.g. role name)
type ExtConfig struct {
	//RawKibanaIndexMode
	RawKibanaIndexMode string `flag:"cl-kibana-index-mode"`

	//KibanaIndexMode enum
	KibanaIndexMode KibanaIndexMode

	//InfraRoleName is the groupname for which a user should be considered an
	//administrator and will be granted the ocp_admin_role
	InfraRoleName string `flag:"cl-infra-role-name"`

	//PermissionExpirationSeconds  the time when permissions expire
	PermissionExpirationSeconds int64 `flag:"cl-permissions-expire-seconds"`
}

//KibanaIndexMode is the mode the proxy uses to generate a user's kibana index
type KibanaIndexMode string

const (
	//KibanaIndexModeSharedOps all users of the InfraGroupName will share a common Kibana index
	KibanaIndexModeSharedOps KibanaIndexMode = "sharedOps"
)

var (
	DefaultPermissionExpirationSeconds int64 = 2 * 60 //2 minutes
)

func RegisterFlagSets(flagSet *flag.FlagSet) {
	flagSet.String("cl-kibana-index-mode", "", "The Kibana Index mode to rewrite the kibana index")
	flagSet.String("cl-infra-role-name", "", "The backend role that is considered an infra role")
	flagSet.Int64("cl-permissions-expire-seconds", DefaultPermissionExpirationSeconds, "The time when permissions expire")
}

func ParseKibanaIndexMode(value string) (KibanaIndexMode, error) {
	if KibanaIndexModeSharedOps == KibanaIndexMode(value) {
		return KibanaIndexModeSharedOps, nil
	}
	return "", fmt.Errorf("Unsupported kibanaIndexMode %q", value)
}
