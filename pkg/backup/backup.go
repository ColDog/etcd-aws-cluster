package backup

import (
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/coldog/etcd-aws-cluster/pkg/aws"
)

var sh = func(cmd string, args ...string) error {
	return exec.Command(cmd, args...).Run()
}

type Config struct {
	NodeID string
	Bucket string
	Prefix string
}

func createBackup(id string) (string, string, error) {
	t := time.Now()
	backupID := "etcd-" + id + "-" + t.Format("2006-01-02T15-04-05Z07-00")
	file := "/var/lib/etcd-backup/" + backupID + ".tar.gz"

	if err := os.MkdirAll("/var/lib/etcd-backup/backup", 0700); err != nil {
		return "", "", err
	}
	if err := sh("/bin/etcdctl", "backup", "--data-dir", "/var/lib/etcd",
		"--backup-dir", "/var/lib/etcd-backup/backup"); err != nil {
		return "", "", err
	}
	if err := sh("/bin/tar", "-C", "/var/lib/etcd-backup", "-zcf",
		file, "."); err != nil {
		return "", "", err
	}
	return backupID, file, nil
}

func Backup(s3 aws.Client, c Config) error {
	backupID, filename, err := createBackup(c.NodeID)
	if err != nil {
		return err
	}
	return s3.Upload(filename, c.Bucket, c.Prefix+"/"+backupID)
}

func Watch(s3 aws.Client, c Config, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for range t.C {
		err := Backup(s3, c)
		if err != nil {
			log.Printf("backup failed: %v", err)
		}
	}
}
