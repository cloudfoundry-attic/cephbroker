package cephbroker

import (
	"strings"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/voldriver"
	"github.com/pivotal-cf/brokerapi"
)

type BindResponse struct {
	voldriver.ErrorResponse
	SharedDevice brokerapi.SharedDevice
}

//go:generate counterfeiter -o ../cephfakes/fake_controller.go . Controller

type Controller interface {
	voldriver.Provisioner
	Bind(logger lager.Logger, instanceID string) BindResponse
}

type controller struct {
	cephClient Client
}

func NewController(cephClient Client) Controller {
	return &controller{cephClient: cephClient}
}

func (p *controller) Create(logger lager.Logger, createRequest voldriver.CreateRequest) voldriver.ErrorResponse {
	logger = logger.Session("provision")
	logger.Info("start")
	defer logger.Info("end")

	mounted := p.cephClient.IsFilesystemMounted(logger)
	if !mounted {
		_, err := p.cephClient.MountFileSystem(logger, "/")
		if err != nil {
			return voldriver.ErrorResponse{Err: err.Error()}
		}
	}
	mountpoint, err := p.cephClient.CreateShare(logger, createRequest.Name)
	if err != nil {
		return voldriver.ErrorResponse{Err: err.Error()}
	}

	logger.Info("mountpoint-created", lager.Data{mountpoint: mountpoint})

	return voldriver.ErrorResponse{}
}

func (p *controller) Remove(logger lager.Logger, removeRequest voldriver.RemoveRequest) voldriver.ErrorResponse {
	logger = logger.Session("remove")
	logger.Info("start")
	defer logger.Info("end")
	err := p.cephClient.DeleteShare(logger, removeRequest.Name)
	if err != nil {
		logger.Error("Error deleting share", err)
		return voldriver.ErrorResponse{Err: err.Error()}
	}
	return voldriver.ErrorResponse{}
}

func (p *controller) Bind(logger lager.Logger, instanceID string) BindResponse {
	logger = logger.Session("bind-service-instance")
	logger.Info("start")
	defer logger.Info("end")
	response := BindResponse{}

	remoteSharePath, localMountPoint, err := p.cephClient.GetPathsForShare(logger, instanceID)
	if err != nil {
		logger.Error("failed-getting-paths-for-share", err)
		response.Err = err.Error()
		return response
	}

	mds, keyring, err := p.cephClient.GetConfigDetails(logger)
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
