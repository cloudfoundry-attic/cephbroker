package cephbroker

import "github.com/tedsuo/rata"

const (
	CatalogRoute               = "catalog"
	CreateServiceInstanceRoute = "create"
	DeleteServiceInstanceRoute = "delete"
	BindServiceInstanceRoute   = "bind"
	UnbindServiceInstanceRoute = "unbind"
)

var Routes = rata.Routes{
	{Path: "/v2/catalog", Method: "GET", Name: CatalogRoute},
	{Path: "/v2/service_instances/:service_instance_guid", Method: "PUT", Name: CreateServiceInstanceRoute},
	{Path: "/v2/service_instances/:service_instance_guid", Method: "DELETE", Name: DeleteServiceInstanceRoute},
	{Path: "/v2/service_instances/:service_instance_guid/service_bindings/:service_binding_id", Method: "PUT", Name: BindServiceInstanceRoute},
	{Path: "/v2/service_instances/:service_instance_guid/service_bindings/:service_binding_id", Method: "DELETE", Name: UnbindServiceInstanceRoute},
}
