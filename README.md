# ETCD AWS Cluster

This container serves to assist in the creation of an etcd (2.x) cluster from an AWS auto scaling group. It writes a file to /etc/etcd/config that contains configuration parameters for using ETCD.

This file can then be loaded as an `EnvironmentFile` in a drop in:

```bash
[Service]
EnvironmentFile=/etc/etcd/config
```

## Usage

Expected usage is to run a container once that will populate the environment file and then also run a second container that will continuously re-populate the configuration in case ETCD is restarted.

Additionally, the process that watches will remove nodes from ETCD that have been terminated by the autoscaling group.

```shell
docker run --rm \
  -e ETCD_CLIENT_SCHEME=https \
  -e ETCD_PEER_SCHEME=https \
  -v /etc/etcd/:/etc/etcd/ \
  coldog/etcd-aws-cluster:latest \
  -watch -interval 1m
```

### Environment Variables

Environment variables used for configuration with preset defaults.

```shell
# This is the file that the environment variables will be written to.
ETCD_ENV_FILE=/etc/etcd/config

# If the client scheme is set to `https` then the certs variables are expected
# to be set.
ETCD_CLIENT_SCHEME=https
ETCD_CLIENT_PORT=2379
ETCD_CLIENT_CA_FILE=/etc/etcd/certs/ca.pem
ETCD_CLIENT_CERT_FILE=/etc/etcd/certs/etcd.pem
ETCD_CLIENT_KEY_FILE=/etc/etcd/certs/etcd-key.pem

# Peer configuration.
ETCD_PEER_SCHEME=https
ETCD_PEER_PORT=2380
ETCD_PEER_CA_FILE=/etc/etcd/certs/peer-ca.pem
ETCD_PEER_CERT_FILE=/etc/etcd/certs/peer-etcd.pem
ETCD_PEER_KEY_FILE=/etc/etcd/certs/peer-etcd-key.pem
```

## Flags

- `-watch`: Configures whether the process should poll every interval or whether it should run once and exit.
- `-interval`: Configures the interval to poll for new updates from the autoscaling group for.

## Output

The following variables will be written to the output file:

- `ETCD_INITIAL_CLUSTER_STATE`: "new" or "existing".
- `ETCD_NAME`: The ID assigned by AWS to this instance.
- `ETCD_INITIAL_CLUSTER`: Initial cluster configuration. These are all nodes in the cluster including the new node.
- `ETCD_LISTEN_CLIENT_URLS`: This is computed by `<Scheme>://0.0.0.0:<ClientPort>`.
- `ETCD_LISTEN_PEER_URLS`: This is computed by `<Scheme>://0.0.0.0:<PeerPort>`.
- `ETCD_INITIAL_ADVERTISE_PEER_URLS`: This is computed by `<Scheme>://<Hostname>:<PeerPort>`.
- `ETCD_ADVERTISE_CLIENT_URLS`: This is computed by `<Scheme>://<Hostname>:<ClientPort>`.
- `ETCD_TRUSTED_CA_FILE`: This is passed through from the input configuration.
- `ETCD_CERT_FILE`: This is passed through from the input configuration.
- `ETCD_KEY_FILE`: This is passed through from the input configuration.
- `ETCD_CLIENT_CERT_AUTH`: This is passed through from the input configuration.
- `ETCD_PEER_TRUSTED_CA_FILE`: This is passed through from the input configuration.
- `ETCD_PEER_CERT_FILE`: This is passed through from the input configuration.
- `ETCD_PEER_KEY_FILE`: : This is passed through from the input configuration.
- `ETCD_PEER_CLIENT_CERT_AUTH`: This is passed through from the input configuration.
