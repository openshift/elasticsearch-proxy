package clients

import (
	"fmt"
	"os/user"
	"path/filepath"
	"strings"

	"k8s.io/client-go/tools/clientcmd"

	osprojectv1 "github.com/openshift/api/project/v1"
	log "github.com/sirupsen/logrus"
	authenticationapi "k8s.io/api/authentication/v1"
	authorizationapi "k8s.io/api/authorization/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

//OpenShiftClient abstracts kubeclient and calls
type OpenShiftClient interface {
	ListNamespaces(token string) ([]Namespace, error)

	//TokenReview performs a tokenreview for a given token submitting to the apiserver
	//using the serviceaccount token. It returns a simplejson object of the response
	TokenReview(token string) (*TokenReview, error)
	SubjectAccessReview(user, namespace, verb, resource, resourceAPIGroup string) (bool, error)
}

//DefaultOpenShiftClient is the default impl of OpenShiftClient
type DefaultOpenShiftClient struct {
	client *kubernetes.Clientset
}

//TokenReview is simple struct wrapper around a kubernetes TokenReview
type TokenReview struct {
	*authenticationapi.TokenReview
}

//UserName returns the username associated with a given token
func (t *TokenReview) UserName() string {
	return t.Status.User.Username
}

//Groups returns the groups associated with a given token
func (t *TokenReview) Groups() []string {
	return t.Status.User.Groups
}

//Namespace wrappers a core kube namespace type
type Namespace struct {
	Ns osprojectv1.Project
}

//UID get the UID of a namespace
func (ns *Namespace) UID() string {
	return string(ns.Ns.UID)
}

//Name get the name of a namespace
func (ns *Namespace) Name() string {
	return ns.Ns.Name
}

//ListNamespaces associated with a given token
func (c *DefaultOpenShiftClient) ListNamespaces(token string) (namespaces []Namespace, err error) {
	client, err := newKubeClient(token)
	if err != nil {
		return nil, err
	}

	result := &osprojectv1.ProjectList{}
	err = client.RESTClient().
		Get().
		Prefix("apis", "project.openshift.io", "v1", "projects").
		Do().
		Into(result)
	if err != nil {
		return nil, err
	}
	for _, ns := range result.Items {
		namespaces = append(namespaces, Namespace{ns})
	}
	return namespaces, nil
}

//TokenReview performs a tokenreview for a given token submitting to the apiserver
//using the serviceaccount token. It returns a simplejson object of the response
func (c *DefaultOpenShiftClient) TokenReview(token string) (*TokenReview, error) {
	log.Debug("Performing TokenReview...")
	review := &authenticationapi.TokenReview{
		Spec: authenticationapi.TokenReviewSpec{
			Token: token,
		},
	}
	result, err := c.client.AuthenticationV1().TokenReviews().Create(review)
	if err != nil {
		return nil, err
	}
	return &TokenReview{result}, nil
}

//SubjectAccessReview performs a SAR and returns true if the user is allowed
func (c *DefaultOpenShiftClient) SubjectAccessReview(user, namespace, verb, resource, resourceAPIGroup string) (bool, error) {
	log.Debug("Performing SubjectAccessReview...")
	sar := &authorizationapi.SubjectAccessReview{
		Spec: authorizationapi.SubjectAccessReviewSpec{
			User: user,
		},
	}
	if strings.HasPrefix(resource, "/") {
		sar.Spec.NonResourceAttributes = &authorizationapi.NonResourceAttributes{
			Path: resource,
			Verb: verb,
		}
	} else {
		sar.Spec.ResourceAttributes = &authorizationapi.ResourceAttributes{
			Resource:  resource,
			Namespace: namespace,
			Group:     resourceAPIGroup,
			Verb:      verb,
		}
	}
	result, err := c.client.AuthorizationV1().SubjectAccessReviews().Create(sar)
	if err != nil {
		return false, err
	}
	return result.Status.Allowed, nil
}

// NewOpenShiftClient returns a client for connecting to the api server.
func NewOpenShiftClient() (OpenShiftClient, error) {
	c, err := newKubeClient("")
	if err != nil {
		return nil, fmt.Errorf("failed not create kubernetes client: %v", err)
	}
	return &DefaultOpenShiftClient{client: c}, nil
}

func newKubeClient(token string) (*kubernetes.Clientset, error) {
	config, err := getConfig()
	if err != nil {
		return nil, err
	}
	if token != "" {
		config.BearerToken = token
		config.BearerTokenFile = ""
	}
	log.Tracef("Creating new OpenShift client %v", config.Host)
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

func getConfig() (*rest.Config, error) {
	// Try the in-cluster config
	c, errInCluster := rest.InClusterConfig()
	if errInCluster == nil {
		log.Trace("Created in-cluster config")
		return c, nil
	}
	log.Tracef("Failed to create in-cluster config: %v", errInCluster)
	// If no in-cluster config, try the default location in the user's home directory
	usr, errKubeConfig := user.Current()
	if errKubeConfig == nil {
		var c *rest.Config
		c, errKubeConfig = clientcmd.BuildConfigFromFlags("", filepath.Join(usr.HomeDir, ".kube", "config"))
		if errKubeConfig == nil {
			log.Trace("Created host based (~/.kube) config")
			return c, nil
		}
	}
	return nil, fmt.Errorf("could not create k8s config for both in-cluster [%v] and kubeconfig [%v]", errInCluster, errKubeConfig)
}
