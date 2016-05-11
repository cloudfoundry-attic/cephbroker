package cephbrokerlocal

import "github.com/cloudfoundry-incubator/cephbroker/model"

type Controller interface {
	GetCatalog() (model.Catalog, error)
	CreateServiceInstance(service_instance_id string) (model.CreateServiceInstanceResponse, error)
	ServiceInstanceExists(service_instance_id string) bool
	ServiceInstancePropertiesMatch(service_instance_id string, properties map[string]interface{}) bool
}

type CephController struct {
}

func NewController() Controller {
	return &CephController{}
}

func (c *CephController) GetCatalog() (model.Catalog, error) {
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

func (c *CephController) CreateServiceInstance(service_instance_id string) (model.CreateServiceInstanceResponse, error) {
	//check if filesystem is mounted

	//else mount

	//create share

	response := model.CreateServiceInstanceResponse{
		DashboardUrl:  "http://dashboard_url",
		LastOperation: nil,
	}
	return response, nil
}

func (c *CephController) ServiceInstanceExists(service_instance_id string) bool {
	return false
}

func (c *CephController) ServiceInstancePropertiesMatch(service_instance_id string, properties map[string]interface{}) bool {
	return false
}
