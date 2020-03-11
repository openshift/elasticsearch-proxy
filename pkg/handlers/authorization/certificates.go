package authorization

import (
	"net/http"
)

//certSubjectExtractor takes a request and extracts the subject from the certificates
//in RFC 2253 Distinguished Names syntax.
type certSubjectExtractor func(req *http.Request) string

func defaultCertSubjectExtractor(req *http.Request) string {
	if req.TLS != nil && len(req.TLS.VerifiedChains) > 0 && len(req.TLS.VerifiedChains[0]) > 0 {
		return req.TLS.VerifiedChains[0][0].Subject.String()
	}
	return ""
}
