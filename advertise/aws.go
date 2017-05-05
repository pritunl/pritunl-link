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

	if config.Config.Aws != nil {
		region := config.Config.Aws.Region
		vpcId := config.Config.Aws.VpcId
		instanceId := config.Config.Aws.InstanceId
		interfaceId := config.Config.Aws.InterfaceId

		if region != "" && vpcId != "" &&
			(instanceId != "" || interfaceId != "") {

			data.Region = region
			data.VpcId = vpcId
			data.InstanceId = instanceId
			data.InterfaceId = interfaceId

			return
		}
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
	tables, err := awsGetRouteTables(route.Region, route.VpcId)
	if err != nil {
		return
	}

	sess, err := awsGetSession(route.Region)
	if err != nil {
		return
	}

	ec2Svc := ec2.New(sess)

	for _, table := range tables {
		input := &ec2.DeleteRouteInput{}

		input.SetDestinationCidrBlock(route.DestNetwork)
		input.SetRouteTableId(table)

		_, err = ec2Svc.DeleteRoute(input)
		if err != nil {
			err = &errortypes.RequestError{
				errors.Wrap(err, "cloud: Failed to get delete route"),
			}
			return
		}
	}

	err = route.Remove()
	if err != nil {
		return
	}

	return
}
