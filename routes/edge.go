package routes

type EdgeRoute struct {
	Network string `json:"network"`
	Nexthop string `json:"nexthop"`
}

func (r *EdgeRoute) Add() (err error) {
	routes, err := GetCurrent()
	if err != nil {
		return
	}

	if routes.Edge == nil {
		routes.Edge = map[string]*EdgeRoute{}
	}

	routes.Edge[r.Network] = r

	err = routes.Commit()
	if err != nil {
		return
	}

	return
}

func (r *EdgeRoute) Remove() (err error) {
	routes, err := GetCurrent()
	if err != nil {
		return
	}

	if routes.Edge != nil {
		if _, ok := routes.Edge[r.Network]; ok {
			delete(routes.Edge, r.Network)
		}

		if len(routes.Edge) == 0 {
			routes.Edge = nil
		}
	}

	err = routes.Commit()
	if err != nil {
		return
	}

	return
}
