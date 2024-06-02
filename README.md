# go-filesystem

This tool allows you to interact with cloud storage services like AWS S3, GCP GCS, and Azure Blob Storage using familiar Linux commands (ls, touch, etc.).

## Current features:
- **Current:** Supports AWS Localstack with commands ls, touch, rm, rm -r.
- **Planned:** Additional commands and support for more cloud providers.

### important
This works only on MacOS(Apple Silicon or Intel) now.

## Installation

### 1. Install FUSE

https://osxfuse.github.io/2024/04/05/macFUSE-4.7.0.html

### 2. Install the Binary

Find the latest version and install:
```shell
go install github.com/kinshotomoya/go-filesystem/go-filesystem@latest
```

### 3. Mount Directory
```shell
go-filesystem -mountdir {mountDir} -provider aws -env local -bucket {bucketName}
```

Example:
```shell
go-filesystem -mountdir /tmp/myown-filesystem -provider aws -env local -bucket my-bucket
```

### 4. Use the Filesystem
Navigate and use as usual:
```shell
cd /tmp/myown-filesystem
ls -lh
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

### 5. Unmount the Directory
Change directory before exiting the go-filesystem process so that the directory is unmounted automatically and successfully.
If there is any trouble as unmounting the directory, do it manually 

```shell
umount {target directory}
```

Example:
```shell
umount /tmp/myown-filesystem/
```


## Development

### Verify Mounted Filesystems
```shell
mount
```

### Run localstack on docker-compose
```shell
docker compose up -d
```

### Setup Local Test Data 
```shell
./test-data/insert-test-data.sh
```

### Run go-filesystem
```shell
go run go-filesystem/main.go -mountdir /tmp/myown-filesystem -provider aws -env local -bucket my-bucket
```

## License
MIT License

Feel free to further adjust this to match your project's specifics and add any missing details.


## Tasks
- [x] sample code (access memory file directory)
- [x] docker set up
- [x] fix code to access localstack
- [x] fix code to run ls linux command `ls`
- [x] fix code to run other linux command `touch`
- [x] fix code to run other linux command `rm`
- [x] fix code to run other linux command `mkdir`
- [x] fix code to run other linux command `rm -r`
- [ ] fix code to run other linux command `mv`
- [ ] fix code to run other linux command `tree`
- [ ] fix code to run other linux command `vi`
- [ ] fix code to access not only localstack but also other cloud storage
- [x] fix code to not display `override rwxrwxrwx root/wheel uchg for hoge.txt?` when removing files that is created on mounted filesystem
- [x] fix code to umount directory when kill go process
- [ ] not to access to s3 carelessly
- [x] go cli
