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
)

type awsMetaData struct {
	Region     string
	InstanceId string
	VpcId      string
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

	data = &awsMetaData{
		Region:     region,
		InstanceId: instanceId,
		VpcId:      vpcId,
	}

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
	region := ""
	vpcId := ""
	instanceId := ""
	interfaceId := ""

	if config.Config.Aws != nil {
		region = config.Config.Aws.Region
		vpcId = config.Config.Aws.VpcId
		instanceId = config.Config.Aws.InstanceId
		interfaceId = config.Config.Aws.InterfaceId
	}

	if region == "" || vpcId == "" || (instanceId == "" && interfaceId == "") {
		data, e := awsGetMetaData()
		if e != nil {
			err = e
			return
		}

		region = data.Region
		vpcId = data.VpcId
		instanceId = data.InstanceId
	}

	tables, err := awsGetRouteTables(region, vpcId)
	if err != nil {
		return
	}

	sess, err := awsGetSession(region)
	if err != nil {
		return
	}

	ec2Svc := ec2.New(sess)

	for _, table := range tables {
		input := &ec2.CreateRouteInput{}
		input.SetDestinationCidrBlock(network)
		input.SetRouteTableId(table)

		if interfaceId != "" {
			input.SetNetworkInterfaceId(interfaceId)
		} else {
			input.SetInstanceId(instanceId)
		}

		_, err = ec2Svc.CreateRoute(input)
		if err != nil {
			input := &ec2.ReplaceRouteInput{}
			input.SetInstanceId(instanceId)
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

	route := &awsRoute{
		Network:     network,
		InterfaceId: interfaceId,
		InstanceId:  instanceId,
	}

	err = route.Add()
	if err != nil {
		return
	}

	return
}
