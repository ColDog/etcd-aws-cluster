package main

import (
	"log"

	"github.com/coldog/etcd-aws-cluster/pkg/aws"
	"github.com/coldog/etcd-aws-cluster/pkg/controller"
	"github.com/coldog/etcd-aws-cluster/pkg/etcd"
)

func main() {
	etcdClient, err := etcd.NewClient(etcd.GetEnvConfig())
	if err != nil {
		log.Fatalf("failed to init etcd client: %v", err)
	}

	awsClient, err := aws.NewClient()
	if err != nil {
		log.Fatalf("failed to init aws client: %v", err)
	}

	err = controller.NewController(awsClient, etcdClient).Run()
	if err != nil {
		log.Fatalf("config failed: %v", err)
	}
}
