package model

type ServiceBinding struct {
	Id                string `json:"id"`
	ServiceId         string `json:"service_id"`
	AppId             string `json:"app_id"`
	ServicePlanId     string `json:"service_plan_id"`
	PrivateKey        string `json:"private_key"`
	ServiceInstanceId string `json:"service_instance_id"`
}

type CreateServiceBindingResponse struct {
	// SyslogDrainUrl string      `json:"syslog_drain_url, omitempty"`
	Credentials  interface{}   `json:"credentials"`
	VolumeMounts []VolumeMount `json:"volume_mounts"`
}

type Credential struct {
}

type VolumeMount struct {
	ContainerPath string `json:"container_path"`
	Mode          string `json:"mode"`
	Driver        string `json:"private.driver"`
	GroupId       string `json:"private.group_id"`
	Mountpath     string `json:"mount_path"`
}
