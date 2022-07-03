package routes

type HcloudRoute struct {
	Destination string `json:"destination"`
	Gateway     string `json:"gateway"`
}

func (r *HcloudRoute) Add() (err error) {
	routes, err := GetCurrent()
	if err != nil {
		return
	}

	if routes.Hcloud == nil {
		routes.Hcloud = map[string]*HcloudRoute{}
	}

	routes.Hcloud[r.Destination] = r

	err = routes.Commit()
	if err != nil {
		return
	}

	return
}

func (r *HcloudRoute) Remove() (err error) {
	routes, err := GetCurrent()
	if err != nil {
		return
	}

	if routes.Hcloud != nil {
		if _, ok := routes.Hcloud[r.Destination]; ok {
			delete(routes.Hcloud, r.Destination)
		}

		if len(routes.Hcloud) == 0 {
			routes.Hcloud = nil
		}
	}

	err = routes.Commit()
	if err != nil {
		return
	}

	return
}
