package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// HealthChecker is an interface that knows how to list buckets on object storage
type HealthChecker interface {
	HealthCheck(hcDuration time.Duration) (context.CancelFunc, error)
}

type fakeHealthChecker struct {
}

func NewFakeHealthChecker() HealthChecker {
	return &fakeHealthChecker{}
}

func (b *fakeHealthChecker) HealthCheck(hcDuration time.Duration) (context.CancelFunc, error) {
	_, cancel := context.WithTimeout(context.Background(), hcDuration)
	return cancel, nil
}

type ClusterHealthChecker struct {
	*madmin.AnonymousClient
}

func NewHealthChecker(host, port, accessKey, accessSecret string, insecure bool) (HealthChecker, error) {
	if len(os.Args) > 2 && os.Args[1] == "server" && strings.HasPrefix(os.Args[2], "http") {
		anonymousClient, err := madmin.NewAnonymousClient(fmt.Sprintf("%s:%s", host, port), insecure)
		if err != nil {
			return nil, err
		}
		return &ClusterHealthChecker{anonymousClient}, nil
	}
	return minio.New(fmt.Sprintf("%s:%s", host, port), &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, accessSecret, ""),
		Secure: insecure,
	})
}

func (b *ClusterHealthChecker) HealthCheck(hcDuration time.Duration) (context.CancelFunc, error) {
	parent, cancel := context.WithTimeout(context.Background(), hcDuration)
	result, err := b.Healthy(parent, madmin.HealthOpts{})
	if !result.Healthy {
		return cancel, errors.New("drycc storage unhealthy")
	}
	return cancel, err
}
