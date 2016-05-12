package cephbrokerlocal_test

import (
	"fmt"
	"path"

	. "github.com/cloudfoundry-incubator/cephbroker/cephbrokerlocal"
	"github.com/cloudfoundry-incubator/cephbroker/model"
	"github.com/cloudfoundry-incubator/cephdriver/cephlocal/cephfakes"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cephbrokerlocal", func() {
	var (
		testLogger      lager.Logger
		cephClient      Client
		controller      Controller
		fakeInvoker     *cephfakes.FakeInvoker
		fakeSystemUtil  *cephfakes.FakeSystemUtil
		localMountPoint string
	)
	BeforeEach(func() {
		testLogger = lagertest.NewTestLogger("ControllerTest")
		fakeInvoker = new(cephfakes.FakeInvoker)
		fakeSystemUtil = new(cephfakes.FakeSystemUtil)
		localMountPoint = "/tmp/share"
		cephClient = NewCephClientWithInvokerAndSystemUtil("some-mds", fakeInvoker, fakeSystemUtil, localMountPoint)
		instanceMap := make(map[string]*model.ServiceInstance)
		bindingMap := make(map[string]*model.ServiceBinding)
		controller = NewController(cephClient, "/tmp/cephbroker", instanceMap, bindingMap)
	})
	Context("Catalog", func() {
		It("should produce a valid catalog", func() {
			catalog, err := controller.GetCatalog(testLogger)
			Expect(err).ToNot(HaveOccurred())
			Expect(catalog).ToNot(BeNil())
			Expect(catalog.Services).ToNot(BeNil())
			Expect(len(catalog.Services)).To(Equal(1))
			Expect(catalog.Services[0].Name).To(Equal("cephfs"))
			Expect(catalog.Services[0].Requires).ToNot(BeNil())
			Expect(len(catalog.Services[0].Requires)).To(Equal(1))
			Expect(catalog.Services[0].Requires[0]).To(Equal("volume_mount"))

			Expect(catalog.Services[0].Plans).ToNot(BeNil())
			Expect(len(catalog.Services[0].Plans)).To(Equal(1))
			Expect(catalog.Services[0].Plans[0].Name).To(Equal("free"))

			Expect(catalog.Services[0].Bindable).To(Equal(true))
		})
		Context("CreateServiceInstance", func() {
			var (
				instance model.ServiceInstance
			)
			BeforeEach(func() {
				instance = model.ServiceInstance{}

			})
			It("should create a valid service instance", func() {
				fakeSystemUtil.MkdirAllReturns(nil)
				properties := map[string]interface{}{"some-property": "some-value"}
				instance.Parameters = properties
				createResponse, err := controller.CreateServiceInstance(testLogger, "service-instance-guid", instance)
				Expect(err).ToNot(HaveOccurred())
				Expect(createResponse.DashboardUrl).ToNot(Equal(""))
				Expect(fakeSystemUtil.MkdirAllCallCount()).To(Equal(2))
			})
			Context("should fail to create service instance", func() {
				It("when base filesystem directory creation errors", func() {
					fakeSystemUtil.MkdirAllReturns(fmt.Errorf("failed to create directory"))
					properties := map[string]interface{}{"some-property": "some-value"}
					instance.Parameters = properties

					_, err := controller.CreateServiceInstance(testLogger, "service-instance-guid", instance)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal(fmt.Sprintf("failed to create local directory '%s', mount filesystem failed", localMountPoint)))
				})
				It("when filesystem mount fails", func() {
					fakeInvoker.InvokeReturns(fmt.Errorf("failed to mount filesystem"))
					properties := map[string]interface{}{"some-property": "some-value"}
					instance.Parameters = properties
					_, err := controller.CreateServiceInstance(testLogger, "service-instance-guid", instance)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("failed to mount filesystem"))
				})
				It("when share creation errors", func() {
					properties := map[string]interface{}{"some-property": "some-value"}
					instance.Parameters = properties
					// to ensure filesystem is mounted(on first creation)
					_, err := controller.CreateServiceInstance(testLogger, "service-instance-guid", instance)
					Expect(err).ToNot(HaveOccurred())

					fakeSystemUtil.MkdirAllReturns(fmt.Errorf("failed to create directory"))
					_, err = controller.CreateServiceInstance(testLogger, "service-instance-guid", instance)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal(fmt.Sprintf("failed to create share '%s'", path.Join(localMountPoint, "service-instance-guid"))))
				})

			})

		})
	})
})
