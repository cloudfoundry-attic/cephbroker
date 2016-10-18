package cephbroker_test

import (
	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/voldriver"
	"github.com/pivotal-cf/brokerapi"

	"encoding/json"

	"sync"

	"os"

	"code.cloudfoundry.org/cephbroker/cephbroker"
	"code.cloudfoundry.org/cephbroker/cephfakes"
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"code.cloudfoundry.org/goshims/ioutilshim/ioutil_fake"
	"context"
)

type dynamicState struct {
	InstanceMap map[string]brokerapi.ProvisionDetails
	BindingMap  map[string]brokerapi.BindDetails
}

var _ = Describe("Broker", func() {
	var (
		broker             brokerapi.ServiceBroker
		fakeController     *cephfakes.FakeController
		fakeIoutil         *ioutil_fake.FakeIoutil
		logger             lager.Logger
		ctx context.Context
		WriteFileCallCount int
		WriteFileWrote     string
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test-broker")
		ctx = context.TODO()
		fakeController = &cephfakes.FakeController{}
		fakeIoutil = &ioutil_fake.FakeIoutil{}
		fakeIoutil.WriteFileStub = func(filename string, data []byte, perm os.FileMode) error {
			WriteFileCallCount++
			WriteFileWrote = string(data)
			return nil
		}
	})

	Context("when recreating", func() {
		It("should be able to bind to previously created service", func() {
			filecontents, err := json.Marshal(dynamicState{
				InstanceMap: map[string]brokerapi.ProvisionDetails{
					"service-name": {
						ServiceID:        "service-id",
						PlanID:           "plan-id",
						OrganizationGUID: "o",
						SpaceGUID:        "s",
					},
				},
				BindingMap: map[string]brokerapi.BindDetails{},
			})
			Expect(err).NotTo(HaveOccurred())
			fakeIoutil.ReadFileReturns(filecontents, nil)

			broker = cephbroker.New(
				logger, fakeController,
				"service-name", "service-id",
				"plan-name", "plan-id", "plan-desc", "/fake-dir",
				fakeIoutil,
			)

			_, err = broker.Bind(ctx, "service-name", "whatever", brokerapi.BindDetails{AppGUID: "guid", Parameters: map[string]interface{}{}})
			Expect(err).NotTo(HaveOccurred())
		})

		It("shouldn't be able to bind to service from invalid state file", func() {
			filecontents := "{serviceName: [some invalid state]}"
			fakeIoutil.ReadFileReturns([]byte(filecontents[:]), nil)

			broker = cephbroker.New(
				logger, fakeController,
				"service-name", "service-id",
				"plan-name", "plan-id", "plan-desc", "/fake-dir",
				fakeIoutil,
			)

			_, err := broker.Bind(ctx, "service-name", "whatever", brokerapi.BindDetails{AppGUID: "guid", Parameters: map[string]interface{}{}})
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when creating first time", func() {
		BeforeEach(func() {
			broker = cephbroker.New(
				logger, fakeController,
				"service-name", "service-id",
				"plan-name", "plan-id", "plan-desc", "/fake-dir",
				fakeIoutil,
			)
		})

		Context(".Services", func() {
			It("returns the service catalog as appropriate", func() {
				result := broker.Services()[0]
				Expect(result.ID).To(Equal("service-id"))
				Expect(result.Name).To(Equal("service-name"))
				Expect(result.Description).To(Equal("CephFS service docs: https://code.cloudfoundry.org/cephfs-bosh-release/"))
				Expect(result.Bindable).To(Equal(true))
				Expect(result.PlanUpdatable).To(Equal(false))
				Expect(result.Tags).To(ContainElement("ceph"))
				Expect(result.Requires).To(ContainElement(brokerapi.RequiredPermission("volume_mount")))

				Expect(result.Plans[0].Name).To(Equal("plan-name"))
				Expect(result.Plans[0].ID).To(Equal("plan-id"))
				Expect(result.Plans[0].Description).To(Equal("plan-desc"))
			})
		})

		Context(".Provision", func() {
			It("should provision the service instance", func() {
				_, err := broker.Provision(ctx, "some-instance-id", brokerapi.ProvisionDetails{}, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeController.CreateCallCount()).To(Equal(1))

				_, details := fakeController.CreateArgsForCall(0)
				Expect(err).NotTo(HaveOccurred())
				Expect(details.Name).To(Equal("some-instance-id"))
				Expect(details.Opts["volume_id"]).To(Equal("some-instance-id"))
			})

			It("should write state", func() {
				WriteFileCallCount = 0
				WriteFileWrote = ""
				_, err := broker.Provision(ctx, "some-instance-id", brokerapi.ProvisionDetails{}, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(WriteFileCallCount).To(Equal(1))
				Expect(WriteFileWrote).To(Equal("{\"InstanceMap\":{\"some-instance-id\":{\"service_id\":\"\",\"plan_id\":\"\",\"organization_guid\":\"\",\"space_guid\":\"\"}},\"BindingMap\":{}}"))
			})

			Context("when provisioning errors", func() {
				BeforeEach(func() {
					fakeController.CreateReturns(voldriver.ErrorResponse{Err: "some-error"})
				})

				It("errors", func() {
					_, err := broker.Provision(ctx, "some-instance-id", brokerapi.ProvisionDetails{}, false)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("when the service instance already exists with different details", func() {
				var details brokerapi.ProvisionDetails
				BeforeEach(func() {
					details = brokerapi.ProvisionDetails{
						ServiceID:        "service-id",
						PlanID:           "plan-id",
						OrganizationGUID: "org-guid",
						SpaceGUID:        "space-guid",
					}
					_, err := broker.Provision(ctx, "some-instance-id", details, false)
					Expect(err).NotTo(HaveOccurred())
				})

				It("should error", func() {
					details.ServiceID = "different-service-id"
					_, err := broker.Provision(ctx, "some-instance-id", details, false)
					Expect(err).To(Equal(brokerapi.ErrInstanceAlreadyExists))
				})
			})
		})

		Context(".Deprovision", func() {
			BeforeEach(func() {
				_, err := broker.Provision(ctx, "some-instance-id", brokerapi.ProvisionDetails{}, false)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should deprovision the service", func() {
				_, err := broker.Deprovision(ctx, "some-instance-id", brokerapi.DeprovisionDetails{}, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeController.RemoveCallCount()).To(Equal(1))

				By("checking that we can reprovision a slightly different service")
				_, err = broker.Provision(ctx, "some-instance-id", brokerapi.ProvisionDetails{ServiceID: "different-service-id"}, false)
				Expect(err).NotTo(Equal(brokerapi.ErrInstanceAlreadyExists))
			})

			It("errors when the service instance does not exist", func() {
				_, err := broker.Deprovision(ctx, "some-nonexistant-instance-id", brokerapi.DeprovisionDetails{}, false)
				Expect(err).To(Equal(brokerapi.ErrInstanceDoesNotExist))
			})

			It("Errors when ceph can't deprovision", func() {
				fakeController.RemoveReturns(voldriver.ErrorResponse{"something"})
				_, err := broker.Deprovision(ctx, "some-instance-id", brokerapi.DeprovisionDetails{}, false)
				Expect(err.Error()).To(Equal("something"))
			})

			It("should write state", func() {
				WriteFileCallCount = 0
				WriteFileWrote = ""
				_, err := broker.Deprovision(ctx, "some-instance-id", brokerapi.DeprovisionDetails{}, false)
				Expect(err).NotTo(HaveOccurred())

				Expect(WriteFileCallCount).To(Equal(1))
				Expect(WriteFileWrote).To(Equal("{\"InstanceMap\":{},\"BindingMap\":{}}"))
			})

			Context("when the provisioner fails to remove", func() {
				BeforeEach(func() {
					fakeController.RemoveReturns(voldriver.ErrorResponse{Err: "some-error"})
				})

				It("should error", func() {
					_, err := broker.Deprovision(ctx, "some-instance-id", brokerapi.DeprovisionDetails{}, false)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context(".Bind", func() {
			var bindDetails brokerapi.BindDetails

			BeforeEach(func() {
				_, err := broker.Provision(ctx, "some-instance-id", brokerapi.ProvisionDetails{}, false)
				Expect(err).NotTo(HaveOccurred())

				bindDetails = brokerapi.BindDetails{AppGUID: "guid", Parameters: map[string]interface{}{}}
			})

			It("includes empty credentials to prevent CAPI crash", func() {
				binding, err := broker.Bind(ctx, "some-instance-id", "binding-id", bindDetails)
				Expect(err).NotTo(HaveOccurred())

				Expect(binding.Credentials).NotTo(BeNil())
			})

			It("uses the instance id in the default container path", func() {
				binding, err := broker.Bind(ctx, "some-instance-id", "binding-id", bindDetails)
				Expect(err).NotTo(HaveOccurred())
				Expect(binding.VolumeMounts[0].ContainerDir).To(Equal("/var/vcap/data/some-instance-id"))
			})

			It("flows container path through", func() {
				bindDetails.Parameters["mount"] = "/var/vcap/otherdir/something"
				binding, err := broker.Bind(ctx, "some-instance-id", "binding-id", bindDetails)
				Expect(err).NotTo(HaveOccurred())
				Expect(binding.VolumeMounts[0].ContainerDir).To(Equal("/var/vcap/otherdir/something"))
			})

			It("uses rw as its default mode", func() {
				binding, err := broker.Bind(ctx, "some-instance-id", "binding-id", bindDetails)
				Expect(err).NotTo(HaveOccurred())
				Expect(binding.VolumeMounts[0].Mode).To(Equal("rw"))
			})

			It("sets mode to `r` when readonly is true", func() {
				bindDetails.Parameters["readonly"] = true
				binding, err := broker.Bind(ctx, "some-instance-id", "binding-id", bindDetails)
				Expect(err).NotTo(HaveOccurred())

				Expect(binding.VolumeMounts[0].Mode).To(Equal("r"))
			})

			It("should write state", func() {
				WriteFileCallCount = 0
				WriteFileWrote = ""
				_, err := broker.Bind(ctx, "some-instance-id", "binding-id", bindDetails)
				Expect(err).NotTo(HaveOccurred())

				Expect(WriteFileCallCount).To(Equal(1))
				Expect(WriteFileWrote).To(Equal("{\"InstanceMap\":{\"some-instance-id\":{\"service_id\":\"\",\"plan_id\":\"\",\"organization_guid\":\"\",\"space_guid\":\"\"}},\"BindingMap\":{\"binding-id\":{\"app_guid\":\"guid\",\"plan_id\":\"\",\"service_id\":\"\"}}}"))
			})

			It("errors if mode is not a boolean", func() {
				bindDetails.Parameters["readonly"] = ""
				_, err := broker.Bind(ctx, "some-instance-id", "binding-id", bindDetails)
				Expect(err).To(Equal(brokerapi.ErrRawParamsInvalid))
			})

			It("fills in the driver name", func() {
				binding, err := broker.Bind(ctx, "some-instance-id", "binding-id", bindDetails)
				Expect(err).NotTo(HaveOccurred())

				Expect(binding.VolumeMounts[0].Driver).To(Equal("cephdriver"))
			})

			It("fills in the group id", func() {
				fakeController.BindReturns(cephbroker.BindResponse{SharedDevice: brokerapi.SharedDevice{VolumeId: "some-instance-id"}})
				binding, err := broker.Bind(ctx, "some-instance-id", "binding-id", bindDetails)
				Expect(err).NotTo(HaveOccurred())

				Expect(binding.VolumeMounts[0].Device.VolumeId).To(Equal("some-instance-id"))
			})

			Context("when the binding already exists", func() {
				BeforeEach(func() {
					_, err := broker.Bind(ctx, "some-instance-id", "binding-id", brokerapi.BindDetails{AppGUID: "guid"})
					Expect(err).NotTo(HaveOccurred())
				})

				It("doesn't error when binding the same details", func() {
					_, err := broker.Bind(ctx, "some-instance-id", "binding-id", brokerapi.BindDetails{AppGUID: "guid"})
					Expect(err).NotTo(HaveOccurred())
				})

				It("errors when binding different details", func() {
					_, err := broker.Bind(ctx, "some-instance-id", "binding-id", brokerapi.BindDetails{AppGUID: "different"})
					Expect(err).To(Equal(brokerapi.ErrBindingAlreadyExists))
				})
			})

			It("errors when the service instance does not exist", func() {
				_, err := broker.Bind(ctx, "nonexistant-instance-id", "binding-id", brokerapi.BindDetails{AppGUID: "guid"})
				Expect(err).To(Equal(brokerapi.ErrInstanceDoesNotExist))
			})

			It("errors when the app guid is not provided", func() {
				_, err := broker.Bind(ctx, "some-instance-id", "binding-id", brokerapi.BindDetails{})
				Expect(err).To(Equal(brokerapi.ErrAppGuidNotProvided))
			})

			It("Errors when ceph can't bind", func() {
				fakeController.BindReturns(cephbroker.BindResponse{voldriver.ErrorResponse{"something"}, brokerapi.SharedDevice{}})
				_, err := broker.Bind(ctx, "some-instance-id", "binding-id", bindDetails)
				Expect(err.Error()).To(Equal("something"))
			})
		})

		Context(".Unbind", func() {
			BeforeEach(func() {
				_, err := broker.Provision(ctx, "some-instance-id", brokerapi.ProvisionDetails{}, false)
				Expect(err).NotTo(HaveOccurred())

				_, err = broker.Bind(ctx, "some-instance-id", "binding-id", brokerapi.BindDetails{AppGUID: "guid"})
				Expect(err).NotTo(HaveOccurred())
			})

			It("unbinds a bound service instance from an app", func() {
				err := broker.Unbind(ctx, "some-instance-id", "binding-id", brokerapi.UnbindDetails{})
				Expect(err).NotTo(HaveOccurred())
			})

			It("fails when trying to unbind a instance that has not been provisioned", func() {
				err := broker.Unbind(ctx, "some-other-instance-id", "binding-id", brokerapi.UnbindDetails{})
				Expect(err).To(Equal(brokerapi.ErrInstanceDoesNotExist))
			})

			It("fails when trying to unbind a binding that has not been bound", func() {
				err := broker.Unbind(ctx, "some-instance-id", "some-other-binding-id", brokerapi.UnbindDetails{})
				Expect(err).To(Equal(brokerapi.ErrBindingDoesNotExist))
			})
			It("should write state", func() {
				WriteFileCallCount = 0
				WriteFileWrote = ""
				err := broker.Unbind(ctx, "some-instance-id", "binding-id", brokerapi.UnbindDetails{})
				Expect(err).NotTo(HaveOccurred())

				Expect(WriteFileCallCount).To(Equal(1))
				Expect(WriteFileWrote).To(Equal("{\"InstanceMap\":{\"some-instance-id\":{\"service_id\":\"\",\"plan_id\":\"\",\"organization_guid\":\"\",\"space_guid\":\"\"}},\"BindingMap\":{}}"))
			})

		})
		Context("when multiple operations happen in parallel", func() {
			It("maintains consistency", func() {
				var wg sync.WaitGroup

				wg.Add(2)

				smash := func(uniqueName string) {
					defer GinkgoRecover()
					defer wg.Done()

					broker.Services()

					_, err := broker.Provision(ctx, uniqueName, brokerapi.ProvisionDetails{}, false)
					Expect(err).NotTo(HaveOccurred())

					_, err = broker.Bind(ctx, uniqueName, "binding-id", brokerapi.BindDetails{AppGUID: "guid"})
					Expect(err).NotTo(HaveOccurred())

					err = broker.Unbind(ctx, uniqueName, "some-other-binding-id", brokerapi.UnbindDetails{})
					Expect(err).To(Equal(brokerapi.ErrBindingDoesNotExist))

					_, err = broker.Deprovision(ctx, uniqueName, brokerapi.DeprovisionDetails{}, false)
					Expect(err).NotTo(HaveOccurred())
				}

				// Note go race detection should kick in if access is unsynchronized
				go smash("some-instance-1")
				go smash("some-instance-2")

				wg.Wait()
			})
		})
	})
})
