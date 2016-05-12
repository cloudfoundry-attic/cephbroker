package cephbrokerhttp_test

import (
	"bytes"
	"encoding/json"
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
				fakeCatalog := model.Catalog{}
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
			})
		})
		Context(".ServiceInstance", func() {
			It("should produce valid create service instance response", func() {
				successfullCreateService(handler, fakeController)
			})
			It("should error if service instance already exists with different properties", func() {
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
			It("should not error if service instance already exists with same properties", func() {
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
