package main

import (
	"flag"
	"log"
	"time"

	"github.com/coldog/etcd-aws-cluster/pkg/aws"
	"github.com/coldog/etcd-aws-cluster/pkg/controller"
	"github.com/coldog/etcd-aws-cluster/pkg/etcd"
)

func main() {
	var (
		interval = "5m"
		watch    = false
	)
	flag.StringVar(&interval, "interval", interval, "Watch interval")
	flag.BoolVar(&watch, "watch", watch, "Watch will poll the autoscaling group and continuously write to the configured file")
	flag.Parse()

	etcdClient, err := etcd.NewClient(etcd.GetEnvConfig())
	if err != nil {
		log.Fatalf("failed to init etcd client: %v", err)
	}

	awsClient, err := aws.NewClient()
	if err != nil {
		log.Fatalf("failed to init aws client: %v", err)
	}

	ctrl := controller.NewController(awsClient, etcdClient)

	if watch {
		intervalTime, iErr := time.ParseDuration(interval)
		if iErr != nil {
			log.Fatalf("failed to parse interval (%s): %v", interval, iErr)
		}
		ctrl.Watch(intervalTime)
		return
	}

	err = ctrl.Run()
	if err != nil {
		log.Fatalf("run failed: %v", err)
	}
}
