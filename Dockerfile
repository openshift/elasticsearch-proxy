### This is a generated file from Dockerfile.in ###
#@follow_tag(openshift-golang-builder:1.14)
FROM registry.ci.openshift.org/ocp/builder:rhel-8-golang-1.16-openshift-4.8 AS builder

ENV BUILD_VERSION=1.0
ENV OS_GIT_MAJOR=1
ENV OS_GIT_MINOR=0
ENV OS_GIT_PATCH=0
ENV SOURCE_GIT_COMMIT=${CI_ELASTICSEARCH_PROXY_UPSTREAM_COMMIT}
ENV SOURCE_GIT_URL=${CI_ELASTICSEARCH_PROXY_UPSTREAM_URL}
ENV REMOTE_SOURCE=${REMOTE_SOURCE:-.}


WORKDIR  /go/src/github.com/openshift/elasticsearch-proxy
COPY ${REMOTE_SOURCE} .

RUN make

#@follow_tag(openshift-ose-base:ubi8)
FROM registry.ci.openshift.org/ocp/4.8:base
COPY --from=builder /go/src/github.com/openshift/elasticsearch-proxy/bin/elasticsearch-proxy /usr/bin/
ENTRYPOINT ["/usr/bin/elasticsearch-proxy"]

LABEL \
        io.k8s.display-name="OpenShift ElasticSearch Proxy" \
        io.k8s.description="OpenShift ElasticSearch Proxy component of OpenShift Cluster Logging" \
        name="openshift/ose-elasticsearch-proxy" \
        com.redhat.component="ose-elasticsearch-proxy-container" \
        io.openshift.maintainer.product="OpenShift Container Platform" \
        io.openshift.maintainer.component="Logging" \
        io.openshift.build.commit.id=${CI_ELASTICSEARCH_PROXY_UPSTREAM_COMMIT} \
        io.openshift.build.source-location=${CI_ELASTICSEARCH_PROXY_UPSTREAM_URL} \
        io.openshift.build.commit.url=${CI_ELASTICSEARCH_PROXY_UPSTREAM_URL}/commit/${CI_ELASTICSEARCH_PROXY_UPSTREAM_COMMIT} \
        version=v1.0

