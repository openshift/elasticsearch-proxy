package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"log"

	"github.com/openshift/elasticsearch-proxy/pkg/config"
)

type RequestContext struct {
	Token    string
	UserName string
	Groups   []string
	Roles    []string
}

type Options struct {
	*config.Options
}

type StructuredError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Error   error  `json:"error,omitempty"`
}

func NewStructuredError(err error) StructuredError {
	code := http.StatusInternalServerError
	message := "Internal Error"
	if strings.HasPrefix(err.Error(), "got") {
		parts := strings.Split(err.Error(), " ")
		log.Printf("Parts %q", parts)
		if len(parts) >= 2 {
			if parsedCode, parseError := strconv.Atoi(parts[1]); parseError == nil {
				code = parsedCode
			} else {
				log.Printf("Unable to parse response code from response %v", err.Error())
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

//FnHandlerRequest is a function that can process a request
type FnHandlerRequest func(req *http.Request, context *RequestContext) (*http.Request, error)

//RequestHandler if a function that modifies a request.  Execution occurs
//after authentication but before proxy to upstream
type RequestHandler interface {
	//Process the request and return the modification or error
	Process(req *http.Request, context *RequestContext) (*http.Request, error)
	//Name of the request handler
	Name() string
}

//SimpleRequestHandler is a simple container to modify requests
type SimpleRequestHandler struct {
	name      string
	processor FnHandlerRequest
}

//Name of the requesthandler
func (h *SimpleRequestHandler) Name() string {
	return h.name
}

//Process the Request
func (h *SimpleRequestHandler) Process(req *http.Request, context *RequestContext) (*http.Request, error) {
	return h.processor(req, context)
}

//NewRequestHandler creates a named wrapper to a requesthandler function
func NewRequestHandler(name string, handler FnHandlerRequest) *SimpleRequestHandler {
	return &SimpleRequestHandler{
		name,
		handler,
	}
}
