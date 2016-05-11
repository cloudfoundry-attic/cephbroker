package cephbrokerhttp

import (
	"net/http"

	"github.com/cloudfoundry-incubator/cephbroker"
	"github.com/cloudfoundry-incubator/cephbroker/cephbrokerlocal"
	"github.com/cloudfoundry-incubator/cephbroker/model"
	cf_http_handlers "github.com/cloudfoundry-incubator/cf_http/handlers"
	"github.com/cloudfoundry-incubator/volman"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/rata"
)

func respondWithError(logger lager.Logger, info string, err error, w http.ResponseWriter) {
	logger.Error(info, err)
	cf_http_handlers.WriteJSONResponse(w, http.StatusInternalServerError, volman.NewError(err))
}

func NewHandler(logger lager.Logger) (http.Handler, error) {
	logger = logger.Session("server")
	logger.Info("start")
	defer logger.Info("end")

	controller := cephbrokerlocal.NewController()

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

		catalog, err := controller.GetCatalog()
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
		serviceInstanceExists := controller.ServiceInstanceExists(instanceId)
		if serviceInstanceExists == true {
			if controller.ServiceInstancePropertiesMatch(instanceId, nil) == true {
				// 200
				response := model.CreateServiceInstanceResponse{
					DashboardUrl:  "http://dashboard_url",
					LastOperation: nil,
				}

				cf_http_handlers.WriteJSONResponse(w, 200, response)
				return
			} else {
				//409
				cf_http_handlers.WriteJSONResponse(w, 409, struct{}{})
				return
			}
		}
		createResponse, err := controller.CreateServiceInstance(instanceId)
		if err != nil {

			cf_http_handlers.WriteJSONResponse(w, 201, createResponse)
			return
		}
		cf_http_handlers.WriteJSONResponse(w, 201, createResponse)
	}
}
