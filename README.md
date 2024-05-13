# cfs

cfs(custom file system)

You can access to cloud file storage such as aws s3, gcp gcs, azure blob 
as if you ran linux command like `ls` `touch`

## Current features:
- Currently only aws localstack can be usued
- following linux command can be usued 
  - `ls` `touch` `rm`

I am implementing the other commands now

## How to use

### 1. install FUSE into your PC

https://osxfuse.github.io/2024/04/05/macFUSE-4.7.0.html


### 2. Install binary

you can confirm the latest version [here](https://github.com/kinshotomoya/myown-filesystem/releases)
```shell
go install github.com/kinshotomoya/myown-filesystem/cfs@version
```

### 3. Mount your directory as follows
```shell
cfs -mountdir {mountDir} -provider aws -env local -bucket {bucketName}
```

Example:
```shell
cfs -mountdir /tmp/myown-filesystem -provider aws -env local -bucket my-bucket
```

### 4. you can your filesystem as usual
```shell
cd /tmp/myown-filesystem
```

Example:
```shell
cd /tmp/myown-filesystem
[]:/tmp/myown-filesystem/ ls -lh                                            
total 32
-rwxrwxrwx  0 root  wheel    86B  5 14 08:07 child1.txt
drwxrwxrwx  0 root  wheel    22B  5 14 08:07 child2
drwxrwxrwx  0 root  wheel    11B  5 14 08:07 child3
-rwxrwxrwx  0 root  wheel   219B  5 14 08:07 insert-test-data.sh
```

## How to develop in local

do umount after kill go process 
```shell
umount /tmp/myown-filesystem
```

if you want to confirm what filesystem are mounted, you run following command
```shell
mount
```

### set up local test data
```shell
./test-data/insert-test-data.sh
```

### execute custom filesystem process
```shell
go run cfs/main.go -mountdir /tmp/myown-filesystem -provider aws -env local -bucket my-bucket
```

## Tasks
- [x] sample code (access memory file directory)
- [x] docker set up
- [x] fix code to access localstack
- [x] fix code to run ls linux command `ls`
- [x] fix code to run other linux command `touch`
- [x] fix code to run other linux command `rm`
- [ ] fix code to run other linux command `rm -r`
- [ ] fix code to run other linux command `mv`
- [ ] fix code to run other linux command `tree`
- [ ] fix code to access not only localstack but also other cloud storage
- [ ] fix code to umount directory when kill go process 
- [x] go cli