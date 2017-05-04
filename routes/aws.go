package routes

type AwsRoute struct {
	DestNetwork string `json:"dest_network"`
	InterfaceId string `json:"interface_id"`
	InstanceId  string `json:"instance_id"`
}

func (r *AwsRoute) Add() (err error) {
	routes, err := GetCurrent()
	if err != nil {
		return
	}

	if routes.Aws == nil {
		routes.Aws = map[string]*AwsRoute{}
	}

	routes.Aws[r.DestNetwork] = r

	err = routes.Commit()
	if err != nil {
		return
	}

	return
}
