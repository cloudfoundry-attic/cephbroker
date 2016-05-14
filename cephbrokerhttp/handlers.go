package cephbrokerhttp

import (
	"net/http"

	"github.com/cloudfoundry-incubator/cephbroker"
	"github.com/cloudfoundry-incubator/cephbroker/cephbrokerlocal"
	"github.com/cloudfoundry-incubator/cephbroker/model"
	"github.com/cloudfoundry-incubator/cephbroker/utils"
	cf_http_handlers "github.com/cloudfoundry-incubator/cf_http/handlers"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/rata"
)

func NewHandler(logger lager.Logger, controller cephbrokerlocal.Controller) (http.Handler, error) {
	logger = logger.Session("server")
	logger.Info("start")
	defer logger.Info("end")

	var handlers = rata.Handlers{
		"catalog": newCatalogHandler(logger, controller),
		"create":  newCreateServiceInstanceHandler(logger, controller),
		"delete":  newDeleteServiceInstanceHandler(logger, controller),
		"bind":    newBindServiceInstanceHandler(logger, controller),
	}

	return rata.NewRouter(cephbroker.Routes, handlers)
}

func newCatalogHandler(logger lager.Logger, controller cephbrokerlocal.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		logger := logger.Session("catalog")
		logger.Info("start")
		defer logger.Info("end")

		catalog, err := controller.GetCatalog(logger)
		if err != nil {
			cf_http_handlers.WriteJSONResponse(w, http.StatusOK, struct{}{})
			return
		}
		cf_http_handlers.WriteJSONResponse(w, http.StatusOK, catalog)

	}
}
func newCreateServiceInstanceHandler(logger lager.Logger, controller cephbrokerlocal.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		logger := logger.Session("create")
		logger.Info("start")
		instanceId := rata.Param(req, "service_instance_guid")
		logger.Info("instance-id", lager.Data{"id": instanceId})
		var instance model.ServiceInstance
		err := utils.UnmarshallDataFromRequest(req, &instance)
		if err != nil {
			cf_http_handlers.WriteJSONResponse(w, 409, struct{}{})
			return
		}
		serviceInstanceExists := controller.ServiceInstanceExists(logger, instanceId)
		if serviceInstanceExists {
			if controller.ServiceInstancePropertiesMatch(logger, instanceId, instance) == true {
				response := model.CreateServiceInstanceResponse{
					DashboardUrl:  "http://dashboard_url",
					LastOperation: nil,
				}
				cf_http_handlers.WriteJSONResponse(w, 200, response)
				return
			} else {
				cf_http_handlers.WriteJSONResponse(w, 409, struct{}{})
				return
			}
		}
		createResponse, err := controller.CreateServiceInstance(logger, instanceId, instance)
		if err != nil {
			cf_http_handlers.WriteJSONResponse(w, 409, struct{}{})
			return
		}
		cf_http_handlers.WriteJSONResponse(w, 201, createResponse)
	}
}
func newDeleteServiceInstanceHandler(logger lager.Logger, controller cephbrokerlocal.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		logger := logger.Session("delete")
		logger.Info("start")
		instanceId := rata.Param(req, "service_instance_guid")
		logger.Info("instance-id", lager.Data{"id": instanceId})
		serviceInstanceExists := controller.ServiceInstanceExists(logger, instanceId)
		if serviceInstanceExists == false {
			cf_http_handlers.WriteJSONResponse(w, 410, struct{}{})
			return
		}
		err := controller.DeleteServiceInstance(logger, instanceId)
		if err != nil {
			cf_http_handlers.WriteJSONResponse(w, 409, struct{}{})
			return
		}
		cf_http_handlers.WriteJSONResponse(w, 200, struct{}{})
	}
}
func newBindServiceInstanceHandler(logger lager.Logger, controller cephbrokerlocal.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		logger := logger.Session("bind")
		logger.Info("start")
		instanceId := rata.Param(req, "service_instance_guid")
		logger.Info("instance-id", lager.Data{"id": instanceId})
		bindingId := rata.Param(req, "service_binding_id")
		logger.Info("binding-id", lager.Data{"id": bindingId})
		var binding model.ServiceBinding
		err := utils.UnmarshallDataFromRequest(req, &binding)
		if err != nil {
			cf_http_handlers.WriteJSONResponse(w, 409, struct{}{})
			return
		}
		serviceBindingExists := controller.ServiceBindingExists(logger, instanceId, bindingId)
		if serviceBindingExists {
			if controller.ServiceBindingPropertiesMatch(logger, instanceId, bindingId, binding) == true {
				response, err := controller.GetBinding(logger, instanceId, bindingId)
				if err != nil {
					cf_http_handlers.WriteJSONResponse(w, 409, struct{}{})
					return
				}
				cf_http_handlers.WriteJSONResponse(w, 200, response)
				return
			} else {
				cf_http_handlers.WriteJSONResponse(w, 409, struct{}{})
				return
			}
		}
		bindResponse, err := controller.BindServiceInstance(logger, instanceId, bindingId, binding)
		if err != nil {
			cf_http_handlers.WriteJSONResponse(w, 409, struct{}{})
			return
		}
		cf_http_handlers.WriteJSONResponse(w, 201, bindResponse)
	}
}
