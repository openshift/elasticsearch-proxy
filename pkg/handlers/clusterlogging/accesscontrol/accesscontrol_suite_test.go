package accesscontrol

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAccessControl(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Access Control Suite")
}
