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
	Driver       string       `json:"driver"`
	ContainerDir string       `json:"container_dir"`
	Mode         string       `json:"mode"`
	DeviceType   string       `json:"device_type"`
	Device       SharedDevice `json:"device"`
}

type SharedDevice struct {
	VolumeId    string     `json:"volume_id"`
	MountConfig CephConfig `json:"mount_config"`
}

type CephConfig struct {
	IP               string `json:"ip"`
	Keyring          string `json:"keyring"`
	RemoteMountPoint string `json:"remote_mount_point"`
	LocalMountPoint  string `json:"local_mount_point"`
}
