package etcd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"time"

	etcd "github.com/coreos/etcd/client"
)

type connectFunc = func(url string) (etcd.MembersAPI, error)

func connector(tp etcd.CancelableTransport) connectFunc {
	return func(url string) (etcd.MembersAPI, error) {
		cl, err := etcd.New(etcd.Config{
			Endpoints: []string{url},
			Transport: tp,
		})
		if err != nil {
			return nil, err
		}
		return etcd.NewMembersAPI(cl), nil
	}
}

type Client interface {
	Config() Config
	Add(clientHostname, candidateHostname string) error
	Remove(clientHostname, candidateHostname string) error
	IsAvailable(hostname string) bool
	Members(hostname string) (map[string]string, error)
}

type Config struct {
	EnvFile        string
	ClientScheme   string
	ClientCertFile string
	ClientCAFile   string
	ClientKeyFile  string
	ClientPort     string
	PeerScheme     string
	PeerCertFile   string
	PeerCAFile     string
	PeerKeyFile    string
	PeerPort       string
}

func (c Config) PeerURL(hostname string) string {
	return c.PeerScheme + "://" + hostname + ":" + c.PeerPort
}

func (c Config) ClientURL(hostname string) string {
	return c.ClientScheme + "://" + hostname + ":" + c.ClientPort
}

func (c Config) PeerURLs(m map[string]string) (urls []string) {
	for id, hostname := range m {
		urls = append(urls, id+"="+c.PeerURL(hostname))
	}
	sort.Strings(urls) // For testing repeatability.
	return
}

func NewClient(c Config) (Client, error) {
	tp := http.DefaultTransport.(*http.Transport)
	if c.ClientScheme == "https" {
		var err error
		tp, err = transport(c.ClientCertFile, c.ClientKeyFile, c.ClientCAFile)
		if err != nil {
			return nil, err
		}
	}
	return &client{
		config:  c,
		connect: connector(tp),
	}, nil
}

type client struct {
	config  Config
	connect connectFunc
}

func (c *client) Config() Config { return c.config }

func (c *client) Add(clientHostname, candidateHostname string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clientURL := c.config.ClientURL(clientHostname)
	candidateURL := c.config.PeerURL(candidateHostname)

	api, err := c.connect(clientURL)
	if err != nil {
		return err
	}
	_, err = api.Add(ctx, candidateURL)
	return err
}

func (c *client) Remove(clientHostname, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clientURL := c.config.ClientURL(clientHostname)

	api, err := c.connect(clientURL)
	if err != nil {
		return err
	}

	var id string
	membs, err := api.List(ctx)
	if err != nil {
		return err
	}
	for _, m := range membs {
		if m.Name == name {
			id = m.ID
			break
		}
	}
	return api.Remove(ctx, id)
}

func (c *client) IsAvailable(hostname string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clientURL := c.config.ClientURL(hostname)

	for i := 0; i < 10; i++ {
		api, err := c.connect(clientURL)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		_, err = api.List(ctx)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		return true
	}
	return false
}

func (c *client) Members(hostname string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clientURL := c.config.ClientURL(hostname)

	api, err := c.connect(clientURL)
	if err != nil {
		return nil, err
	}
	l, err := api.List(ctx)
	if err != nil {
		return nil, err
	}

	membs := map[string]string{}
	for _, m := range l {
		if len(m.ClientURLs) == 0 {
			continue
		}
		u, err := url.Parse(m.ClientURLs[0])
		if err != nil {
			return nil, err
		}
		membs[m.Name] = u.Hostname()
	}
	return membs, nil
}

func transport(certFile, keyFile, caFile string) (*http.Transport, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	return transport, nil
}
