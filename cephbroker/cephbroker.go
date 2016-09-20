package cephbroker

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sync"

	ioutilshim "code.cloudfoundry.org/goshims/ioutil"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/voldriver"
	"github.com/pivotal-cf/brokerapi"
)

const (
	PermissionVolumeMount = brokerapi.RequiredPermission("volume_mount")
	DefaultContainerPath  = "/var/vcap/data"
)

type staticState struct {
	ServiceName string `json:"ServiceName"`
	ServiceId   string `json:"ServiceId"`
	PlanName    string `json:"PlanName"`
	PlanId      string `json:"PlanId"`
	PlanDesc    string `json:"PlanDesc"`
}

type dynamicState struct {
	InstanceMap map[string]brokerapi.ProvisionDetails
	BindingMap  map[string]brokerapi.BindDetails
}

type lock interface {
	Lock()
	Unlock()
}

type broker struct {
	logger     lager.Logger
	controller Controller
	dataDir    string
	ioutil     ioutilshim.Ioutil
	mutex      lock

	static  staticState
	dynamic dynamicState
}

func New(
	logger lager.Logger, controller Controller,
	serviceName, serviceId, planName, planId, planDesc, dataDir string,
	ioutil ioutilshim.Ioutil,
) *broker {

	theBroker := broker{
		logger:     logger,
		controller: controller,
		dataDir:    dataDir,
		ioutil:     ioutil,
		mutex:      &sync.Mutex{},
		static: staticState{
			ServiceName: serviceName,
			ServiceId:   serviceId,
			PlanName:    planName,
			PlanId:      planId,
			PlanDesc:    planDesc,
		},
		dynamic: dynamicState{
			InstanceMap: map[string]brokerapi.ProvisionDetails{},
			BindingMap:  map[string]brokerapi.BindDetails{},
		},
	}

	theBroker.restoreDynamicState()

	return &theBroker
}

func (b *broker) Services() []brokerapi.Service {
	logger := b.logger.Session("services")
	logger.Info("start")
	defer logger.Info("end")

	return []brokerapi.Service{{
		ID:            b.static.ServiceId,
		Name:          b.static.ServiceName,
		Description:   "CephFS service docs: https://code.cloudfoundry.org/cephfs-bosh-release/",
		Bindable:      true,
		PlanUpdatable: false,
		Tags:          []string{"ceph"},
		Requires:      []brokerapi.RequiredPermission{PermissionVolumeMount},

		Plans: []brokerapi.ServicePlan{{
			Name:        b.static.PlanName,
			ID:          b.static.PlanId,
			Description: b.static.PlanDesc,
			Free:        new(bool),
		}},
	}}
}

func (b *broker) Provision(instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, error) {
	logger := b.logger.Session("provision")
	logger.Info("start")
	defer logger.Info("end")

	b.mutex.Lock()
	defer b.mutex.Unlock()

	defer b.serialize(b.dynamic)

	if b.instanceConflicts(details, instanceID) {
		logger.Error("instance-already-exists", brokerapi.ErrInstanceAlreadyExists)
		return brokerapi.ProvisionedServiceSpec{}, brokerapi.ErrInstanceAlreadyExists
	}

	errResp := b.controller.Create(logger, voldriver.CreateRequest{
		Name: instanceID,
		Opts: map[string]interface{}{"volume_id": instanceID},
	})

	if errResp.Err != "" {
		err := errors.New(errResp.Err)
		logger.Error("provisioner-create-failed", err)
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	b.dynamic.InstanceMap[instanceID] = details

	return brokerapi.ProvisionedServiceSpec{}, nil
}

func (b *broker) Deprovision(instanceID string, details brokerapi.DeprovisionDetails, asyncAllowed bool) (brokerapi.DeprovisionServiceSpec, error) {
	logger := b.logger.Session("deprovision")
	logger.Info("start")
	defer logger.Info("end")

	b.mutex.Lock()
	defer b.mutex.Unlock()

	defer b.serialize(b.dynamic)

	if _, ok := b.dynamic.InstanceMap[instanceID]; !ok {
		return brokerapi.DeprovisionServiceSpec{}, brokerapi.ErrInstanceDoesNotExist
	}

	errResp := b.controller.Remove(logger, voldriver.RemoveRequest{
		Name: instanceID,
	})

	if errResp.Err != "" {
		err := errors.New(errResp.Err)
		logger.Error("provisioner-remove-failed", err)
		return brokerapi.DeprovisionServiceSpec{}, err
	}

	delete(b.dynamic.InstanceMap, instanceID)

	return brokerapi.DeprovisionServiceSpec{}, nil
}

func (b *broker) Bind(instanceID string, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, error) {
	logger := b.logger.Session("bind")
	logger.Info("start")
	defer logger.Info("end")

	b.mutex.Lock()
	defer b.mutex.Unlock()

	defer b.serialize(b.dynamic)

	if _, ok := b.dynamic.InstanceMap[instanceID]; !ok {
		return brokerapi.Binding{}, brokerapi.ErrInstanceDoesNotExist
	}

	if details.AppGUID == "" {
		return brokerapi.Binding{}, brokerapi.ErrAppGuidNotProvided
	}

	mode, err := evaluateMode(details.Parameters)
	if err != nil {
		return brokerapi.Binding{}, err
	}

	if b.bindingConflicts(bindingID, details) {
		return brokerapi.Binding{}, brokerapi.ErrBindingAlreadyExists
	}

	response := b.controller.Bind(logger, instanceID)

	if response.Err != "" {
		err := errors.New(response.Err)
		logger.Error("provisioner-remove-failed", err)
		return brokerapi.Binding{}, err
	}

	b.dynamic.BindingMap[bindingID] = details

	return brokerapi.Binding{
		Credentials: struct{}{}, // if nil, cloud controller chokes on response
		VolumeMounts: []brokerapi.VolumeMount{{
			ContainerDir: evaluateContainerPath(details.Parameters, instanceID),
			Mode:         mode,
			Driver:       "cephdriver",
			DeviceType:   "shared",
			Device:       response.SharedDevice,
		}},
	}, nil
}

func (b *broker) Unbind(instanceID string, bindingID string, details brokerapi.UnbindDetails) error {
	logger := b.logger.Session("unbind")
	logger.Info("start")
	defer logger.Info("end")

	b.mutex.Lock()
	defer b.mutex.Unlock()

	defer b.serialize(b.dynamic)

	if _, ok := b.dynamic.InstanceMap[instanceID]; !ok {
		return brokerapi.ErrInstanceDoesNotExist
	}

	if _, ok := b.dynamic.BindingMap[bindingID]; !ok {
		return brokerapi.ErrBindingDoesNotExist
	}

	delete(b.dynamic.BindingMap, bindingID)

	return nil
}

func (b *broker) Update(instanceID string, details brokerapi.UpdateDetails, asyncAllowed bool) (brokerapi.UpdateServiceSpec, error) {
	panic("not implemented")
}

func (b *broker) LastOperation(instanceID string, operationData string) (brokerapi.LastOperation, error) {
	panic("not implemented")
}

func (b *broker) instanceConflicts(details brokerapi.ProvisionDetails, instanceID string) bool {
	if existing, ok := b.dynamic.InstanceMap[instanceID]; ok {
		if !reflect.DeepEqual(details, existing) {
			return true
		}
	}
	return false
}

func evaluateContainerPath(parameters map[string]interface{}, volId string) string {
	if containerPath, ok := parameters["mount"]; ok && containerPath != "" {
		return containerPath.(string)
	}

	return path.Join(DefaultContainerPath, volId)
}

func evaluateMode(parameters map[string]interface{}) (string, error) {
	if ro, ok := parameters["readonly"]; ok {
		switch ro := ro.(type) {
		case bool:
			return readOnlyToMode(ro), nil
		default:
			return "", brokerapi.ErrRawParamsInvalid
		}
	}
	return "rw", nil
}

func readOnlyToMode(ro bool) string {
	if ro {
		return "r"
	}
	return "rw"
}

func (b *broker) bindingConflicts(bindingID string, details brokerapi.BindDetails) bool {
	if existing, ok := b.dynamic.BindingMap[bindingID]; ok {
		if !reflect.DeepEqual(details, existing) {
			return true
		}
	}
	return false
}

func (b *broker) serialize(state interface{}) {
	logger := b.logger.Session("serialize-state")
	logger.Info("start")
	defer logger.Info("end")

	stateFile := filepath.Join(b.dataDir, fmt.Sprintf("%s-services.json", b.static.ServiceName))

	stateData, err := json.Marshal(state)
	if err != nil {
		b.logger.Error("failed-to-marshall-state", err)
		return
	}

	err = b.ioutil.WriteFile(stateFile, stateData, os.ModePerm)
	if err != nil {
		b.logger.Error(fmt.Sprintf("failed-to-write-state-file: %s", stateFile), err)
		return
	}

	logger.Info("state-saved", lager.Data{"state-file": stateFile})
}

func (b *broker) restoreDynamicState() {
	logger := b.logger.Session("restore-services")
	logger.Info("start")
	defer logger.Info("end")

	stateFile := filepath.Join(b.dataDir, fmt.Sprintf("%s-services.json", b.static.ServiceName))

	serviceData, err := b.ioutil.ReadFile(stateFile)
	if err != nil {
		b.logger.Error(fmt.Sprintf("failed-to-read-state-file: %s", stateFile), err)
		return
	}

	dynamicState := dynamicState{}
	err = json.Unmarshal(serviceData, &dynamicState)
	if err != nil {
		b.logger.Error(fmt.Sprintf("failed-to-unmarshall-state from state-file: %s", stateFile), err)
		return
	}
	logger.Info("state-restored", lager.Data{"state-file": stateFile})
	b.dynamic = dynamicState
}
