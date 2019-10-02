h1. Developing the proxy against a running instance of OKD and Cluster logging

* Login as an administrator
* create secrets dir: `sudo mkdir -p /var/run/secrets/kubernetes.io/serviceaccount/`
* Forward traffic to Elasticsearch:
  `oc -n openshift-logging port-forward $espod 9200:9200`
* Build `make clean && make`
* `export KUBERNETES_SERVICE_HOST=192.168.122.199.nip.io`
* `export KUBERNETES_SERVICE_PORT=8443`
* `export LOGLEVEL=trace`
* Run: `make-prep-for-run && make run`