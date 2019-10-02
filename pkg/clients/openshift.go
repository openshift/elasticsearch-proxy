package clients

import (
	"strings"

	authenticationapi "k8s.io/api/authentication/v1"
	authorizationapi "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	log "github.com/sirupsen/logrus"
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
	corev1.Namespace
}

//UID get the UID of a namespace
func (ns *Namespace) UID() string {
	return ns.UID()
}

//Name get the name of a namespace
func (ns *Namespace) Name() string {
	return ns.Name()
}

//ListNamespaces associated with a given token
func (c *DefaultOpenShiftClient) ListNamespaces(token string) (namespaces []Namespace, err error) {

	var nsList *corev1.NamespaceList
	if nsList, err = newKubeClient(token).CoreV1().Namespaces().List(metav1.ListOptions{}); err != nil {
		for _, ns := range nsList.Items {
			namespaces = append(namespaces, Namespace{ns})
		}
	} else {
		return []Namespace{}, err
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
	return &DefaultOpenShiftClient{
		newKubeClient(""),
	}, nil
}

func newKubeClient(token string) *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	if token != "" {
		config.BearerToken = token
	}
	log.Tracef("Creating new OpenShift client with: %+v", config)
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return clientset
}
