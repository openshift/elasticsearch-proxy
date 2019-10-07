# Developing the proxy

The proxy performs calls to `tokenreviews.authentication.k8s.io` therefore it has to run 
under a user which is able to perform calls to this API. The proxy also talks to Elasticsearch via TLS
therefore it has to be configured with Elasticsearch root certificates.

The proxy first gets k8s credentials from `/var/run/secrets/kubernetes.io/serviceaccount` and then fallbacks 
to settings in `{HOME}/.kube`. Therefore you can use two methods to start the proxy with different kubernetes
credentials.

Elasticsearch root certificates can be injected from runing Elasticsearch pod or copied manually.
Make target `make copy-es-certs` copies Elasticsearch certificates to host system.

## Run the proxy with Elasticsearch service account

Use this method if there is Elasticsearch pod running.

1. Run `make copy-k8s-sa` to copy Elasticsearch service account credentials to your host.
2. `KUBERNETES_SERVICE_HOST=192.168.122.199.nip.io KUBERNETES_SERVICE_PORT=8443 make run` to run the proxy.

## Run the proxy with local Kubernetes configuration

1. `make run`

## Make request to proxy
```bash
curl -kivX GET -H "Authorization: Bearer $(oc whoami -t)" https://localhost:60000/project.myproject.bcc99fbb-e67e-11e9-8e6a-8c16456c84e7.*/_search\?pretty
```

## Start local instance of Elasticsearch
1. Download OSS Elasticsearch 6.x distribution. The version has to match OpenDistro supported version.
2. Install OpenDistro security plugin and initialize it
3. Install roles, roles_mapping and config.yaml from https://github.com/jcantrill/origin-aggregated-logging/tree/6x_proxy_opendistro/elasticsearch/sgconfig
    ```bash
    cp elasticsearch/sgconfig/config.yml ~/tmp/elasticsearch-6.8.1/plugins/opendistro_security/securityconfig/config.yml 
    cp elasticsearch/sgconfig/roles.yml ~/tmp/elasticsearch-6.8.1/plugins/opendistro_security/securityconfig/roles.yml
    cp elasticsearch/sgconfig/roles_mapping.yml ~/tmp/elasticsearch-6.8.1/plugins/opendistro_security/securityconfig/roles_mapping.yml
    ```
4. Start ES and verify curl works `curl -kivX GET  --cert config/kirk.pem --key config/kirk-key.pem --cacert config/root-ca.pem  https://localhost:9200`
5. Copy certificates to `__output` directory to be used by proxy `make copy-es-certs`
6. Start proxy
