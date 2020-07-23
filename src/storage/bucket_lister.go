package storage

import (
    "context"
	"github.com/minio/minio-go/v7"
)

// BucketLister is an interface that knows how to list buckets on object storage
type BucketLister interface {
	ListBuckets(ctx context.Context) ([]minio.BucketInfo, error)
}

type fakeBucketLister struct {
	bucketInfos []minio.BucketInfo
}

func NewFakeBucketLister(buckets []minio.BucketInfo) BucketLister {
	return &fakeBucketLister{bucketInfos: buckets}
}

func (b *fakeBucketLister) ListBuckets(ctx context.Context) ([]minio.BucketInfo, error) {
	return b.bucketInfos, nil
}
