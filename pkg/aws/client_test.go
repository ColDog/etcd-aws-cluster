package aws

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var mockSession = func() *session.Session {
	// server is the mock server that simply writes a 200 status back to the client
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	}))

	return session.Must(session.NewSession(&aws.Config{
		DisableSSL: aws.Bool(true),
		Endpoint:   aws.String(server.URL),
		Region:     aws.String("test"),
	}))
}()

type EC2Mock struct {
	ec2iface.EC2API
	mock.Mock
}

func (m *EC2Mock) DescribeInstances(in *ec2.DescribeInstancesInput) (
	*ec2.DescribeInstancesOutput, error) {
	a := m.Called(in)
	return a.Get(0).(*ec2.DescribeInstancesOutput), a.Error(1)
}

type ASGMock struct {
	autoscalingiface.AutoScalingAPI
	mock.Mock
}

func (m *ASGMock) DescribeAutoScalingGroups(
	in *autoscaling.DescribeAutoScalingGroupsInput) (
	*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	a := m.Called(in)
	return a.Get(0).(*autoscaling.DescribeAutoScalingGroupsOutput), a.Error(1)
}

func (m *ASGMock) DescribeAutoScalingGroupsPages(
	in *autoscaling.DescribeAutoScalingGroupsInput,
	fn func(*autoscaling.DescribeAutoScalingGroupsOutput, bool) bool) error {
	a := m.Called(in)
	fn(a.Get(0).(*autoscaling.DescribeAutoScalingGroupsOutput), true)
	return a.Error(1)
}

type S3Mock struct {
	s3iface.S3API
	mock.Mock
}

func (m *S3Mock) PutObject(in *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	a := m.Called(in)
	return a.Get(0).(*s3.PutObjectOutput), a.Error(1)
}

func TestClient_Load(t *testing.T) {
	createSession = func(...*aws.Config) (*session.Session, error) { return mockSession, nil }
	_, err := NewClient()
	require.Equal(t, "aws: autoscaling group not found", err.Error())
}

func TestClient_Instances(t *testing.T) {
	a := &ASGMock{}
	e := &EC2Mock{}

	c := &client{
		asg:        a,
		ec2:        e,
		hostname:   "1.ec2.internal",
		region:     "us-west-2",
		instanceID: "1",
		groupName:  "test",
	}

	a.On("DescribeAutoScalingGroups", &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{aws.String("test")},
	}).Return(&autoscaling.DescribeAutoScalingGroupsOutput{
		AutoScalingGroups: []*autoscaling.Group{
			{
				Instances: []*autoscaling.Instance{
					{InstanceId: aws.String("1")},
					{InstanceId: aws.String("2")},
				},
			},
		},
	}, nil)
	e.On("DescribeInstances", &ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String("1"), aws.String("2")},
	}).Return(&ec2.DescribeInstancesOutput{
		Reservations: []*ec2.Reservation{{
			Instances: []*ec2.Instance{
				{
					InstanceId: aws.String("1"),
					NetworkInterfaces: []*ec2.InstanceNetworkInterface{{
						PrivateIpAddress: aws.String("1.ec2.internal"),
					}},
				},
				{
					InstanceId: aws.String("2"),
					NetworkInterfaces: []*ec2.InstanceNetworkInterface{{
						PrivateIpAddress: aws.String("2.ec2.internal"),
					}},
				},
			},
		}},
	}, nil)

	m, err := c.GroupInstances()
	require.Nil(t, err)
	require.Equal(t, map[string]string{
		"1": "1.ec2.internal",
		"2": "2.ec2.internal",
	}, m)

	e.AssertExpectations(t)
	a.AssertExpectations(t)
}

func TestClient_LoadName(t *testing.T) {
	a := &ASGMock{}

	c := &client{
		asg:        a,
		hostname:   "1.ec2.internal",
		region:     "us-west-2",
		instanceID: "1",
		groupName:  "test",
	}

	a.On("DescribeAutoScalingGroupsPages",
		&autoscaling.DescribeAutoScalingGroupsInput{}).
		Return(&autoscaling.DescribeAutoScalingGroupsOutput{
			AutoScalingGroups: []*autoscaling.Group{
				{
					AutoScalingGroupName: aws.String("new"),
					Instances: []*autoscaling.Instance{
						{InstanceId: aws.String("1")},
						{InstanceId: aws.String("2")},
					},
				},
			},
		}, nil)

	err := c.loadGroupName()
	require.Nil(t, err)
	require.Equal(t, "new", c.groupName)

	a.AssertExpectations(t)
}

func TestClient_Upload(t *testing.T) {
	s := &S3Mock{}

	c := &client{
		s3:         s,
		hostname:   "1.ec2.internal",
		region:     "us-west-2",
		instanceID: "1",
		groupName:  "test",
	}

	f, _ := os.Open("testdata/test.txt")
	defer f.Close()

	s.On("PutObject",
		mock.MatchedBy(func(in *s3.PutObjectInput) bool {
			return *in.Bucket == "test" && *in.Key == "test"
		})).
		Return(&s3.PutObjectOutput{}, nil)

	err := c.Upload("testdata/test.txt", "test", "test")
	require.Nil(t, err)

	s.AssertExpectations(t)
}
