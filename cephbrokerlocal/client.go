package cephbrokerlocal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/cephbroker/utils"
	"github.com/cloudfoundry/gunk/os_wrap/exec_wrap"
)

type Client interface {
	IsFilesystemMounted(lager.Logger) bool
	MountFileSystem(lager.Logger, string) (string, error)
	CreateShare(lager.Logger, string) (string, error)
	DeleteShare(lager.Logger, string) error
	GetPathsForShare(lager.Logger, string) (string, string, error)
	GetConfigDetails(lager.Logger) (string, string, error)
}

type cephClient struct {
	mds                 string
	invoker             Invoker
	systemUtil          utils.SystemUtil
	baseLocalMountPoint string
	mounted             bool
	keyring             string
	remoteMountPath     string
}

const CellBasePath string = "/var/vcap/data/volumes/"

var (
	ShareNotFound   error = errors.New("share not found, internal error")
	KeyringNotFound error = errors.New("unable to open cephfs keyring")
)

func NewCephClientWithInvokerAndSystemUtil(mds string, useInvoker Invoker, useSystemUtil utils.SystemUtil, localMountPoint string, keyringFile string) Client {
	return &cephClient{
		mds:                 mds,
		invoker:             useInvoker,
		systemUtil:          useSystemUtil,
		baseLocalMountPoint: localMountPoint,
		mounted:             false,
		keyring:             keyringFile,
	}
}
func NewCephClient(mds string, localMountPoint string, keyringFile string, remoteMountPath string) Client {
	return &cephClient{
		mds:                 mds,
		invoker:             NewRealInvoker(),
		systemUtil:          utils.NewRealSystemUtil(),
		baseLocalMountPoint: localMountPoint,
		mounted:             false,
		keyring:             keyringFile,
		remoteMountPath:     remoteMountPath,
	}
}
func (c *cephClient) IsFilesystemMounted(logger lager.Logger) bool {
	logger = logger.Session("is-filesystem-mounted")
	logger.Info("start")
	defer logger.Info("end")
	return c.mounted
}

func (c *cephClient) MountFileSystem(logger lager.Logger, remoteMountPoint string) (string, error) {
	logger = logger.Session("mount-filesystem", lager.Data{"remoteMountPoint": remoteMountPoint})
	logger.Info("start")
	defer logger.Info("end")

	err := c.systemUtil.MkdirAll(c.baseLocalMountPoint, os.ModePerm)
	if err != nil {
		logger.Error("failed-to-create-local-dir", err)
		return "", fmt.Errorf("failed to create local directory '%s', mount filesystem failed", c.baseLocalMountPoint)
	}

	cmdArgs := []string{"-m", c.mds, "-k", c.keyring, "-r", remoteMountPoint, c.baseLocalMountPoint}
	err = c.invokeCeph(logger, cmdArgs)
	if err != nil {
		logger.Error("cephfs-error", err)
		return "", err
	}
	c.mounted = true
	return c.baseLocalMountPoint, nil
}

func (c *cephClient) CreateShare(logger lager.Logger, shareName string) (string, error) {
	logger = logger.Session("create-share", lager.Data{"shareName": shareName})
	logger.Info("start")
	defer logger.Info("end")

	sharePath := filepath.Join(c.baseLocalMountPoint, shareName)
	err := c.systemUtil.MkdirAll(sharePath, os.ModePerm)
	if err != nil {
		logger.Error("failed-to-create-share", err)
		return "", fmt.Errorf("failed to create share '%s'", sharePath)
	}
	return sharePath, nil
}

func (c *cephClient) DeleteShare(logger lager.Logger, shareName string) error {
	logger = logger.Session("delete-share", lager.Data{"shareName": shareName})
	logger.Info("start")
	defer logger.Info("end")

	sharePath := filepath.Join(c.baseLocalMountPoint, shareName)
	err := c.systemUtil.Remove(sharePath)
	if err != nil {
		logger.Error("failed-to-delete-share", err)
		return fmt.Errorf("failed to delete share '%s'", sharePath)
	}
	return nil
}

func (c *cephClient) GetPathsForShare(logger lager.Logger, shareName string) (string, string, error) {
	logger = logger.Session("get-paths-for-share", lager.Data{shareName: shareName})
	logger.Info("start")
	defer logger.Info("end")

	shareLocalPath := filepath.Join(c.baseLocalMountPoint, shareName)
	exists := c.systemUtil.Exists(shareLocalPath)
	if exists == false {
		logger.Error("share-not-found", ShareNotFound)
		return "", "", ShareNotFound
	}

	shareAbsPath := filepath.Join(c.remoteMountPath, shareName)
	cellPath := filepath.Join(CellBasePath, shareName)
	return shareAbsPath, cellPath, nil
}

func (c *cephClient) GetConfigDetails(logger lager.Logger) (string, string, error) {
	if c.mds == "" || c.keyring == "" {
		return "", "", fmt.Errorf("Error retreiving Ceph config details")
	}
	contents, err := c.systemUtil.ReadFile(c.keyring)
	if err != nil {
		logger.Error("failed-to-get-keyring", KeyringNotFound)
		return "", "", KeyringNotFound
	}
	return c.mds, string(contents), nil
}

func (c *cephClient) invokeCeph(logger lager.Logger, args []string) error {
	cmd := "ceph-fuse"
	logger.Info("invoking-ceph", lager.Data{"cmd": cmd, "args": args})
	defer logger.Debug("done-invoking-ceph")
	return c.invoker.Invoke(logger, cmd, args)
}

//go:generate counterfeiter -o ../cephfakes/fake_invoker.go . Invoker

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
		logger.Error("unable-to-get-stdout", err)
		return err
	}

	if err = cmdHandle.Start(); err != nil {
		logger.Error("starting-command", err)
		return err
	}

	if err = cmdHandle.Wait(); err != nil {

		logger.Error("command-exited", err)
		return err
	}

	// could validate stdout, but defer until actually need it
	return nil
}
