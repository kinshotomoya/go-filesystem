package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsv2cfg "github.com/aws/aws-sdk-go-v2/config"
	s3v2 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hanwen/go-fuse/v2/fs"
	"log"
	"myown-filesystem/internal"
)

func connectProvider(provider string, env string, bucketName string) (internal.ClientBase, error) {
	ctx := context.Background()

	endpoint := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if env == "local" {
			return aws.Endpoint{
				URL:           "http://localhost:4566",
				SigningRegion: "ap-northeast-1",
			}, nil
		} else {
			return aws.Endpoint{}, nil
		}
	})

	cfg, err := awsv2cfg.LoadDefaultConfig(
		ctx,
		awsv2cfg.WithRegion("ap-northeast-1"),
		awsv2cfg.WithEndpointResolverWithOptions(endpoint),
		// TODO: local以外の場合は指定されたprofileを利用
		//awsv2cfg.WithSharedConfigProfile()
	)
	if err != nil {
		return nil, err
	}

	clientV2 := s3v2.NewFromConfig(cfg, func(options *s3v2.Options) {
		// trueにしないと、localの場合にhttp://my-bucket.localhost:4566とアドレスがなってしまう
		if env == "local" {
			options.UsePathStyle = true
		}
	})

	return internal.S3Client{
		Client:     clientV2,
		BucketName: bucketName,
	}, nil
}

// myown -mountdir /tmp/myown-filesystem -provider aws -env local -bucket my-bucket
func main() {
	// fs.api.goに定義されているそれぞれのinterfaceを実装することで、ファイルシステムに対するシステムコールをハンドリングできるようになる
	// 例えば、Readdirメソッドを実装すると、lsコマンドで発行されるシステムコールをgoプロセス内でハンドリングできる
	mountDir := flag.String("mountdir", "/tmp/myown-filesystem", "mount directory")
	provider := flag.String("provider", "aws", "cloud provider aws, gcp, azure")
	env := flag.String("env", "local", "environment")
	bucketName := flag.String("bucket", "default-bucket", "bucket name")
	flag.Parse()

	if mountDir == nil {
		log.Fatal("mountdir flag is required")
	}

	if provider == nil {
		log.Fatal("provider flag is required")
	}

	client, err := connectProvider(*provider, *env, *bucketName)
	if err != nil {
		log.Println(err)
		log.Fatal("fatal connect provider")
	}
	fmt.Println("connected to target provider")

	opts := &fs.Options{}
	// ルートディレクトリにマウントしている
	server, err := fs.Mount(*mountDir, &internal.Node{Client: client, IsDirectory: true}, opts)
	if err != nil {
		log.Fatal("fatal mount")
	}
	fmt.Println("mounted to target directory")
	server.Wait()
	// TODO: シグナルハンドリング
	server.Unmount()
}
