package cephbrokerlocal

import (
	"github.com/cloudfoundry-incubator/cephbroker/model"
	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter -o ./cephfakes/fake_controller.go . Controller

type Controller interface {
	GetCatalog(logger lager.Logger) (model.Catalog, error)
	CreateServiceInstance(logger lager.Logger, service_instance_id string, properties interface{}) (model.CreateServiceInstanceResponse, error)
	ServiceInstanceExists(logger lager.Logger, service_instance_id string) bool
	ServiceInstancePropertiesMatch(logger lager.Logger, service_instance_id string, properties interface{}) bool
}

type cephController struct {
	cephClient Client
}

func NewController(cephClient Client) Controller {
	return &cephController{cephClient: cephClient}
}

func (c *cephController) GetCatalog(logger lager.Logger) (model.Catalog, error) {
	logger = logger.Session("get-catalog")
	logger.Info("start")
	defer logger.Info("end")
	plan := model.ServicePlan{
		Name:        "free",
		Id:          "free-plan-guid",
		Description: "free ceph filesystem",
		Metadata:    nil,
		Free:        true,
	}

	service := model.Service{
		Name:            "cephfs",
		Id:              "cephfs-service-guid",
		Description:     "Provides the Ceph FS volume service, including volume creation and volume mounts",
		Bindable:        true,
		PlanUpdateable:  false,
		Tags:            nil,
		Requires:        []string{"volume_mount"},
		Metadata:        nil,
		Plans:           []model.ServicePlan{plan},
		DashboardClient: nil,
	}
	catalog := model.Catalog{
		Services: []model.Service{service},
	}
	return catalog, nil
}

func (c *cephController) CreateServiceInstance(logger lager.Logger, service_instance_id string, properties interface{}) (model.CreateServiceInstanceResponse, error) {
	logger = logger.Session("create-service-instance")
	logger.Info("start")
	defer logger.Info("end")
	mounted := c.cephClient.IsFilesystemMounted(logger)
	if !mounted {
		_, err := c.cephClient.MountFileSystem(logger, "/")
		if err != nil {
			return model.CreateServiceInstanceResponse{}, err
		}

	}
	mountpoint, err := c.cephClient.CreateShare(logger, service_instance_id)
	if err != nil {
		return model.CreateServiceInstanceResponse{}, err
	}
	logger.Info("mountpoint-created", lager.Data{mountpoint: mountpoint})
	response := model.CreateServiceInstanceResponse{
		DashboardUrl:  "http://dashboard_url",
		LastOperation: nil,
	}
	return response, nil
}

func (c *cephController) ServiceInstanceExists(logger lager.Logger, service_instance_id string) bool {
	return false
}

func (c *cephController) ServiceInstancePropertiesMatch(logger lager.Logger, service_instance_id string, properties interface{}) bool {
	return false
}
