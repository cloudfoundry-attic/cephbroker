package cephbroker

import "github.com/tedsuo/rata"

const (
	CatalogRoute               = "catalog"
	CreateServiceInstanceRoute = "create"
	DeleteServiceInstanceRoute = "delete"
)

var Routes = rata.Routes{
	{Path: "/v2/catalog", Method: "GET", Name: CatalogRoute},
	{Path: "/v2/service_instances/:service_instance_guid", Method: "PUT", Name: CreateServiceInstanceRoute},
	{Path: "/v2/service_instances/:service_instance_guid", Method: "DELETE", Name: DeleteServiceInstanceRoute},
}
