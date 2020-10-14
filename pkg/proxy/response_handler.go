package proxy

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

type responseLogger struct {
	rw http.ResponseWriter
}

func (rl *responseLogger) Header() http.Header {
	log.Debugf("Response Write header %v", rl.rw.Header())
	return rl.rw.Header()
}

func (rl *responseLogger) Write(content []byte) (int, error) {
	log.Debugf("Response Write content %v", string(content))
	length, err := rl.rw.Write(content)
	log.Debugf("Response Write %d, %v", length, err)
	return length, err
}

func (rl *responseLogger) WriteHeader(statusCode int) {
	log.Debugf("Response statusCode %d", statusCode)
	rl.rw.WriteHeader(statusCode)
}

func NewResponseWriter(rw http.ResponseWriter) http.ResponseWriter {
	if log.GetLevel() >= log.DebugLevel {
		return &responseLogger{
			rw,
		}
	}
	return rw
}
