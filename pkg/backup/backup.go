package backup

import (
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/coldog/etcd-aws-cluster/pkg/aws"
)

var (
	backupRoot = "/var/lib/etcd-backup"
	backupDir  = backupRoot + "/backup"
)

var sh = func(cmd string, args ...string) error {
	return exec.Command(cmd, args...).Run()
}

type Config struct {
	NodeID string
	Bucket string
	Prefix string
}

func createBackup(id string, ts time.Time) (string, string, error) {
	backupID := "etcd-" + id + "-" + ts.Format("2006-01-02T15-04-05Z07-00")
	file := backupRoot + "/" + backupID + ".tar.gz"

	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return "", "", err
	}
	if err := sh("/bin/etcdctl", "backup", "--data-dir", "/var/lib/etcd",
		"--backup-dir", backupDir); err != nil {
		return "", "", err
	}
	if err := sh("/bin/tar", "-C", backupRoot, "-zcf",
		file, "."); err != nil {
		return "", "", err
	}
	return backupID, file, nil
}

func Backup(s3 aws.Client, c Config, ts time.Time) error {
	backupID, filename, err := createBackup(c.NodeID, ts)
	if err != nil {
		return err
	}
	return s3.Upload(filename, c.Bucket, c.Prefix+"/"+backupID)
}

func Watch(s3 aws.Client, c Config, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for range t.C {
		err := Backup(s3, c, time.Now())
		if err != nil {
			log.Printf("backup failed: %v", err)
		}
	}
}
