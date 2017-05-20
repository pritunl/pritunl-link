package advertise

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/routes"
	"time"
)

type awsMetaData struct {
	Region      string
	InstanceId  string
	InterfaceId string
	VpcId       string
}

func awsGetSession(region string) (sess *session.Session, err error) {
	opts := session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}

	if region != "" {
		opts.Config = aws.Config{
			Region: &region,
		}
	}

	sess, err = session.NewSessionWithOptions(opts)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "cloud: Failed to create AWS session"),
		}
		return
	}

	return
}

func awsGetMetaData() (data *awsMetaData, err error) {
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

	sess, err := awsGetSession("")
	if err != nil {
		return
	}

	ec2metadataSvc := ec2metadata.New(sess)

	region, err := ec2metadataSvc.Region()
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "cloud: Failed to get AWS region"),
		}
		return
	}

	instanceId, err := ec2metadataSvc.GetMetadata("instance-id")
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "cloud: Failed to get EC2 instance ID"),
		}
		return
	}

	macAddr, err := ec2metadataSvc.GetMetadata("mac")
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "cloud: Failed to get EC2 MAC address"),
		}
		return
	}

	vpcId, err := ec2metadataSvc.GetMetadata(
		fmt.Sprintf("network/interfaces/macs/%s/vpc-id", macAddr))
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "cloud: Failed to get EC2 VPC ID"),
		}
		return
	}

	data.Region = region
	data.VpcId = vpcId
	data.InstanceId = instanceId

	return
}

func awsGetRouteTables(region, vpcId string) (tables []string, err error) {
	tables = []string{}

	sess, err := awsGetSession(region)
	if err != nil {
		return
	}

	ec2Svc := ec2.New(sess)

	filterName := "vpc-id"
	filters := []*ec2.Filter{
		{
			Name: &filterName,
			Values: []*string{
				&vpcId,
			},
		},
	}

	input := &ec2.DescribeRouteTablesInput{}
	input.SetFilters(filters)

	vpcTables, err := ec2Svc.DescribeRouteTables(input)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "cloud: Failed to get VPC route tables"),
		}
		return
	}

	for _, table := range vpcTables.RouteTables {
		tables = append(tables, *table.RouteTableId)
	}

	return
}

func AwsAddRoute(network string) (err error) {
	time.Sleep(150 * time.Millisecond)

	data, err := awsGetMetaData()
	if err != nil {
		return
	}

	tables, err := awsGetRouteTables(data.Region, data.VpcId)
	if err != nil {
		return
	}

	sess, err := awsGetSession(data.Region)
	if err != nil {
		return
	}

	ec2Svc := ec2.New(sess)

	for _, table := range tables {
		input := &ec2.ReplaceRouteInput{}

		if data.InterfaceId != "" {
			input.SetNetworkInterfaceId(data.InterfaceId)
		} else {
			input.SetInstanceId(data.InstanceId)
		}

		input.SetInstanceId(data.InstanceId)
		input.SetDestinationCidrBlock(network)
		input.SetRouteTableId(table)

		_, err = ec2Svc.ReplaceRoute(input)

		if err != nil {
			input := &ec2.CreateRouteInput{}
			input.SetDestinationCidrBlock(network)
			input.SetRouteTableId(table)

			if data.InterfaceId != "" {
				input.SetNetworkInterfaceId(data.InterfaceId)
			} else {
				input.SetInstanceId(data.InstanceId)
			}

			_, err = ec2Svc.CreateRoute(input)
			if err != nil {
				err = &errortypes.RequestError{
					errors.Wrap(err, "cloud: Failed to get create route"),
				}
				return
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

		tables, e := awsGetRouteTables(route.Region, route.VpcId)
		if e != nil {
			err = e
			return
		}

		sess, e := awsGetSession(route.Region)
		if e != nil {
			err = e
			return
		}

		ec2Svc := ec2.New(sess)

		for _, table := range tables {
			input := &ec2.DeleteRouteInput{}

			input.SetDestinationCidrBlock(route.DestNetwork)
			input.SetRouteTableId(table)

			ec2Svc.DeleteRoute(input)
		}
	}

	err = route.Remove()
	if err != nil {
		return
	}

	return
}
