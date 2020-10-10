package proxy

import (
	log "github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NewResponseWriter", func() {

	var (
		rw = &responseLogger{}
	)

	Context("when LOG_LEVEL >= debug", func() {
		It("should return a responseLogger", func() {
			log.SetLevel(log.TraceLevel)
			act := NewResponseWriter(rw)
			Expect(act).To(Not(BeIdenticalTo(rw)))
		})
	})
	Context("when LOG_LEVEL < debug", func() {
		It("should return the http.ResponseWriter it was given", func() {
			log.SetLevel(log.InfoLevel)
			Expect(NewResponseWriter(rw)).To(BeIdenticalTo(rw))
		})
	})

})
