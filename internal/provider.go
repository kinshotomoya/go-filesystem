package internal

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	s3v2 "github.com/aws/aws-sdk-go-v2/service/s3"
	"io"
	"time"
)

type ClientBase interface {
	List(ctx context.Context, key string) ([]string, error)
	GetObject(ctx context.Context, key string) (*Object, error)
	IsDirectory(ctx context.Context, key string) (bool, error)
	GetDirectoryInfo(ctx context.Context, key string) (*DirectoryInfo, error)
	CreateObject(ctx context.Context, key string) (*Object, error)
	DeleteObject(ctx context.Context, key string) error
	Close()
}

var _ = (ClientBase)((*S3Client)(nil))

type S3Client struct {
	Client     *s3v2.Client
	BucketName string
}

type Object struct {
	Body              io.Reader
	ContentLengthByte int64
	LastModified      int64 // 最終更新日
}

type DirectoryInfo struct {
	SumContentByte int64
	LastModified   int64 // 最終更新日
}

func (receiver *S3Client) DeleteObject(ctx context.Context, key string) error {
	_, err := receiver.Client.DeleteObject(ctx, &s3v2.DeleteObjectInput{
		Bucket: aws.String(receiver.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}
	return nil
}

func (receiver *S3Client) CreateObject(ctx context.Context, key string) (*Object, error) {
	_, err := receiver.Client.PutObject(ctx, &s3v2.PutObjectInput{
		Bucket: aws.String(receiver.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	return &Object{
		LastModified: time.Now().Unix(),
	}, nil
}

func (receiver *S3Client) List(ctx context.Context, key string) ([]string, error) {
	resp, err := receiver.Client.ListObjectsV2(ctx, &s3v2.ListObjectsV2Input{
		Bucket: aws.String(receiver.BucketName),
		Prefix: aws.String(key),
	})

	if err != nil {
		return nil, err
	}

	entries := make([]string, len(resp.Contents))
	for i := range resp.Contents {
		entries[i] = *resp.Contents[i].Key
	}

	return entries, nil

}

func (receiver *S3Client) GetObject(ctx context.Context, key string) (*Object, error) {
	object, err := receiver.Client.GetObject(ctx, &s3v2.GetObjectInput{
		Bucket: aws.String(receiver.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	var contentLengthByte int64
	if object.ContentLength != nil {
		contentLengthByte = *object.ContentLength
	}
	return &Object{
		Body:              object.Body,
		ContentLengthByte: contentLengthByte,
		LastModified:      object.LastModified.Unix(),
	}, nil

}

func (receiver *S3Client) IsDirectory(ctx context.Context, key string) (bool, error) {
	output, err := receiver.Client.ListObjectsV2(ctx, &s3v2.ListObjectsV2Input{
		Bucket: aws.String(receiver.BucketName),
		Prefix: aws.String(key + "/"),
	})
	if err != nil {
		return false, err
	}
	var isDirectory bool
	if output.KeyCount != nil && *output.KeyCount >= 1 {
		isDirectory = true
	}
	return isDirectory, nil

}

func (receiver *S3Client) GetDirectoryInfo(ctx context.Context, key string) (*DirectoryInfo, error) {
	// 指定されたkey配下のファイルをトラバースして、合計ファイルサイズと一番更新日が遅いファイルの更新日を取得する
	output, err := receiver.Client.ListObjectsV2(ctx, &s3v2.ListObjectsV2Input{
		Bucket: aws.String(receiver.BucketName),
		Prefix: aws.String(key + "/"),
	})

	if err != nil {
		return nil, err
	}

	var sumContentByte int64
	var lastModifiedTimeUnix int64
	for _, v := range output.Contents {
		object, err := receiver.GetObject(ctx, *v.Key)
		if err != nil {
			return nil, err
		}
		sumContentByte += object.ContentLengthByte
		if lastModifiedTimeUnix < object.LastModified {
			lastModifiedTimeUnix = object.LastModified
		}
	}

	return &DirectoryInfo{
		SumContentByte: sumContentByte,
		LastModified:   lastModifiedTimeUnix,
	}, nil

}

func (receiver *S3Client) Close() {

}
