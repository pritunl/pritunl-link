package routes

type GoogleRoute struct {
	DestNetwork  string `json:"dest_network"`
	Network      string `json:"network"`
	NetworkShort string `json:"network_short"`
}

func (r *GoogleRoute) Add() (err error) {
	routes, err := GetCurrent()
	if err != nil {
		return
	}

	if routes.Google == nil {
		routes.Google = map[string]*GoogleRoute{}
	}

	routes.Google[r.DestNetwork] = r

	err = routes.Commit()
	if err != nil {
		return
	}

	return
}

func (r *GoogleRoute) Remove() (err error) {
	routes, err := GetCurrent()
	if err != nil {
		return
	}

	if routes.Google != nil {
		if _, ok := routes.Google[r.DestNetwork]; ok {
			delete(routes.Google, r.DestNetwork)
		}

		if len(routes.Google) == 0 {
			routes.Google = nil
		}
	}

	err = routes.Commit()
	if err != nil {
		return
	}

	return
}
