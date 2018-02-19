provider "aws" {
  region  = "us-west-2"
}

module "etcd" {
  source = "./aws"

  namespace = "default"
  vpc_id    = "vpc-f2b2f696"
  subnet_ids = ["subnet-fef2879a", "subnet-016ce077", "subnet-5214c00a"]
  security_groups = ["sg-cda331b2"]
}
