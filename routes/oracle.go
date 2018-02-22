package routes

type OracleRoute struct {
	DestNetwork     string `json:"dest_network"`
	Region          string `json:"region"`
	UserOcid        string `json:"user_ocid"`
	TenancyOcid     string `json:"tenancy_ocid"`
	CompartmentOcid string `json:"compartment_ocid"`
	VncOcid         string `json:"vnc_ocid"`
	PrivateIpOcid   string `json:"private_ip_ocid"`
}

func (r *OracleRoute) Add() (err error) {
	routes, err := GetCurrent()
	if err != nil {
		return
	}

	if routes.Oracle == nil {
		routes.Oracle = map[string]*OracleRoute{}
	}

	routes.Oracle[r.DestNetwork] = r

	err = routes.Commit()
	if err != nil {
		return
	}

	return
}

func (r *OracleRoute) Remove() (err error) {
	routes, err := GetCurrent()
	if err != nil {
		return
	}

	if routes.Oracle != nil {
		if _, ok := routes.Oracle[r.DestNetwork]; ok {
			delete(routes.Oracle, r.DestNetwork)
		}

		if len(routes.Oracle) == 0 {
			routes.Oracle = nil
		}
	}

	err = routes.Commit()
	if err != nil {
		return
	}

	return
}
