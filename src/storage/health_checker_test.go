package storage

import (
	"os"
	"testing"

	"github.com/minio/minio-go/v7"
)

func TestNewHealthChecker(t *testing.T) {
	os.Args = []string{"minio", "server", "http://node{1...16}.example.com/mnt/export{1...32}"}
	healthChecker, err := NewHealthChecker("localhost", "9000", "test", "test", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := healthChecker.(*ClusterHealthChecker); !ok {
		t.Error("Assertion error")
	}

	os.Args = []string{"minio", "server", "/data"}
	healthChecker, err = NewHealthChecker("localhost", "9000", "test", "test", false)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := healthChecker.(*minio.Client); !ok {
		t.Error("Assertion error")
	}

	os.Args = []string{"minio", "gateway", "s3", "https://pay.minio.io"}
	healthChecker, err = NewHealthChecker("localhost", "9000", "test", "test", false)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := healthChecker.(*minio.Client); !ok {
		t.Error("Assertion error")
	}
}
