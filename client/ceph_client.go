package client

import (
	"fmt"
	"path/filepath"

	"github.com/cloudfoundry-incubator/cephbroker/utils"
	"github.com/cloudfoundry/gunk/os_wrap/exec_wrap"
	"github.com/pivotal-golang/lager"
)

type Client interface {
	IsFilesystemMounted(lager.Logger) bool
	RootMountPoint() string
	MountFileSystem(lager.Logger, string) (string, error)
	CreateShare(lager.Logger, string) (string, error)
	DeleteShare(lager.Logger, string) error
}

type CephClient struct {
	Ceph_mds            string
	MountTargetLocation string
	Cmd                 exec_wrap.Exec
}

func NewCephClient(mds string, targetLocation string) *CephClient {
	return &CephClient{
		Ceph_mds:            mds,
		MountTargetLocation: targetLocation,
	}
}
func (c *CephClient) RootMountPoint() string {
	return c.MountTargetLocation
}
func (c *CephClient) IsFilesystemMounted(logger lager.Logger) bool {
	logger = logger.Session("is-filesystem-mounted")
	logger.Info("start")
	defer logger.Info("end")
	return false
}

func (c *CephClient) MountFileSystem(logger lager.Logger, mountRoot string) (string, error) {
	logger = logger.Session("mount-filesystem")
	logger.Info("start")
	defer logger.Info("end")

	mountPoint := filepath.Join(c.MountTargetLocation, mountRoot)

	err := utils.MkDir(mountPoint)
	if err != nil {
		logger.Error("error-in-mkdir", err)
		return "", err
	}

	cmdArgs := []string{"-m", c.Ceph_mds, mountPoint}

	err = c.InvokeCeph(logger, "ceph-fuse", cmdArgs)
	if err != nil {
		logger.Error("error-in-invoking-ceph", err)
		return "", err

	}

	return mountPoint, nil
}

func (c *CephClient) CreateShare(logger lager.Logger, shareName string) (string, error) {
	logger = logger.Session("create-share")
	logger.Info("start")
	defer logger.Info("end")

	mountPoint := filepath.Join(c.MountTargetLocation, shareName)
	err := utils.MkDir(mountPoint)
	if err != nil {

		logger.Error("error-in-mkdir", err)
		return "", err
	}
	cmdArgs := []string{"-m", c.Ceph_mds, "-r", fmt.Sprintf("/", shareName), mountPoint}

	err = c.InvokeCeph(logger, "ceph-fuse", cmdArgs)
	if err != nil {

		logger.Error("error-invoking-ceph", err)
		return "", err
	}

	return mountPoint, nil
}

func (c *CephClient) DeleteShare(logger lager.Logger, shareName string) error {
	logger = logger.Session("delete-share")
	logger.Info("start")
	defer logger.Info("end")

	mountPoint := filepath.Join(c.MountTargetLocation, shareName)
	cmdArgs := []string{mountPoint}

	err := c.InvokeCeph(logger, "umount", cmdArgs)
	if err != nil {

		logger.Error("error-invoking-ceph", err)
		return err
	}

	return nil
}

func (c *CephClient) InvokeCeph(logger lager.Logger, executable string, cmdArgs []string) error {
	logger = logger.Session("invoke")
	logger.Info("start")
	defer logger.Info("end")

	exec := exec_wrap.NewExec()
	cmdHandle := exec.Command(executable, cmdArgs...)
	logger.Info("cmd-data", lager.Data{"args": cmdArgs})
	_, err := cmdHandle.StdoutPipe()
	if err != nil {
		logger.Error("unable to get stdout", err)
		return err
	}

	if err = cmdHandle.Start(); err != nil {
		logger.Error("starting command", err)
		return err
	}

	if err = cmdHandle.Wait(); err != nil {
		logger.Error("command-exited", err)
		return err
	}

	return nil
}
