//
// minio.go
// Copyright (C) 2018 Odin <Odin@Odin-Pro.local>
//
// Distributed under terms of the MIT license.
//

package rest

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"strings"

	"wgo/resty"
	"wgo/whttp"

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

func fileExtByMimeType(mt string) string {
	switch strings.ToLower(mt) {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	default:
		return ""
	}
}

func PutPicByUrl(url, bucketName string) (string, error) {
	objectStorage, err := NewObjectStorage(mio["endpoint"].(string), mio["access_key"].(string), mio["secret_key"].(string), mio["secure"].(bool))
	if err != nil {
		return "", err
	}
	path, err := objectStorage.PutPicByUrl(url, bucketName)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%s", mio["cdn"].(string), path), nil

}

// put object by url
func (o *ObjectStorage) PutPicByUrl(url, bucketName string) (string, error) {
	// download pic
	resty.SetRedirectPolicy(resty.FlexibleRedirectPolicy(15))
	if picRes, err := resty.R().Get(url); err == nil {
		mimeType := picRes.Header().Get(whttp.HeaderContentType)
		body := picRes.Body()
		checksum := fmt.Sprintf("%x", md5.Sum(body))
		objectName := fmt.Sprintf("%s/%s/%s%s", checksum[0:2], checksum[2:4], checksum, fileExtByMimeType(mimeType))
		Debug("[PutPicByUrl]objectName: %s, checksum: %s, mimeType: %s, url: %s", objectName, checksum, mimeType, url)
		// return PutObject(bucketName, objectName, mimeType, bytes.NewReader(body), int64(len(body)))
		if _, err := o.PutObject(bucketName, objectName, mimeType, bytes.NewReader(body), int64(len(body))); err != nil {
			// Debug("[PutPicByUrl]upload image failed: %s", err)
			return "", err
		}
		return fmt.Sprintf("/%s/%s", bucketName, objectName), nil
	}
	return "", errors.New("[PutPicByUrl]failed")
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
