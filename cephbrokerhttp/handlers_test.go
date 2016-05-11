package cephbrokerhttp_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry-incubator/cephbroker/cephbrokerhttp"
	"github.com/cloudfoundry-incubator/cephbroker/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("Cephbroker Handlers", func() {

	Context("when generating http handlers", func() {

		It("should produce handler with catalog route", func() {
			testLogger := lagertest.NewTestLogger("HandlersTest")
			handler, _ := cephbrokerhttp.NewHandler(testLogger)
			w := httptest.NewRecorder()
			r, _ := http.NewRequest("GET", "http://0.0.0.0/v2/catalog", nil)
			handler.ServeHTTP(w, r)
			Expect(w.Code).Should(Equal(200))
			catalog := model.Catalog{}
			body, err := ioutil.ReadAll(w.Body)
			Expect(err).ToNot(HaveOccurred())
			err = json.Unmarshal(body, &catalog)
			Expect(err).ToNot(HaveOccurred())
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
		It("should produce handler with create service instance route", func() {
			testLogger := lagertest.NewTestLogger("HandlersTest")
			handler, _ := cephbrokerhttp.NewHandler(testLogger)
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

		})

	})
})
