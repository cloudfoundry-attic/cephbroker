package cephbrokerlocal_test

import (
	"bytes"
	"fmt"
	"path"

	. "code.cloudfoundry.org/cephbroker/cephbrokerlocal"
	"code.cloudfoundry.org/cephbroker/cephfakes"
	"code.cloudfoundry.org/cephbroker/model"
	"github.com/cloudfoundry/gunk/os_wrap/exec_wrap/execfakes"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"os"
)

const AN_ERROR = "An Error"

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
		planId          string
		planName        string
		planDesc        string
	)
	BeforeEach(func() {
		planName = "free"
		planId = "free-plan-guid"
		planDesc = "free ceph filesystem"
		testLogger = lagertest.NewTestLogger("ControllerTest")
		fakeInvoker = new(cephfakes.FakeInvoker)
		serviceGuid = "some-service-guid"
		fakeSystemUtil = new(cephfakes.FakeSystemUtil)
		localMountPoint = "/tmp/share"
		cephClient = NewCephClientWithInvokerAndSystemUtil("some-mds-url:9999", fakeInvoker, fakeSystemUtil, localMountPoint, "/some-keyring-file")
		instanceMap = make(map[string]*model.ServiceInstance)
		bindingMap = make(map[string]*model.ServiceBinding)
		controller = NewController(cephClient, "service-name", "service-id", planId, planName, planDesc, "/tmp/cephbroker", instanceMap, bindingMap, fakeSystemUtil)

	})
	Context(".Catalog", func() {
		It("should produce a valid catalog", func() {
			catalog, err := controller.GetCatalog(testLogger)
			Expect(err).ToNot(HaveOccurred())

			Expect(catalog).ToNot(BeNil())
			Expect(catalog.Services).ToNot(BeNil())
			Expect(len(catalog.Services)).To(Equal(1))

			Expect(catalog.Services[0].Name).To(Equal("service-name"))
			Expect(catalog.Services[0].Id).To(Equal("service-id"))

			Expect(catalog.Services[0].Requires).ToNot(BeNil())
			Expect(len(catalog.Services[0].Requires)).To(Equal(1))
			Expect(catalog.Services[0].Requires[0]).To(Equal("volume_mount"))

			Expect(catalog.Services[0].Plans).ToNot(BeNil())
			Expect(len(catalog.Services[0].Plans)).To(Equal(1))
			Expect(catalog.Services[0].Plans[0].Name).To(Equal(planName))

			Expect(catalog.Services[0].Bindable).To(Equal(true))
			Expect(catalog.Services[0].PlanUpdateable).To(Equal(false))
		})
	})
	DONT_CARE_ERROR := "An Error"
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
				const BAD_PATH = "/non-existent-path"
				controller = NewController(cephClient, "service-name", "service-id", planId, planName, planDesc, BAD_PATH, instanceMap, bindingMap, fakeSystemUtil)
				fakeSystemUtil.MkdirAllStub = func(path string, _ os.FileMode) error {
					if path == BAD_PATH {
  					return fmt.Errorf(DONT_CARE_ERROR)
					}
					return nil
				}

				_, err := controller.CreateServiceInstance(testLogger, "service-instance-guid", instance)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(fmt.Sprintf(DONT_CARE_ERROR)))
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
			controller = NewController(cephClient, "service-name", "service-id", planId, planName, planDesc, "/non-existent-path", instanceMap, bindingMap, fakeSystemUtil)
			fakeSystemUtil.MkdirAllReturns(fmt.Errorf(AN_ERROR))
			err := controller.DeleteServiceInstance(testLogger, serviceGuid)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf(AN_ERROR)))
		})

	})
	Context(".BindServiceInstance", func() {
		var (
			instance    model.ServiceInstance
			bindingInfo model.ServiceBinding
		)
		BeforeEach(func() {
			instance = model.ServiceInstance{}
			instance.PlanId = "some-planId"
			instance.Parameters = map[string]interface{}{"some-property": "some-value"}
			bindingInfo = model.ServiceBinding{}
			successfullServiceInstanceCreate(testLogger, fakeSystemUtil, instance, controller, serviceGuid)
		})
		It("should be able bind service instance", func() {
			fakeSystemUtil.ExistsReturns(true)
			fakeSystemUtil.ReadFileReturns([]byte("some keyring content"), nil)
			bindingInfo.Parameters = map[string]interface{}{"container_path": "/some-user-specified-path"}
			bindingResponse, err := controller.BindServiceInstance(testLogger, serviceGuid, "some-binding-id", bindingInfo)
			Expect(err).ToNot(HaveOccurred())
			Expect(bindingResponse.VolumeMounts).ToNot(BeNil())
			Expect(len(bindingResponse.VolumeMounts)).To(Equal(1))
			Expect(bindingResponse.VolumeMounts[0].ContainerPath).To(Equal("/some-user-specified-path"))
			Expect(bindingResponse.VolumeMounts[0].Private).ToNot(BeNil())
			Expect(bindingResponse.VolumeMounts[0].Private.Driver).To(Equal("cephdriver"))
			Expect(bindingResponse.VolumeMounts[0].Private.Config).ToNot(BeNil())
			Expect(bindingResponse.VolumeMounts[0].Private.Config).To(ContainSubstring("some-mds"))
			Expect(bindingResponse.VolumeMounts[0].Private.Config).NotTo(ContainSubstring("9999"))
			Expect(bindingResponse.VolumeMounts[0].Private.Config).To(ContainSubstring("some keyring content"))
		})
		Context("should fail", func() {
			It("when unable to find the backing share", func() {
				fakeSystemUtil.ExistsReturns(false)
				_, err := controller.BindServiceInstance(testLogger, serviceGuid, "some-binding-id", bindingInfo)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("share not found, internal error"))
			})
			It("when updating internal bookkeeping fails", func() {
				controller = NewController(cephClient, "service-name", "service-id", planId, planName, planDesc, "/non-existent-path", instanceMap, bindingMap, fakeSystemUtil)
				fakeSystemUtil.MkdirAllReturns(fmt.Errorf(DONT_CARE_ERROR))
				fakeSystemUtil.ExistsReturns(true)
				_, err := controller.BindServiceInstance(testLogger, serviceGuid, "some-binding-id", bindingInfo)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(fmt.Sprintf(DONT_CARE_ERROR)))
			})
		})
	})
	Context(".ServiceBindingExists", func() {
		var (
			instance  model.ServiceInstance
			bindingId string
		)
		BeforeEach(func() {
			instance = model.ServiceInstance{}
			instance.PlanId = "some-planId"
			instance.Parameters = map[string]interface{}{"some-property": "some-value"}
			bindingId = "some-binding-id"
		})
		It("should confirm existence of service instance", func() {
			binding := model.ServiceBinding{}
			successfullServiceInstanceCreate(testLogger, fakeSystemUtil, instance, controller, serviceGuid)
			successfullServiceBindingCreate(testLogger, fakeSystemUtil, binding, controller, serviceGuid, bindingId)
			bindingExists := controller.ServiceBindingExists(testLogger, serviceGuid, bindingId)
			Expect(bindingExists).To(Equal(true))
		})
		It("should confirm non-existence of service binding", func() {
			bindingExists := controller.ServiceBindingExists(testLogger, serviceGuid, bindingId)
			Expect(bindingExists).To(Equal(false))
		})
	})
	Context(".ServiceBindingPropertiesMatch", func() {
		var (
			instance  model.ServiceInstance
			bindingId string
		)
		BeforeEach(func() {
			instance = model.ServiceInstance{}
			instance.PlanId = "some-planId"
			instance.Parameters = map[string]interface{}{"some-property": "some-value"}
			bindingId = "some-binding-id"

		})
		It("should return true if properties match", func() {
			binding := model.ServiceBinding{}
			successfullServiceInstanceCreate(testLogger, fakeSystemUtil, instance, controller, serviceGuid)
			successfullServiceBindingCreate(testLogger, fakeSystemUtil, binding, controller, serviceGuid, bindingId)
			anotherBinding := model.ServiceBinding{}
			propertiesMatch := controller.ServiceBindingPropertiesMatch(testLogger, serviceGuid, bindingId, anotherBinding)
			Expect(propertiesMatch).To(Equal(true))
		})
		It("should return false if properties do not match", func() {
			binding := model.ServiceBinding{}
			successfullServiceInstanceCreate(testLogger, fakeSystemUtil, instance, controller, serviceGuid)
			successfullServiceBindingCreate(testLogger, fakeSystemUtil, binding, controller, serviceGuid, bindingId)
			anotherBinding := model.ServiceBinding{}
			anotherBinding.AppId = "some-other-appId"
			propertiesMatch := controller.ServiceBindingPropertiesMatch(testLogger, serviceGuid, bindingId, anotherBinding)
			Expect(propertiesMatch).ToNot(Equal(true))
		})
	})
	Context(".ServiceInstanceUnbind", func() {
		var (
			instance  model.ServiceInstance
			bindingId string
		)
		BeforeEach(func() {
			instance = model.ServiceInstance{}
			instance.PlanId = "some-planId"
			instance.Parameters = map[string]interface{}{"some-property": "some-value"}
			bindingId = "some-binding-id"

			binding := model.ServiceBinding{}
			successfullServiceInstanceCreate(testLogger, fakeSystemUtil, instance, controller, serviceGuid)
			successfullServiceBindingCreate(testLogger, fakeSystemUtil, binding, controller, serviceGuid, bindingId)
		})
		It("should delete service binding", func() {
			err := controller.UnbindServiceInstance(testLogger, serviceGuid, bindingId)
			Expect(err).ToNot(HaveOccurred())

			exists := controller.ServiceBindingExists(testLogger, serviceGuid, bindingId)
			Expect(exists).To(Equal(false))
		})
		It("when updating internal bookkeeping fails", func() {
			controller = NewController(cephClient, "service-name", "service-id", planId, planName, planDesc, "/non-existent-path", instanceMap, bindingMap, fakeSystemUtil)
			fakeSystemUtil.MkdirAllReturns(fmt.Errorf(AN_ERROR))
			err := controller.UnbindServiceInstance(testLogger, serviceGuid, bindingId)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(AN_ERROR))
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
	Expect(fakeSystemUtil.MkdirAllCallCount()).To(Equal(3))
}

func successfullServiceBindingCreate(testLogger lager.Logger, fakeSystemUtil *cephfakes.FakeSystemUtil, binding model.ServiceBinding, controller Controller, serviceGuid string, bindingId string) {
	fakeSystemUtil.ExistsReturns(true)
	bindResponse, err := controller.BindServiceInstance(testLogger, serviceGuid, bindingId, binding)
	Expect(err).ToNot(HaveOccurred())
	Expect(bindResponse.VolumeMounts).ToNot(BeNil())
	Expect(len(bindResponse.VolumeMounts)).To(Equal(1))
}
