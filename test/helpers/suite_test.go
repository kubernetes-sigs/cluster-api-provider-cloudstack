package helpers_test

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
)

func TestCloud(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Cloud Suite")
}
