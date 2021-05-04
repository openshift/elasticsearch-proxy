export GOROOT=$(shell go env GOROOT)
export GOFLAGS=-mod=vendor
export GO111MODULE=on

ARTIFACT_DIR?=./tmp
CURPATH=$(PWD)
GOFLAGS?=
BIN_NAME=elasticsearch-proxy
IMAGE_REPOSITORY_NAME=quay.io/openshift/origin-${BIN_NAME}:latest
LOCAL_IMAGE_TAG=openshift/origin-${BIN_NAME}
MAIN_PKG=cmd/proxy/main.go
TARGET_DIR=$(CURPATH)/_output
TARGET=$(CURPATH)/bin/$(BIN_NAME)
BUILD_GOPATH=$(TARGET_DIR)

#inputs to 'run' which may need to change
TLS_CERTS_BASEDIR=_output
NAMESPACE ?= "openshift-logging"
ES_CERTS_DIR ?= ""
CACHE_EXPIRY ?= "5s"

PKGS=$(shell go list ./... | grep -v -E '/vendor/')
TEST_OPTIONS?=

ELASTICSEARCH_NAME ?=elasticsearch

KUBERNETES_SERVICE_HOST ?= $(shell oc get svc kubernetes -n default -o jsonpath='{.spec.clusterIP}')
KUBERNETES_SERVICE_PORT ?= $(shell oc get svc kubernetes -n default -o jsonpath='{.spec.ports[?(@.name == "https")].port}')

all: build

artifactdir:
	@mkdir -p $(ARTIFACT_DIR)

fmt:
	@gofmt -l -w cmd && \
	gofmt -l -w pkg
.PHONY: fmt

build: fmt
	@mkdir -p $(TARGET_DIR)/src/$(APP_REPO)
	go build $(LDFLAGS) -o $(TARGET) $(MAIN_PKG)
.PHONY: build

vendor:
	go mod vendor
.PHONY: vendor

image:
	podman build -f Dockerfile -t $(LOCAL_IMAGE_TAG) .
.PHONY: image

deploy-image: image
	IMAGE_TAG=$(LOCAL_IMAGE_TAG) hack/deploy-image.sh
.PHONY: deploy-image


clean:
	rm -rf $(TARGET_DIR)
	rm -rf $(TLS_CERTS_BASEDIR)
.PHONY: clean

COVERAGE_DIR=$(ARTIFACT_DIR)/coverage
test: artifactdir
	@mkdir -p $(COVERAGE_DIR)
	@go test -race -coverprofile=$(COVERAGE_DIR)/test-unit.cov ./pkg/...
	@go tool cover -html=$(COVERAGE_DIR)/test-unit.cov -o $(COVERAGE_DIR)/test-unit-coverage.html
	@go tool cover -func=$(COVERAGE_DIR)/test-unit.cov | tail -n 1
.PHONY: test

copy-k8s-sa:
	mkdir -p ${TLS_CERTS_BASEDIR} || true
	oc -n ${NAMESPACE} get pod -l component=elasticsearch -o jsonpath={.items[0].metadata.name} > _output/espod && \
	oc -n ${NAMESPACE} exec -c elasticsearch $$(cat _output/espod) -- cat /var/run/secrets/kubernetes.io/serviceaccount/ca.crt > _output/ca.crt && \
	oc -n ${NAMESPACE} serviceaccounts get-token elasticsearch > _output/sa-token && \
	echo ${NAMESPACE} > _output/namespace && \
	sudo mkdir -p /var/run/secrets/kubernetes.io/serviceaccount/||:  && \
	sudo ln -sf $${PWD}/_output/ca.crt /var/run/secrets/kubernetes.io/serviceaccount/ca.crt && \
	sudo ln -sf $${PWD}/_output/sa-token /var/run/secrets/kubernetes.io/serviceaccount/token
.PHONY: copy-k8s-sa

copy-es-certs:
	mkdir -p ${TLS_CERTS_BASEDIR} ||:
ifneq ($(ES_CERTS_DIR), "")
	cp ${ES_CERTS_DIR}/kirk.pem ${TLS_CERTS_BASEDIR}/admin-cert
	cp ${ES_CERTS_DIR}/kirk-key.pem ${TLS_CERTS_BASEDIR}/admin-key
	cp ${ES_CERTS_DIR}/root-ca.pem ${TLS_CERTS_BASEDIR}/admin-ca
else
	for n in ca cert key ; do \
		oc -n ${NAMESPACE} extract secret/${ELASTICSEARCH_NAME} --keys=admin-$$n --to=${TLS_CERTS_BASEDIR} --confirm ; \
	done
endif
.PHONY: copy-es-certs

run: copy-es-certs
	KUBERNETES_SERVICE_HOST="${KUBERNETES_SERVICE_HOST}" \
	KUBERNETES_SERVICE_PORT="${KUBERNETES_SERVICE_PORT}" \
	LOGLEVEL=trace go run ${MAIN_PKG} --listening-address=':60000' \
        --tls-cert=$(TLS_CERTS_BASEDIR)/admin-cert \
        --tls-key=$(TLS_CERTS_BASEDIR)/admin-key \
        --upstream-ca=$(TLS_CERTS_BASEDIR)/admin-ca \
        --cache-expiry=$(CACHE_EXPIRY) \
		--auth-backend-role=sg_role_admin='{"namespace": "default", "verb": "view", "resource": "pods/metrics"}' \
		--auth-backend-role=prometheus='{"verb": "get", "resource": "/metrics"}' \
		--auth-backend-role=jaeger='{"verb": "get", "resource": "/jaeger", "resourceAPIGroup": "elasticsearch.jaegertracing.io"}' \
		--cl-infra-role-name=sg_role_admin \
		--ssl-insecure-skip-verify
.PHONY: run

lint:
	@hack/run-linter
.PHONY: lint
gen-dockerfiles:
	./hack/generate-dockerfile-from-midstream > Dockerfile
.PHONY: gen-dockerfiles
