package cephbrokerlocal

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/gunk/os_wrap/exec_wrap"
	"github.com/pivotal-golang/lager"
)

type Client interface {
	IsFilesystemMounted(lager.Logger) bool
	MountFileSystem(lager.Logger, string) (string, error)
	CreateShare(lager.Logger, string) (string, error)
	DeleteShare(lager.Logger, string) error
}

type CephClient struct {
	mds                 string
	invoker             Invoker
	systemUtil          SystemUtil
	baseLocalMountPoint string
	mounted             bool
}

func NewCephClientWithInvokerAndSystemUtil(mds string, useInvoker Invoker, useSystemUtil SystemUtil, localMountPoint string) *CephClient {
	return &CephClient{
		mds:                 mds,
		invoker:             useInvoker,
		systemUtil:          useSystemUtil,
		baseLocalMountPoint: localMountPoint,
		mounted:             false,
	}
}
func NewCephClient(mds string, localMountPoint string) *CephClient {
	return &CephClient{
		mds:                 mds,
		invoker:             NewRealInvoker(),
		systemUtil:          NewRealSystemUtil(),
		baseLocalMountPoint: localMountPoint,
		mounted:             false,
	}
}
func (c *CephClient) IsFilesystemMounted(logger lager.Logger) bool {
	logger = logger.Session("is-filesystem-mounted")
	logger.Info("start")
	defer logger.Info("end")
	return c.mounted
}

func (c *CephClient) MountFileSystem(logger lager.Logger, remoteMountPoint string) (string, error) {
	logger = logger.Session("mount-filesystem")
	logger.Info("start")
	defer logger.Info("end")

	err := c.systemUtil.MkdirAll(c.baseLocalMountPoint, os.ModePerm)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create local directory '%s', mount filesystem failed", c.baseLocalMountPoint), err)
		return "", fmt.Errorf("failed to create local directory '%s', mount filesystem failed", c.baseLocalMountPoint)
	}

	cmdArgs := []string{"-m", c.mds, "-r", remoteMountPoint, c.baseLocalMountPoint}
	err = c.invokeCeph(logger, cmdArgs)
	if err != nil {
		logger.Error("cephfs-error", err)
		return "", err
	}
	c.mounted = true
	return c.baseLocalMountPoint, nil
}

func (c *CephClient) CreateShare(logger lager.Logger, shareName string) (string, error) {
	logger = logger.Session("create-share")
	logger.Info("start")
	defer logger.Info("end")
	logger.Info("share-name", lager.Data{shareName: shareName})
	sharePath := filepath.Join(c.baseLocalMountPoint, shareName)
	err := c.systemUtil.MkdirAll(sharePath, os.ModePerm)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create share '%s'", sharePath), err)
		return "", fmt.Errorf("failed to create share '%s'", sharePath)
	}
	return sharePath, nil
}

func (c *CephClient) DeleteShare(logger lager.Logger, shareName string) error {
	logger = logger.Session("delete-share")
	logger.Info("start")
	defer logger.Info("end")

	sharePath := filepath.Join(c.baseLocalMountPoint, shareName)
	err := c.systemUtil.Remove(sharePath)
	if err != nil {
		logger.Error("error-in-delete-share", err)
		return err
	}
	return nil
}

func (c *CephClient) invokeCeph(logger lager.Logger, args []string) error {
	cmd := "ceph-fuse"
	return c.invoker.Invoke(logger, cmd, args)
}

//go:generate counterfeiter -o ./cephfakes/fake_system_util.go . SystemUtil

type SystemUtil interface {
	MkdirAll(path string, perm os.FileMode) error
	WriteFile(filename string, data []byte, perm os.FileMode) error
	Remove(string) error
}
type realSystemUtil struct{}

func NewRealSystemUtil() SystemUtil {
	return &realSystemUtil{}
}

func (f *realSystemUtil) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (f *realSystemUtil) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return ioutil.WriteFile(filename, data, perm)
}

func (f *realSystemUtil) Remove(path string) error {
	return os.Remove(path)
}

//go:generate counterfeiter -o ./cephfakes/fake_invoker.go . Invoker

type Invoker interface {
	Invoke(logger lager.Logger, executable string, args []string) error
}

type realInvoker struct {
	useExec exec_wrap.Exec
}

func NewRealInvoker() Invoker {
	return NewRealInvokerWithExec(exec_wrap.NewExec())
}

func NewRealInvokerWithExec(useExec exec_wrap.Exec) Invoker {
	return &realInvoker{useExec}
}

func (r *realInvoker) Invoke(logger lager.Logger, executable string, cmdArgs []string) error {
	cmdHandle := r.useExec.Command(executable, cmdArgs...)

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

	// could validate stdout, but defer until actually need it
	return nil
}
