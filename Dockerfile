FROM registry.svc.ci.openshift.org/openshift/release:golang-1.10 AS builder
WORKDIR  /go/src/github.com/openshift/elasticsearch-proxy
COPY . .
RUN make

FROM registry.svc.ci.openshift.org/openshift/origin-v4.0:base
COPY --from=builder bin/elasticsearch-proxy /usr/bin/
ENTRYPOINT ["/usr/bin/elasticsearch-proxy"]
LABEL io.k8s.display-name="OpenShift ElasticSearch Proxy" \
      io.k8s.description="OpenShift ElasticSearch Proxy component of OpenShift Cluster Logging" 
