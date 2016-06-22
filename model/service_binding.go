package model

type ServiceBinding struct {
	Id                string                 `json:"id"`
	ServiceId         string                 `json:"service_id"`
	AppId             string                 `json:"app_id"`
	ServicePlanId     string                 `json:"service_plan_id"`
	PrivateKey        string                 `json:"private_key"`
	ServiceInstanceId string                 `json:"service_instance_id"`
	Parameters        map[string]interface{} `json:"parameters"`
}

type CreateServiceBindingResponse struct {
	Credentials  Credentials   `json:"credentials"`
	VolumeMounts []VolumeMount `json:"volume_mounts"`
}

type Credentials struct {
	URI      string `json:"uri"`
	Hostname string `json:"hostname"`
	Port     string `json:"port"`
	Name     string `json:"name"`
	VHost    string `json:"vhost"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type VolumeMount struct {
	ContainerPath string                    `json:"container_path"`
	Mode          string                    `json:"mode"`
	Private       VolumeMountPrivateDetails `json:"private"`
}

type VolumeMountPrivateDetails struct {
	Driver  string `json:"driver"`
	GroupId string `json:"group_id"`
	Config  string `json:"config"`
}

type CephConfig struct {
	IP               string `json:"ip"`
	Keyring          string `json:"keyring"`
	RemoteMountPoint string `json:"remotemountpoint"`
	LocalMountPoint  string `json:"localmountpoint"`
}
