package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"syscall"

	"code.cloudfoundry.org/cephbroker/cephbrokerhttp"
	"code.cloudfoundry.org/cephbroker/cephbrokerlocal"
	"code.cloudfoundry.org/cephbroker/model"
	"code.cloudfoundry.org/cephbroker/utils"
	cf_lager "code.cloudfoundry.org/cflager"
	cf_debug_server "code.cloudfoundry.org/debugserver"
	"code.cloudfoundry.org/goshims/ioutil"
	"code.cloudfoundry.org/goshims/os"
	"code.cloudfoundry.org/lager"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"
)

var atAddress = flag.String(
	"listenAddr",
	"0.0.0.0:8999",
	"host:port to serve cephfs service broker functions",
)
var mds = flag.String(
	"mds",
	"10.0.0.106:6789",
	"host:port for ceph mds server",
)
var keyringFile = flag.String(
	"keyringFile",
	"/etc/ceph/ceph.client.admin.keyring",
	"keyring file for ceph authentication",
)
var configPath = flag.String(
	"configPath",
	"/tmp/cephbroker",
	"config directory to store book-keeping info",
)
var serviceName = flag.String(
	"serviceName",
	"cephfs",
	"name of the service to register with cloud controller",
)
var serviceId = flag.String(
	"serviceId",
	"cephfs-service-guid",
	"ID of the service to register with cloud controller",
)
var planName = flag.String(
	"planName",
	"free",
	"name of the service plan to register with cloud controller",
)
var planId = flag.String(
	"planId",
	"free-plan-guid",
	"ID of the service plan to register with cloud controller",
)
var planDesc = flag.String(
	"planDesc",
	"free ceph filesystem",
	"description of the service plan to register with cloud controller",
)

var baseMountPath = flag.String(
	"baseMountPath",
	"/tmp/share",
	"local directory to mount within on the service broker host",
)
var baseRemoteMountPath = flag.String(
	"baseRemoteMountPath",
	"/",
	"directory to mount on ceph file system server",
)

func main() {
	parseCommandLine()
	withLogger, logTap := logger()
	defer withLogger.Info("ends")

	syscall.Umask(000)

	servers, err := createCephBrokerServer(withLogger, *atAddress)

	if err != nil {
		panic("failed to load services metadata.....aborting")
	}
	if dbgAddr := cf_debug_server.DebugAddress(flag.CommandLine); dbgAddr != "" {
		servers = append(grouper.Members{
			{"debug-server", cf_debug_server.Runner(dbgAddr, logTap)},
		}, servers...)
	}
	process := ifrit.Invoke(processRunnerFor(servers))
	withLogger.Info("started")
	untilTerminated(withLogger, process)
}

func exitOnFailure(logger lager.Logger, err error) {
	if err != nil {
		logger.Error("Fatal err.. aborting", err)
		panic(err.Error())
	}
}

func untilTerminated(logger lager.Logger, process ifrit.Process) {
	err := <-process.Wait()
	exitOnFailure(logger, err)
}

func processRunnerFor(servers grouper.Members) ifrit.Runner {
	return sigmon.New(grouper.NewOrdered(os.Interrupt, servers))
}

func createCephBrokerServer(logger lager.Logger, atAddress string) (grouper.Members, error) {
	cephClient := cephbrokerlocal.NewCephClient(*mds, *baseMountPath, *keyringFile, *baseRemoteMountPath)
	existingServiceInstances, err := loadServiceInstances()
	if err != nil {
		return nil, err
	}
	existingServiceBindings, err := loadServiceBindings()
	if err != nil {
		return nil, err
	}
	controller := cephbrokerlocal.NewController(cephClient, *serviceName, *serviceId, *planId, *planName, *planDesc, *configPath, existingServiceInstances, existingServiceBindings, osshim.OsShim{}, ioutilshim.IoutilShim{})
	handler, err := cephbrokerhttp.NewHandler(logger, controller)
	exitOnFailure(logger, err)

	return grouper.Members{
		{"http-server", http_server.New(atAddress, handler)},
	}, nil
}

func logger() (lager.Logger, *lager.ReconfigurableSink) {

	logger, reconfigurableSink := cf_lager.New("ceph-broker")
	return logger, reconfigurableSink
}

func parseCommandLine() {
	cf_lager.AddFlags(flag.CommandLine)
	cf_debug_server.AddFlags(flag.CommandLine)
	flag.Parse()
}

func loadServiceInstances() (map[string]*model.ServiceInstance, error) {
	var serviceInstancesMap map[string]*model.ServiceInstance

	err := utils.ReadAndUnmarshal(&serviceInstancesMap, *configPath, "service_instances.json", ioutilshim.IoutilShim{})
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("WARNING: service instance data file '%s' does not exist: \n", "service_instances.json")
			serviceInstancesMap = make(map[string]*model.ServiceInstance)
		} else {
			return nil, errors.New(fmt.Sprintf("Could not load the service instances, message: %s", err.Error()))
		}
	}

	return serviceInstancesMap, nil
}

func loadServiceBindings() (map[string]*model.ServiceBinding, error) {
	var bindingMap map[string]*model.ServiceBinding
	err := utils.ReadAndUnmarshal(&bindingMap, *configPath, "service_bindings.json", ioutilshim.IoutilShim{})
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("WARNING: key map data file '%s' does not exist: \n", "service_bindings.json")
			bindingMap = make(map[string]*model.ServiceBinding)
		} else {
			return nil, errors.New(fmt.Sprintf("Could not load the service instances, message: %s", err.Error()))
		}
	}

	return bindingMap, nil
}
