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


## Installing, configuring, and starting local instance of Elasticsearch and Proxy
1. Download ES
  `curl https://artifacts.elastic.co/downloads/elasticsearch/elasticsearch-6.6.2.tar.gz -o elasticsearch-6.6.2.tar.gz`

2. Extract ES
  `tar -xzf elasticsearch-6.6.2.tar.gz`

3. Install OpenDistro plugin
  `elasticsearch-6.6.2/bin/elasticsearch-plugin install -b com.amazon.opendistroforelasticsearch:opendistro_security:0.8.0.0`

4. Initialize OpenDistro
  `sh elasticsearch-6.6.2/plugins/opendistro_security/tools/install_demo_configuration.sh -y`

5. Disable x-pack-security
  ```
  echo "xpack.security.enabled: false" >> elasticsearch-6.6.2/config/elasticsearch.yml && \
  echo "xpack.monitoring.enabled: true" >> elasticsearch-6.6.2/config/elasticsearch.yml && \
  echo "xpack.graph.enabled: false" >> elasticsearch-6.6.2/config/elasticsearch.yml && \
  echo "xpack.watcher.enabled: false" >> elasticsearch-6.6.2/config/elasticsearch.yml
  ```

6. Install roles, roles_mapping, config.yaml
  ```
  export CONFIG_URI="https://raw.githubusercontent.com/jcantrill/origin-aggregated-logging/6x_proxy_opendistro/elasticsearch/sgconfig/"
  export CONFIG_DEST="elasticsearch-6.6.2/plugins/opendistro_security/securityconfig/"

  for file in "config.yml" "roles.yml" "roles_mapping.yml"; do
    curl ${CONFIG_URI}${file} -o ${CONFIG_DEST}${file};
  done
  ```

7. Start ES
  `nohup elasticsearch-6.6.2/bin/elasticsearch & \
   export ES_PID=$!`

   Watch ES logs
  `tail -f nohup.out`

8. Initialize Security config
  `sh securityadmin_demo.sh`

9. Start proxy
  ```
  export ES_CERTS_DIR="__output"
  mkdir -p ${ES_CERTS_DIR}
  sudo cp /etc/elasticsearch/{kirk,kirk-key,root-ca}.pem ${ES_CERTS_DIR}
  make run
  ```

10. Stopping ES
  `kill $ES_PID`
