resource "aws_security_group" "etcd" {
  name        = "${var.namespace}_etcd"
  description = "Internal etcd security group."
  vpc_id      = "${var.vpc_id}"

  ingress {
    from_port       = 0
    to_port         = 0
    protocol        = "-1"
    security_groups = ["${var.security_groups}"]
  }

  ingress {
    from_port = 0
    to_port   = 0
    protocol  = "-1"
    self      = true
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    KubernetesCluster                           = "${var.namespace}"
    "kubernetes.io/cluster/${var.namespace}" = "etcd"
  }
}
