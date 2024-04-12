# myown-filesystem

local file system

## motivation

you can access to cloud file storage such as aws s3, gcp gcs, azure blob 
as if you ran linux command like `ls` `touch`

## TODO:
- [ ] sample code (access memory file directory)
- [ ] docker set up
- [ ] fix code to access localstack
- [ ] fix code to run other linux command `rm` `touch` `mv` `tree`
- [ ] fix code to access not only localstack but also other cloud storage
- [ ] fix code to umount directory when kill go process 


## how to develop in local

do umount after kill go process 
```shell
$ umount /tmp/myown-filesystem
```

if you want to confirm what filesystem are mounted, you run following command
```shell
$ mount
```

### set up local test data

create bucket
```shell
$ aws --endpoint-url=http://localhost:4566 s3 mb s3://my-bucket
```

confirm the inside of the bucket 
```shell
$ aws --endpoint-url=http://localhost:4566 s3 ls s3://my-bucket
```

copy test-data directory to localstack
```shell
$ aws --endpoint-url=http://localhost:4566 s3 cp --recursive test-data s3://my-bucket/
```

go run
```shell
$ go run cmd/main.go -mountdir /tmp/myown-filesystem -provider aws -env local -bucket my-bucket
```

 ## how to use
install FUSE into your PC
https://osxfuse.github.io/2024/04/05/macFUSE-4.7.0.html



