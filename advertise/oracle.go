package advertise

import (
	"bytes"
	"crypto/md5"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/oraclesdk"
	"github.com/pritunl/pritunl-link/routes"
	"github.com/pritunl/pritunl-link/utils"
	"time"
)

type oracleMetadata struct {
	UserOcid        string
	PrivateKey      string
	RegionName      string
	TenancyOcid     string
	CompartmentOcid string
	VnicOcid        string
}

type oracleOciMetaVnic struct {
	Id        string `json:"vnicId"`
	MacAddr   string `json:"macAddr"`
	PrivateIp string `json:"privateIp"`
}

type oracleOciMetaInstance struct {
	Id            string `json:"id"`
	DisplayName   string `json:"displayName"`
	CompartmentId string `json:"compartmentId"`
	RegionName    string `json:"canonicalRegionName"`
}

type oracleOciMeta struct {
	Instance oracleOciMetaInstance `json:"instance"`
	Vnics    []oracleOciMetaVnic   `json:"vnics"`
}

func oracleGetMetadata() (mdata *oracleMetadata, err error) {
	userOcid := config.Config.Oracle.UserOcid
	privateKey := config.Config.Oracle.PrivateKey

	region := config.Config.Oracle.Region
	tenancyOcid := config.Config.Oracle.TenancyOcid
	compartmentOcid := config.Config.Oracle.CompartmentOcid
	vnicOcidConf := config.Config.Oracle.VnicOcid

	if region != "" && tenancyOcid != "" && compartmentOcid != "" &&
		vnicOcidConf != "" {

		mdata = &oracleMetadata{
			UserOcid:        userOcid,
			PrivateKey:      privateKey,
			RegionName:      region,
			TenancyOcid:     tenancyOcid,
			CompartmentOcid: compartmentOcid,
			VnicOcid:        vnicOcidConf,
		}
		return
	}

	output, err := utils.ExecOutput("", "oci-metadata", "--json")
	if err != nil {
		return
	}

	data := &oracleOciMeta{}

	err = json.Unmarshal([]byte(output), data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "oracle: Failed to parse metadata"),
		}
		return
	}

	vnicOcid := ""
	if data.Vnics != nil {
		for _, vnic := range data.Vnics {
			vnicOcid = vnic.Id
			break
		}
	}

	if vnicOcid == "" {
		err = &errortypes.ParseError{
			errors.Wrap(err, "oracle: Failed to get vnic in metadata"),
		}
		return
	}

	mdata = &oracleMetadata{
		UserOcid:        userOcid,
		PrivateKey:      privateKey,
		RegionName:      data.Instance.RegionName,
		TenancyOcid:     data.Instance.CompartmentId,
		CompartmentOcid: data.Instance.CompartmentId,
		VnicOcid:        vnicOcid,
	}

	return
}

func oracleParseBase64Key(data string) (pemKey []byte, fingerprint string,
	err error) {

	pemKey, err = base64.StdEncoding.DecodeString(data)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "oracle: Failed to parse base64 private key"),
		}
		return
	}
	block, _ := pem.Decode(pemKey)
	if block == nil {
		err = &errortypes.ParseError{
			errors.New("oracle: Failed to decode private key"),
		}
		return
	}

	if block.Type != "RSA PRIVATE KEY" {
		err = &errortypes.ParseError{
			errors.New("oracle: Invalid private key type"),
		}
		return
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Failed to parse rsa key"),
		}
		return
	}

	pubKey, err := x509.MarshalPKIXPublicKey(key.Public())
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Failed to marshal public key"),
		}
		return
	}

	keyHash := md5.New()
	keyHash.Write(pubKey)
	fingerprint = fmt.Sprintf("%x", keyHash.Sum(nil))
	fingerprintBuf := bytes.Buffer{}

	for i, run := range fingerprint {
		fingerprintBuf.WriteRune(run)
		if i%2 == 1 && i != len(fingerprint)-1 {
			fingerprintBuf.WriteRune(':')
		}
	}
	fingerprint = fingerprintBuf.String()

	return
}

func oracleNewClient(region, privateKey, userOcid, tenancyOcid string) (
	client *oraclesdk.Client, err error) {

	key, fingerprint, err := oracleParseBase64Key(privateKey)
	if err != nil {
		return
	}

	client, err = oraclesdk.NewClient(
		userOcid,
		tenancyOcid,
		fingerprint,
		oraclesdk.Region(region),
		oraclesdk.PrivateKeyBytes(key),
		oraclesdk.ShortRetryTime(10*time.Second),
		oraclesdk.LongRetryTime(10*time.Second),
	)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "oracle: Failed to create oracle client"),
		}
		return
	}

	return
}

func OracleAddRoute(network string) (err error) {
	time.Sleep(150 * time.Millisecond)

	mdata, err := oracleGetMetadata()
	if err != nil {
		return
	}

	if constants.Interrupt {
		err = &errortypes.UnknownError{
			errors.Wrap(err, "advertise: Interrupt"),
		}
		return
	}

	client, err := oracleNewClient(
		mdata.RegionName,
		mdata.PrivateKey,
		mdata.UserOcid,
		mdata.TenancyOcid,
	)
	if err != nil {
		return
	}

	vnic, err := client.GetVnic(mdata.VnicOcid)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "oracle: Failed to get vnic"),
		}
		return
	}

	if !vnic.SkipSourceDestCheck {
		skipSrcDest := true
		vnicOpts := &oraclesdk.UpdateVnicOptions{
			SkipSourceDestCheck: &skipSrcDest,
		}
		_, err = client.UpdateVnic(mdata.VnicOcid, vnicOpts)
		if err != nil {
			err = &errortypes.RequestError{
				errors.Wrap(err,
					"oracle: Failed to update vnic source dest check"),
			}
			return
		}

		time.Sleep(250 * time.Millisecond)
	}

	listIpOpt := &oraclesdk.ListPrivateIPsOptions{
		VnicID: mdata.VnicOcid,
	}
	privateIps, err := client.ListPrivateIPs(listIpOpt)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "oracle: Failed to get private ips"),
		}
		return
	}

	privateIpOcid := ""
	subnetOcid := ""
	if privateIps.PrivateIPs != nil {
		for _, privateIp := range privateIps.PrivateIPs {
			privateIpOcid = privateIp.ID
			subnetOcid = privateIp.SubnetID
			break
		}
	}

	if privateIpOcid == "" || subnetOcid == "" {
		err = &errortypes.ParseError{
			errors.New("oracle: Failed to get private ip ocid"),
		}
		return
	}

	subnet, err := client.GetSubnet(subnetOcid)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "oracle: Failed to get subnet"),
		}
		return
	}
	vcnOcid := subnet.VcnID

	tables, err := client.ListRouteTables(
		mdata.CompartmentOcid, vcnOcid, nil)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "oracle: Failed to get routing tables"),
		}
		return
	}

	if len(tables.RouteTables) == 0 {
		err = &errortypes.ParseError{
			errors.New("oracle: Failed to find route tables"),
		}
		return
	}

	for _, table := range tables.RouteTables {
		exists := false
		replace := false

		routeRules := []oraclesdk.RouteRule{}
		for _, route := range table.RouteRules {
			if route.CidrBlock == network {
				exists = true

				if route.NetworkEntityID != privateIpOcid {
					route.NetworkEntityID = privateIpOcid
					replace = true
				}
			}

			routeRules = append(routeRules, route)
		}

		if exists && !replace {
			continue
		}

		if !replace {
			route := oraclesdk.RouteRule{
				CidrBlock:       network,
				NetworkEntityID: privateIpOcid,
			}
			routeRules = append(routeRules, route)
		}

		opts := &oraclesdk.UpdateRouteTableOptions{
			RouteRules: routeRules,
		}

		_, err = client.UpdateRouteTable(table.ID, opts)
		if err != nil {
			err = &errortypes.RequestError{
				errors.Wrap(err, "oracle: Failed to update routing table"),
			}
			return
		}
	}

	route := &routes.OracleRoute{
		DestNetwork:   network,
		VncOcid:       mdata.VnicOcid,
		PrivateIpOcid: privateIpOcid,
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

		privateKey := config.Config.Oracle.PrivateKey
		userOcid := config.Config.Oracle.UserOcid

		mdata, e := oracleGetMetadata()
		if e != nil {
			err = e
			return
		}

		if constants.Interrupt {
			err = &errortypes.UnknownError{
				errors.Wrap(err, "advertise: Interrupt"),
			}
			return
		}

		client, e := oracleNewClient(
			mdata.RegionName,
			privateKey,
			userOcid,
			mdata.TenancyOcid,
		)
		if e != nil {
			err = e
			return
		}

		tables, e := client.ListRouteTables(
			mdata.CompartmentOcid, mdata.VnicOcid, nil)
		if e != nil {
			err = &errortypes.RequestError{
				errors.Wrap(e, "oracle: Failed to get routing tables"),
			}
			return
		}

		for _, table := range tables.RouteTables {
			update := false

			routeRules := []oraclesdk.RouteRule{}
			for _, route := range table.RouteRules {
				if route.CidrBlock == oracleRoute.DestNetwork &&
					route.NetworkEntityID == oracleRoute.PrivateIpOcid {

					update = true
					continue
				}

				routeRules = append(routeRules, route)
			}

			if !update {
				continue
			}

			opts := &oraclesdk.UpdateRouteTableOptions{
				RouteRules: routeRules,
			}

			_, err = client.UpdateRouteTable(table.ID, opts)
			if err != nil {
				err = &errortypes.RequestError{
					errors.Wrap(err,
						"oracle: Failed to update routing table"),
				}
				return
			}
		}
	}

	err = oracleRoute.Remove()
	if err != nil {
		return
	}

	return
}
