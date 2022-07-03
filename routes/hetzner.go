package routes

type HetznerRoute struct {
	Destination string `json:"destination"`
	Gateway     string `json:"gateway"`
}

func (r *HetznerRoute) Add() (err error) {
	routes, err := GetCurrent()
	if err != nil {
		return
	}

	if routes.Hetzner == nil {
		routes.Hetzner = map[string]*HetznerRoute{}
	}

	routes.Hetzner[r.Destination] = r

	err = routes.Commit()
	if err != nil {
		return
	}

	return
}

func (r *HetznerRoute) Remove() (err error) {
	routes, err := GetCurrent()
	if err != nil {
		return
	}

	if routes.Hetzner != nil {
		if _, ok := routes.Hetzner[r.Destination]; ok {
			delete(routes.Hetzner, r.Destination)
		}

		if len(routes.Hetzner) == 0 {
			routes.Hetzner = nil
		}
	}

	err = routes.Commit()
	if err != nil {
		return
	}

	return
}
