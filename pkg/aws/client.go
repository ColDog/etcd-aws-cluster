package aws

import (
	"errors"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

var createSession = session.NewSession

type Client interface {
	Hostname() string
	InstanceID() string
	Region() string
	GroupName() string

	GroupInstances() (map[string]string, error)

	Upload(filename, bucket, key string) error
}

func NewClient() (Client, error) {
	sess, err := createSession()
	if err != nil {
		return nil, err
	}
	meta := ec2metadata.New(sess)
	doc, err := meta.GetInstanceIdentityDocument()
	if err != nil {
		return nil, err
	}
	hostname, err := meta.GetMetadata("local-hostname")
	if err != nil {
		return nil, err
	}
	instanceID, err := meta.GetMetadata("instance-id")
	if err != nil {
		return nil, err
	}
	sess, err = createSession(&aws.Config{
		Region: &doc.Region,
	})
	if err != nil {
		return nil, err
	}
	c := &client{
		asg:        autoscaling.New(sess),
		ec2:        ec2.New(sess),
		s3:         s3.New(sess),
		hostname:   hostname,
		region:     doc.Region,
		instanceID: instanceID,
	}
	err = c.loadGroupName()
	if err != nil {
		return nil, err
	}
	return c, nil
}

type client struct {
	asg        autoscalingiface.AutoScalingAPI
	ec2        ec2iface.EC2API
	s3         s3iface.S3API
	hostname   string
	region     string
	instanceID string
	groupName  string
}

func (c *client) Region() string     { return c.region }
func (c *client) Hostname() string   { return c.hostname }
func (c *client) InstanceID() string { return c.instanceID }
func (c *client) GroupName() string  { return c.groupName }

func (c *client) loadGroupName() error {
	var name string
	err := c.asg.DescribeAutoScalingGroupsPages(
		&autoscaling.DescribeAutoScalingGroupsInput{},
		func(page *autoscaling.DescribeAutoScalingGroupsOutput, lastPage bool) bool {
			for _, g := range page.AutoScalingGroups {
				for _, inst := range g.Instances {
					if c.instanceID == *inst.InstanceId {
						name = *g.AutoScalingGroupName
						return false
					}
				}
			}
			return true
		},
	)
	if err != nil {
		return err
	}
	if name == "" {
		return errors.New("aws: autoscaling group not found")
	}
	c.groupName = name
	return nil
}

func (c *client) GroupInstances() (map[string]string, error) {
	groups, err := c.asg.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{&c.groupName},
	})
	if err != nil {
		return nil, err
	}
	instances := []*string{}
	for _, group := range groups.AutoScalingGroups {
		for _, inst := range group.Instances {
			instances = append(instances, inst.InstanceId)
		}
	}
	ec2Inst, err := c.ec2.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: instances,
	})
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, rev := range ec2Inst.Reservations {
		for _, inst := range rev.Instances {
			if len(inst.NetworkInterfaces) == 0 {
				continue
			}
			out[*inst.InstanceId] = *inst.NetworkInterfaces[0].PrivateIpAddress
		}
	}
	return out, nil
}

func (c *client) Upload(filename, bucket, key string) (err error) {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() {
		if cErr := f.Close(); cErr != nil {
			err = cErr
		}
	}()
	_, err = c.s3.PutObject(&s3.PutObjectInput{
		Key:    &key,
		Bucket: &bucket,
		Body:   f,
	})
	return err
}
