package cephbrokerlocal

import (
	"github.com/cloudfoundry-incubator/cephbroker/model"
	"github.com/cloudfoundry-incubator/cephbroker/utils"
	"github.com/pivotal-golang/lager"
)

const (
	DEFAULT_POLLING_INTERVAL_SECONDS = 10
)

//go:generate counterfeiter -o ./cephfakes/fake_controller.go . Controller

type Controller interface {
	GetCatalog(logger lager.Logger) (model.Catalog, error)
	CreateServiceInstance(logger lager.Logger, instanceId string, instance model.ServiceInstance) (model.CreateServiceInstanceResponse, error)
	ServiceInstanceExists(logger lager.Logger, service_instance_id string) bool
	ServiceInstancePropertiesMatch(logger lager.Logger, service_instance_id string, instance model.ServiceInstance) bool
}

type cephController struct {
	cephClient  Client
	instanceMap map[string]*model.ServiceInstance
	bindingMap  map[string]*model.ServiceBinding
	configPath  string
}

func NewController(cephClient Client, configPath string, instanceMap map[string]*model.ServiceInstance, bindingMap map[string]*model.ServiceBinding) Controller {
	return &cephController{cephClient: cephClient, configPath: configPath, instanceMap: instanceMap, bindingMap: bindingMap}
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

func (c *cephController) CreateServiceInstance(logger lager.Logger, instanceId string, instance model.ServiceInstance) (model.CreateServiceInstanceResponse, error) {
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
	mountpoint, err := c.cephClient.CreateShare(logger, instanceId)
	if err != nil {
		return model.CreateServiceInstanceResponse{}, err
	}

	instance.DashboardUrl = "http://dashboard_url"
	instance.Id = instanceId
	instance.LastOperation = &model.LastOperation{
		State:                    "in progress",
		Description:              "creating service instance...",
		AsyncPollIntervalSeconds: DEFAULT_POLLING_INTERVAL_SECONDS,
	}

	c.instanceMap[instance.Id] = &instance
	err = utils.MarshalAndRecord(c.instanceMap, c.configPath, "service_instances.json")
	if err != nil {
		return model.CreateServiceInstanceResponse{}, err
	}

	logger.Info("mountpoint-created", lager.Data{mountpoint: mountpoint})
	response := model.CreateServiceInstanceResponse{
		DashboardUrl:  instance.DashboardUrl,
		LastOperation: instance.LastOperation,
	}
	return response, nil
}

func (c *cephController) ServiceInstanceExists(logger lager.Logger, service_instance_id string) bool {
	logger = logger.Session("service-instance-exists")
	logger.Info("start")
	defer logger.Info("end")
	_, exists := c.instanceMap[service_instance_id]
	return exists
}

func (c *cephController) ServiceInstancePropertiesMatch(logger lager.Logger, service_instance_id string, instance model.ServiceInstance) bool {
	return false
}
