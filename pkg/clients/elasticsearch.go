package clients

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/bitly/go-simplejson"
	es "github.com/elastic/go-elasticsearch/v6"
	"github.com/elastic/go-elasticsearch/v6/esapi"
	log "github.com/sirupsen/logrus"
)

const (
	contentTypeJSON = "application/json"
)

type MGetRequest struct {
	Docs []MGetItem `json:"docs"`
}

type MGetItem struct {
	Type string `json:"_type,omitempty"`
	Id   string `json:"_id,omitempty"`
}

type MGetResponse struct {
	Docs []MGetResponseItem `json:"docs"`
}
type MGetResponseItem struct {
	Index   string                 `json:"_index,omitempty"`
	Version int                    `json:"_version,omitempty"`
	Found   bool                   `json:"found,omitempty"`
	Source  map[string]interface{} `json:"_source,omitempty"`
	MGetItem
}

//ElasticsearchClient is an admin client to query a local instance of Elasticsearch
type ElasticsearchClient interface {
	Get(index, docType, id string) (string, error)
	Index(index, docType, id, body string, version int) (string, error)
	MGet(index string, items MGetRequest) (*MGetResponse, error)
	Delete(index, docType, id string) (string, error)
}

//DefaultElasticsearchClient is an admin client to query a local instance of Elasticsearch
type DefaultElasticsearchClient struct {
	serverURL string
	client    *es.Client
}

//NewElasticsearchClient is the initializer to create an instance of ES client
func NewElasticsearchClient(skipVerify bool, serverURL, adminCert, adminKey string, adminCA []string) (ElasticsearchClient, error) {
	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	for _, ca := range adminCA {
		caCert, err := ioutil.ReadFile(ca)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		caCertPool.AppendCertsFromPEM(caCert)
	}

	cert, err := tls.LoadX509KeyPair(adminCert, adminKey)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	cfg := es.Config{
		Addresses: []string{
			serverURL,
		},
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: time.Second,
			DialContext:           (&net.Dialer{Timeout: time.Second}).DialContext,
			TLSClientConfig: &tls.Config{
				RootCAs:            caCertPool,
				Certificates:       []tls.Certificate{cert},
				InsecureSkipVerify: skipVerify,
			},
		},
	}

	client, err := es.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &DefaultElasticsearchClient{serverURL, client}, nil
}

func url(elasticsearchURL, path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return elasticsearchURL + path
}

//MGet the items
func (es *DefaultElasticsearchClient) MGet(index string, items MGetRequest) (*MGetResponse, error) {
	log.Tracef("Converting MGet items to json: %+v", items)
	var out []byte
	var err error
	if out, err = json.Marshal(items); err != nil {
		return nil, err
	}
	resp, err := es.client.Mget(bytes.NewReader(out), es.client.Mget.WithIndex(index))
	if err != nil {
		log.Tracef("Error executing Elasticsearch MGet %v", err)
		return nil, err
	}
	bodyAsString, err := readBody(resp)
	if err != nil {
		log.Tracef("Eror reading response body in MGet %v", err)
		return nil, err
	}
	log.Tracef("Unmarshalling response body in MGet: %v", bodyAsString)
	mgetResponse := &MGetResponse{}
	if err = json.Unmarshal([]byte(bodyAsString), mgetResponse); err != nil {
		return nil, err
	}
	return mgetResponse, nil
}

//Get the Document
func (es *DefaultElasticsearchClient) Get(index, docType, id string) (string, error) {
	log.Tracef("Get: %s, %s, %s", index, docType, id)
	resp, err := es.client.Get(index, id, es.client.Get.WithDocumentType(docType))
	if err != nil {
		log.Tracef("Error executing Elasticsearch GET %v", err)
		return "", err
	}
	log.Tracef("Response code: %v", resp.StatusCode)
	body, err := readBody(resp)
	if err != nil {
		return "", err
	}
	return body, nil
}

func readBody(resp *esapi.Response) (string, error) {
	defer resp.Body.Close()
	body, err := simplejson.NewFromReader(resp.Body)
	if err != nil {
		return "", err
	}
	log.Tracef("Response body: %v", body)
	if resp.StatusCode != 200 {
		log.Trace("Additionally inspecting result of non 200 response...")
		errorBody := body.Get("error")
		log.Tracef("errBody: %v", errorBody)
		return errorBody.MustString(), nil
	}
	result, err := body.Encode()
	if err != nil {
		return "", err
	}
	return string(result), nil
}

//Index submits an index request to ES
func (es *DefaultElasticsearchClient) Index(index, docType, id, body string, version int) (string, error) {
	resp, err := es.client.Index(index, strings.NewReader(body),
		es.client.Index.WithDocumentType(docType),
		es.client.Index.WithDocumentID(id),
		es.client.Index.WithVersion(version))
	if err != nil {
		log.Tracef("Error executing Elasticsearch PUT %v", err)
		return "", err
	}
	if err != nil {
		return "", err
	}
	return readBody(resp)
}

//Delete submits a Delete request to ES assuming the given body is of type 'application/json'
func (es *DefaultElasticsearchClient) Delete(index, docType, id string) (string, error) {
	resp, err := es.client.Delete(index, id, es.client.Delete.WithDocumentType(docType))
	if err != nil {
		log.Tracef("Error executing Elasticsearch DELETE %v", err)
		return "", err
	}
	if err != nil {
		return "", err
	}
	return readBody(resp)
}
