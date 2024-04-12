package internal

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	s3v2 "github.com/aws/aws-sdk-go-v2/service/s3"
)

type ClientBase interface {
	List(ctx context.Context, key string) ([]string, error)
	Close()
}

type S3Client struct {
	Client     *s3v2.Client
	BucketName string
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

func (receiver S3Client) Close() {

}
