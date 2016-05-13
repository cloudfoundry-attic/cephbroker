package cephbrokerlocal_test

import (
	"fmt"
	"io"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCephbrokerlocal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cephbrokerlocal Suite")
}

type errCloser struct{ io.Reader }

func (errCloser) Close() error                     { return nil }
func (errCloser) Read(p []byte) (n int, err error) { return 0, fmt.Errorf("any") }
