package config

import (
	"encoding/json"
)

type AuthConfig struct {
	RawAuthBackEndRole []string `flag:"auth-backend-role"`
	AuthBackEndRoles   map[string]BackendRoleConfig
}

//BackendRoleConfig is for executing a SAR against the API server
type BackendRoleConfig struct {
	Namespace        string `json:"namespace,omitempty"`
	Verb             string `json:"verb,omitempty"`
	Resource         string `json:"resource,omitempty"`
	ResourceAPIGroup string `json:"resourceAPIGroup,omitempty"`
}

func parseBackendRoleConfig(value string) (*BackendRoleConfig, error) {
	roleConfig := &BackendRoleConfig{}
	if err := json.Unmarshal([]byte(value), roleConfig); err != nil {
		return nil, err
	}
	return roleConfig, nil
}
