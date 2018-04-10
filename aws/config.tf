data "ignition_config" "etcd" {
  systemd = [
    "${data.ignition_systemd_unit.locksmithd.id}",
    "${data.ignition_systemd_unit.etcd.id}",
    "${data.ignition_systemd_unit.etcd_config.id}",
    "${data.ignition_systemd_unit.etcd_watcherd.id}",
    "${data.ignition_systemd_unit.etcd_server_cert.id}",
    "${data.ignition_systemd_unit.etcd_peer_cert.id}",
  ]

  files = [
    "${data.ignition_file.config.id}",
  ]
}

data "ignition_systemd_unit" "locksmithd" {
  name    = "locksmithd.service"
  enabled = true

  dropin = [
    {
      name = "40-etcd-lock.conf"

      content = <<EOF
[Service]
Environment=REBOOT_STRATEGY=etcd-lock
Environment="LOCKSMITHD_ETCD_CAFILE=/etc/etcd/certs/etcd-server-ca.pem"
Environment="LOCKSMITHD_ETCD_CERTFILE=/etc/etcd/certs/etcd-server.pem"
Environment="LOCKSMITHD_ETCD_KEYFILE=/etc/etcd/certs/etcd-server-key.pem"
Environment="LOCKSMITHD_ENDPOINT=https://127.0.0.1:2379"
EOF
    },
  ]
}

data "ignition_systemd_unit" "etcd" {
  name    = "etcd-member.service"
  enabled = true

  dropin = [
    {
      name = "20-clct-etcd-member.conf"

      content = <<EOF
[Unit]
After=etcd-config.service

[Service]
EnvironmentFile=/etc/etcd/config
Environment="RKT_RUN_ARGS=--volume etc-etcd,kind=host,source=/etc/etcd,readOnly=true --mount volume=etc-etcd,target=/etc/etcd"
EOF
    },
  ]
}

data "ignition_systemd_unit" "etcd_server_cert" {
  name    = "etcd-server-cert.service"
  enabled = true

  content = <<EOF
[Unit]
Description=ETCDServerCerts
Requires=coreos-metadata.service
After=coreos-metadata.service

[Service]
Type=oneshot
ExecStartPre=-/usr/bin/docker pull ${var.pki_image}
ExecStart=/usr/bin/docker run --rm -i \
  -e ETCD_SERVER_KEY=${var.pki_etcd_server_key} \
  -e CA_URL=${var.pki_ca_url} \
  -e INSTANCE_IP=$${COREOS_EC2_IPV4_LOCAL} \
  -e INSTANCE_ID=$${COREOS_EC2_INSTANCE_ID} \
  -e INSTANCE_HOSTNAME=$${COREOS_EC2_HOSTNAME} \
  -v /etc/etcd/certs:/certs \
  ${var.pki_image} gencert etcd server
RemainAfterExit=true

[Install]
WantedBy=multi-user.target
EOF
}

data "ignition_systemd_unit" "etcd_peer_cert" {
  name    = "etcd-peer-cert.service"
  enabled = true

  content = <<EOF
[Unit]
Description=ETCDPeerCerts
Requires=coreos-metadata.service
After=coreos-metadata.service

[Service]
Type=oneshot
ExecStartPre=-/usr/bin/docker pull ${var.pki_image}
ExecStart=/usr/bin/docker run --rm -i \
  -e ETCD_SERVER_KEY=${var.pki_etcd_server_key} \
  -e CA_URL=${var.pki_ca_url} \
  -e INSTANCE_IP=$${COREOS_EC2_IPV4_LOCAL} \
  -e INSTANCE_ID=$${COREOS_EC2_INSTANCE_ID} \
  -e INSTANCE_HOSTNAME=$${COREOS_EC2_HOSTNAME} \
  -v /etc/etcd/certs:/certs \
  ${var.pki_image} gencert etcd peer
RemainAfterExit=true

[Install]
WantedBy=multi-user.target
EOF
}

data "ignition_systemd_unit" "etcd_config" {
  name    = "etcd-config.service"
  enabled = true

  content = <<EOF
[Unit]
Description=ETCDConfig

[Service]
Type=oneshot
ExecStartPre=-/usr/bin/docker pull ${var.controller_image}
ExecStart=/usr/bin/docker run --rm \
  --env-file /etc/etcd/config \
  -v /etc/etcd/:/etc/etcd/ \
  ${var.controller_image} \
  /bin/etcd-config
RemainAfterExit=true

[Install]
WantedBy=multi-user.target
EOF
}

data "ignition_systemd_unit" "etcd_watcherd" {
  name    = "etcd-watcherd.service"
  enabled = true

  content = <<EOF
[Unit]
Description=Configure etcd configuration file.

[Service]
ExecStartPre=-/usr/bin/docker pull ${var.controller_image}
ExecStart=/usr/bin/docker run --rm \
  --env-file /etc/etcd/config \
  -v /etc/etcd/:/etc/etcd/ \
  ${var.controller_image} \
  /bin/etcd-watcherd
Restart=on-failure
RestartSec=30

[Install]
WantedBy=multi-user.target
EOF
}

data "ignition_file" "config" {
  path       = "/etc/etcd/config"
  mode       = 0644
  filesystem = "root"

  content {
    content = <<EOF
ETCD_ENV_FILE=/etc/etcd/config
ETCD_CLIENT_SCHEME=https
ETCD_CLIENT_PORT=2379
ETCD_CLIENT_CA_FILE=/etc/etcd/certs/etcd-server-ca.pem
ETCD_CLIENT_CERT_FILE=etc/etcd/certs/etcd-server.pem
ETCD_CLIENT_KEY_FILE=etc/etcd/certs/etcd-server-key.pem
ETCD_PEER_SCHEME=https
ETCD_PEER_PORT=2380
ETCD_PEER_CA_FILE=/etc/etcd/certs/etcd-peer-ca.pem
ETCD_PEER_CERT_FILE=/etc/etcd/certs/etcd-peer.pem
ETCD_PEER_KEY_FILE=/etc/etcd/certs/etcd-peer-key.pem
EOF
  }
}
