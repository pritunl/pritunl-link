package routes

type AwsRoute struct {
	DestNetwork string `json:"dest_network"`
	Region      string `json:"region"`
	VpcId       string `json:"vpc_id"`
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

func (r *AwsRoute) Remove() (err error) {
	routes, err := GetCurrent()
	if err != nil {
		return
	}

	if routes.Aws != nil {
		if _, ok := routes.Aws[r.DestNetwork]; ok {
			delete(routes.Aws, r.DestNetwork)
		}

		if len(routes.Aws) == 0 {
			routes.Aws = nil
		}
	}

	err = routes.Commit()
	if err != nil {
		return
	}

	return
}
