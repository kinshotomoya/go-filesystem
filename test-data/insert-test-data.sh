#/bin/sh

aws --endpoint-url=http://localhost:4566 s3 mb s3://my-bucket
aws --endpoint-url=http://localhost:4566 s3 ls s3://my-bucket
aws --endpoint-url=http://localhost:4566 s3 cp --recursive test-data s3://my-bucket/
