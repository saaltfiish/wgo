//
// minio.go
// Copyright (C) 2018 Odin <Odin@Odin-Pro.local>
//
// Distributed under terms of the MIT license.
//

package rest

import (
	"fmt"
	"io"

	minio "github.com/minio/minio-go"
	"github.com/minio/minio-go/pkg/policy"
)

type ObjectStorage struct {
	client *minio.Client
}

func NewObjectStorage(endpoint, accessKey, secteKey string, secure bool) (*ObjectStorage, error) {
	minioClient, err := minio.New(endpoint, accessKey, secteKey, secure)
	if err != nil {
		return nil, err
	}
	return &ObjectStorage{
		client: minioClient,
	}, nil
}

// open minio
func openObjectStorage() error {
	if _, err := minio.New(mio["endpoint"].(string), mio["access_key"].(string), mio["secret_key"].(string), mio["secure"].(bool)); err != nil {
		Error("open minio error: %s", err)
		return err
	} else {
		Debug("open object storage ok")
		// if err := os.SetBucketRead("avatars", "12/33"); err != nil {
		// 	Debug("set policy error: %s", err)
		// }
	}
	return nil
}

// set policy
func (o *ObjectStorage) SetBucketRead(bucketName, objectPrefix string) error {
	return o.client.SetBucketPolicy(bucketName, objectPrefix, policy.BucketPolicyReadOnly)
}

// pub object
func PutObject(bucketName, objectName, contentType string, reader io.Reader, objectSize int64) (string, error) {
	objectStorage, _ := NewObjectStorage(mio["endpoint"].(string), mio["access_key"].(string), mio["secret_key"].(string), mio["secure"].(bool))
	_, err := objectStorage.PutObject(bucketName, objectName, contentType, reader, objectSize)
	url := fmt.Sprintf("%s/%s/%s", mio["cdn"].(string), mio["bucket"].(string), objectName)
	return url, err
}
func (o *ObjectStorage) PutObject(bucketName, objectName, contentType string, reader io.Reader, objectSize int64) (int64, error) {
	opts := minio.PutObjectOptions{ContentType: contentType}
	return o.client.PutObject(bucketName, objectName, reader, objectSize, opts)
}
