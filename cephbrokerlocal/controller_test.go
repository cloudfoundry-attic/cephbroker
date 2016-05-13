package cephbrokerlocal_test

import (
	"bytes"
	"fmt"
	"path"

	. "github.com/cloudfoundry-incubator/cephbroker/cephbrokerlocal"
	"github.com/cloudfoundry-incubator/cephbroker/cephbrokerlocal/cephfakes"
	"github.com/cloudfoundry-incubator/cephbroker/model"
	"github.com/cloudfoundry/gunk/os_wrap/exec_wrap/execfakes"
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
		serviceGuid     string
		instanceMap     map[string]*model.ServiceInstance
		bindingMap      map[string]*model.ServiceBinding
	)
	BeforeEach(func() {
		testLogger = lagertest.NewTestLogger("ControllerTest")
		fakeInvoker = new(cephfakes.FakeInvoker)
		serviceGuid = "some-service-guid"
		fakeSystemUtil = new(cephfakes.FakeSystemUtil)
		localMountPoint = "/tmp/share"
		cephClient = NewCephClientWithInvokerAndSystemUtil("some-mds", fakeInvoker, fakeSystemUtil, localMountPoint)
		instanceMap = make(map[string]*model.ServiceInstance)
		bindingMap = make(map[string]*model.ServiceBinding)
		controller = NewController(cephClient, "/tmp/cephbroker", instanceMap, bindingMap)
	})
	Context(".Catalog", func() {
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
		Context(".CreateServiceInstance", func() {
			var (
				instance model.ServiceInstance
			)
			BeforeEach(func() {
				instance = model.ServiceInstance{}
				instance.PlanId = "some-planId"
				instance.Parameters = map[string]interface{}{"some-property": "some-value"}

			})
			It("should create a valid service instance", func() {
				successfullServiceInstanceCreate(testLogger, fakeSystemUtil, instance, controller, serviceGuid)
			})
			Context("should fail to create service instance", func() {
				It("when base filesystem directory creation errors", func() {
					fakeSystemUtil.MkdirAllReturns(fmt.Errorf("failed to create directory"))

					_, err := controller.CreateServiceInstance(testLogger, "service-instance-guid", instance)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal(fmt.Sprintf("failed to create local directory '%s', mount filesystem failed", localMountPoint)))
				})
				It("when filesystem mount fails", func() {
					fakeInvoker.InvokeReturns(fmt.Errorf("failed to mount filesystem"))
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
				It("should error when updating internal bookkeeping fails", func() {
					controller = NewController(cephClient, "/non-existent-path", instanceMap, bindingMap)
					_, err := controller.CreateServiceInstance(testLogger, "service-instance-guid", instance)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal(fmt.Sprintf("open /non-existent-path/service_instances.json: no such file or directory")))
				})

			})
		})
		Context(".ServiceInstanceExists", func() {
			var (
				instance model.ServiceInstance
			)
			BeforeEach(func() {
				instance = model.ServiceInstance{}
				instance.PlanId = "some-planId"
				instance.Parameters = map[string]interface{}{"some-property": "some-value"}

			})
			It("should confirm existence of service instance", func() {
				successfullServiceInstanceCreate(testLogger, fakeSystemUtil, instance, controller, serviceGuid)
				serviceExists := controller.ServiceInstanceExists(testLogger, serviceGuid)
				Expect(serviceExists).To(Equal(true))
			})
			It("should confirm non-existence of service instance", func() {
				serviceExists := controller.ServiceInstanceExists(testLogger, serviceGuid)
				Expect(serviceExists).To(Equal(false))
			})
		})
		Context(".ServiceInstancePropertiesMatch", func() {
			var (
				instance model.ServiceInstance
			)
			BeforeEach(func() {
				instance = model.ServiceInstance{}
				instance.PlanId = "some-planId"
				instance.Parameters = map[string]interface{}{"some-property": "some-value"}

			})
			It("should return true if properties match", func() {
				successfullServiceInstanceCreate(testLogger, fakeSystemUtil, instance, controller, serviceGuid)
				anotherInstance := model.ServiceInstance{}
				properties := map[string]interface{}{"some-property": "some-value"}
				anotherInstance.Parameters = properties
				anotherInstance.PlanId = "some-planId"
				propertiesMatch := controller.ServiceInstancePropertiesMatch(testLogger, serviceGuid, anotherInstance)
				Expect(propertiesMatch).To(Equal(true))
			})
			It("should return false if properties do not match", func() {
				successfullServiceInstanceCreate(testLogger, fakeSystemUtil, instance, controller, serviceGuid)
				anotherInstance := model.ServiceInstance{}
				properties := map[string]interface{}{"some-property": "some-value"}
				anotherInstance.Parameters = properties
				anotherInstance.PlanId = "some-other-planId"
				propertiesMatch := controller.ServiceInstancePropertiesMatch(testLogger, serviceGuid, anotherInstance)
				Expect(propertiesMatch).ToNot(Equal(true))
			})
		})
		Context(".ServiceInstanceDelete", func() {
			var (
				instance model.ServiceInstance
			)
			BeforeEach(func() {
				instance = model.ServiceInstance{}
				instance.PlanId = "some-planId"
				instance.Parameters = map[string]interface{}{"some-property": "some-value"}
			})
			It("should delete service instance", func() {
				successfullServiceInstanceCreate(testLogger, fakeSystemUtil, instance, controller, serviceGuid)
				err := controller.DeleteServiceInstance(testLogger, serviceGuid)
				Expect(err).ToNot(HaveOccurred())

				serviceExists := controller.ServiceInstanceExists(testLogger, serviceGuid)
				Expect(serviceExists).To(Equal(false))
			})
			It("should error when trying to delete non-existence service instance", func() {
				fakeSystemUtil.RemoveReturns(fmt.Errorf("error-in-delete-share"))
				err := controller.DeleteServiceInstance(testLogger, serviceGuid)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(fmt.Sprintf("failed to delete share '%s'", path.Join(localMountPoint, serviceGuid))))
			})
			It("should error when updating internal bookkeeping fails", func() {
				controller = NewController(cephClient, "/non-existent-path", instanceMap, bindingMap)
				err := controller.DeleteServiceInstance(testLogger, serviceGuid)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(fmt.Sprintf("open /non-existent-path/service_instances.json: no such file or directory")))
			})

		})
	})
})

var _ = Describe("RealInvoker", func() {
	var (
		subject    Invoker
		fakeCmd    *execfakes.FakeCmd
		fakeExec   *execfakes.FakeExec
		testLogger = lagertest.NewTestLogger("InvokerTest")
		cmd        = "some-fake-command"
		args       = []string{"fake-args-1"}
	)
	Context("when invoking an executable", func() {
		BeforeEach(func() {
			fakeExec = new(execfakes.FakeExec)
			fakeCmd = new(execfakes.FakeCmd)
			fakeExec.CommandReturns(fakeCmd)
			subject = NewRealInvokerWithExec(fakeExec)
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

func successfullServiceInstanceCreate(testLogger lager.Logger, fakeSystemUtil *cephfakes.FakeSystemUtil, instance model.ServiceInstance, controller Controller, serviceGuid string) {
	fakeSystemUtil.MkdirAllReturns(nil)
	createResponse, err := controller.CreateServiceInstance(testLogger, serviceGuid, instance)
	Expect(err).ToNot(HaveOccurred())
	Expect(createResponse.DashboardUrl).ToNot(Equal(""))
	Expect(fakeSystemUtil.MkdirAllCallCount()).To(Equal(2))
}
