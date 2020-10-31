package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/openshift/elasticsearch-proxy/pkg/config"
)

type ContextKey string

const (
	UsernameKey ContextKey = "username"
	ProjectsKey ContextKey = "projects"
	RolesKey    ContextKey = "roles"
	SubjectKey  ContextKey = "subject"
)

type Options struct {
	*config.Options
}

type StructuredError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Error   error  `json:"error,omitempty"`
}

//NewError returns an error with a code and message that can be returned
//as a structured error understandable by Kibana
func NewError(code, message string) error {
	return fmt.Errorf("got %s %s", code, message)
}

func NewStructuredError(err error) StructuredError {
	code := http.StatusInternalServerError
	message := "Internal Error"
	if strings.HasPrefix(err.Error(), "got") {
		parts := strings.Split(err.Error(), " ")
		log.Debugf("Parts %q", parts)
		if len(parts) >= 2 {
			if parsedCode, parseError := strconv.Atoi(parts[1]); parseError == nil {
				code = parsedCode
			} else {
				log.Errorf("Unable to parse response code from response %v", err.Error())
			}
		}
		if len(parts) >= 3 {
			message = strings.Join(parts[2:], " ")
		}
	}
	return StructuredError{
		code,
		message,
		err,
	}
}

//RequestHandler if a function that modifies a request.  Execution occurs
//after authentication but before proxy to upstream
type RequestHandler interface {
	//Process the request and return the modification or error
	Process(req *http.Request) (*http.Request, error)
	//Name of the request handler
	Name() string
}
