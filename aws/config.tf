data "ignition_config" "etcd" {
  systemd = [
    "${data.ignition_systemd_unit.locksmithd.id}",
    "${data.ignition_systemd_unit.etcd.id}",
    "${data.ignition_systemd_unit.etcd_config.id}",
    "${data.ignition_systemd_unit.etcd_watcherd.id}",
  ]

  files = [
    "${data.ignition_file.etcd_ca.id}",
    "${data.ignition_file.etcd_cert.id}",
    "${data.ignition_file.etcd_key.id}",
    "${data.ignition_file.etcd_peer_ca.id}",
    "${data.ignition_file.etcd_peer_cert.id}",
    "${data.ignition_file.etcd_peer_key.id}",
  ]
}

data "ignition_systemd_unit" "locksmithd" {
  name   = "locksmithd.service"
  enabled = true

  dropin = [
    {
      name    = "40-etcd-lock.conf"
      content = <<EOF
[Service]
Environment=REBOOT_STRATEGY=etcd-lock
Environment="LOCKSMITHD_ETCD_CAFILE=/etc/etcd/certs/ca.pem"
Environment="LOCKSMITHD_ETCD_CERTFILE=/etc/etcd/certs/etcd.pem"
Environment="LOCKSMITHD_ETCD_KEYFILE=/etc/etcd/certs/etcd-key.pem"
Environment="LOCKSMITHD_ENDPOINT=https://127.0.0.1:2379"
EOF
    },
  ]
}

data "ignition_systemd_unit" "etcd" {
  name   = "etcd-member.service"
  enabled = true

  dropin = [
    {
      name    = "20-clct-etcd-member.conf"
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

data "ignition_systemd_unit" "etcd_config" {
  name    = "etcd-config.service"
  enabled = true

  content = <<EOF
[Unit]
Description=Configure etcd configuration file.

[Service]
Type=oneshot
ExecStartPre=-/usr/bin/docker pull coldog/etcd-aws-cluster:latest
ExecStart=/usr/bin/docker run --rm \
  -e ETCD_CLIENT_SCHEME=https \
  -e ETCD_PEER_SCHEME=https \
  -v /etc/etcd/:/etc/etcd/ \
  coldog/etcd-aws-cluster:latest \
  /bin/etcd-config
RemainAfterExit=true
Restart=on-failure

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
ExecStartPre=-/usr/bin/docker pull coldog/etcd-aws-cluster:latest
ExecStart=/usr/bin/docker run --rm \
  -e ETCD_CLIENT_SCHEME=https \
  -e ETCD_PEER_SCHEME=https \
  -v /etc/etcd/:/etc/etcd/ \
  coldog/etcd-aws-cluster:latest \
  /bin/etcd-watcherd
Restart=on-failure
RestartSec=30

[Install]
WantedBy=multi-user.target
EOF
}

data "ignition_file" "etcd_ca" {
  path       = "/etc/etcd/certs/ca.pem"
  mode       = 0644
  filesystem = "root"

  content {
    content = "${tls_self_signed_cert.etcd_ca.cert_pem}"
  }
}

data "ignition_file" "etcd_cert" {
  path       = "/etc/etcd/certs/etcd.pem"
  mode       = 0644
  filesystem = "root"

  content {
    content = "${tls_locally_signed_cert.etcd.cert_pem}"
  }
}

data "ignition_file" "etcd_key" {
  path       = "/etc/etcd/certs/etcd-key.pem"
  mode       = 0644
  filesystem = "root"

  content {
    content = "${tls_private_key.etcd.private_key_pem}"
  }
}

data "ignition_file" "etcd_peer_ca" {
  path       = "/etc/etcd/certs/peer-ca.pem"
  mode       = 0644
  filesystem = "root"

  content {
    content = "${tls_self_signed_cert.etcd_ca.cert_pem}"
  }
}

data "ignition_file" "etcd_peer_cert" {
  path       = "/etc/etcd/certs/peer-etcd.pem"
  mode       = 0644
  filesystem = "root"

  content {
    content = "${tls_locally_signed_cert.etcd_peer.cert_pem}"
  }
}

data "ignition_file" "etcd_peer_key" {
  path       = "/etc/etcd/certs/peer-etcd-key.pem"
  mode       = 0644
  filesystem = "root"

  content {
    content = "${tls_private_key.etcd_peer.private_key_pem}"
  }
}
