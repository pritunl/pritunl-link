package advertise

type AwsRoute struct {
	InterfaceId string `json:"interface_id"`
	InstanceId  string `json:"instance_id"`
}

type currentRoutes struct {
	Aws map[string]*AwsRoute `json:"aws"`
}
