package cephbroker_test

import (
	"bytes"
	"fmt"

	"code.cloudfoundry.org/cephbroker/cephbroker"
	"code.cloudfoundry.org/cephbroker/cephfakes"
	"code.cloudfoundry.org/goshims/execshim/exec_fake"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"code.cloudfoundry.org/goshims/ioutilshim/ioutil_fake"
	"code.cloudfoundry.org/goshims/osshim/os_fake"
)

var CephClient = Describe("CephClient", func() {
	var (
		logger      lager.Logger
		subject     cephbroker.Client
		fakeInvoker *cephfakes.FakeInvoker
		fakeOs      *os_fake.FakeOs
		fakeIoutil  *ioutil_fake.FakeIoutil
	)
	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test-broker")
		fakeInvoker = &cephfakes.FakeInvoker{}
		fakeOs = &os_fake.FakeOs{}
		fakeIoutil = &ioutil_fake.FakeIoutil{}
		subject = cephbroker.NewCephClientWithInvokerAndSystemUtil("mds", fakeInvoker, fakeOs, fakeIoutil, "localMountPoint", "keyringFile")
	})
	Context(".MountFileSystem", func() {
		It("should mount", func() {
			localMountPoint, err := subject.MountFileSystem(logger, "remoteMountPoint")
			Expect(err).NotTo(HaveOccurred())
			Expect(localMountPoint).To(Equal("localMountPoint"))
		})
	})
	Context(".CreateShare", func() {
		It("should create share", func() {
			share, err := subject.CreateShare(logger, "shareName")
			Expect(err).NotTo(HaveOccurred())
			Expect(share).To(Equal("localMountPoint/shareName"))
		})
	})
	Context(".DeleteShare", func() {
		It("should delete share", func() {
			err := subject.DeleteShare(logger, "shareName")
			Expect(err).NotTo(HaveOccurred())
		})
	})
	Context(".GetPathsForShare", func() {
		It("should be able to get paths", func() {
			path1, path2, err := subject.GetPathsForShare(logger, "sharename")
			Expect(err).NotTo(HaveOccurred())
			Expect(path1).To(Equal("sharename"))
			Expect(path2).To(Equal("/var/vcap/data/volumes/sharename"))
		})
	})
	Context(".GetConfigDetails", func() {
		It("should be able to get config details", func() {
			detail1, detail2, err := subject.GetConfigDetails(logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(detail1).To(Equal("mds"))
			Expect(detail2).To(Equal(""))
		})
	})
})

var Invoker = Describe("RealInvoker", func() {
	var (
		subject    cephbroker.Invoker
		fakeCmd    *exec_fake.FakeCmd
		fakeExec   *exec_fake.FakeExec
		testLogger = lagertest.NewTestLogger("InvokerTest")
		cmd        = "some-fake-command"
		args       = []string{"fake-args-1"}
	)
	Context("when invoking an executable", func() {
		BeforeEach(func() {
			fakeExec = new(exec_fake.FakeExec)
			fakeCmd = new(exec_fake.FakeCmd)
			fakeExec.CommandReturns(fakeCmd)
			subject = cephbroker.NewRealInvokerWithExec(fakeExec)
		})

		It("should report an error when unable to attach to stdout", func() {
			fakeCmd.StdoutPipeReturns(errCloser{bytes.NewBufferString("")}, fmt.Errorf("unable to attach to stdout"))
			err := subject.Invoke(testLogger, cmd, args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unable to attach to stdout"))
		})

		It("should report an error when unable to start binary", func() {
			fakeCmd.StdoutPipeReturns(errCloser{bytes.NewBufferString("cmdfails")}, nil)
			fakeCmd.StartReturns(fmt.Errorf("unable to start binary"))
			err := subject.Invoke(testLogger, cmd, args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unable to start binary"))
		})
		It("should report an error when executing the driver binary fails", func() {
			fakeCmd.WaitReturns(fmt.Errorf("executing driver binary fails"))

			err := subject.Invoke(testLogger, cmd, args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("executing driver binary fails"))
		})
		It("should successfully invoke cli", func() {
			err := subject.Invoke(testLogger, cmd, args)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
