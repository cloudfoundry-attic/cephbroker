package cephbrokerlocal_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCephbrokerlocal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cephbrokerlocal Suite")
}
