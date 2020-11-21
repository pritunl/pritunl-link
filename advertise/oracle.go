package advertise

import (
	"time"

	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/oracle"
	"github.com/pritunl/pritunl-link/routes"
)

func OracleAddRoute(network string) (err error) {
	time.Sleep(150 * time.Millisecond)

	mdata, err := oracle.GetMetadata()
	if err != nil {
		return
	}

	pv, err := oracle.NewProvider(mdata)
	if err != nil {
		return
	}

	vnic, err := oracle.GetVnic(pv, mdata.VnicOcid)
	if err != nil {
		return
	}

	if !vnic.SkipSourceDestCheck {
		err = vnic.SetSkipSourceDestCheck(pv, true)
		if err != nil {
			return
		}
	}

	subnet, err := oracle.GetSubnet(pv, vnic.SubnetId)
	if err != nil {
		return
	}

	tables, err := oracle.GetRouteTables(pv, subnet.VcnId)
	if err != nil {
		return
	}

	for _, table := range tables {
		if table.RouteUpsert(network, vnic.PrivateIpId) {
			err = table.CommitRouteRules(pv)
			if err != nil {
				return
			}
		}
	}

	route := &routes.OracleRoute{
		DestNetwork:   network,
		VncOcid:       mdata.VnicOcid,
		PrivateIpOcid: vnic.PrivateIpId,
	}

	err = route.Add()
	if err != nil {
		return
	}

	return
}

func OracleDeleteRoute(oracleRoute *routes.OracleRoute) (err error) {
	if config.Config.DeleteRoutes {
		time.Sleep(150 * time.Millisecond)

		mdata, e := oracle.GetMetadata()
		if e != nil {
			err = e
			return
		}

		pv, e := oracle.NewProvider(mdata)
		if e != nil {
			err = e
			return
		}

		vnic, e := oracle.GetVnic(pv, mdata.VnicOcid)
		if e != nil {
			err = e
			return
		}

		subnet, e := oracle.GetSubnet(pv, vnic.SubnetId)
		if e != nil {
			err = e
			return
		}

		tables, e := oracle.GetRouteTables(pv, subnet.VcnId)
		if e != nil {
			err = e
			return
		}

		for _, table := range tables {
			if table.RouteRemove(oracleRoute.DestNetwork,
				oracleRoute.PrivateIpOcid) {

				err = table.CommitRouteRules(pv)
				if err != nil {
					return
				}
			}
		}
	}

	err = oracleRoute.Remove()
	if err != nil {
		return
	}

	return
}
