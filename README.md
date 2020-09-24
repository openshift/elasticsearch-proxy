OKD Elasticsearch Proxy
=====================

A reverse proxy to Elasticsearch that relies on either client certificate or Bearer token for use with OKD Cluster Logging

More information can be found in the [design document](https://github.com/openshift/enhancements/blob/master/enhancements/cluster-logging/cluster-logging-elasticsearch-proxy.md).


Features:

- [x] Dynamically seeds a user's permissions based on their OKD projects and ability to satisfy subjectaccessreviews
- [x] Utilizes OKD Bearer token for authorization
- [ ] Defaults a set of kibana index patterns for non infra users
- [ ] Dynamically creates a kibana index for non infra users

This proxy is inspired by the [oauth-proxy](https://raw.githubusercontent.com/openshift/oauth-proxy) and the openshift-elasticsearch-plugin

## Contributions

To contribute to the development of elasticsearch-proxy, see  [REVIEW.md](./REVIEW.md)
