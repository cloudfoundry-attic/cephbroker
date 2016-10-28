package cephbroker_test

import (
	"bytes"
	"fmt"

	"context"

	"code.cloudfoundry.org/cephbroker/cephbroker"
	"code.cloudfoundry.org/cephbroker/cephfakes"
	"code.cloudfoundry.org/goshims/execshim/exec_fake"
	"code.cloudfoundry.org/goshims/ioutilshim/ioutil_fake"
	"code.cloudfoundry.org/goshims/osshim/os_fake"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/voldriver"
	"code.cloudfoundry.org/voldriver/driverhttp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var CephClient = Describe("CephClient", func() {
	var (
		logger      lager.Logger
		ctx         context.Context
		env         voldriver.Env
		subject     cephbroker.Client
		fakeInvoker *cephfakes.FakeInvoker
		fakeOs      *os_fake.FakeOs
		fakeIoutil  *ioutil_fake.FakeIoutil
	)
	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test-broker")
		ctx = context.TODO()
		env = driverhttp.NewHttpDriverEnv(logger, ctx)
		fakeInvoker = &cephfakes.FakeInvoker{}
		fakeOs = &os_fake.FakeOs{}
		fakeIoutil = &ioutil_fake.FakeIoutil{}
		subject = cephbroker.NewCephClientWithInvokerAndSystemUtil("mds", fakeInvoker, fakeOs, fakeIoutil, "localMountPoint", "keyringFile")
	})
	Context(".MountFileSystem", func() {
		It("should mount", func() {
			localMountPoint, err := subject.MountFileSystem(env, "remoteMountPoint")
			Expect(err).NotTo(HaveOccurred())
			Expect(localMountPoint).To(Equal("localMountPoint"))
		})
	})
	Context(".CreateShare", func() {
		It("should create share", func() {
			share, err := subject.CreateShare(env, "shareName")
			Expect(err).NotTo(HaveOccurred())
			Expect(share).To(Equal("localMountPoint/shareName"))
		})
	})
	Context(".DeleteShare", func() {
		It("should delete share", func() {
			err := subject.DeleteShare(env, "shareName")
			Expect(err).NotTo(HaveOccurred())
		})
	})
	Context(".GetPathsForShare", func() {
		It("should be able to get paths", func() {
			path1, path2, err := subject.GetPathsForShare(env, "sharename")
			Expect(err).NotTo(HaveOccurred())
			Expect(path1).To(Equal("sharename"))
			Expect(path2).To(Equal("/var/vcap/data/volumes/ceph/sharename"))
		})
	})
	Context(".GetConfigDetails", func() {
		It("should be able to get config details", func() {
			detail1, detail2, err := subject.GetConfigDetails(env)
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
		testCtx    = context.TODO()
		testEnv    voldriver.Env
		cmd        = "some-fake-command"
		args       = []string{"fake-args-1"}
	)
	Context("when invoking an executable", func() {
		BeforeEach(func() {
			testEnv = driverhttp.NewHttpDriverEnv(testLogger, testCtx)
			fakeExec = new(exec_fake.FakeExec)
			fakeCmd = new(exec_fake.FakeCmd)
			fakeExec.CommandContextReturns(fakeCmd)
			subject = cephbroker.NewRealInvokerWithExec(fakeExec)
		})

		It("should report an error when unable to attach to stdout", func() {
			fakeCmd.StdoutPipeReturns(errCloser{bytes.NewBufferString("")}, fmt.Errorf("unable to attach to stdout"))
			err := subject.Invoke(testEnv, cmd, args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unable to attach to stdout"))
		})

		It("should report an error when unable to start binary", func() {
			fakeCmd.StdoutPipeReturns(errCloser{bytes.NewBufferString("cmdfails")}, nil)
			fakeCmd.StartReturns(fmt.Errorf("unable to start binary"))
			err := subject.Invoke(testEnv, cmd, args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unable to start binary"))
		})
		It("should report an error when executing the driver binary fails", func() {
			fakeCmd.WaitReturns(fmt.Errorf("executing driver binary fails"))

			err := subject.Invoke(testEnv, cmd, args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("executing driver binary fails"))
		})
		It("should successfully invoke cli", func() {
			err := subject.Invoke(testEnv, cmd, args)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
