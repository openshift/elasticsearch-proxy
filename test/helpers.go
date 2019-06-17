package test

import (
	"fmt"
	"os"
	"strings"

	// "github.com/bmizerany/assert"
	expectations "github.com/onsi/gomega"
)

//Debug is a convenient log mechnism to spit content to STDOUT
func Debug(object interface{}) {
	if os.Getenv("TEST_DEBUG") != "" {
		fmt.Println(object)
	}
}

//TestExpectation is a helper struct to allow chaining expectations
type TestExpectation struct {
	act string
}

func Expect(act string) *TestExpectation {
	return &TestExpectation{act}
}

func (t *TestExpectation) ToMatchYaml(exp string) {
	//normalize as expectations doesn't like tabs
	t.act = strings.Replace(t.act, "\t", "  ", -1)
	exp = strings.Replace(exp, "\t", "  ", -1)

	expectations.Expect(t.act).To(expectations.MatchYAML(exp))
}
