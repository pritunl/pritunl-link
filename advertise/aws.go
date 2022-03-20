package advertise

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/routes"
	"github.com/pritunl/pritunl-link/utils"
)

type awsMetaData struct {
	Region      string
	InstanceId  string
	InterfaceId string
	VpcId       string
}

type awsRoute struct {
	DestinationCidrBlock     string
	DestinationIpv6CidrBlock string
	InstanceId               string
	NetworkInterfaceId       string
}

func awsGetSession(region string) (cfg aws.Config, err error) {
	if region != "" {
		cfg, err = awsconfig.LoadDefaultConfig(
			context.Background(),
			awsconfig.WithRegion(region),
		)
	} else {
		cfg, err = awsconfig.LoadDefaultConfig(
			context.Background(),
		)
	}
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "cloud: Failed to create AWS config"),
		}
		return
	}

	return
}

func awsGetMetaData(ctx context.Context) (data *awsMetaData, err error) {
	data = &awsMetaData{}

	confRegion := config.Config.Aws.Region
	confVpcId := config.Config.Aws.VpcId
	confInstanceId := config.Config.Aws.InstanceId
	confInterfaceId := config.Config.Aws.InterfaceId

	if confRegion != "" && confVpcId != "" &&
		(confInstanceId != "" || confInterfaceId != "") {

		data.Region = confRegion
		data.VpcId = confVpcId
		data.InstanceId = confInstanceId
		data.InterfaceId = confInterfaceId

		return
	}

	cfg, err := awsGetSession("")
	if err != nil {
		return
	}

	client := imds.NewFromConfig(cfg)

	region, err := client.GetRegion(ctx, &imds.GetRegionInput{})
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "cloud: Failed to get AWS region"),
		}
		return
	}

	instanceIdResp, err := client.GetMetadata(ctx, &imds.GetMetadataInput{
		Path: "instance-id",
	})
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "cloud: Failed to get EC2 instance ID"),
		}
		return
	}

	instanceId, err := ioutil.ReadAll(instanceIdResp.Content)
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "cloud: Failed to read EC2 instance ID"),
		}
		return
	}

	macAddrResp, err := client.GetMetadata(ctx, &imds.GetMetadataInput{
		Path: "mac",
	})
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "cloud: Failed to get EC2 MAC address"),
		}
		return
	}

	macAddr, err := ioutil.ReadAll(macAddrResp.Content)
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "cloud: Failed to read EC2 MAC address"),
		}
		return
	}

	vpcIdResp, err := client.GetMetadata(ctx, &imds.GetMetadataInput{
		Path: fmt.Sprintf(
			"network/interfaces/macs/%s/vpc-id",
			string(macAddr),
		),
	})
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "cloud: Failed to get EC2 VPC ID"),
		}
		return
	}

	vpcId, err := ioutil.ReadAll(vpcIdResp.Content)
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "cloud: Failed to read EC2 VPC ID"),
		}
		return
	}

	data.Region = region.Region
	data.VpcId = string(vpcId)
	data.InstanceId = string(instanceId)

	return
}

func awsGetRouteTables(ctx context.Context, region, vpcId string) (
	tables map[string][]*awsRoute, err error) {

	tables = map[string][]*awsRoute{}

	cfg, err := awsGetSession(region)
	if err != nil {
		return
	}

	client := ec2.NewFromConfig(cfg)

	input := &ec2.DescribeRouteTablesInput{
		Filters: []types.Filter{
			types.Filter{
				Name: utils.StringX("vpc-id"),
				Values: []string{
					vpcId,
				},
			},
		},
	}

	vpcTables, err := client.DescribeRouteTables(ctx, input)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "cloud: Failed to get VPC route tables"),
		}
		return
	}

	for _, table := range vpcTables.RouteTables {
		rtes := []*awsRoute{}

		for _, route := range table.Routes {
			destinationCidrBlock := ""
			if route.DestinationCidrBlock != nil {
				destinationCidrBlock = *route.DestinationCidrBlock
			}

			destinationIpv6CidrBlock := ""
			if route.DestinationIpv6CidrBlock != nil {
				destinationIpv6CidrBlock = *route.DestinationIpv6CidrBlock
			}

			instanceId := ""
			if route.InstanceId != nil {
				instanceId = *route.InstanceId
			}

			networkInterfaceId := ""
			if route.NetworkInterfaceId != nil {
				networkInterfaceId = *route.NetworkInterfaceId
			}

			rte := &awsRoute{
				DestinationCidrBlock:     destinationCidrBlock,
				DestinationIpv6CidrBlock: destinationIpv6CidrBlock,
				InstanceId:               instanceId,
				NetworkInterfaceId:       networkInterfaceId,
			}

			rtes = append(rtes, rte)
		}

		tableId := ""
		if table.RouteTableId != nil {
			tableId = *table.RouteTableId
		}

		if tableId == "" {
			continue
		}

		tables[tableId] = rtes
	}

	return
}

func AwsAddRoute(network string) (err error) {
	time.Sleep(150 * time.Millisecond)

	ipv6 := strings.Contains(network, ":")

	if constants.Interrupt {
		err = &errortypes.UnknownError{
			errors.Wrap(err, "advertise: Interrupt"),
		}
		return
	}

	ctx := context.Background()

	data, err := awsGetMetaData(ctx)
	if err != nil {
		return
	}

	tables, err := awsGetRouteTables(ctx, data.Region, data.VpcId)
	if err != nil {
		return
	}

	cfg, err := awsGetSession(data.Region)
	if err != nil {
		return
	}

	client := ec2.NewFromConfig(cfg)

	for tableId, rtes := range tables {
		exists := false
		replace := false

		for _, route := range rtes {
			if ipv6 {
				if route.DestinationIpv6CidrBlock != network {
					continue
				}
			} else {
				if route.DestinationCidrBlock != network {
					continue
				}
			}

			exists = true

			if data.InterfaceId != "" {
				if route.NetworkInterfaceId != data.InterfaceId {
					replace = true
				}
			} else {
				if route.InstanceId != data.InstanceId {
					replace = true
				}
			}

			break
		}

		if exists && !replace {
			continue
		}

		if replace {
			input := &ec2.ReplaceRouteInput{}

			if data.InterfaceId != "" {
				input.NetworkInterfaceId = utils.StringX(data.InterfaceId)
			} else {
				input.InstanceId = utils.StringX(data.InstanceId)
			}

			if ipv6 {
				input.DestinationIpv6CidrBlock = utils.StringX(network)
			} else {
				input.DestinationCidrBlock = utils.StringX(network)
			}
			input.RouteTableId = utils.StringX(tableId)

			_, err = client.ReplaceRoute(ctx, input)
			if err != nil {
				input := &ec2.CreateRouteInput{}
				input.DestinationCidrBlock = utils.StringX(network)
				input.RouteTableId = utils.StringX(tableId)

				if data.InterfaceId != "" {
					input.NetworkInterfaceId = utils.StringX(data.InterfaceId)
				} else {
					input.InstanceId = utils.StringX(data.InstanceId)
				}

				_, err = client.CreateRoute(ctx, input)
				if err != nil {
					err = &errortypes.RequestError{
						errors.Wrap(err, "cloud: Failed to get create route"),
					}
					return
				}
			}
		} else {
			input := &ec2.CreateRouteInput{}
			if ipv6 {
				input.DestinationIpv6CidrBlock = utils.StringX(network)
			} else {
				input.DestinationCidrBlock = utils.StringX(network)
			}
			input.RouteTableId = utils.StringX(tableId)

			if data.InterfaceId != "" {
				input.NetworkInterfaceId = utils.StringX(data.InterfaceId)
			} else {
				input.InstanceId = utils.StringX(data.InstanceId)
			}

			_, err = client.CreateRoute(ctx, input)
			if err != nil {
				input := &ec2.ReplaceRouteInput{}

				if data.InterfaceId != "" {
					input.NetworkInterfaceId = utils.StringX(data.InterfaceId)
				} else {
					input.InstanceId = utils.StringX(data.InstanceId)
				}

				if ipv6 {
					input.DestinationIpv6CidrBlock = utils.StringX(network)
				} else {
					input.DestinationCidrBlock = utils.StringX(network)
				}
				input.RouteTableId = utils.StringX(tableId)

				_, err = client.ReplaceRoute(ctx, input)
				if err != nil {
					err = &errortypes.RequestError{
						errors.Wrap(err, "cloud: Failed to get create route"),
					}
					return
				}
			}
		}
	}

	route := &routes.AwsRoute{
		DestNetwork: network,
		Region:      data.Region,
		VpcId:       data.VpcId,
		InterfaceId: data.InterfaceId,
		InstanceId:  data.InstanceId,
	}

	err = route.Add()
	if err != nil {
		return
	}

	return
}

func AwsDeleteRoute(route *routes.AwsRoute) (err error) {
	if config.Config.DeleteRoutes {
		time.Sleep(150 * time.Millisecond)

		ipv6 := strings.Contains(route.DestNetwork, ":")

		if constants.Interrupt {
			err = &errortypes.UnknownError{
				errors.Wrap(err, "advertise: Interrupt"),
			}
			return
		}

		ctx := context.Background()

		tables, e := awsGetRouteTables(ctx, route.Region, route.VpcId)
		if e != nil {
			err = e
			return
		}

		cfg, e := awsGetSession(route.Region)
		if e != nil {
			err = e
			return
		}

		client := ec2.NewFromConfig(cfg)

		for tableId := range tables {
			input := &ec2.DeleteRouteInput{}

			if ipv6 {
				input.DestinationIpv6CidrBlock = utils.StringX(
					route.DestNetwork)
			} else {
				input.DestinationCidrBlock = utils.StringX(route.DestNetwork)
			}
			input.RouteTableId = utils.StringX(tableId)

			client.DeleteRoute(ctx, input)
		}
	}

	err = route.Remove()
	if err != nil {
		return
	}

	return
}
