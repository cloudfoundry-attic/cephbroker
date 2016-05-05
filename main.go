package main

import (
	"flag"

	"github.com/cloudfoundry-incubator/cephbroker/config"
	webs "github.com/cloudfoundry-incubator/cephbroker/web_server"

	cf_lager "github.com/cloudfoundry-incubator/cf-lager"
)

type Options struct {
	ConfigPath string
	Cloud      string
}

var options Options

func main() {
	config := config.Config{}
	parseCommandLine(&config)

	webs.CreateServer(config)

}

// Private func

func checkCloudName(name string) error {
	return nil
}
func parseCommandLine(config *config.Config) {
	flag.StringVar(&config.AtAddress, "listenAddr", "0.0.0.0:8001", "host:port to serve cephbroker functions")
	flag.StringVar(&config.DataPath, "dataPath", "", "Path to directory where files are saved")
	flag.StringVar(&config.CatalogPath, "catalogPath", "", "Path to directory where files are saved")

	cf_lager.AddFlags(flag.CommandLine)

	flag.Parse()
}
