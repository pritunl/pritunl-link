package advertise

import (
	"bytes"
	"crypto/md5"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/oraclesdk"
	"github.com/pritunl/pritunl-link/routes"
	"time"
)

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

	region := config.Config.Oracle.Region
	privateKey := config.Config.Oracle.PrivateKey
	userOcid := config.Config.Oracle.UserOcid
	tenancyOcid := config.Config.Oracle.TenancyOcid
	compartmentOcid := config.Config.Oracle.CompartmentOcid
	vncOcid := config.Config.Oracle.VncOcid
	privateIpOcid := config.Config.Oracle.PrivateIpOcid

	if constants.Interrupt {
		err = &errortypes.UnknownError{
			errors.Wrap(err, "advertise: Interrupt"),
		}
		return
	}

	client, err := oracleNewClient(
		region,
		privateKey,
		userOcid,
		tenancyOcid,
	)
	if err != nil {
		return
	}

	tables, err := client.ListRouteTables(compartmentOcid, vncOcid, nil)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "oracle: Failed to get routing tables"),
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
		DestNetwork:     network,
		Region:          region,
		UserOcid:        userOcid,
		TenancyOcid:     tenancyOcid,
		CompartmentOcid: compartmentOcid,
		VncOcid:         vncOcid,
		PrivateIpOcid:   privateIpOcid,
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

		region := config.Config.Oracle.Region
		privateKey := config.Config.Oracle.PrivateKey
		userOcid := config.Config.Oracle.UserOcid
		tenancyOcid := config.Config.Oracle.TenancyOcid
		compartmentOcid := config.Config.Oracle.CompartmentOcid
		vncOcid := config.Config.Oracle.VncOcid

		if constants.Interrupt {
			err = &errortypes.UnknownError{
				errors.Wrap(err, "advertise: Interrupt"),
			}
			return
		}

		client, e := oracleNewClient(
			region,
			privateKey,
			userOcid,
			tenancyOcid,
		)
		if e != nil {
			err = e
			return
		}

		tables, e := client.ListRouteTables(compartmentOcid, vncOcid, nil)
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
				if route.CidrBlock == oracleRoute.DestNetwork {
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
					errors.Wrap(err, "oracle: Failed to update routing table"),
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
