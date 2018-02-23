package backup

import (
	"strings"
	"testing"
	"time"

	"github.com/coldog/etcd-aws-cluster/pkg/aws"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type AWSMock struct {
	aws.Client
	mock.Mock
}

func (m *AWSMock) Upload(filename, bucket, key string) error {
	return m.Called(filename, bucket, key).Error(0)
}

func TestUpload(t *testing.T) {
	a := &AWSMock{}

	ts := time.Date(2011, 01, 01, 01, 01, 0, 0, time.UTC)

	a.On("Upload",
		"testdata/etcd-1-2011-01-01T01-01-00Z-00.tar.gz",
		"test",
		"pre/etcd-1-2011-01-01T01-01-00Z-00",
	).Return(nil)

	commands := []string{}

	sh = func(cmd string, args ...string) error {
		commands = append(commands, cmd+" "+strings.Join(args, " "))
		return nil
	}

	backupRoot = "testdata"
	backupDir = backupRoot + "/backup"

	err := Backup(a, Config{
		NodeID: "1",
		Bucket: "test",
		Prefix: "pre",
	}, ts)
	require.Nil(t, err)

	require.Equal(t, []string{
		"/bin/etcdctl backup --data-dir /var/lib/etcd --backup-dir testdata/backup",
		"/bin/tar -C testdata -zcf testdata/etcd-1-2011-01-01T01-01-00Z-00.tar.gz .",
	}, commands)

	a.AssertExpectations(t)
}
