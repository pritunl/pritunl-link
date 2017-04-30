package cloud

import (
	"crypto/md5"
	"fmt"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/errortypes"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type googleMetaData struct {
	Project       string
	Instance      string
	InstanceShort string
	Network       string
	NetworkShort  string
}

type googleRoute struct {
	Name                 string
	DestRange            string
	Network              string
	NetworkShort         string
	NextHopInstance      string
	NextHopInstanceShort string
}

var client = &http.Client{
	Timeout: 500 * time.Millisecond,
}

func googleInternal(path string) (val string, err error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("http://metadata.google.internal/%s", path),
		nil,
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "cloud: Failed to request Google metadata"),
		}
		return
	}

	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "cloud: Failed to get Google metadata"),
		}
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "cloud: Failed to read Google metadata"),
		}
		return
	}

	val = string(body)

	return
}

func googleGetMetaData() (data *googleMetaData, err error) {
	project, err := googleInternal(
		"computeMetadata/v1/project/project-id")
	if err != nil {
		return
	}

	name, err := googleInternal("computeMetadata/v1/instance/name")
	if err != nil {
		return
	}

	zone, err := googleInternal("computeMetadata/v1/instance/zone")
	if err != nil {
		return
	}

	network, err := googleInternal(
		"computeMetadata/v1/instance/network-interfaces/0/network")
	if err != nil {
		return
	}

	if !strings.Contains(network, "/global/") {
		network = strings.Replace(
			network, "/networks/", "/global/networks/", 1)
	}

	networks := strings.Split(network, "/")

	data = &googleMetaData{
		Project:       project,
		Instance:      fmt.Sprintf("%s/instances/%s", zone, name),
		InstanceShort: name,
		Network:       network,
		NetworkShort:  networks[len(networks)-1],
	}

	return
}

func googleGetRoutes(svc *compute.Service, project string) (
	routes map[string]*googleRoute, err error) {

	routes = map[string]*googleRoute{}
	call := svc.Routes.List(project)

	resp, err := call.Do()
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "cloud: Failed to get Google routes"),
		}
		return
	}

	for _, route := range resp.Items {
		network := strings.Split(route.Network, "/")
		instance := strings.Split(route.NextHopInstance, "/")

		routes[route.DestRange] = &googleRoute{
			Name:                 route.Name,
			DestRange:            route.DestRange,
			Network:              route.Network,
			NetworkShort:         network[len(network)-1],
			NextHopInstance:      route.NextHopInstance,
			NextHopInstanceShort: instance[len(instance)-1],
		}
	}

	return
}

func googleHasRoute(svc *compute.Service, project, destRange,
	networkShort, instanceShort string) (exists bool, err error) {

	routes, err := googleGetRoutes(svc, project)
	if err != nil {
		return
	}

	if route, ok := routes[destRange]; ok {
		if route.DestRange != destRange ||
			route.NetworkShort != networkShort ||
			route.NextHopInstanceShort != instanceShort {

			call := svc.Routes.Delete(project, route.Name)

			_, err = call.Do()
			if err != nil {
				err = &errortypes.RequestError{
					errors.Wrap(err, "cloud: Failed to remove Google route"),
				}
				return
			}

			for i := 0; i < 20; i++ {
				routes, e := googleGetRoutes(svc, project)
				if e != nil {
					err = e
					return
				}

				if _, ok := routes[destRange]; !ok {
					break
				}

				time.Sleep(250 * time.Millisecond)
			}
		} else {
			exists = true
			return
		}
	}

	return
}

func GoogleAddRoute(network string) (err error) {
	ctx := context.Background()

	data, err := googleGetMetaData()
	if err != nil {
		return
	}

	client, err := google.DefaultClient(ctx, compute.CloudPlatformScope)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "cloud: Failed to get Google client"),
		}
		return
	}

	svc, err := compute.New(client)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "cloud: Failed to get Google compute"),
		}
		return
	}

	exists, err := googleHasRoute(svc, data.Project, network,
		data.NetworkShort, data.InstanceShort)
	if err != nil {
		return
	}
	if exists {
		return
	}

	route := &compute.Route{
		Name:            fmt.Sprintf("pritunl-%x", md5.Sum([]byte(network))),
		DestRange:       network,
		Priority:        1000,
		Network:         data.Network,
		NextHopInstance: data.Instance,
	}

	call := svc.Routes.Insert(data.Project, route)

	_, err = call.Do()
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "cloud: Failed to insert Google route"),
		}
		return
	}

	return
}
