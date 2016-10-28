package cephbroker

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/cephbroker/utils"
	"code.cloudfoundry.org/goshims/execshim"
	"code.cloudfoundry.org/goshims/ioutilshim"
	"code.cloudfoundry.org/goshims/osshim"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/voldriver"
	"code.cloudfoundry.org/voldriver/driverhttp"
)

//go:generate counterfeiter -o ../cephfakes/fake_ceph_client.go . Client

type Client interface {
	IsFilesystemMounted(voldriver.Env) bool
	MountFileSystem(voldriver.Env, string) (string, error)
	CreateShare(voldriver.Env, string) (string, error)
	DeleteShare(voldriver.Env, string) error
	GetPathsForShare(voldriver.Env, string) (string, string, error)
	GetConfigDetails(voldriver.Env) (string, string, error)
}

type cephClient struct {
	mds                 string
	invoker             Invoker
	os                  osshim.Os
	ioutil              ioutilshim.Ioutil
	baseLocalMountPoint string
	mounted             bool
	keyring             string
	remoteMountPath     string
}

const CellBasePath string = "/var/vcap/data/volumes/ceph/"

var (
	ShareNotFound   error = errors.New("share not found, internal error")
	KeyringNotFound error = errors.New("unable to open cephfs keyring")
)

func NewCephClientWithInvokerAndSystemUtil(mds string, useInvoker Invoker, os osshim.Os, ioutil ioutilshim.Ioutil, localMountPoint string, keyringFile string) Client {
	return &cephClient{
		mds:                 mds,
		invoker:             useInvoker,
		os:                  os,
		ioutil:              ioutil,
		baseLocalMountPoint: localMountPoint,
		mounted:             false,
		keyring:             keyringFile,
	}
}
func NewCephClient(mds string, localMountPoint string, keyringFile string, remoteMountPath string) Client {
	return &cephClient{
		mds:                 mds,
		invoker:             NewRealInvoker(),
		os:                  &osshim.OsShim{},
		ioutil:              &ioutilshim.IoutilShim{},
		baseLocalMountPoint: localMountPoint,
		mounted:             false,
		keyring:             keyringFile,
		remoteMountPath:     remoteMountPath,
	}
}
func (c *cephClient) IsFilesystemMounted(env voldriver.Env) bool {
	logger := env.Logger().Session("is-filesystem-mounted")
	logger.Info("start")
	defer logger.Info("end")
	return c.mounted
}

func (c *cephClient) MountFileSystem(env voldriver.Env, remoteMountPoint string) (string, error) {
	logger := env.Logger().Session("mount-filesystem", lager.Data{"remoteMountPoint": remoteMountPoint})
	logger.Info("start")
	defer logger.Info("end")

	err := c.os.MkdirAll(c.baseLocalMountPoint, os.ModePerm)
	if err != nil {
		logger.Error("failed-to-create-local-dir", err)
		return "", fmt.Errorf("failed to create local directory '%s', mount filesystem failed", c.baseLocalMountPoint)
	}

	cmdArgs := []string{"-m", c.mds, "-k", c.keyring, "-r", remoteMountPoint, c.baseLocalMountPoint}
	err = c.invokeCeph(driverhttp.EnvWithLogger(logger, env), cmdArgs)
	if err != nil {
		logger.Error("cephfs-error", err)
		return "", err
	}
	c.mounted = true
	return c.baseLocalMountPoint, nil
}

func (c *cephClient) CreateShare(env voldriver.Env, shareName string) (string, error) {
	logger := env.Logger().Session("create-share", lager.Data{"shareName": shareName})
	logger.Info("start")
	defer logger.Info("end")

	sharePath := filepath.Join(c.baseLocalMountPoint, shareName)
	err := c.os.MkdirAll(sharePath, os.ModePerm)
	if err != nil {
		logger.Error("failed-to-create-share", err)
		return "", fmt.Errorf("failed to create share '%s'", sharePath)
	}
	return sharePath, nil
}

func (c *cephClient) DeleteShare(env voldriver.Env, shareName string) error {
	logger := env.Logger().Session("delete-share", lager.Data{"shareName": shareName})
	logger.Info("start")
	defer logger.Info("end")

	sharePath := filepath.Join(c.baseLocalMountPoint, shareName)
	err := c.os.RemoveAll(sharePath)
	if err != nil {
		logger.Error("failed-to-delete-share", err)
		return fmt.Errorf("failed to delete share '%s'", sharePath)
	}
	return nil
}

func (c *cephClient) GetPathsForShare(env voldriver.Env, shareName string) (string, string, error) {
	logger := env.Logger().Session("get-paths-for-share", lager.Data{shareName: shareName})
	logger.Info("start")
	defer logger.Info("end")

	shareLocalPath := filepath.Join(c.baseLocalMountPoint, shareName)
	exists := utils.Exists(shareLocalPath, c.os)
	if exists == false {
		logger.Error("share-not-found", ShareNotFound)
		return "", "", ShareNotFound
	}

	shareAbsPath := filepath.Join(c.remoteMountPath, shareName)
	cellPath := filepath.Join(CellBasePath, shareName)
	return shareAbsPath, cellPath, nil
}

func (c *cephClient) GetConfigDetails(env voldriver.Env) (string, string, error) {
	logger := env.Logger().Session("get-config-details")
	if c.mds == "" || c.keyring == "" {
		return "", "", fmt.Errorf("Error retreiving Ceph config details")
	}
	contents, err := c.ioutil.ReadFile(c.keyring)
	if err != nil {
		logger.Error("failed-to-get-keyring", KeyringNotFound)
		return "", "", KeyringNotFound
	}
	return c.mds, string(contents), nil
}

func (c *cephClient) invokeCeph(env voldriver.Env, args []string) error {
	logger := env.Logger().Session("invoke-ceph")
	cmd := "ceph-fuse"
	logger.Info("invoking-ceph", lager.Data{"cmd": cmd, "args": args})
	defer logger.Debug("done-invoking-ceph")
	return c.invoker.Invoke(driverhttp.EnvWithLogger(logger, env), cmd, args)
}

//go:generate counterfeiter -o ../cephfakes/fake_invoker.go . Invoker

type Invoker interface {
	Invoke(env voldriver.Env, executable string, args []string) error
}

type realInvoker struct {
	useExec execshim.Exec
}

func NewRealInvoker() Invoker {
	return NewRealInvokerWithExec(&execshim.ExecShim{})
}

func NewRealInvokerWithExec(useExec execshim.Exec) Invoker {
	return &realInvoker{useExec}
}

func (r *realInvoker) Invoke(env voldriver.Env, executable string, cmdArgs []string) error {
	logger := env.Logger().Session("invoke")
	logger.Info("start")
	defer logger.Info("end")

	cmdHandle := r.useExec.CommandContext(env.Context(), executable, cmdArgs...)

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
