variable "namespace" {}

variable "ssh_key" {
  default = "default"
}

variable "min" {
  default = 3
}

variable "max" {
  default = 3
}

variable "desired" {
  default = 3
}

variable "root_volume_size" {
  default = 64
}

variable "instance_type" {
  default = "t2.small"
}

variable "vpc_id" {}

variable "subnet_ids" {
  type = "list"
}

variable "pki_ca_url" {}

variable "pki_etcd_server_key" {}

variable "pki_image" {
  default = "coldog/pki:latest"
}

variable "controller_image" {
  default = "coldog/etcd-aws-cluster:latest"
}
