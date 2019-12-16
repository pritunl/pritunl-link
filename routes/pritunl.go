package routes

type PritunlRoute struct {
	DestNetwork    string `json:"dest_network"`
	OrganizationId string `json:"organization_id"`
	VpcId          string `json:"vpc_id"`
	Target         string `json:"target"`
}

func (r *PritunlRoute) Add() (err error) {
	routes, err := GetCurrent()
	if err != nil {
		return
	}

	if routes.Pritunl == nil {
		routes.Pritunl = map[string]*PritunlRoute{}
	}

	routes.Pritunl[r.DestNetwork] = r

	err = routes.Commit()
	if err != nil {
		return
	}

	return
}

func (r *PritunlRoute) Remove() (err error) {
	routes, err := GetCurrent()
	if err != nil {
		return
	}

	if routes.Pritunl != nil {
		if _, ok := routes.Pritunl[r.DestNetwork]; ok {
			delete(routes.Pritunl, r.DestNetwork)
		}

		if len(routes.Pritunl) == 0 {
			routes.Pritunl = nil
		}
	}

	err = routes.Commit()
	if err != nil {
		return
	}

	return
}
