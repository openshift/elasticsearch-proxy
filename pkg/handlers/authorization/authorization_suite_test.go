package authorization

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAuthorization(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Authorization Suite")
}
