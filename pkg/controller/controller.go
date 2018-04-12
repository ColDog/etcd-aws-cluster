package controller

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"text/template"
	"time"

	"github.com/coldog/etcd-aws-cluster/pkg/aws"
	"github.com/coldog/etcd-aws-cluster/pkg/etcd"
)

type Config struct {
	etcd.Config

	InstanceID   string
	InstanceHost string
	GroupName    string

	Instances        map[string]string
	AvailableMembers map[string]bool
	ActiveMembers    map[string]string
}

func (cfg *Config) AnyAvailable() bool {
	for _, avail := range cfg.AvailableMembers {
		if avail {
			return true
		}
	}
	return false
}

func (cfg *Config) AnyAvailableHost() string {
	for id, avail := range cfg.AvailableMembers {
		if avail {
			return cfg.Instances[id]
		}
	}
	return ""
}

type RealizedConfig struct {
	etcd.Config

	ClusterState              string
	InitialCluster            []string
	Name                      string
	InitialAdvertisePeerURL   string
	InitialAdvertiseClientURL string
	ListenClientURL           string
	ListenPeerURL             string
}

func (r *RealizedConfig) ConfigVars() []byte {
	b := bytes.NewBuffer(nil)
	const tpl = `
ETCD_INITIAL_CLUSTER_STATE="{{.ClusterState}}"
ETCD_NAME="{{.Name}}"
ETCD_INITIAL_CLUSTER="{{range $i, $el := .InitialCluster}}{{if $i}},{{end}}{{$el}}{{end}}"
ETCD_LISTEN_CLIENT_URLS="{{.ListenClientURL}}"
ETCD_LISTEN_PEER_URLS="{{.ListenPeerURL}}"
ETCD_INITIAL_ADVERTISE_PEER_URLS="{{.InitialAdvertisePeerURL}}"
ETCD_ADVERTISE_CLIENT_URLS="{{.InitialAdvertiseClientURL}}"
ETCD_TRUSTED_CA_FILE={{.ClientCAFile}}
ETCD_CERT_FILE={{.ClientCertFile}}
ETCD_KEY_FILE={{.ClientKeyFile}}
ETCD_CLIENT_CERT_AUTH={{eq .ClientScheme "https"}}
ETCD_PEER_TRUSTED_CA_FILE={{.PeerCAFile}}
ETCD_PEER_CERT_FILE={{.PeerCertFile}}
ETCD_PEER_KEY_FILE={{.PeerKeyFile}}
ETCD_PEER_CLIENT_CERT_AUTH={{eq .PeerScheme "https"}}
`
	err := template.Must(template.New("").Parse(tpl)).Execute(b, r)
	if err != nil {
		panic(err)
	}
	return b.Bytes()
}

func NewController(a aws.Client, e etcd.Client) *Controller {
	return &Controller{aws: a, etcd: e}
}

type Controller struct {
	aws  aws.Client
	etcd etcd.Client
}

func (c *Controller) refreshConfig() (*Config, error) {
	instances, err := c.aws.GroupInstances()
	if err != nil {
		return nil, err
	}

	availableMembers := map[string]bool{}
	activeMembers := map[string]string{}
	for id, host := range instances {
		available := c.etcd.IsAvailable(host)
		availableMembers[id] = available

		if available {
			membs, err := c.etcd.Members(host)
			if err != nil {
				continue
			}
			for id, host := range membs {
				activeMembers[id] = host
			}
		}
	}

	next := &Config{
		Config:           c.etcd.Config(),
		InstanceID:       c.aws.InstanceID(),
		GroupName:        c.aws.GroupName(),
		InstanceHost:     c.aws.IP(),
		Instances:        instances,
		AvailableMembers: availableMembers,
		ActiveMembers:    activeMembers,
	}
	return next, nil
}

func (c *Controller) getRealizedConfig(config *Config) *RealizedConfig {
	realized := &RealizedConfig{
		Config:                    config.Config,
		Name:                      config.InstanceID,
		ListenClientURL:           config.ClientURL("0.0.0.0"),
		ListenPeerURL:             config.PeerURL("0.0.0.0"),
		InitialAdvertiseClientURL: config.ClientURL(config.InstanceHost),
		InitialAdvertisePeerURL:   config.PeerURL(config.InstanceHost),
	}

	// If any are available, join an existing cluster.
	if config.AnyAvailable() {
		members := map[string]string{}
		for k, v := range config.ActiveMembers {
			members[k] = v
		}
		members[config.InstanceID] = config.InstanceHost
		realized.ClusterState = "existing"
		realized.InitialCluster = config.PeerURLs(members)
	} else {
		realized.ClusterState = "new"
		realized.InitialCluster = config.PeerURLs(config.Instances)
	}
	return realized
}

func (c *Controller) getRemovalCandidates(config *Config) (out []string) {
	for id := range config.ActiveMembers {
		if _, ok := config.Instances[id]; !ok {
			out = append(out, id)
		}
	}
	return
}

func (c *Controller) Run() error {
	config, err := c.refreshConfig()
	if err != nil {
		return err
	}

	log.Printf("starting run")
	logConfig(config)

	configFile := c.etcd.Config().EnvFile

	if config.AnyAvailable() {
		toRemove := c.getRemovalCandidates(config)
		for _, id := range toRemove {
			log.Printf("removing etcd node: %s", id)
			err = c.etcd.Remove(config.AnyAvailableHost(), id)
			if err != nil {
				return err
			}
			delete(config.ActiveMembers, id)
		}
	}

	log.Println("finding realized config")
	realized := c.getRealizedConfig(config)
	logConfig(realized)

	if config.AnyAvailable() && !config.AvailableMembers[config.InstanceID] {
		log.Printf("adding self to cluster: %s", config.InstanceHost)
		err = c.etcd.Add(config.AnyAvailableHost(), config.InstanceHost)
		if err != nil {
			return err
		}
	}

	log.Printf("writing config: %s", configFile)
	return ioutil.WriteFile(configFile, realized.ConfigVars(), 0700)
}

func (c *Controller) Watch(interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for range t.C {
		err := c.Run()
		if err != nil {
			log.Printf("run failed: %v", err)
		}
	}
}

func logConfig(c interface{}) {
	out, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		panic(err)
	}
	log.Printf("config: %s\n", string(out))
}
