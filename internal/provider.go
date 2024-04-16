package internal

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	s3v2 "github.com/aws/aws-sdk-go-v2/service/s3"
	"io"
)

type ClientBase interface {
	List(ctx context.Context, key string) ([]string, error)
	GetObject(ctx context.Context, key string) (*Object, error)
	IsDirectory(ctx context.Context, key string) (bool, error)
	Close()
}

type S3Client struct {
	Client     *s3v2.Client
	BucketName string
}

type Object struct {
	Body              io.Reader
	ContentLengthByte int64
}

func (receiver S3Client) List(ctx context.Context, key string) ([]string, error) {
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

func (receiver S3Client) GetObject(ctx context.Context, key string) (*Object, error) {
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
	}, nil

}

func (receiver S3Client) IsDirectory(ctx context.Context, key string) (bool, error) {
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

func (receiver S3Client) Close() {

}
