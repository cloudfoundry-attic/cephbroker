package web_server

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"github.com/cloudfoundry-incubator/cephbroker/config"
	"github.com/cloudfoundry-incubator/cephbroker/model"
	"github.com/cloudfoundry-incubator/cephbroker/utils"
)

var (
	conf = config.GetConfig()
)

type Server struct {
	controller *Controller
}

func CreateServer(configuration config.Config) {
	serviceInstances, err := loadServiceInstances(configuration)
	if err != nil {
		panic(err)
	}

	serviceBindings, err := loadServiceBindings(configuration)
	if err != nil {
		panic(err)
	}

	controller, err := CreateController(serviceInstances, serviceBindings)
	if err != nil {
		panic(err)
	}

	router := mux.NewRouter()

	router.HandleFunc("/v2/catalog", controller.Catalog).Methods("GET")
	router.HandleFunc("/v2/service_instances/{service_instance_guid}", controller.GetServiceInstance).Methods("GET")
	router.HandleFunc("/v2/service_instances/{service_instance_guid}", controller.CreateServiceInstance).Methods("PUT")
	router.HandleFunc("/v2/service_instances/{service_instance_guid}", controller.RemoveServiceInstance).Methods("DELETE")
	router.HandleFunc("/v2/service_instances/{service_instance_guid}/service_bindings/{service_binding_guid}", controller.Bind).Methods("PUT")
	router.HandleFunc("/v2/service_instances/{service_instance_guid}/service_bindings/{service_binding_guid}", controller.UnBind).Methods("DELETE")

	http.Handle("/", router)

	fmt.Println("Server started, listening on " + configuration.AtAddress + "...")
	fmt.Println("CTL-C to break out of broker")
	http.ListenAndServe(configuration.AtAddress, nil)
}

// private methods
func loadServiceInstances(conf config.Config) (map[string]*model.ServiceInstance, error) {
	var serviceInstancesMap map[string]*model.ServiceInstance

	err := utils.ReadAndUnmarshal(&serviceInstancesMap, conf.DataPath, "ServiceInstances.json")
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("WARNING: service instance data file '%s' does not exist: \n")
			serviceInstancesMap = make(map[string]*model.ServiceInstance)
		} else {
			return nil, errors.New(fmt.Sprintf("Could not load the service instances, message: %s", err.Error()))
		}
	}

	return serviceInstancesMap, nil
}

func loadServiceBindings(conf config.Config) (map[string]*model.ServiceBinding, error) {
	var bindingMap map[string]*model.ServiceBinding

	err := utils.ReadAndUnmarshal(&bindingMap, conf.DataPath, "ServiceBindings.json")

	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("WARNING: key map data file '%s' does not exist: \n", conf.ServiceBindingsFileName)
			bindingMap = make(map[string]*model.ServiceBinding)
		} else {
			return nil, errors.New(fmt.Sprintf("Could not load the service instances, message: %s", err.Error()))
		}
	}

	return bindingMap, nil
}
