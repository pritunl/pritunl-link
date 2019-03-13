package advertise

import (
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/routes"
)

//var (
//	azureClient = &http.Client{
//		Timeout: 10 * time.Second,
//	}
//)
//
//type azureMetadata struct {
//	SubscriptionId string
//	ResourceGroup  string
//	InstanceName   string
//	Location       string
//	PublicIp       string
//	PrivateIp      string
//}
//
//type azureAddress struct {
//	PrivateIpAddress string `json:"privateIpAddress"`
//	PublicIpAddress  string `json:"publicIpAddress"`
//}
//
//type azureIfaceAddress struct {
//	IpAddress []azureAddress `json:"ipAddress"`
//}
//
//type azureInterface struct {
//	Ipv4 azureIfaceAddress `json:"ipv4"`
//}
//
//type azureNet struct {
//	Interface []azureInterface `json:"interface"`
//}
//
//type azureCompute struct {
//	Location          string `json:"location"`
//	Name              string `json:"name"`
//	ResourceGroupName string `json:"resourceGroupName"`
//	SubscriptionId    string `json:"subscriptionId"`
//}
//
//type azureInstanceMetadata struct {
//	Compute azureCompute `json:"compute"`
//	Network azureNet     `json:"network"`
//}
//
//func azureGetMetaData() (mdata *azureMetadata, err error) {
//	req, err := http.NewRequest(
//		"GET",
//		"http://169.254.169.254/metadata/instance?api-version=2018-02-01",
//		nil,
//	)
//	if err != nil {
//		err = &errortypes.RequestError{
//			errors.Wrap(err, "azure: Azure metadata request error"),
//		}
//		return
//	}
//
//	req.Header.Set("Metadata", "true")
//
//	resp, err := azureClient.Do(req)
//	if err != nil {
//		err = &errortypes.RequestError{
//			errors.Wrap(err, "azure: Azure metadata request failed"),
//		}
//		return
//	}
//	defer resp.Body.Close()
//
//	data := &azureInstanceMetadata{}
//	err = json.NewDecoder(resp.Body).Decode(data)
//	if err != nil {
//		err = &errortypes.ParseError{
//			errors.Wrap(
//				err, "azure: Failed to parse azure metadata response",
//			),
//		}
//		return
//	}
//
//	if resp.StatusCode != 200 {
//		err = &errortypes.RequestError{
//			errors.New("azure: Azure metadata request bad status"),
//		}
//		return
//	}
//
//	publicIp := ""
//	privateIp := ""
//	if data.Network.Interface != nil && len(data.Network.Interface) > 0 {
//		iface := data.Network.Interface[0]
//
//		if iface.Ipv4.IpAddress != nil && len(iface.Ipv4.IpAddress) > 0 {
//			publicIp = iface.Ipv4.IpAddress[0].PublicIpAddress
//			privateIp = iface.Ipv4.IpAddress[0].PrivateIpAddress
//		}
//	}
//
//	if privateIp == "" {
//		err = &errortypes.RequestError{
//			errors.New("azure: Azure metadata private ip nil"),
//		}
//		return
//	}
//
//	mdata = &azureMetadata{
//		SubscriptionId: data.Compute.SubscriptionId,
//		ResourceGroup:  data.Compute.ResourceGroupName,
//		InstanceName:   data.Compute.Name,
//		Location:       data.Compute.Location,
//		PublicIp:       publicIp,
//		PrivateIp:      privateIp,
//	}
//
//	return
//}
//
//type azureSubnet struct {
//	Name         string
//	RouteTableId string
//}
//
//type azureNetwork struct {
//	mdata        *azureMetadata
//	authr        autorest.Authorizer
//	Name         string
//	Subnets      []azureSubnet
//	RouteTableId string
//}
//
//func (n *azureNetwork) TableIds() (tables []string) {
//	tables = []string{}
//	tablesSet := set.NewSet()
//
//	for _, subnet := range n.Subnets {
//		if !tablesSet.Contains(subnet.RouteTableId) {
//			tables = append(tables, subnet.RouteTableId)
//			tablesSet.Add(subnet.RouteTableId)
//		}
//	}
//
//	return
//}
//
//func (n *azureNetwork) TableExists() (exists bool, err error) {
//	tableClient := network.NewRouteTablesClient(n.mdata.SubscriptionId)
//	tableClient.Authorizer = n.authr
//
//	res, err := tableClient.Get(context.Background(),
//		n.mdata.ResourceGroup, n.Name, "")
//	if err != nil {
//		if res.StatusCode == 404 {
//			exists = false
//			err = nil
//			return
//		}
//
//		err = &errortypes.RequestError{
//			errors.Wrap(err, "azure: Azure route table update error"),
//		}
//		return
//	}
//
//	if res.ID == nil || *res.ID == "" {
//		err = &errortypes.RequestError{
//			errors.New("azure: Azure compute table id nil"),
//		}
//		return
//	}
//
//	n.RouteTableId = *res.ID
//	exists = true
//
//	return
//}
//
//func (n *azureNetwork) UpsertTable() (err error) {
//	if n.RouteTableId != "" {
//		return
//	}
//
//	exists, err := n.TableExists()
//	if err != nil {
//		return
//	}
//
//	if exists {
//		return
//	}
//
//	tableName := n.Name
//	location := n.mdata.Location
//
//	tableClient := network.NewRouteTablesClient(n.mdata.SubscriptionId)
//	tableClient.Authorizer = n.authr
//
//	params := network.RouteTable{
//		Name:     &tableName,
//		Location: &location,
//	}
//
//	res, err := tableClient.CreateOrUpdate(context.Background(),
//		n.mdata.ResourceGroup, tableName, params)
//	if err != nil {
//		err = &errortypes.RequestError{
//			errors.Wrap(err, "azure: Azure route table update error"),
//		}
//		return
//	}
//
//	err = res.WaitForCompletionRef(
//		context.Background(), tableClient.Client)
//	if err != nil {
//		err = &errortypes.RequestError{
//			errors.Wrap(err, "azure: Azure route table wait error"),
//		}
//		return
//	}
//
//	tableRes, err := tableClient.Get(context.Background(),
//		n.mdata.ResourceGroup, n.Name, "")
//	if err != nil {
//		err = &errortypes.RequestError{
//			errors.Wrap(err, "azure: Azure route table update error"),
//		}
//		return
//	}
//
//	if tableRes.ID == nil || *tableRes.ID == "" {
//		err = &errortypes.RequestError{
//			errors.New("azure: Azure compute table id nil"),
//		}
//		return
//	}
//
//	n.RouteTableId = *tableRes.ID
//
//	return
//}
//
//func (n *azureNetwork) AttachTables() (err error) {
//	subnetClient := network.NewSubnetsClient(n.mdata.SubscriptionId)
//	subnetClient.Authorizer = n.authr
//
//	if n.RouteTableId == "" {
//		err = &errortypes.RequestError{
//			errors.New("azure: Cannot attach table without id"),
//		}
//		return
//	}
//
//	newSubnets := []azureSubnet{}
//
//	for _, subnet := range n.Subnets {
//		if subnet.RouteTableId != "" {
//			newSubnets = append(newSubnets, subnet)
//			continue
//		}
//
//		subnetRes, e := subnetClient.Get(context.Background(),
//			n.mdata.ResourceGroup, n.Name, subnet.Name, "")
//		if e != nil {
//			if subnetRes.StatusCode == 404 {
//				continue
//			}
//
//			err = &errortypes.RequestError{
//				errors.Wrap(e, "azure: Azure subnet get error"),
//			}
//			return
//		}
//
//		subnetRes.RouteTable = &network.RouteTable{
//			ID: &n.RouteTableId,
//		}
//
//		res, e := subnetClient.CreateOrUpdate(context.Background(),
//			n.mdata.ResourceGroup, n.Name, subnet.Name, subnetRes)
//		if e != nil {
//			err = &errortypes.RequestError{
//				errors.Wrap(e, "azure: Azure subnet table update error"),
//			}
//			return
//		}
//
//		err = res.WaitForCompletionRef(
//			context.Background(), subnetClient.Client)
//		if err != nil {
//			err = &errortypes.RequestError{
//				errors.Wrap(err, "azure: Azure subnet table wait error"),
//			}
//			return
//		}
//
//		newSubnet := azureSubnet{
//			Name:         subnet.Name,
//			RouteTableId: n.RouteTableId,
//		}
//		newSubnets = append(newSubnets, newSubnet)
//	}
//
//	n.Subnets = newSubnets
//
//	return
//}
//
//func (n *azureNetwork) AddRoute(destination string) (
//	changed bool, err error) {
//
//	tableClient := network.NewRouteTablesClient(n.mdata.SubscriptionId)
//	tableClient.Authorizer = n.authr
//
//	if strings.Contains(destination, ":") {
//		return
//	}
//
//	for _, tableId := range n.TableIds() {
//		tableIds := strings.Split(tableId, "/")
//		tableName := tableIds[len(tableIds)-1]
//
//		tableRes, e := tableClient.Get(context.Background(),
//			n.mdata.ResourceGroup, tableName, "")
//		if e != nil {
//			err = &errortypes.RequestError{
//				errors.Wrap(e, "azure: Azure table get error"),
//			}
//			return
//		}
//
//		if tableRes.Routes == nil {
//			err = &errortypes.RequestError{
//				errors.New("azure: Azure table routes nil"),
//			}
//			return
//		}
//
//		exists := false
//		nextHopType := network.RouteNextHopTypeVirtualAppliance
//		newRoutes := []network.Route{}
//
//		for _, route := range *tableRes.Routes {
//			if route.ID == nil || route.AddressPrefix == nil ||
//				*route.AddressPrefix != destination {
//
//				newRoutes = append(newRoutes, route)
//				continue
//			}
//
//			exists = true
//
//			if route.NextHopType == nextHopType &&
//				*route.NextHopIPAddress == n.mdata.PrivateIp {
//
//				newRoutes = append(newRoutes, route)
//				continue
//			}
//
//			route.NextHopType = network.RouteNextHopTypeVirtualAppliance
//			route.NextHopIPAddress = &n.mdata.PrivateIp
//			newRoutes = append(newRoutes, route)
//			changed = true
//		}
//
//		if !exists {
//			destinations := strings.Split(destination, "/")
//			if len(destinations) != 2 {
//				err = &errortypes.RequestError{
//					errors.New("azure: Azure route len error"),
//				}
//				return
//			}
//
//			routeName := fmt.Sprintf("%s-%s-%s",
//				n.Name, destinations[0], destinations[1])
//			route := network.Route{
//				Name: &routeName,
//				RoutePropertiesFormat: &network.RoutePropertiesFormat{
//					AddressPrefix:    &destination,
//					NextHopType:      nextHopType,
//					NextHopIPAddress: &n.mdata.PrivateIp,
//				},
//			}
//
//			newRoutes = append(newRoutes, route)
//			changed = true
//		}
//
//		if changed {
//			tableRes.Routes = &newRoutes
//
//			res, e := tableClient.CreateOrUpdate(context.Background(),
//				n.mdata.ResourceGroup, tableName, tableRes)
//			if e != nil {
//				err = &errortypes.RequestError{
//					errors.Wrap(e, "azure: Azure table update error"),
//				}
//				return
//			}
//
//			err = res.WaitForCompletionRef(
//				context.Background(), tableClient.Client)
//			if err != nil {
//				err = &errortypes.RequestError{
//					errors.Wrap(err, "azure: Azure table wait error"),
//				}
//				return
//			}
//		}
//	}
//
//	return
//}
//
//func (n *azureNetwork) RemoveRoute(rte *routes.AzureRoute) (
//	changed bool, err error) {
//
//	tableClient := network.NewRouteTablesClient(n.mdata.SubscriptionId)
//	tableClient.Authorizer = n.authr
//
//	if strings.Contains(rte.DestNetwork, ":") {
//		return
//	}
//
//	for _, tableId := range n.TableIds() {
//		tableIds := strings.Split(tableId, "/")
//		tableName := tableIds[len(tableIds)-1]
//
//		tableRes, e := tableClient.Get(context.Background(),
//			n.mdata.ResourceGroup, tableName, "")
//		if e != nil {
//			err = &errortypes.RequestError{
//				errors.Wrap(e, "azure: Azure table get error"),
//			}
//			return
//		}
//
//		if tableRes.Routes == nil {
//			err = &errortypes.RequestError{
//				errors.New("azure: Azure table routes nil"),
//			}
//			return
//		}
//
//		nextHopType := network.RouteNextHopTypeVirtualAppliance
//		newRoutes := []network.Route{}
//
//		for _, route := range *tableRes.Routes {
//			if route.ID == nil || route.AddressPrefix == nil ||
//				*route.AddressPrefix != rte.DestNetwork ||
//				route.NextHopType != nextHopType ||
//				*route.NextHopIPAddress != rte.NextHop {
//
//				newRoutes = append(newRoutes, route)
//				continue
//			}
//
//			changed = true
//		}
//
//		if changed {
//			tableRes.Routes = &newRoutes
//
//			res, e := tableClient.CreateOrUpdate(context.Background(),
//				n.mdata.ResourceGroup, tableName, tableRes)
//			if e != nil {
//				err = &errortypes.RequestError{
//					errors.Wrap(e, "azure: Azure table update error"),
//				}
//				return
//			}
//
//			err = res.WaitForCompletionRef(
//				context.Background(), tableClient.Client)
//			if err != nil {
//				err = &errortypes.RequestError{
//					errors.Wrap(err, "azure: Azure table wait error"),
//				}
//				return
//			}
//		}
//	}
//
//	return
//}
//
//func azureGetInstanceNetwork(mdata *azureMetadata) (
//	net *azureNetwork, err error) {
//
//	authr, err := auth.NewAuthorizerFromEnvironment()
//	if err != nil {
//		err = &errortypes.RequestError{
//			errors.Wrap(err, "azure: Azure authorizer init error"),
//		}
//		return
//	}
//
//	computeClient := compute.NewVirtualMachinesClient(mdata.SubscriptionId)
//	computeClient.Authorizer = authr
//
//	computeRes, err := computeClient.Get(
//		context.Background(), mdata.ResourceGroup, mdata.InstanceName,
//		compute.InstanceView)
//	if err != nil {
//		err = &errortypes.RequestError{
//			errors.Wrap(err, "azure: Azure compute request error"),
//		}
//		return
//	}
//
//	if computeRes.NetworkProfile == nil {
//		err = &errortypes.RequestError{
//			errors.New("azure: Azure compute network profile nil"),
//		}
//		return
//	}
//
//	if computeRes.NetworkProfile.NetworkInterfaces == nil {
//		err = &errortypes.RequestError{
//			errors.New("azure: Azure compute network interfaces nil"),
//		}
//		return
//	}
//
//	primaryIface := ""
//	for _, iface := range *computeRes.NetworkProfile.NetworkInterfaces {
//		primaryIface = *iface.ID
//		break
//	}
//
//	if primaryIface == "" {
//		err = &errortypes.RequestError{
//			errors.New("azure: Azure compute primary interface nil"),
//		}
//		return
//	}
//
//	primaryIfaces := strings.Split(primaryIface, "/")
//	primaryIface = primaryIfaces[len(primaryIfaces)-1]
//
//	ifaceClient := network.NewInterfaceIPConfigurationsClient(
//		mdata.SubscriptionId)
//	ifaceClient.Authorizer = authr
//
//	ifaceRes, err := ifaceClient.List(
//		context.Background(), mdata.ResourceGroup, primaryIface)
//	if err != nil {
//		err = &errortypes.RequestError{
//			errors.Wrap(err, "azure: Azure interface request error"),
//		}
//		return
//	}
//
//	ifaces := ifaceRes.Values()
//	if ifaces == nil {
//		err = &errortypes.RequestError{
//			errors.New("azure: Azure interface configuration nil"),
//		}
//		return
//	}
//
//	networkName := ""
//	for _, iface := range ifaces {
//		if iface.Subnet == nil {
//			err = &errortypes.RequestError{
//				errors.New("azure: Azure interface subnet nil"),
//			}
//			return
//		}
//
//		if iface.Subnet.ID == nil {
//			err = &errortypes.RequestError{
//				errors.New("azure: Azure interface subnet id nil"),
//			}
//			return
//		}
//
//		networkName = *iface.Subnet.ID
//		break
//	}
//
//	if networkName == "" {
//		err = &errortypes.RequestError{
//			errors.New("azure: Azure compute network id nil"),
//		}
//		return
//	}
//
//	networkNames := strings.Split(networkName, "/")
//	if len(networkNames) < 4 {
//		err = &errortypes.RequestError{
//			errors.New("azure: Azure compute network id parse error"),
//		}
//		return
//	}
//	networkName = networkNames[len(networkNames)-3]
//
//	vnetClient := network.NewVirtualNetworksClient(mdata.SubscriptionId)
//	vnetClient.Authorizer = authr
//
//	res, err := vnetClient.Get(context.Background(),
//		mdata.ResourceGroup, networkName, "")
//	if err != nil {
//		err = &errortypes.RequestError{
//			errors.Wrap(err, "azure: Azure networks request error"),
//		}
//		return
//	}
//
//	if res.Subnets == nil {
//		err = &errortypes.RequestError{
//			errors.New("azure: Azure network subnets nil"),
//		}
//		return
//	}
//
//	net = &azureNetwork{
//		mdata:   mdata,
//		authr:   authr,
//		Name:    networkName,
//		Subnets: []azureSubnet{},
//	}
//
//	for _, subnet := range *res.Subnets {
//		if subnet.Name == nil {
//			continue
//		}
//
//		if subnet.AddressPrefix == nil {
//			err = &errortypes.RequestError{
//				errors.New("azure: Azure compute subnet address prefix nil"),
//			}
//			return
//		}
//
//		routeTableId := ""
//		if subnet.RouteTable != nil && subnet.RouteTable.ID != nil {
//			routeTableId = *subnet.RouteTable.ID
//		}
//
//		if routeTableId != "" && net.RouteTableId == "" {
//			net.RouteTableId = routeTableId
//		}
//
//		snet := azureSubnet{
//			Name:         *subnet.Name,
//			RouteTableId: routeTableId,
//		}
//
//		net.Subnets = append(net.Subnets, snet)
//	}
//
//	return
//}

func AzureAddRoute(network string) (err error) {
	//mdata, err := azureGetMetaData()
	//if err != nil {
	//	return
	//}
	//
	//net, err := azureGetInstanceNetwork(mdata)
	//if err != nil {
	//	return
	//}
	//
	//err = net.UpsertTable()
	//if err != nil {
	//	return
	//}
	//
	//err = net.AttachTables()
	//if err != nil {
	//	return
	//}
	//
	//changed, err := net.AddRoute(network)
	//if err != nil {
	//	return
	//}
	//
	//if !changed {
	//	return
	//}
	//
	//route := &routes.AzureRoute{
	//	DestNetwork: network,
	//	NextHop:     mdata.PrivateIp,
	//}
	//
	//err = route.Add()
	//if err != nil {
	//	return
	//}

	err = &errortypes.UnknownError{
		errors.New("cloud: Azure not supported"),
	}

	return
}

func AzureDeleteRoute(rte *routes.AzureRoute) (err error) {
	//if !config.Config.DeleteRoutes {
	//	err = rte.Remove()
	//	if err != nil {
	//		return
	//	}
	//
	//	return
	//}
	//
	//mdata, err := azureGetMetaData()
	//if err != nil {
	//	return
	//}
	//
	//net, err := azureGetInstanceNetwork(mdata)
	//if err != nil {
	//	return
	//}
	//
	//err = net.UpsertTable()
	//if err != nil {
	//	return
	//}
	//
	//err = net.AttachTables()
	//if err != nil {
	//	return
	//}
	//
	//_, err = net.RemoveRoute(rte)
	//if err != nil {
	//	return
	//}
	//
	//err = rte.Remove()
	//if err != nil {
	//	return
	//}

	err = &errortypes.UnknownError{
		errors.New("cloud: Azure not supported"),
	}

	return
}
