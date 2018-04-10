data "aws_vpc" "main" {
  id = "${var.vpc_id}"
}

resource "aws_security_group" "etcd" {
  name        = "${var.namespace}-etcd"
  description = "Internal etcd security group."
  vpc_id      = "${var.vpc_id}"

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["${data.aws_vpc.main.cidr_block}"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
