package config_test

import (
	"net/url"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/elasticsearch-proxy/pkg/config"
	cltypes "github.com/openshift/elasticsearch-proxy/pkg/handlers/clusterlogging/types"
)

func errorMessage(msgs ...string) string {
	result := make([]string, 0)
	result = append(result, "Invalid configuration:")
	result = append(result, msgs...)
	return strings.Join(result, "\n  ")
}

var _ = Describe("Initializing Config options", func() {

	Describe("when defining tls-client-ca without key or certs", func() {
		It("should fail", func() {
			args := []string{"--tls-client-ca=/foo/bar"}
			options, err := config.Init(args)
			Expect(options).Should(BeNil())

			Expect(err.Error()).Should(
				Equal(errorMessage("tls-client-ca requires tls-key-file or tls-cert-file to be set to listen on tls")))
		})
	})

	Describe("when defining tls-client-ca and key without certs", func() {
		It("should fail", func() {
			args := []string{"--tls-client-ca=/foo/bar", "--tls-key=/foo/bar"}
			options, err := config.Init(args)
			Expect(options).Should(BeNil())
			Expect(err.Error()).Should(
				Equal(errorMessage("tls-client-ca requires tls-key-file or tls-cert-file to be set to listen on tls")))
		})
	})

	Describe("when defining no options", func() {
		It("should not fail", func() {
			args := []string{}
			options, err := config.Init(args)
			Expect(err).Should(BeNil())
			Expect(options).Should(Not(BeNil()))
			Expect(&url.URL{Scheme: "https", Host: "127.0.0.1:9200", Path: "/"}).Should(Equal(options.ElasticsearchURL))
		})
	})

	Describe("when defining auth backend role", func() {
		Describe("without a valid backendname", func() {

			It("should fail", func() {
				args := []string{"--auth-backend-role={'verb':'get'}"}
				options, err := config.Init(args)
				Expect(options).Should(BeNil())
				Expect(err.Error()).Should(
					Equal(errorMessage("auth-backend-role \"{'verb':'get'}\" should be name=SAR")))
			})
		})
		Describe("that is the same as one that exists", func() {

			It("should fail", func() {
				args := []string{"--auth-backend-role=foo={\"verb\":\"get\"}", "--auth-backend-role=foo={\"verb\":\"get\"}"}
				options, err := config.Init(args)
				Expect(options).Should(BeNil())
				Expect(err.Error()).Should(
					Equal(errorMessage("Backend role with that name \"foo={\\\"verb\\\":\\\"get\\\"}\" already exists")))
			})
		})
		Describe("with unique backend roles", func() {

			It("should succeed", func() {
				args := []string{"--auth-backend-role=foo={\"verb\":\"get\"}", "--auth-backend-role=bar={\"verb\":\"get\"}"}
				options, err := config.Init(args)
				Expect(err).Should(BeNil())
				Expect(options).Should(Not(BeNil()))
				exp := map[string]config.BackendRoleConfig{
					"foo": config.BackendRoleConfig{Verb: "get"},
					"bar": config.BackendRoleConfig{Verb: "get"},
				}
				Expect(options.AuthBackEndRoles).Should(Equal(exp))
			})
		})
	})

	Describe("when defining kibana index mode", func() {
		Describe("with an unsupported value", func() {
			It("should fail", func() {
				args := []string{"--cl-kibana-index-mode=foo"}
				options, err := config.Init(args)
				Expect(options).Should(BeNil())
				Expect(err.Error()).Should(
					Equal(errorMessage("Unsupported kibanaIndexMode \"foo\"")))
			})
		})
		Describe("with a supported value", func() {
			It("should succeed", func() {
				args := []string{"--cl-kibana-index-mode=sharedOps"}
				options, err := config.Init(args)
				Expect(err).Should(BeNil())
				Expect(options).Should(Not(BeNil()))
				Expect(options.KibanaIndexMode).Should(
					Equal(cltypes.KibanaIndexModeSharedOps))
			})
		})
	})

})
