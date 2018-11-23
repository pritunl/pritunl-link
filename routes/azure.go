package routes

type AzureRoute struct {
	DestNetwork string `json:"dest_network"`
	NextHop     string `json:"next_hop"`
}

func (r *AzureRoute) Add() (err error) {
	routes, err := GetCurrent()
	if err != nil {
		return
	}

	if routes.Azure == nil {
		routes.Azure = map[string]*AzureRoute{}
	}

	routes.Azure[r.DestNetwork] = r

	err = routes.Commit()
	if err != nil {
		return
	}

	return
}

func (r *AzureRoute) Remove() (err error) {
	routes, err := GetCurrent()
	if err != nil {
		return
	}

	if routes.Azure != nil {
		if _, ok := routes.Azure[r.DestNetwork]; ok {
			delete(routes.Azure, r.DestNetwork)
		}

		if len(routes.Azure) == 0 {
			routes.Azure = nil
		}
	}

	err = routes.Commit()
	if err != nil {
		return
	}

	return
}
