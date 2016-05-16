package cephbrokerhttp_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry-incubator/cephbroker/cephbrokerhttp"
	"github.com/cloudfoundry-incubator/cephbroker/cephbrokerlocal/cephfakes"
	"github.com/cloudfoundry-incubator/cephbroker/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("Cephbroker Handlers", func() {

	Context("when generating handlers", func() {
		var (
			testLogger     lager.Logger
			fakeController *cephfakes.FakeController
			handler        http.Handler
		)
		BeforeEach(func() {
			testLogger = lagertest.NewTestLogger("HandlersTest")
			fakeController = new(cephfakes.FakeController)
			handler, _ = cephbrokerhttp.NewHandler(testLogger, fakeController)
		})
		Context(".Catalog", func() {
			It("should produce valid catalog response", func() {
				fakeServices := []model.Service{model.Service{Id: "some-service-id"}}
				fakeCatalog := model.Catalog{
					Services: fakeServices,
				}
				fakeController.GetCatalogReturns(fakeCatalog, nil)
				w := httptest.NewRecorder()
				r, _ := http.NewRequest("GET", "http://0.0.0.0/v2/catalog", nil)
				handler.ServeHTTP(w, r)
				Expect(w.Code).Should(Equal(200))
				catalog := model.Catalog{}
				body, err := ioutil.ReadAll(w.Body)
				Expect(err).ToNot(HaveOccurred())
				err = json.Unmarshal(body, &catalog)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(catalog.Services)).To(Equal(1))
			})
			It("should error on catalog generation error", func() {
				fakeCatalog := model.Catalog{}
				fakeController.GetCatalogReturns(fakeCatalog, fmt.Errorf("Error building catalog"))
				w := httptest.NewRecorder()
				r, _ := http.NewRequest("GET", "http://0.0.0.0/v2/catalog", nil)
				handler.ServeHTTP(w, r)
				Expect(w.Code).Should(Equal(200))
				catalog := model.Catalog{}
				body, err := ioutil.ReadAll(w.Body)
				Expect(err).ToNot(HaveOccurred())
				err = json.Unmarshal(body, &catalog)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(catalog.Services)).To(Equal(0))
			})

		})
		Context(".ServiceInstanceCreate", func() {
			It("should produce valid create service instance response", func() {
				successfullCreateService(handler, fakeController)
			})
			It("should return 409 if service instance already exists with different properties", func() {
				successfullCreateService(handler, fakeController)
				fakeController.ServiceInstanceExistsReturns(true)
				fakeController.ServiceInstancePropertiesMatchReturns(false)
				fakeCreateResponse := model.CreateServiceInstanceResponse{}
				fakeController.CreateServiceInstanceReturns(fakeCreateResponse, nil)
				serviceInstance := model.ServiceInstance{
					Id:               "ceph-service-guid",
					DashboardUrl:     "http://dashboard_url",
					InternalId:       "ceph-service-guid",
					ServiceId:        "ceph-service-guid",
					PlanId:           "free-plan-guid",
					OrganizationGuid: "organization-guid",
					SpaceGuid:        "space-guid",
					LastOperation:    nil,
					Parameters:       "parameters",
				}
				w := httptest.NewRecorder()
				payload, err := json.Marshal(serviceInstance)
				Expect(err).ToNot(HaveOccurred())
				reader := bytes.NewReader(payload)
				r, _ := http.NewRequest("PUT", "http://0.0.0.0/v2/service_instances/cephfs-service-guid", reader)
				handler.ServeHTTP(w, r)
				Expect(w.Code).Should(Equal(409))
			})
			It("should return 409 if service details not valid json", func() {
				w := httptest.NewRecorder()
				reader := bytes.NewReader([]byte(""))
				r, _ := http.NewRequest("PUT", "http://0.0.0.0/v2/service_instances/cephfs-service-guid", reader)
				handler.ServeHTTP(w, r)
				Expect(w.Code).Should(Equal(409))
			})
			It("should return 409 if service creation fails", func() {
				serviceInstance := model.ServiceInstance{
					Id:               "ceph-service-guid",
					DashboardUrl:     "http://dashboard_url",
					InternalId:       "ceph-service-guid",
					ServiceId:        "ceph-service-guid",
					PlanId:           "free-plan-guid",
					OrganizationGuid: "organization-guid",
					SpaceGuid:        "space-guid",
					LastOperation:    nil,
					Parameters:       "parameters",
				}
				payload, err := json.Marshal(serviceInstance)
				Expect(err).ToNot(HaveOccurred())
				reader := bytes.NewReader(payload)
				fakeController.ServiceInstanceExistsReturns(false)
				fakeCreateResponse := model.CreateServiceInstanceResponse{}
				fakeController.CreateServiceInstanceReturns(fakeCreateResponse, fmt.Errorf("Error creating service instance"))
				w := httptest.NewRecorder()
				r, _ := http.NewRequest("PUT", "http://0.0.0.0/v2/service_instances/cephfs-service-guid", reader)
				handler.ServeHTTP(w, r)
				Expect(w.Code).Should(Equal(409))
			})
			It("should return 200 if service instance already exists with same properties", func() {
				successfullCreateService(handler, fakeController)
				fakeController.ServiceInstanceExistsReturns(true)
				fakeController.ServiceInstancePropertiesMatchReturns(true)
				fakeCreateResponse := model.CreateServiceInstanceResponse{}
				fakeController.CreateServiceInstanceReturns(fakeCreateResponse, nil)
				serviceInstance := model.ServiceInstance{
					Id:               "ceph-service-guid",
					DashboardUrl:     "http://dashboard_url",
					InternalId:       "ceph-service-guid",
					ServiceId:        "ceph-service-guid",
					PlanId:           "free-plan-guid",
					OrganizationGuid: "organization-guid",
					SpaceGuid:        "space-guid",
					LastOperation:    nil,
					Parameters:       "parameters",
				}
				w := httptest.NewRecorder()
				payload, err := json.Marshal(serviceInstance)
				Expect(err).ToNot(HaveOccurred())
				reader := bytes.NewReader(payload)
				r, _ := http.NewRequest("PUT", "http://0.0.0.0/v2/service_instances/cephfs-service-guid", reader)
				handler.ServeHTTP(w, r)
				Expect(w.Code).Should(Equal(200))
			})
		})
		Context(".ServiceInstanceDelete", func() {
			It("should produce valid delete service instance response", func() {
				successfullCreateService(handler, fakeController)
				successfullDeleteService(handler, fakeController)
			})
			It("should return 410 if service instance does not exist", func() {
				serviceInstance := model.ServiceInstance{
					Id:               "ceph-service-guid",
					DashboardUrl:     "http://dashboard_url",
					InternalId:       "ceph-service-guid",
					ServiceId:        "ceph-service-guid",
					PlanId:           "free-plan-guid",
					OrganizationGuid: "organization-guid",
					SpaceGuid:        "space-guid",
					LastOperation:    nil,
					Parameters:       "parameters",
				}
				w := httptest.NewRecorder()
				payload, err := json.Marshal(serviceInstance)
				Expect(err).ToNot(HaveOccurred())
				reader := bytes.NewReader(payload)
				r, _ := http.NewRequest("DELETE", "http://0.0.0.0/v2/service_instances/cephfs-service-guid", reader)
				handler.ServeHTTP(w, r)
				Expect(w.Code).Should(Equal(410))
			})
			It("should return 409 if service instance deletion fails", func() {
				fakeController.ServiceInstanceExistsReturns(true)
				fakeController.DeleteServiceInstanceReturns(fmt.Errorf("error deleting service instance"))
				serviceInstance := model.ServiceInstance{
					Id:               "ceph-service-guid",
					DashboardUrl:     "http://dashboard_url",
					InternalId:       "ceph-service-guid",
					ServiceId:        "ceph-service-guid",
					PlanId:           "free-plan-guid",
					OrganizationGuid: "organization-guid",
					SpaceGuid:        "space-guid",
					LastOperation:    nil,
					Parameters:       "parameters",
				}
				w := httptest.NewRecorder()
				payload, err := json.Marshal(serviceInstance)
				Expect(err).ToNot(HaveOccurred())
				reader := bytes.NewReader(payload)
				r, _ := http.NewRequest("DELETE", "http://0.0.0.0/v2/service_instances/cephfs-service-guid", reader)
				handler.ServeHTTP(w, r)
				Expect(w.Code).Should(Equal(409))
			})
		})
		Context(".ServiceInstanceBind", func() {
			It("should produce valid bind service instance response", func() {
				successfullCreateService(handler, fakeController)
				successfullBindService(handler, fakeController)
			})
			It("should return 409 if binding already exists with different properties", func() {
				successfullCreateService(handler, fakeController)
				successfullBindService(handler, fakeController)
				fakeController.ServiceBindingExistsReturns(true)
				fakeController.ServiceBindingPropertiesMatchReturns(false)
				fakeBindResponse := model.CreateServiceBindingResponse{}
				fakeController.BindServiceInstanceReturns(fakeBindResponse, nil)
				binding := model.ServiceBinding{
					Id: "ceph-service-guid",
				}
				w := httptest.NewRecorder()
				payload, err := json.Marshal(binding)
				Expect(err).ToNot(HaveOccurred())
				reader := bytes.NewReader(payload)
				r, _ := http.NewRequest("PUT", "http://0.0.0.0/v2/service_instances/cephfs-service-guid/service_bindings/cephfs-service-binding-guid", reader)
				handler.ServeHTTP(w, r)
				Expect(w.Code).Should(Equal(409))
			})
			It("should return 409 if service details not valid json", func() {
				w := httptest.NewRecorder()
				reader := bytes.NewReader([]byte(""))
				r, _ := http.NewRequest("PUT", "http://0.0.0.0/v2/service_instances/cephfs-service-guid/service_bindings/cephfs-service-binding-guid", reader)
				handler.ServeHTTP(w, r)
				Expect(w.Code).Should(Equal(409))
			})
			It("should return 409 if service binding fails", func() {
				binding := model.ServiceBinding{
					Id: "ceph-service-guid",
				}
				payload, err := json.Marshal(binding)
				Expect(err).ToNot(HaveOccurred())
				reader := bytes.NewReader(payload)
				fakeController.ServiceBindingExistsReturns(false)
				fakeBindingResponse := model.CreateServiceBindingResponse{}
				fakeController.BindServiceInstanceReturns(fakeBindingResponse, fmt.Errorf("Error binding service instance"))
				w := httptest.NewRecorder()
				r, _ := http.NewRequest("PUT", "http://0.0.0.0/v2/service_instances/cephfs-service-guid/service_bindings/cephfs-service-binding-guid", reader)
				handler.ServeHTTP(w, r)
				Expect(w.Code).Should(Equal(409))
			})
			It("should return 200 if service instance already exists with same properties", func() {
				successfullCreateService(handler, fakeController)
				successfullBindService(handler, fakeController)
				fakeController.ServiceBindingExistsReturns(true)
				fakeController.ServiceBindingPropertiesMatchReturns(true)
				fakeBindingResponse := model.CreateServiceBindingResponse{}
				fakeController.BindServiceInstanceReturns(fakeBindingResponse, nil)
				binding := model.ServiceBinding{
					Id: "ceph-service-guid",
				}
				w := httptest.NewRecorder()
				payload, err := json.Marshal(binding)
				Expect(err).ToNot(HaveOccurred())
				reader := bytes.NewReader(payload)
				r, _ := http.NewRequest("PUT", "http://0.0.0.0/v2/service_instances/cephfs-service-guid/service_bindings/cephfs-service-binding-guid", reader)
				handler.ServeHTTP(w, r)
				Expect(w.Code).Should(Equal(200))
			})
		})
		Context(".ServiceInstanceUnBind", func() {
			It("should produce valid unbind service instance response", func() {
				successfullCreateService(handler, fakeController)
				successfullBindService(handler, fakeController)

				fakeController.ServiceBindingExistsReturns(true)
				binding := model.ServiceBinding{
					Id:            "ceph-service-guid",
					ServiceId:     "ceph-service-guid",
					ServicePlanId: "some-plan_id",
				}
				w := httptest.NewRecorder()
				payload, err := json.Marshal(binding)
				Expect(err).ToNot(HaveOccurred())
				reader := bytes.NewReader(payload)
				r, _ := http.NewRequest("DELETE", "http://0.0.0.0/v2/service_instances/cephfs-service-guid/service_bindings/cephfs-service-binding-guid", reader)
				handler.ServeHTTP(w, r)
				Expect(w.Code).Should(Equal(200))

			})
			It("should return 410 if binding does not exist", func() {
				successfullCreateService(handler, fakeController)
				successfullBindService(handler, fakeController)
				fakeController.ServiceBindingExistsReturns(false)
				fakeController.UnbindServiceInstanceReturns(fmt.Errorf("binding does not exist"))
				binding := model.ServiceBinding{
					Id: "ceph-service-guid",
				}
				w := httptest.NewRecorder()
				payload, err := json.Marshal(binding)
				Expect(err).ToNot(HaveOccurred())
				reader := bytes.NewReader(payload)
				r, _ := http.NewRequest("DELETE", "http://0.0.0.0/v2/service_instances/cephfs-service-guid/service_bindings/cephfs-service-binding-guid", reader)
				handler.ServeHTTP(w, r)
				Expect(w.Code).Should(Equal(410))
			})
		})

	})
})

func successfullCreateService(handler http.Handler, fakeController *cephfakes.FakeController) {
	fakeCreateResponse := model.CreateServiceInstanceResponse{}
	fakeController.CreateServiceInstanceReturns(fakeCreateResponse, nil)
	serviceInstance := model.ServiceInstance{
		Id:               "ceph-service-guid",
		DashboardUrl:     "http://dashboard_url",
		InternalId:       "ceph-service-guid",
		ServiceId:        "ceph-service-guid",
		PlanId:           "free-plan-guid",
		OrganizationGuid: "organization-guid",
		SpaceGuid:        "space-guid",
		LastOperation:    nil,
		Parameters:       "parameters",
	}
	w := httptest.NewRecorder()
	payload, err := json.Marshal(serviceInstance)
	Expect(err).ToNot(HaveOccurred())
	reader := bytes.NewReader(payload)
	r, _ := http.NewRequest("PUT", "http://0.0.0.0/v2/service_instances/cephfs-service-guid", reader)
	handler.ServeHTTP(w, r)
	Expect(w.Code).Should(Equal(201))
	body, err := ioutil.ReadAll(w.Body)
	Expect(err).ToNot(HaveOccurred())
	createServiceResponse := model.CreateServiceInstanceResponse{}
	err = json.Unmarshal(body, &createServiceResponse)
	Expect(err).ToNot(HaveOccurred())
}

func successfullDeleteService(handler http.Handler, fakeController *cephfakes.FakeController) {
	serviceInstance := model.ServiceInstance{
		Id:               "ceph-service-guid",
		DashboardUrl:     "http://dashboard_url",
		InternalId:       "ceph-service-guid",
		ServiceId:        "ceph-service-guid",
		PlanId:           "free-plan-guid",
		OrganizationGuid: "organization-guid",
		SpaceGuid:        "space-guid",
		LastOperation:    nil,
		Parameters:       "parameters",
	}
	fakeController.ServiceInstanceExistsReturns(true)
	w := httptest.NewRecorder()
	payload, err := json.Marshal(serviceInstance)
	Expect(err).ToNot(HaveOccurred())
	reader := bytes.NewReader(payload)
	r, _ := http.NewRequest("DELETE", "http://0.0.0.0/v2/service_instances/cephfs-service-guid", reader)
	handler.ServeHTTP(w, r)
	Expect(w.Code).Should(Equal(200))
}

func successfullBindService(handler http.Handler, fakeController *cephfakes.FakeController) {
	fakeBindResponse := model.CreateServiceBindingResponse{}
	fakeController.BindServiceInstanceReturns(fakeBindResponse, nil)
	binding := model.ServiceBinding{
		Id: "ceph-service-guid",
	}
	w := httptest.NewRecorder()
	payload, err := json.Marshal(binding)
	Expect(err).ToNot(HaveOccurred())
	reader := bytes.NewReader(payload)
	r, _ := http.NewRequest("PUT", "http://0.0.0.0/v2/service_instances/cephfs-service-guid/service_bindings/cephfs-service-binding-guid", reader)
	handler.ServeHTTP(w, r)
	Expect(w.Code).Should(Equal(201))
	body, err := ioutil.ReadAll(w.Body)
	Expect(err).ToNot(HaveOccurred())
	bindingResponse := model.CreateServiceBindingResponse{}
	err = json.Unmarshal(body, &bindingResponse)
	Expect(err).ToNot(HaveOccurred())
}
