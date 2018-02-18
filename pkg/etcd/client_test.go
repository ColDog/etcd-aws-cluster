package etcd

import (
	"context"
	"os"
	"testing"

	etcd "github.com/coreos/etcd/client"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

type MockAPI struct {
	etcd.MembersAPI
	mock.Mock
}

func (m *MockAPI) connect(url string) (etcd.MembersAPI, error) {
	return m, nil
}

func (m *MockAPI) Add(ctx context.Context, peerURL string) (*etcd.Member, error) {
	a := m.Called(peerURL)
	return nil, a.Error(1)
}

func (m *MockAPI) Remove(ctx context.Context, id string) error {
	a := m.Called(id)
	return a.Error(0)
}

func (m *MockAPI) List(ctx context.Context) ([]etcd.Member, error) {
	a := m.Called()
	return a.Get(0).([]etcd.Member), a.Error(1)
}

var etcdTestConfig = Config{
	PeerScheme:   "https",
	ClientScheme: "https",
	ClientPort:   "2380",
	PeerPort:     "2379",
}

func TestTransport(t *testing.T) {
	dir, _ := os.Getwd()
	_, err := transport(
		dir+"/testdata/etcd.pem",
		dir+"/testdata/etcd-key.pem",
		dir+"/testdata/etcd-ca.pem",
	)
	require.Nil(t, err)
}

func TestClient_Add(t *testing.T) {
	m := &MockAPI{}

	m.On("Add", "https://1.ec2.internal:2379").Return(nil, nil)

	c := &client{
		config:  etcdTestConfig,
		connect: m.connect,
	}

	err := c.Add("2.ec2.internal", "1.ec2.internal")
	require.Nil(t, err)

	m.AssertExpectations(t)
}

func TestClient_Remove(t *testing.T) {
	m := &MockAPI{}

	m.On("List").Return([]etcd.Member{
		{
			ID:   "xxxxxx",
			Name: "1",
			ClientURLs: []string{
				"https://2.ec2.internal:2380",
			},
			PeerURLs: []string{
				"https://2.ec2.internal:2379",
			},
		},
	}, nil)
	m.On("Remove", "xxxxxx").Return(nil)

	c := &client{
		config:  etcdTestConfig,
		connect: m.connect,
	}

	err := c.Remove("2.ec2.internal", "1")
	require.Nil(t, err)

	m.AssertExpectations(t)
}

func TestClient_Available(t *testing.T) {
	m := &MockAPI{}

	m.On("List").Return([]etcd.Member{
		{
			ID:   "xxxxxx",
			Name: "1",
			ClientURLs: []string{
				"https://2.ec2.internal:2380",
			},
			PeerURLs: []string{
				"https://2.ec2.internal:2379",
			},
		},
	}, nil)

	c := &client{
		config:  etcdTestConfig,
		connect: m.connect,
	}

	ok := c.IsAvailable("2.ec2.internal")
	require.True(t, ok)

	m.AssertExpectations(t)
}

func TestClient_Members(t *testing.T) {
	m := &MockAPI{}

	m.On("List").Return([]etcd.Member{
		{
			ID:   "xxxxxx",
			Name: "1",
			ClientURLs: []string{
				"https://2.ec2.internal:2380",
			},
			PeerURLs: []string{
				"https://2.ec2.internal:2379",
			},
		},
	}, nil)

	c := &client{
		config:  etcdTestConfig,
		connect: m.connect,
	}

	b, err := c.Members("2.ec2.internal")
	require.Nil(t, err)
	require.Equal(t, map[string]string{
		"1": "2.ec2.internal",
	}, b)

	m.AssertExpectations(t)
}
