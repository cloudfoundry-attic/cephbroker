package cephbrokerhttp

import (
	"net/http"

	"github.com/cloudfoundry-incubator/cephbroker"
	"github.com/cloudfoundry-incubator/cephbroker/cephbrokerlocal"
	"github.com/cloudfoundry-incubator/cephbroker/model"
	"github.com/cloudfoundry-incubator/cephbroker/utils"
	cf_http_handlers "github.com/cloudfoundry-incubator/cf_http/handlers"
	"github.com/cloudfoundry-incubator/volman"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/rata"
)

func respondWithError(logger lager.Logger, info string, err error, w http.ResponseWriter) {
	logger.Error(info, err)
	cf_http_handlers.WriteJSONResponse(w, http.StatusInternalServerError, volman.NewError(err))
}

func NewHandler(logger lager.Logger, controller cephbrokerlocal.Controller) (http.Handler, error) {
	logger = logger.Session("server")
	logger.Info("start")
	defer logger.Info("end")

	var handlers = rata.Handlers{
		"catalog": newCatalogHandler(logger, controller),
		"create":  newCreateServiceInstanceHandler(logger, controller),
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
		serviceInstanceExists := controller.ServiceInstanceExists(logger, instanceId)
		var instance model.ServiceInstance
		if serviceInstanceExists {
			err := utils.ProvisionDataFromRequest(req, &instance)

			if err != nil {
				cf_http_handlers.WriteJSONResponse(w, 409, struct{}{})
				return
			}

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
