CURPATH=$(PWD)
GOFLAGS?=
BIN_NAME=elasticsearch-proxy
IMAGE_REPOSITORY_NAME ?=github.com/openshift/$(BIN_NAME)
MAIN_PKG=cmd/proxy/main.go
TARGET_DIR=$(CURPATH)/_output
TARGET=$(CURPATH)/bin/$(BIN_NAME)
BUILD_GOPATH=$(TARGET_DIR)

#inputs to 'run' which may need to change
TLS_CERTS_BASEDIR=_output

PKGS=$(shell go list ./... | grep -v -E '/vendor/')
TEST_PKGS=$(shell go list ./... | grep -v -E '/vendor/' | grep -v -E 'cmd')
TEST_OPTIONS?=

all: build

fmt:
	@gofmt -l -w cmd && \
	gofmt -l -w pkg
.PHONY: fmt

build: fmt
	@mkdir -p $(TARGET_DIR)/src/$(APP_REPO)
	go build $(LDFLAGS) -o $(TARGET) $(MAIN_PKG)
.PHONY: build

image:
	imagebuilder -f Dockerfile -t $(IMAGE_REPOSITORY_NAME)/$(BIN_NAME) .
.PHONY: images

clean:
	rm -rf $(TARGET_DIR)
	rm -rf $(TLS_CERTS_BASEDIR)
.PHONY: clean

test:
	@for pkg in $(TEST_PKGS) ; do \
		go test $(TEST_OPTIONS) $$pkg  ; \
	done
.PHONY: test

prep-for-run:
	mkdir -p ${TLS_CERTS_BASEDIR}||:  && \
	for n in "ca" "cert" "key" ; do \
		oc -n openshift-logging get secret elasticsearch -o jsonpath={.data.admin-$$n} | base64 -d > _output/admin-$$n ; \
	done && \
	oc -n openshift-logging  get pod -l component=elasticsearch -o jsonpath={.items[0].metadata.name} > _output/espod && \
	oc -n openshift-logging exec -c elasticsearch $$(cat _output/espod) -- cat /var/run/secrets/kubernetes.io/serviceaccount/ca.crt > _output/ca.crt && \
	oc -n openshift-logging serviceaccounts get-token elasticsearch > _output/sa-token && \
	echo openshift-logging > _output/namespace && \
	mkdir -p /var/run/secrets/kubernetes.io/serviceaccount/||:  && \
	sudo ln -sf $${PWD}/_output/ca.crt /var/run/secrets/kubernetes.io/serviceaccount/ca.crt && \
	sudo ln -sf $${PWD}/_output/sa-token /var/run/secrets/kubernetes.io/serviceaccount/token && \
	sudo ln -sf $${PWD}/_output/namespace /var/run/secrets/kubernetes.io/serviceaccount/namespace
	
.PHONY: prep-for-run

run:
	$(TARGET) --https-address=':60000' \
        --tls-cert=$(TLS_CERTS_BASEDIR)/admin-cert \
        --tls-key=$(TLS_CERTS_BASEDIR)/admin-key \
        --upstream-ca=$(TLS_CERTS_BASEDIR)/admin-ca \
		--auth-backend-role=sg_role_admin='{"namespace": "default", "verb": "view", "resource": "pods/metrics"}' \
		--auth-backend-role=prometheus='{"verb": "get", "resource": "/metrics"}' \
		--auth-backend-role=jaeger='{"verb": "get", "resource": "/jaeger", "resourceAPIGroup": "elasticsearch.jaegertracing.io"}' \
		--cl-infra-role-name=sg_role_admin \
		--ssl-insecure-skip-verify
.PHONY: run
