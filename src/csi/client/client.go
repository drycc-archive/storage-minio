package client

import (
	"bytes"
	"context"
	"fmt"
	"net/url"

	"github.com/golang/glog"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	Config *Config
	minio  *minio.Client
	ctx    context.Context
}

// Config holds values to configure the driver
type Config struct {
	Lookup          string
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string
}

func NewClient(cfg *Config) (*Client, error) {
	var client = &Client{}

	client.Config = cfg
	u, err := url.Parse(client.Config.Endpoint)
	if err != nil {
		return nil, err
	}
	ssl := u.Scheme == "https"
	endpoint := u.Hostname()
	if u.Port() != "" {
		endpoint = u.Hostname() + ":" + u.Port()
	}
	bucketLookupType := minio.BucketLookupAuto
	if cfg.Lookup == "path" {
		bucketLookupType = minio.BucketLookupPath
	} else if cfg.Lookup == "dns" {
		bucketLookupType = minio.BucketLookupDNS
	}

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(client.Config.AccessKeyID, client.Config.SecretAccessKey, ""),
		Secure:       ssl,
		BucketLookup: bucketLookupType,
	})
	if err != nil {
		return nil, err
	}
	client.minio = minioClient
	client.ctx = context.Background()
	return client, nil
}

func NewClientFromSecret(secret map[string]string) (*Client, error) {
	return NewClient(&Config{
		Lookup:          secret["lookup"],
		AccessKeyID:     secret["accesskey"],
		SecretAccessKey: secret["secretkey"],
		Endpoint:        secret["endpoint"],
	})
}

func (client *Client) BucketExists(bucketName string) (bool, error) {
	return client.minio.BucketExists(client.ctx, bucketName)
}

func (client *Client) CreateBucket(bucketName string) error {
	return client.minio.MakeBucket(client.ctx, bucketName, minio.MakeBucketOptions{Region: ""})
}

func (client *Client) CreatePrefix(bucketName string, prefix string) error {
	if prefix != "" {
		_, err := client.minio.PutObject(client.ctx, bucketName, prefix+"/", bytes.NewReader([]byte("")), 0, minio.PutObjectOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (client *Client) RemovePrefix(bucketName string, prefix string) error {
	var err error

	if err = client.removeObjects(bucketName, prefix); err == nil {
		return client.minio.RemoveObject(client.ctx, bucketName, prefix, minio.RemoveObjectOptions{})
	}

	glog.Warningf("removeObjects failed with: %s, will try removeObjectsOneByOne", err)

	if err = client.removeObjectsOneByOne(bucketName, prefix); err == nil {
		return client.minio.RemoveObject(client.ctx, bucketName, prefix, minio.RemoveObjectOptions{})
	}

	return err
}

func (client *Client) RemoveBucket(bucketName string) error {
	var err error

	if err = client.removeObjects(bucketName, ""); err == nil {
		return client.minio.RemoveBucket(client.ctx, bucketName)
	}

	glog.Warningf("removeObjects failed with: %s, will try removeObjectsOneByOne", err)

	if err = client.removeObjectsOneByOne(bucketName, ""); err == nil {
		return client.minio.RemoveBucket(client.ctx, bucketName)
	}

	return err
}

func (client *Client) removeObjects(bucketName, prefix string) error {
	objectsCh := make(chan minio.ObjectInfo)
	var listErr error

	go func() {
		defer close(objectsCh)

		for object := range client.minio.ListObjects(
			client.ctx,
			bucketName,
			minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
			if object.Err != nil {
				listErr = object.Err
				return
			}
			objectsCh <- object
		}
	}()

	if listErr != nil {
		glog.Error("error listing objects", listErr)
		return listErr
	}

	opts := minio.RemoveObjectsOptions{
		GovernanceBypass: true,
	}
	haveErrWhenRemoveObjects := false
	for e := range client.minio.RemoveObjects(client.ctx, bucketName, objectsCh, opts) {
		glog.Errorf("failed to remove object %s, error: %s", e.ObjectName, e.Err)
		haveErrWhenRemoveObjects = true
	}
	if haveErrWhenRemoveObjects {
		return fmt.Errorf("failed to remove all objects of bucket %s", bucketName)
	}

	return nil
}

// will delete files one by one without file lock
func (client *Client) removeObjectsOneByOne(bucketName, prefix string) error {
	parallelism := 16
	objectsCh := make(chan minio.ObjectInfo, 1)
	guardCh := make(chan int, parallelism)
	var listErr error
	totalObjects := 0
	removeErrors := 0

	go func() {
		defer close(objectsCh)

		for object := range client.minio.ListObjects(client.ctx, bucketName,
			minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
			if object.Err != nil {
				listErr = object.Err
				return
			}
			totalObjects++
			objectsCh <- object
		}
	}()

	if listErr != nil {
		glog.Error("error listing objects", listErr)
		return listErr
	}

	for obj := range objectsCh {
		guardCh <- 1
		go func(obj minio.ObjectInfo) {
			err := client.minio.RemoveObject(client.ctx, bucketName, obj.Key,
				minio.RemoveObjectOptions{VersionID: obj.VersionID})
			if err != nil {
				glog.Errorf("failed to remove object %s, error: %s", obj.Key, err)
				removeErrors++
			}
			<-guardCh
		}(obj)
	}
	for i := 0; i < parallelism; i++ {
		guardCh <- 1
	}
	for i := 0; i < parallelism; i++ {
		<-guardCh
	}

	if removeErrors > 0 {
		return fmt.Errorf("Failed to remove %v objects out of total %v of path %s", removeErrors, totalObjects, bucketName)
	}

	return nil
}
