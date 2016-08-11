package cephbrokerlocal

import (
	"fmt"
	"path"
	"reflect"
	"strings"

	"encoding/json"

	"code.cloudfoundry.org/cephbroker/model"
	"code.cloudfoundry.org/cephbroker/utils"
	"code.cloudfoundry.org/goshims/ioutil"
	"code.cloudfoundry.org/goshims/os"
	"code.cloudfoundry.org/lager"
)

const (
	DEFAULT_POLLING_INTERVAL_SECONDS = 10
	DEFAULT_CONTAINER_PATH           = "/var/vcap/data/"
)

//go:generate counterfeiter -o ../cephfakes/fake_controller.go . Controller

type Controller interface {
	GetCatalog(logger lager.Logger) (model.Catalog, error)
	CreateServiceInstance(logger lager.Logger, serverInstanceId string, instance model.ServiceInstance) (model.CreateServiceInstanceResponse, error)
	ServiceInstanceExists(logger lager.Logger, serviceInstanceId string) bool
	ServiceInstancePropertiesMatch(logger lager.Logger, serviceInstanceId string, instance model.ServiceInstance) bool
	DeleteServiceInstance(logger lager.Logger, serviceInstanceId string) error
	BindServiceInstance(logger lager.Logger, serverInstanceId string, bindingId string, bindingInfo model.ServiceBinding) (model.CreateServiceBindingResponse, error)
	ServiceBindingExists(logger lager.Logger, serviceInstanceId string, bindingId string) bool
	ServiceBindingPropertiesMatch(logger lager.Logger, serviceInstanceId string, bindingId string, binding model.ServiceBinding) bool
	GetBinding(logger lager.Logger, serviceInstanceId, bindingId string) (model.ServiceBinding, error)
	UnbindServiceInstance(logger lager.Logger, serviceInstanceId string, bindingId string) error
}

type cephController struct {
	cephClient  Client
	instanceMap map[string]*model.ServiceInstance
	bindingMap  map[string]*model.ServiceBinding
	configPath  string
	serviceName string
	serviceId   string
	planId      string
	planName    string
	planDesc    string
	os          osshim.Os
	ioutil      ioutilshim.Ioutil
}

func NewController(cephClient Client, serviceName, serviceId, planId, planName, planDesc, configPath string, instanceMap map[string]*model.ServiceInstance, bindingMap map[string]*model.ServiceBinding, os osshim.Os, ioutil ioutilshim.Ioutil) Controller {
	return &cephController{
		cephClient:  cephClient,
		serviceName: serviceName,
		serviceId:   serviceId,
		planId:      planId,
		planName:    planName,
		planDesc:    planDesc,
		configPath:  configPath,
		instanceMap: instanceMap,
		bindingMap:  bindingMap,
		os:          os,
		ioutil:      ioutil,
	}
}

func (c *cephController) GetCatalog(logger lager.Logger) (model.Catalog, error) {
	logger = logger.Session("get-catalog")
	logger.Info("start")
	defer logger.Info("end")
	plan := model.ServicePlan{
		Name:        c.planName, // "free"
		Id:          c.planId,   // "free-plan-guid"
		Description: c.planDesc, // "free ceph filesystem"
		Metadata:    nil,
		Free:        true,
	}

	service := model.Service{
		Name:            c.serviceName,
		Id:              c.serviceId,
		Description:     "CephFS service docs: https://code.cloudfoundry.org/cephfs-bosh-release/",
		Bindable:        true,
		PlanUpdateable:  false,
		Tags:            []string{"ceph"},
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

func (c *cephController) CreateServiceInstance(logger lager.Logger, serviceInstanceId string, instance model.ServiceInstance) (model.CreateServiceInstanceResponse, error) {
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
	mountpoint, err := c.cephClient.CreateShare(logger, serviceInstanceId)
	if err != nil {
		return model.CreateServiceInstanceResponse{}, err
	}

	instance.DashboardUrl = "http://dashboard_url"
	instance.Id = serviceInstanceId
	instance.LastOperation = &model.LastOperation{
		State:                    "in progress",
		Description:              "creating service instance...",
		AsyncPollIntervalSeconds: DEFAULT_POLLING_INTERVAL_SECONDS,
	}

	c.instanceMap[serviceInstanceId] = &instance
	err = utils.MarshalAndRecord(c.instanceMap, c.configPath, "service_instances.json", c.os, c.ioutil)
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

func (c *cephController) ServiceInstanceExists(logger lager.Logger, serviceInstanceId string) bool {
	logger = logger.Session("service-instance-exists")
	logger.Info("start")
	defer logger.Info("end")
	_, exists := c.instanceMap[serviceInstanceId]
	return exists
}

func (c *cephController) ServiceInstancePropertiesMatch(logger lager.Logger, serviceInstanceId string, instance model.ServiceInstance) bool {
	logger = logger.Session("service-instance-properties-match")
	logger.Info("start")
	defer logger.Info("end")
	existingServiceInstance, exists := c.instanceMap[serviceInstanceId]
	if exists == false {
		return false
	}
	if existingServiceInstance.PlanId != instance.PlanId {
		return false
	}
	if existingServiceInstance.SpaceGuid != instance.SpaceGuid {
		return false
	}
	if existingServiceInstance.OrganizationGuid != instance.OrganizationGuid {
		return false
	}
	areParamsEqual := reflect.DeepEqual(existingServiceInstance.Parameters, instance.Parameters)
	return areParamsEqual
}

func (c *cephController) DeleteServiceInstance(logger lager.Logger, serviceInstanceId string) error {
	logger = logger.Session("delete-service-instance")
	logger.Info("start")
	defer logger.Info("end")
	err := c.cephClient.DeleteShare(logger, serviceInstanceId)
	if err != nil {
		logger.Error("Error deleting share", err)
		return err
	}
	delete(c.instanceMap, serviceInstanceId)
	err = utils.MarshalAndRecord(c.instanceMap, c.configPath, "service_instances.json", c.os, c.ioutil)
	if err != nil {
		return err
	}
	return nil
}
func (c *cephController) BindServiceInstance(logger lager.Logger, serviceInstanceId string, bindingId string, bindingInfo model.ServiceBinding) (model.CreateServiceBindingResponse, error) {
	logger = logger.Session("bind-service-instance")
	logger.Info("start")
	defer logger.Info("end")
	c.bindingMap[bindingId] = &bindingInfo
	remoteSharePath, localMountPoint, err := c.cephClient.GetPathsForShare(logger, serviceInstanceId)
	if err != nil {
		logger.Error("failed-getting-paths-for-share", err)
		return model.CreateServiceBindingResponse{}, err
	}
	containerMountPath := determineContainerMountPath(bindingInfo.Parameters, serviceInstanceId)
	mds, keyring, err := c.cephClient.GetConfigDetails(logger)
	if err != nil {
		logger.Error("failed-to-determine-container-mountpath", err)
		return model.CreateServiceBindingResponse{}, err
	}

	mdsParts := strings.Split(mds, ":")
	cephConfig := model.CephConfig{IP: mdsParts[0], Keyring: keyring, RemoteMountPoint: remoteSharePath, LocalMountPoint: localMountPoint}
	config, err := json.Marshal(cephConfig)
	if err != nil {
		logger.Error("failed-to-marshal-cephconfig", err)
		return model.CreateServiceBindingResponse{}, err
	}
	privateDetails := model.VolumeMountPrivateDetails{Driver: "cephdriver", GroupId: serviceInstanceId, Config: string(config)}

	volumeMount := model.VolumeMount{ContainerPath: containerMountPath, Mode: "rw", Private: privateDetails}
	volumeMounts := []model.VolumeMount{volumeMount}
	creds := model.Credentials{}
	createBindingResponse := model.CreateServiceBindingResponse{Credentials: creds, VolumeMounts: volumeMounts}
	err = utils.MarshalAndRecord(c.bindingMap, c.configPath, "service_bindings.json", c.os, c.ioutil)
	if err != nil {
		logger.Error("failed-to-json-marshal-service-bindings", err)
		return model.CreateServiceBindingResponse{}, err
	}
	return createBindingResponse, nil
}

func (c *cephController) ServiceBindingExists(logger lager.Logger, serviceInstanceId string, bindingId string) bool {
	logger = logger.Session("service-binding-exists")
	logger.Info("start")
	defer logger.Info("end")
	_, exists := c.bindingMap[bindingId]
	return exists
}

func (c *cephController) ServiceBindingPropertiesMatch(logger lager.Logger, serviceInstanceId string, bindingId string, binding model.ServiceBinding) bool {
	logger = logger.Session("service-binding-properties-match")
	logger.Info("start")
	defer logger.Info("end")
	existingBinding, exists := c.bindingMap[bindingId]
	if exists == false {
		return false
	}
	if existingBinding.AppId != binding.AppId {
		return false
	}
	if existingBinding.ServicePlanId != binding.ServicePlanId {
		return false
	}
	if existingBinding.ServiceId != binding.ServiceId {
		return false
	}
	if existingBinding.ServiceInstanceId != binding.ServiceInstanceId {
		return false
	}
	if existingBinding.Id != binding.Id {
		return false
	}
	return true
}

func (c *cephController) UnbindServiceInstance(logger lager.Logger, serviceInstanceId string, bindingId string) error {
	logger = logger.Session("unbind")
	logger.Info("start")
	defer logger.Info("end")
	delete(c.bindingMap, bindingId)
	err := utils.MarshalAndRecord(c.bindingMap, c.configPath, "service_bindings.json", c.os, c.ioutil)
	if err != nil {
		logger.Error("error-unbind", err)
		return err
	}
	return nil
}

func (c *cephController) GetBinding(logger lager.Logger, instanceId, bindingId string) (model.ServiceBinding, error) {
	logger = logger.Session("get-binding")
	logger.Info("start")
	defer logger.Info("end")
	binding, exists := c.bindingMap[bindingId]
	if exists == true {
		return *binding, nil
	}
	return model.ServiceBinding{}, fmt.Errorf("binding not found")
}

func determineContainerMountPath(parameters map[string]interface{}, volId string) string {
	if containerPath, ok := parameters["container_path"]; ok {
		return containerPath.(string)
	}
	if containerPath, ok := parameters["path"]; ok {
		return containerPath.(string)
	}
	return path.Join(DEFAULT_CONTAINER_PATH, volId)
}
