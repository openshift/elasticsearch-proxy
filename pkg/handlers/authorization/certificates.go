package authorization

import (
	"net/http"
)

//certCNExtractor takes a request and extracts the CN from the certificates
type certCNExtractor func(req *http.Request) string

func defaultCertCNExtractor(req *http.Request) string {
	if req.TLS != nil && len(req.TLS.VerifiedChains) > 0 && len(req.TLS.VerifiedChains[0]) > 0 {
		return req.TLS.VerifiedChains[0][0].Subject.CommonName
	}
	return ""
}
