package cephbroker_test

import (
	"context"

	"code.cloudfoundry.org/cephbroker/cephbroker"
	"code.cloudfoundry.org/goshims/ioutilshim/ioutil_fake"
	"code.cloudfoundry.org/goshims/osshim/os_fake"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/voldriver"
	"code.cloudfoundry.org/voldriver/driverhttp"
	"code.cloudfoundry.org/voldriver/voldriverfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var CephClient = Describe("CephClient", func() {
	var (
		logger      lager.Logger
		ctx         context.Context
		env         voldriver.Env
		subject     cephbroker.Client
		fakeInvoker *voldriverfakes.FakeInvoker
		fakeOs      *os_fake.FakeOs
		fakeIoutil  *ioutil_fake.FakeIoutil
	)
	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test-broker")
		ctx = context.TODO()
		env = driverhttp.NewHttpDriverEnv(logger, ctx)
		fakeInvoker = &voldriverfakes.FakeInvoker{}
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
