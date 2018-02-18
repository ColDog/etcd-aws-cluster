resource "tls_private_key" "etcd_peer" {
  algorithm = "RSA"
  rsa_bits  = "2048"
}

resource "tls_cert_request" "etcd_peer" {
  key_algorithm   = "${tls_private_key.etcd_peer.algorithm}"
  private_key_pem = "${tls_private_key.etcd_peer.private_key_pem}"

  subject {
    common_name  = "etcd"
    organization = "etcd"
  }

  dns_names = [
    "*.${data.aws_region.current.name}.compute.internal",
  ]

  ip_addresses = [
    "127.0.0.1",
  ]
}

resource "tls_locally_signed_cert" "etcd_peer" {
  cert_request_pem = "${tls_cert_request.etcd_peer.cert_request_pem}"

  ca_key_algorithm   = "${tls_self_signed_cert.etcd_ca.key_algorithm}"
  ca_private_key_pem = "${tls_private_key.etcd_ca.private_key_pem}"
  ca_cert_pem        = "${tls_self_signed_cert.etcd_ca.cert_pem}"

  validity_period_hours = 87600

  allowed_uses = [
    "key_encipherment",
    "digital_signature",
    "cert_signing",
    "server_auth",
    "client_auth",
  ]
}
