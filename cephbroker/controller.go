package cephbroker

import (
	"strings"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/voldriver"
	"github.com/pivotal-cf/brokerapi"
	"code.cloudfoundry.org/voldriver/driverhttp"
)

type BindResponse struct {
	voldriver.ErrorResponse
	SharedDevice brokerapi.SharedDevice
}

//go:generate counterfeiter -o ../cephfakes/fake_controller.go . Controller

type Controller interface {
	voldriver.Provisioner
	Bind(env voldriver.Env, instanceID string) BindResponse
}

type controller struct {
	cephClient Client
}

func NewController(cephClient Client) Controller {
	return &controller{cephClient: cephClient}
}

func (p *controller) Create(env voldriver.Env, createRequest voldriver.CreateRequest) voldriver.ErrorResponse {
	logger := env.Logger().Session("provision")
	logger.Info("start")
	defer logger.Info("end")

	mounted := p.cephClient.IsFilesystemMounted(driverhttp.EnvWithLogger(logger,env))
	if !mounted {
		_, err := p.cephClient.MountFileSystem(driverhttp.EnvWithLogger(logger,env), "/")
		if err != nil {
			return voldriver.ErrorResponse{Err: err.Error()}
		}
	}
	mountpoint, err := p.cephClient.CreateShare(driverhttp.EnvWithLogger(logger,env), createRequest.Name)
	if err != nil {
		return voldriver.ErrorResponse{Err: err.Error()}
	}

	logger.Info("mountpoint-created", lager.Data{mountpoint: mountpoint})

	return voldriver.ErrorResponse{}
}

func (p *controller) Remove(env voldriver.Env, removeRequest voldriver.RemoveRequest) voldriver.ErrorResponse {
	logger := env.Logger().Session("remove")
	logger.Info("start")
	defer logger.Info("end")
	err := p.cephClient.DeleteShare(driverhttp.EnvWithLogger(logger,env), removeRequest.Name)
	if err != nil {
		logger.Error("Error deleting share", err)
		return voldriver.ErrorResponse{Err: err.Error()}
	}
	return voldriver.ErrorResponse{}
}

func (p *controller) Bind(env voldriver.Env, instanceID string) BindResponse {
	logger := env.Logger().Session("bind-service-instance")
	logger.Info("start")
	defer logger.Info("end")
	response := BindResponse{}

	remoteSharePath, localMountPoint, err := p.cephClient.GetPathsForShare(driverhttp.EnvWithLogger(logger,env), instanceID)
	if err != nil {
		logger.Error("failed-getting-paths-for-share", err)
		response.Err = err.Error()
		return response
	}

	mds, keyring, err := p.cephClient.GetConfigDetails(driverhttp.EnvWithLogger(logger,env))
	if err != nil {
		logger.Error("failed-to-determine-container-mountpath", err)
		response.Err = err.Error()
		return response
	}

	mdsParts := strings.Split(mds, ":")

	return BindResponse{
		SharedDevice: brokerapi.SharedDevice{
			VolumeId: instanceID,
			MountConfig: map[string]interface{}{
				"ip":                 mdsParts[0],
				"keyring":            keyring,
				"remote_mount_point": remoteSharePath,
				"local_mount_point":  localMountPoint,
			},
		},
	}
}
