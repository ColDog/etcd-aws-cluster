data "ignition_config" "etcd" {
  systemd = [
    "${data.ignition_systemd_unit.locksmithd.id}",
    "${data.ignition_systemd_unit.etcd.id}",
    "${data.ignition_systemd_unit.etcd_config.id}",
    "${data.ignition_systemd_unit.etcd_watcherd.id}",
    "${data.ignition_systemd_unit.etcd_server_cert.id}",
    "${data.ignition_systemd_unit.etcd_peer_cert.id}",
    "${data.ignition_systemd_unit.etcd_backup.id}",
    "${data.ignition_systemd_unit.etcd_backup_timer.id}",
    "${data.ignition_systemd_unit.etcd_server_cert_timer.id}",
    "${data.ignition_systemd_unit.etcd_peer_cert_timer.id}",
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
EnvironmentFile=/run/metadata/coreos
Environment="ETCD_UID=232"
Type=oneshot
ExecStartPre=-/usr/bin/docker pull ${var.pki_image}
ExecStartPre=/usr/bin/mkdir -p /etc/etcd/certs
ExecStartPost=/usr/bin/chown -R etcd:etcd /etc/etcd/certs
ExecStart=/usr/bin/docker run --rm -i \
  --user $${ETCD_UID} \
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
EnvironmentFile=/run/metadata/coreos
Environment="ETCD_UID=232"
Type=oneshot
ExecStartPre=-/usr/bin/docker pull ${var.pki_image}
ExecStartPre=/usr/bin/mkdir -p /etc/etcd/certs
ExecStartPost=/usr/bin/chown -R etcd:etcd /etc/etcd/certs
ExecStart=/usr/bin/docker run --rm -i \
  --user $${ETCD_UID} \
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
Requires=etcd-peer-cert.service
Requires=etcd-server-cert.service
After=etcd-peer-cert.service
After=etcd-server-cert.service

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
Description=ETCDConfigWatcherd
Requires=etcd-peer-cert.service
Requires=etcd-server-cert.service
After=etcd-peer-cert.service
After=etcd-server-cert.service

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
ETCD_CLIENT_CERT_FILE=/etc/etcd/certs/etcd-server.pem
ETCD_CLIENT_KEY_FILE=/etc/etcd/certs/etcd-server-key.pem
ETCD_PEER_SCHEME=https
ETCD_PEER_PORT=2380
ETCD_PEER_CA_FILE=/etc/etcd/certs/etcd-peer-ca.pem
ETCD_PEER_CERT_FILE=/etc/etcd/certs/etcd-peer.pem
ETCD_PEER_KEY_FILE=/etc/etcd/certs/etcd-peer-key.pem
EOF
  }
}

data "ignition_systemd_unit" "etcd_backup" {
  name    = "etcd-backup.service"
  enabled = true

  content = <<EOF
[Unit]
Description=ETCDBackup

[Service]
Type=oneshot

EnvironmentFile=/run/metadata/coreos
Environment="ETCD_DATA_DIR=/var/lib/etcd"
Environment="ETCD_BACKUP_DIR=/var/lib/etcd-backup"
Environment="S3_PATH=s3://${aws_s3_bucket.etcd.bucket}"
Environment="BACKUP_FILE=/tmp/etcd-backup.tar.gz"

ExecStartPre=/usr/bin/rm -rf $${ETCD_BACKUP_DIR}
ExecStartPre=/usr/bin/mkdir -p $${ETCD_BACKUP_DIR}/member/snap
ExecStartPre=/usr/bin/echo ETCD_DATA_DIR: $${ETCD_DATA_DIR}
ExecStartPre=/usr/bin/echo ETCD_BACKUP_DIR: $${ETCD_BACKUP_DIR}
ExecStartPre=/usr/bin/etcdctl backup --data-dir=$${ETCD_DATA_DIR} --backup-dir=$${ETCD_BACKUP_DIR}
ExecStartPre=/usr/bin/touch $${ETCD_BACKUP_DIR}/member/snap/backup
ExecStartPre=/usr/bin/tar tar -zcvf $${BACKUP_FILE} -C $${ETCD_BACKUP_DIR} .

ExecStart=/usr/bin/docker run --rm -v /tmp:/tmp mesosphere/aws-cli s3 cp $${BACKUP_FILE} $${S3_PATH}/backups/$${COREOS_EC2_INSTANCE_ID}.tar.gz
EOF
}

data "ignition_systemd_unit" "etcd_backup_timer" {
  name    = "etcd-backup.timer"
  enabled = true

  content = <<EOF

[Unit]
Description=ETCDBackupTimer

[Timer]
OnBootSec=1min
OnUnitActiveSec=5min

[Install]
WantedBy=timers.target
EOF
}

data "ignition_systemd_unit" "etcd_peer_cert_timer" {
  name    = "etcd-peer-cert.timer"
  enabled = true

  content = <<EOF

[Unit]
Description=ETCDPeerCertTimer

[Timer]
OnActiveSec=7d

[Install]
WantedBy=timers.target
EOF
}

data "ignition_systemd_unit" "etcd_server_cert_timer" {
  name    = "etcd-server-cert.timer"
  enabled = true

  content = <<EOF

[Unit]
Description=ETCDServerCertTimer

[Timer]
OnActiveSec=7d

[Install]
WantedBy=timers.target
EOF
}
