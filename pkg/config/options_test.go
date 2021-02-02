package config_test

import (
	"net/url"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/elasticsearch-proxy/pkg/config"
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
			args := []string{"--tls-client-ca=/foo/bar", "--metrics-tls-cert=/foo/bar", "--metrics-tls-key=/foo/bar"}
			options, err := config.Init(args)
			Expect(options).Should(BeNil())

			Expect(err.Error()).Should(
				Equal(errorMessage("tls-client-ca requires tls-key-file or tls-cert-file to be set to listen on tls")))
		})
	})

	Describe("when defining tls-client-ca and key without certs", func() {
		It("should fail", func() {
			args := []string{"--tls-client-ca=/foo/bar", "--tls-key=/foo/bar", "--metrics-tls-cert=/foo/bar", "--metrics-tls-key=/foo/bar"}
			options, err := config.Init(args)
			Expect(options).Should(BeNil())
			Expect(err.Error()).Should(
				Equal(errorMessage("tls-client-ca requires tls-key-file or tls-cert-file to be set to listen on tls")))
		})
	})

	Describe("when defining metrics-listening-address", func() {
		It("should fail without metrics-tls-cert", func() {
			args := []string{"--metrics-listening-address=:60001", "--metrics-tls-key=/foo/bar"}
			options, err := config.Init(args)
			Expect(options).Should(BeNil())
			Expect(err.Error()).Should(
				Equal(errorMessage("metrics-listening-address requires metrics-tls-cert and metrics-tls-key to be set")))
		})

		It("should fail without metrics-tls-key", func() {
			args := []string{"--metrics-listening-address=:60001", "--metrics-tls-cert=/foo/bar"}
			options, err := config.Init(args)
			Expect(options).Should(BeNil())
			Expect(err.Error()).Should(
				Equal(errorMessage("metrics-listening-address requires metrics-tls-cert and metrics-tls-key to be set")))
		})
	})

	Describe("when defining no options", func() {
		It("should not fail", func() {
			args := []string{}
			options, err := config.Init(args)
			Expect(err).Should(BeNil())
			Expect(options).Should(Not(BeNil()))
			Expect(&url.URL{Scheme: "https", Host: "localhost:9200", Path: "/"}).Should(Equal(options.ElasticsearchURL))
		})
	})

	Describe("when defining the admin role", func() {
		It("should succeed", func() {
			args := []string{"--auth-admin-role=foo"}
			options, err := config.Init(args)
			Expect(err).Should(BeNil())
			Expect(options).Should(Not(BeNil()))
			Expect(options.AuthAdminRole).Should(Equal("foo"))
		})
	})

	Describe("when defining the default role", func() {
		It("should succeed", func() {
			args := []string{"--auth-default-role=foo"}
			options, err := config.Init(args)
			Expect(err).Should(BeNil())
			Expect(options).Should(Not(BeNil()))
			Expect(options.AuthDefaultRole).Should(Equal("foo"))
		})

	})

	Describe("when defining whitelisted names", func() {
		It("should succeed", func() {
			args := []string{"--auth-whitelisted-name=foo", "--auth-whitelisted-name=bar"}
			options, err := config.Init(args)
			Expect(err).Should(BeNil())
			Expect(options).Should(Not(BeNil()))
			Expect(options.AuthWhiteListedNames).Should(Equal([]string{"foo", "bar"}))
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

	// HTTPReadTimeout
	Describe("when defining HTTP server read timeout", func() {
		Describe("to be non-negative", func() {
			It("should succeed", func() {
				args := []string{"--http-read-timeout=7ns"}
				options, err := config.Init(args)
				Expect(err).Should(BeNil())
				Expect(options).Should(Not(BeNil()))
				Expect(options.HTTPReadTimeout).Should(Equal(7 * time.Nanosecond))
			})
		})
		Describe("to be negative", func() {
			It("should fail", func() {
				args := []string{"--http-read-timeout=-7ns"}
				options, err := config.Init(args)
				Expect(options).Should(BeNil())
				Expect(err.Error()).Should(
					Equal(errorMessage("http-read-timeout can not be negative")))
			})
		})
	})

	// HTTPWriteTimeout
	Describe("when defining HTTP server write timeout", func() {
		Describe("to be non-negative", func() {
			It("should succeed", func() {
				args := []string{"--http-write-timeout=1ms"}
				options, err := config.Init(args)
				Expect(err).Should(BeNil())
				Expect(options).Should(Not(BeNil()))
				Expect(options.HTTPWriteTimeout).Should(Equal(time.Millisecond))
			})
		})
		Describe("to be negative", func() {
			It("should fail", func() {
				args := []string{"--http-write-timeout=-1ms"}
				options, err := config.Init(args)
				Expect(options).Should(BeNil())
				Expect(err.Error()).Should(
					Equal(errorMessage("http-write-timeout can not be negative")))
			})
		})
	})

	// HTTPIdleTimeout
	Describe("when defining HTTP server idle timeout", func() {
		Describe("to be non-negative", func() {
			It("should succeed", func() {
				args := []string{"--http-idle-timeout=7s"}
				options, err := config.Init(args)
				Expect(err).Should(BeNil())
				Expect(options).Should(Not(BeNil()))
				Expect(options.HTTPIdleTimeout).Should(Equal(7 * time.Second))
			})
		})
		Describe("to be negative", func() {
			It("should fail", func() {
				args := []string{"--http-idle-timeout=-7s"}
				options, err := config.Init(args)
				Expect(options).Should(BeNil())
				Expect(err.Error()).Should(
					Equal(errorMessage("http-idle-timeout can not be negative")))
			})
		})
	})

	// HTTPMaxConnsPerHost
	Describe("when defining HTTP transport max connections per host", func() {
		Describe("to be non-negative", func() {
			It("should succeed", func() {
				args := []string{"--http-max-conns-per-host=1"}
				options, err := config.Init(args)
				Expect(err).Should(BeNil())
				Expect(options).Should(Not(BeNil()))
				Expect(options.HTTPMaxConnsPerHost).Should(Equal(1))
			})
		})
		Describe("to be negative", func() {
			It("should fail", func() {
				args := []string{"--http-max-conns-per-host=-1"}
				options, err := config.Init(args)
				Expect(options).Should(BeNil())
				Expect(err.Error()).Should(
					Equal(errorMessage("http-max-conns-per-host can not be negative")))
			})
		})
	})

	// HTTPMaxIdleConns
	Describe("when defining HTTP transport max idle connections", func() {
		Describe("to be non-negative", func() {
			It("should succeed", func() {
				args := []string{"--http-max-idle-conns=1"}
				options, err := config.Init(args)
				Expect(err).Should(BeNil())
				Expect(options).Should(Not(BeNil()))
				Expect(options.HTTPMaxIdleConns).Should(Equal(1))
			})
		})
		Describe("to be negative", func() {
			It("should fail", func() {
				args := []string{"--http-max-idle-conns=-1"}
				options, err := config.Init(args)
				Expect(options).Should(BeNil())
				Expect(err.Error()).Should(
					Equal(errorMessage("http-max-idle-conns can not be negative")))
			})
		})
	})

	// HTTPMaxIdleConnsPerHost
	Describe("when defining HTTP transport max idle connections per host", func() {
		Describe("to be non-negative", func() {
			It("should succeed", func() {
				args := []string{"--http-max-idle-conns-per-host=1"}
				options, err := config.Init(args)
				Expect(err).Should(BeNil())
				Expect(options).Should(Not(BeNil()))
				Expect(options.HTTPMaxIdleConnsPerHost).Should(Equal(1))
			})
		})
		Describe("to be negative", func() {
			It("should fail", func() {
				args := []string{"--http-max-idle-conns-per-host=-1"}
				options, err := config.Init(args)
				Expect(options).Should(BeNil())
				Expect(err.Error()).Should(
					Equal(errorMessage("http-max-idle-conns-per-host can not be negative")))
			})
		})
	})

	// HTTPIdleConnTimeout
	Describe("when defining negative HTTP transport idle connection timeout", func() {
		Describe("to be non-negative", func() {
			It("should succeed", func() {
				args := []string{"--http-idle-conn-timeout=2m"}
				options, err := config.Init(args)
				Expect(err).Should(BeNil())
				Expect(options).Should(Not(BeNil()))
				Expect(options.HTTPIdleConnTimeout).Should(Equal(2 * time.Minute))
			})
		})
		Describe("to be negative", func() {
			It("should fail", func() {
				args := []string{"--http-idle-conn-timeout=-2m"}
				options, err := config.Init(args)
				Expect(options).Should(BeNil())
				Expect(err.Error()).Should(
					Equal(errorMessage("http-idle-conn-timeout can not be negative")))
			})
		})
	})

	// HTTPTLSHandshakeTimeout
	Describe("when defining HTTP transport TLS handshake timeout", func() {
		Describe("to be non-negative", func() {
			It("should succeed", func() {
				args := []string{"--http-tls-handshake-timeout=3h"}
				options, err := config.Init(args)
				Expect(err).Should(BeNil())
				Expect(options).Should(Not(BeNil()))
				Expect(options.HTTPTLSHandshakeTimeout).Should(Equal(3 * time.Hour))
			})
		})
		Describe("to be negative", func() {
			It("should fail", func() {
				args := []string{"--http-tls-handshake-timeout=-3h"}
				options, err := config.Init(args)
				Expect(options).Should(BeNil())
				Expect(err.Error()).Should(
					Equal(errorMessage("http-tls-handshake-timeout can not be negative")))
			})
		})
	})

	// HTTPExpectContinueTimeout
	Describe("when defining HTTP transport expect continue timeout", func() {
		Describe("to be non-negative", func() {
			It("should succeed", func() {
				args := []string{"--http-expect-continue-timeout=1h2m3s4ms5us6ns"}
				options, err := config.Init(args)
				Expect(err).Should(BeNil())
				Expect(options).Should(Not(BeNil()))
				Expect(options.HTTPExpectContinueTimeout).Should(Equal(
					1*time.Hour + 2*time.Minute + 3*time.Second + 4*time.Millisecond +
						5*time.Microsecond + 6*time.Nanosecond))
			})
		})
		Describe("to be negative", func() {
			It("should fail", func() {
				args := []string{"--http-expect-continue-timeout=-1h2m3s4ms5us6ns"}
				options, err := config.Init(args)
				Expect(options).Should(BeNil())
				Expect(err.Error()).Should(
					Equal(errorMessage("http-expect-continue-timeout can not be negative")))
			})
		})
	})
})
