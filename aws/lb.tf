resource "aws_lb" "etcd" {
  name                             = "${var.namespace}-etcd"
  subnets                          = ["${var.subnet_ids}"]
  internal                         = true
  enable_cross_zone_load_balancing = true
  ip_address_type                  = "ipv4"
  load_balancer_type               = "network"
  idle_timeout                     = 3600
}

resource "aws_lb_listener" "etcd" {
  load_balancer_arn = "${aws_lb.etcd.arn}"
  port              = 2379
  protocol          = "TCP"

  default_action {
    target_group_arn = "${aws_lb_target_group.etcd.arn}"
    type             = "forward"
  }
}

resource "aws_lb_target_group" "etcd" {
  name     = "${var.namespace}-etcd"
  port     = 2379
  protocol = "TCP"
  vpc_id   = "${var.vpc_id}"

  health_check {
    port     = 2379
    protocol = "TCP"
  }
}
