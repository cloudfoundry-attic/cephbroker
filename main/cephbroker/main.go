package main

import (
	"flag"

	"code.cloudfoundry.org/cflager"
	"code.cloudfoundry.org/debugserver"

	"syscall"

	"code.cloudfoundry.org/cephbroker/cephbroker"
	"code.cloudfoundry.org/cephbroker/utils"
	ioutilshim "code.cloudfoundry.org/goshims/ioutil"
	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf/brokerapi"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
)

var dataDir = flag.String(
	"dataDir",
	"",
	"[REQUIRED] - Broker's state will be stored here to persist across reboots",
)

var atAddress = flag.String(
	"listenAddr",
	"0.0.0.0:8999",
	"host:port to serve service broker API",
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
	"localvolume",
	"name of the service to register with cloud controller",
)
var serviceId = flag.String(
	"serviceId",
	"service-guid",
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
	"free local filesystem",
	"description of the service plan to register with cloud controller",
)
var username = flag.String(
	"username",
	"admin",
	"basic auth username to verify on incoming requests",
)
var password = flag.String(
	"password",
	"admin",
	"basic auth password to verify on incoming requests",
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
	syscall.Umask(000)

	logger, logSink := cflager.New("localbroker")
	logger.Info("starting")
	defer logger.Info("ends")

	server := createServer(logger)

	if dbgAddr := debugserver.DebugAddress(flag.CommandLine); dbgAddr != "" {
		server = utils.ProcessRunnerFor(grouper.Members{
			{"debug-server", debugserver.Runner(dbgAddr, logSink)},
			{"broker-api", server},
		})
	}

	process := ifrit.Invoke(server)
	logger.Info("started")
	utils.UntilTerminated(logger, process)
}

func parseCommandLine() {
	cflager.AddFlags(flag.CommandLine)
	debugserver.AddFlags(flag.CommandLine)
	flag.Parse()
}

func createServer(logger lager.Logger) ifrit.Runner {
	controller := cephbroker.NewController(cephbroker.NewCephClient(
		*mds,
		*baseMountPath,
		*keyringFile,
		*baseRemoteMountPath,
	))
	serviceBroker := cephbroker.New(
		logger, controller,
		*serviceName, *serviceId, *planName, *planId, *planDesc, *dataDir,
		&ioutilshim.IoutilShim{},
	)
	credentials := brokerapi.BrokerCredentials{Username: *username, Password: *password}
	handler := brokerapi.New(serviceBroker, logger.Session("broker-api"), credentials)

	return http_server.New(*atAddress, handler)
}
