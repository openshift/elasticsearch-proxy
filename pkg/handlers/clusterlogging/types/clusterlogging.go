package types

//Project is a simple representation of an OpeShift project
type Project struct {
	Name string
	UUID string
}

//UserInfo is a simple representation of an OpenShift User
type UserInfo struct {
	Username string
	Groups   []string
	Projects []Project
}
