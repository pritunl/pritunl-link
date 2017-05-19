package routes

type UnifiRoute struct {
	Network string `json:"network"`
	Nexthop string `json:"nexthop"`
}

func (r *UnifiRoute) Add() (err error) {
	routes, err := GetCurrent()
	if err != nil {
		return
	}

	if routes.Unifi == nil {
		routes.Unifi = map[string]*UnifiRoute{}
	}

	routes.Unifi[r.Network] = r

	err = routes.Commit()
	if err != nil {
		return
	}

	return
}

func (r *UnifiRoute) Remove() (err error) {
	routes, err := GetCurrent()
	if err != nil {
		return
	}

	if routes.Unifi != nil {
		if _, ok := routes.Unifi[r.Network]; ok {
			delete(routes.Unifi, r.Network)
		}

		if len(routes.Unifi) == 0 {
			routes.Unifi = nil
		}
	}

	err = routes.Commit()
	if err != nil {
		return
	}

	return
}
