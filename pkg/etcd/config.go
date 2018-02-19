package etcd

import (
	"os"
)

func env(name, defaults string) string {
	val := os.Getenv(name)
	if val == "" {
		return defaults
	}
	return val
}

func GetEnvConfig() Config {
	return Config{
		EnvFile:        env("ETCD_ENV_FILE", "/etc/etcd/config"),
		ClientScheme:   env("ETCD_CLIENT_SCHEME", "https"),
		ClientPort:     env("ETCD_CLIENT_PORT", "2379"),
		ClientCAFile:   env("ETCD_CLIENT_CA_FILE", "/etc/etcd/certs/ca.pem"),
		ClientCertFile: env("ETCD_CLIENT_CERT_FILE", "/etc/etcd/certs/etcd.pem"),
		ClientKeyFile:  env("ETCD_CLIENT_KEY_FILE", "/etc/etcd/certs/etcd-key.pem"),
		PeerScheme:     env("ETCD_PEER_SCHEME", "https"),
		PeerPort:       env("ETCD_PEER_PORT", "2380"),
		PeerCAFile:     env("ETCD_PEER_CA_FILE", "/etc/etcd/certs/peer-ca.pem"),
		PeerCertFile:   env("ETCD_PEER_CERT_FILE", "/etc/etcd/certs/peer-etcd.pem"),
		PeerKeyFile:    env("ETCD_PEER_KEY_FILE", "/etc/etcd/certs/peer-etcd-key.pem"),
	}
}
