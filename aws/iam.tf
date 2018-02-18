resource "aws_iam_instance_profile" "etcd_profile" {
  name = "${var.namespace}_etcd_profile"
  role = "${aws_iam_role.etcd_role.name}"
}

resource "aws_iam_role" "etcd_role" {
  name = "${var.namespace}_etcd_role"
  path = "/"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "EtcdRole",
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "etcd_policy" {
  name = "${var.namespace}_etcd_policy"
  role = "${aws_iam_role.etcd_role.id}"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "EnableLogs",
      "Action": "logs:*",
      "Resource": "*",
      "Effect": "Allow"
    },
    {
      "Sid": "AutoscalingDescribe",
      "Action": "autoscaling:Describe*",
      "Resource": "*",
      "Effect": "Allow"
    },
    {
      "Sid": "EC2Describe",
      "Action": "ec2:Describe*",
      "Resource": "*",
      "Effect": "Allow"
    }
  ]
}
EOF
}
