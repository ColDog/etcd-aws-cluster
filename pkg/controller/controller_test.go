package controller

import (
	"io/ioutil"
	"testing"

	"github.com/coldog/etcd-aws-cluster/pkg/aws"
	"github.com/coldog/etcd-aws-cluster/pkg/etcd"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func tempFileName() string {
	f, _ := ioutil.TempFile("", "conf-")
	f.Close()
	return f.Name()
}

var etcdTestConfig = etcd.Config{
	EnvFile:      tempFileName(),
	PeerScheme:   "https",
	ClientScheme: "https",
	ClientPort:   "2380",
	PeerPort:     "2379",
}

type MockAWS struct {
	aws.Client
	mock.Mock
}

func (m *MockAWS) Hostname() string   { return m.Called().String(0) }
func (m *MockAWS) Region() string     { return m.Called().String(0) }
func (m *MockAWS) InstanceID() string { return m.Called().String(0) }
func (m *MockAWS) GroupName() string  { return m.Called().String(0) }

func (m *MockAWS) GroupInstances() (map[string]string, error) {
	a := m.Called()
	return a.Get(0).(map[string]string), a.Error(1)
}

type MockETCD struct {
	mock.Mock
}

func (m *MockETCD) Config() etcd.Config {
	return m.Called().Get(0).(etcd.Config)
}

func (m *MockETCD) Add(clientHostname, candidateHostname string) error {
	return m.Called(clientHostname, candidateHostname).Error(0)
}

func (m *MockETCD) Remove(clientHostname, candidateHostname string) error {
	return m.Called(clientHostname, candidateHostname).Error(0)
}

func (m *MockETCD) IsAvailable(hostname string) bool {
	return m.Called(hostname).Bool(0)
}

func (m *MockETCD) Members(hostname string) (map[string]string, error) {
	a := m.Called(hostname)
	return a.Get(0).(map[string]string), nil
}

func TestConfig_Available(t *testing.T) {
	c := &Config{
		AvailableMembers: map[string]bool{
			"1": true,
		},
		Instances: map[string]string{
			"1": "1.ec2.local",
		},
	}

	require.True(t, c.AnyAvailable())
	require.Equal(t, "1.ec2.local", c.AnyAvailableHost())
}

func TestController_NewClusterGetConfig(t *testing.T) {
	a := &MockAWS{}
	e := &MockETCD{}

	c := &Controller{
		aws:  a,
		etcd: e,
	}

	a.On("InstanceID").Return("1")
	a.On("Hostname").Return("1.ec2.internal")
	a.On("GroupName").Return("test")
	a.On("GroupInstances").Return(map[string]string{
		"1": "1.ec2.internal",
		"2": "2.ec2.internal",
	}, nil)

	e.On("IsAvailable", "1.ec2.internal").Return(false)
	e.On("IsAvailable", "2.ec2.internal").Return(false)
	e.On("Config").Return(etcdTestConfig)

	expected := &Config{
		Config:     etcdTestConfig,
		InstanceID: "1",
		Hostname:   "1.ec2.internal",
		GroupName:  "test",
		Instances: map[string]string{
			"1": "1.ec2.internal",
			"2": "2.ec2.internal",
		},
		AvailableMembers: map[string]bool{
			"1": false,
			"2": false,
		},
		ActiveMembers: map[string]string{},
	}

	config, err := c.refreshConfig()
	require.Nil(t, err)
	require.Equal(t, expected, config)
}

func TestController_NewClusterRealized(t *testing.T) {
	config := &Config{
		Config:     etcdTestConfig,
		InstanceID: "1",
		Hostname:   "1.ec2.internal",
		GroupName:  "test",
		Instances: map[string]string{
			"1": "1.ec2.internal",
			"2": "2.ec2.internal",
		},
		AvailableMembers: map[string]bool{
			"1": false,
			"2": false,
		},
		ActiveMembers: map[string]string{},
	}

	expected := &RealizedConfig{
		Config:       etcdTestConfig,
		ClusterState: "new",
		InitialCluster: []string{
			"1=https://1.ec2.internal:2379",
			"2=https://2.ec2.internal:2379",
		},
		Name: "1",
		InitialAdvertisePeerURL:   "https://1.ec2.internal:2379",
		InitialAdvertiseClientURL: "https://1.ec2.internal:2380",
		ListenClientURL:           "https://0.0.0.0:2380",
		ListenPeerURL:             "https://0.0.0.0:2379",
	}

	realized := (&Controller{}).getRealizedConfig(config)

	require.Equal(t, expected, realized)

	const expectedVars = `
ETCD_INITIAL_CLUSTER_STATE="new"
ETCD_NAME="1"
ETCD_INITIAL_CLUSTER="1=https://1.ec2.internal:2379,2=https://2.ec2.internal:2379"
ETCD_LISTEN_CLIENT_URLS="https://0.0.0.0:2380"
ETCD_LISTEN_PEER_URLS="https://0.0.0.0:2379"
ETCD_INITIAL_ADVERTISE_PEER_URLS="https://1.ec2.internal:2379"
ETCD_ADVERTISE_CLIENT_URLS="https://1.ec2.internal:2380"
ETCD_TRUSTED_CA_FILE=
ETCD_CERT_FILE=
ETCD_KEY_FILE=
ETCD_CLIENT_CERT_AUTH=true
ETCD_PEER_TRUSTED_CA_FILE=
ETCD_PEER_CERT_FILE=
ETCD_PEER_KEY_FILE=
ETCD_PEER_CLIENT_CERT_AUTH=true
`
	require.Equal(t, expectedVars, string(realized.ConfigVars()))
}

func TestController_NewClusterRemoval(t *testing.T) {
	config := &Config{
		Config:     etcdTestConfig,
		InstanceID: "1",
		Hostname:   "1.ec2.internal",
		GroupName:  "test",
		Instances: map[string]string{
			"1": "1.ec2.internal",
			"2": "2.ec2.internal",
		},
		AvailableMembers: map[string]bool{
			"1": false,
			"2": false,
		},
		ActiveMembers: map[string]string{},
	}

	require.Empty(t, (&Controller{}).getRemovalCandidates(config))
}

func TestController_NewClusterRun(t *testing.T) {
	a := &MockAWS{}
	e := &MockETCD{}

	c := &Controller{
		aws:  a,
		etcd: e,
	}

	a.On("InstanceID").Return("1")
	a.On("Hostname").Return("1.ec2.internal")
	a.On("GroupName").Return("test")
	a.On("GroupInstances").Return(map[string]string{
		"1": "1.ec2.internal",
		"2": "2.ec2.internal",
	}, nil)

	e.On("IsAvailable", "1.ec2.internal").Return(false)
	e.On("IsAvailable", "2.ec2.internal").Return(false)
	e.On("Config").Return(etcdTestConfig)

	err := c.Run()
	require.Nil(t, err)

	const expectedVars = `
ETCD_INITIAL_CLUSTER_STATE="new"
ETCD_NAME="1"
ETCD_INITIAL_CLUSTER="1=https://1.ec2.internal:2379,2=https://2.ec2.internal:2379"
ETCD_LISTEN_CLIENT_URLS="https://0.0.0.0:2380"
ETCD_LISTEN_PEER_URLS="https://0.0.0.0:2379"
ETCD_INITIAL_ADVERTISE_PEER_URLS="https://1.ec2.internal:2379"
ETCD_ADVERTISE_CLIENT_URLS="https://1.ec2.internal:2380"
ETCD_TRUSTED_CA_FILE=
ETCD_CERT_FILE=
ETCD_KEY_FILE=
ETCD_CLIENT_CERT_AUTH=true
ETCD_PEER_TRUSTED_CA_FILE=
ETCD_PEER_CERT_FILE=
ETCD_PEER_KEY_FILE=
ETCD_PEER_CLIENT_CERT_AUTH=true
`
	data, err := ioutil.ReadFile(etcdTestConfig.EnvFile)
	require.Nil(t, err)

	require.Equal(t, expectedVars, string(data))
}

func TestController_ExistingClusterGetConfig(t *testing.T) {
	a := &MockAWS{}
	e := &MockETCD{}

	c := &Controller{
		aws:  a,
		etcd: e,
	}

	a.On("InstanceID").Return("1")
	a.On("Hostname").Return("1.ec2.internal")
	a.On("GroupName").Return("test")
	a.On("GroupInstances").Return(map[string]string{
		"1": "1.ec2.internal",
		"2": "2.ec2.internal",
	}, nil)

	e.On("IsAvailable", "1.ec2.internal").Return(true)
	e.On("IsAvailable", "2.ec2.internal").Return(false)
	e.On("Config").Return(etcdTestConfig)
	e.On("Members", "1.ec2.internal").Return(map[string]string{
		"1": "1.ec2.internal",
	}, nil)
	e.On("Members", "2.ec2.internal").Return(map[string]string{
		"1": "1.ec2.internal",
	}, nil)

	expected := &Config{
		Config:     etcdTestConfig,
		InstanceID: "1",
		Hostname:   "1.ec2.internal",
		GroupName:  "test",
		Instances: map[string]string{
			"1": "1.ec2.internal",
			"2": "2.ec2.internal",
		},
		AvailableMembers: map[string]bool{
			"1": true,
			"2": false,
		},
		ActiveMembers: map[string]string{
			"1": "1.ec2.internal",
		},
	}

	config, err := c.refreshConfig()
	require.Nil(t, err)
	require.Equal(t, expected, config)
}

func TestController_ExistingClusterRealized(t *testing.T) {
	config := &Config{
		Config:     etcdTestConfig,
		InstanceID: "1",
		Hostname:   "1.ec2.internal",
		GroupName:  "test",
		Instances: map[string]string{
			"1": "1.ec2.internal",
			"2": "2.ec2.internal",
			"3": "3.ec2.internal",
		},
		AvailableMembers: map[string]bool{
			"1": true,
			"2": true,
			"3": false,
		},
		ActiveMembers: map[string]string{
			"2": "2.ec2.internal",
		},
	}

	expected := &RealizedConfig{
		Config:       etcdTestConfig,
		ClusterState: "existing",
		InitialCluster: []string{
			"1=https://1.ec2.internal:2379",
			"2=https://2.ec2.internal:2379",
		},
		Name: "1",
		InitialAdvertisePeerURL:   "https://1.ec2.internal:2379",
		InitialAdvertiseClientURL: "https://1.ec2.internal:2380",
		ListenClientURL:           "https://0.0.0.0:2380",
		ListenPeerURL:             "https://0.0.0.0:2379",
	}

	realized := (&Controller{}).getRealizedConfig(config)

	require.Equal(t, expected, realized)

	const expectedVars = `
ETCD_INITIAL_CLUSTER_STATE="existing"
ETCD_NAME="1"
ETCD_INITIAL_CLUSTER="1=https://1.ec2.internal:2379,2=https://2.ec2.internal:2379"
ETCD_LISTEN_CLIENT_URLS="https://0.0.0.0:2380"
ETCD_LISTEN_PEER_URLS="https://0.0.0.0:2379"
ETCD_INITIAL_ADVERTISE_PEER_URLS="https://1.ec2.internal:2379"
ETCD_ADVERTISE_CLIENT_URLS="https://1.ec2.internal:2380"
ETCD_TRUSTED_CA_FILE=
ETCD_CERT_FILE=
ETCD_KEY_FILE=
ETCD_CLIENT_CERT_AUTH=true
ETCD_PEER_TRUSTED_CA_FILE=
ETCD_PEER_CERT_FILE=
ETCD_PEER_KEY_FILE=
ETCD_PEER_CLIENT_CERT_AUTH=true
`
	require.Equal(t, expectedVars, string(realized.ConfigVars()))
}

func TestController_NeedsRemoval(t *testing.T) {
	config := &Config{
		Config:     etcdTestConfig,
		InstanceID: "1",
		Hostname:   "1.ec2.internal",
		GroupName:  "test",
		Instances: map[string]string{
			"1": "1.ec2.internal",
		},
		AvailableMembers: map[string]bool{
			"1": false,
		},
		ActiveMembers: map[string]string{
			"1": "1.ec2.internal",
			"2": "2.ec2.internal",
		},
	}

	require.Equal(t, []string{"2"}, (&Controller{}).getRemovalCandidates(config))
}

func TestController_ExistingClusterRun(t *testing.T) {
	a := &MockAWS{}
	e := &MockETCD{}

	c := &Controller{
		aws:  a,
		etcd: e,
	}

	a.On("InstanceID").Return("1")
	a.On("Hostname").Return("1.ec2.internal")
	a.On("GroupName").Return("test")
	a.On("GroupInstances").Return(map[string]string{
		"1": "1.ec2.internal",
		"2": "2.ec2.internal",
	}, nil)

	e.On("IsAvailable", "1.ec2.internal").Return(false)
	e.On("IsAvailable", "2.ec2.internal").Return(true)
	e.On("Config").Return(etcdTestConfig)
	e.On("Members", "1.ec2.internal").Return(map[string]string{
		"1": "1.ec2.internal",
		"2": "1.ec2.internal",
	}, nil)
	e.On("Members", "2.ec2.internal").Return(map[string]string{
		"1": "1.ec2.internal",
		"2": "2.ec2.internal",
	}, nil)
	e.On("Add", "2.ec2.internal", "1.ec2.internal").Return(nil)

	err := c.Run()
	require.Nil(t, err)

	const expectedVars = `
ETCD_INITIAL_CLUSTER_STATE="existing"
ETCD_NAME="1"
ETCD_INITIAL_CLUSTER="1=https://1.ec2.internal:2379,2=https://2.ec2.internal:2379"
ETCD_LISTEN_CLIENT_URLS="https://0.0.0.0:2380"
ETCD_LISTEN_PEER_URLS="https://0.0.0.0:2379"
ETCD_INITIAL_ADVERTISE_PEER_URLS="https://1.ec2.internal:2379"
ETCD_ADVERTISE_CLIENT_URLS="https://1.ec2.internal:2380"
ETCD_TRUSTED_CA_FILE=
ETCD_CERT_FILE=
ETCD_KEY_FILE=
ETCD_CLIENT_CERT_AUTH=true
ETCD_PEER_TRUSTED_CA_FILE=
ETCD_PEER_CERT_FILE=
ETCD_PEER_KEY_FILE=
ETCD_PEER_CLIENT_CERT_AUTH=true
`
	data, err := ioutil.ReadFile(etcdTestConfig.EnvFile)
	require.Nil(t, err)

	require.Equal(t, expectedVars, string(data))
}

func TestController_ExistingClusterRemovalRun(t *testing.T) {
	a := &MockAWS{}
	e := &MockETCD{}

	c := &Controller{
		aws:  a,
		etcd: e,
	}

	a.On("InstanceID").Return("1")
	a.On("Hostname").Return("1.ec2.internal")
	a.On("GroupName").Return("test")
	a.On("GroupInstances").Return(map[string]string{
		"1": "1.ec2.internal",
		"2": "2.ec2.internal",
	}, nil)

	e.On("IsAvailable", "1.ec2.internal").Return(true)
	e.On("IsAvailable", "2.ec2.internal").Return(true)
	e.On("Config").Return(etcdTestConfig)
	e.On("Members", "1.ec2.internal").Return(map[string]string{
		"1": "1.ec2.internal",
		"2": "2.ec2.internal",
		"3": "3.ec2.internal",
	}, nil)
	e.On("Members", "2.ec2.internal").Return(map[string]string{
		"1": "1.ec2.internal",
		"2": "2.ec2.internal",
	}, nil)
	e.On("Remove", "1.ec2.internal", "3").Return(nil)
	e.On("Remove", "2.ec2.internal", "3").Return(nil)

	err := c.Run()
	require.Nil(t, err)

	const expectedVars = `
ETCD_INITIAL_CLUSTER_STATE="existing"
ETCD_NAME="1"
ETCD_INITIAL_CLUSTER="1=https://1.ec2.internal:2379,2=https://2.ec2.internal:2379"
ETCD_LISTEN_CLIENT_URLS="https://0.0.0.0:2380"
ETCD_LISTEN_PEER_URLS="https://0.0.0.0:2379"
ETCD_INITIAL_ADVERTISE_PEER_URLS="https://1.ec2.internal:2379"
ETCD_ADVERTISE_CLIENT_URLS="https://1.ec2.internal:2380"
ETCD_TRUSTED_CA_FILE=
ETCD_CERT_FILE=
ETCD_KEY_FILE=
ETCD_CLIENT_CERT_AUTH=true
ETCD_PEER_TRUSTED_CA_FILE=
ETCD_PEER_CERT_FILE=
ETCD_PEER_KEY_FILE=
ETCD_PEER_CLIENT_CERT_AUTH=true
`
	data, err := ioutil.ReadFile(etcdTestConfig.EnvFile)
	require.Nil(t, err)

	require.Equal(t, expectedVars, string(data))
}

func TestController_RealizedSSL(t *testing.T) {
	config := &Config{
		Config:     etcd.GetEnvConfig(),
		InstanceID: "1",
		Hostname:   "1.ec2.internal",
		GroupName:  "test",
		Instances: map[string]string{
			"1": "1.ec2.internal",
			"2": "2.ec2.internal",
			"3": "3.ec2.internal",
		},
		AvailableMembers: map[string]bool{
			"1": true,
			"2": true,
			"3": false,
		},
		ActiveMembers: map[string]string{
			"1": "1.ec2.internal",
			"2": "2.ec2.internal",
		},
	}

	expected := &RealizedConfig{
		Config:       etcd.GetEnvConfig(),
		ClusterState: "existing",
		InitialCluster: []string{
			"1=https://1.ec2.internal:2380",
			"2=https://2.ec2.internal:2380",
		},
		Name: "1",
		InitialAdvertisePeerURL:   "https://1.ec2.internal:2380",
		InitialAdvertiseClientURL: "https://1.ec2.internal:2379",
		ListenClientURL:           "https://0.0.0.0:2379",
		ListenPeerURL:             "https://0.0.0.0:2380",
	}

	realized := (&Controller{}).getRealizedConfig(config)

	require.Equal(t, expected, realized)

	const expectedVars = `
ETCD_INITIAL_CLUSTER_STATE="existing"
ETCD_NAME="1"
ETCD_INITIAL_CLUSTER="1=https://1.ec2.internal:2380,2=https://2.ec2.internal:2380"
ETCD_LISTEN_CLIENT_URLS="https://0.0.0.0:2379"
ETCD_LISTEN_PEER_URLS="https://0.0.0.0:2380"
ETCD_INITIAL_ADVERTISE_PEER_URLS="https://1.ec2.internal:2380"
ETCD_ADVERTISE_CLIENT_URLS="https://1.ec2.internal:2379"
ETCD_TRUSTED_CA_FILE=/etc/etcd/certs/ca.pem
ETCD_CERT_FILE=/etc/etcd/certs/etcd.pem
ETCD_KEY_FILE=/etc/etcd/certs/etcd-key.pem
ETCD_CLIENT_CERT_AUTH=true
ETCD_PEER_TRUSTED_CA_FILE=/etc/etcd/certs/peer-ca.pem
ETCD_PEER_CERT_FILE=/etc/etcd/certs/peer-etcd.pem
ETCD_PEER_KEY_FILE=/etc/etcd/certs/peer-etcd-key.pem
ETCD_PEER_CLIENT_CERT_AUTH=true
`
	require.Equal(t, expectedVars, string(realized.ConfigVars()))
}
