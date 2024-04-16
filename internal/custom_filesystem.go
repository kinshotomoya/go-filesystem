package internal

import (
	"context"
	"fmt"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"io"
	"strings"
	"syscall"
)

type Root struct {
	fs.Inode

	Client ClientBase
	name   string // ディレクトリ、ファイルの名前（key name）
}

// これは型アサーションをすることでinterfaceの実装ミスをコンパイル時に防ぐために定義している
// 以下詳細：
// (*Root)(nil)でHelloRoot型のnilポインタを返す
// (fs.NodeGetattrer)((*Root)(nil))で↑で作成したHelloRoot型のnilポインタをfs.NodeGetattrer型に型アサーションしようとしている
// こうすることで、HelloRoot構造体が、fs.NodeGetattrer interfaceを実装していない場合にコンパイルエラーが発生するので、コンパイル時に実装ミスに気づける
var _ = (fs.NodeGetattrer)((*Root)(nil))
var _ = (fs.NodeReaddirer)((*Root)(nil))
var _ = (fs.NodeLookuper)((*Root)(nil))

// TODO: ここで指定したデータとlookupなどの各メソッドで定義した情報どちらが使われるのか調べる
func (r *Root) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = 0755
	return 0
}

func (r *Root) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	path := r.Path(r.Root())
	var key string
	if path != "" {
		key = path + "/" + name
	} else {
		key = name
	}
	fmt.Println(path, name, key)
	isDirectory, err := r.Client.IsDirectory(ctx, key)
	if err != nil {
		return nil, syscall.ENOENT
	}
	if isDirectory {
		// ディレクトリの場合
		chile := r.NewInode(ctx, &Root{name: name, Client: r.Client}, fs.StableAttr{Mode: syscall.S_IFDIR})

		// ディレクトリであるフラグとファイルパーミッションをORでビット演算している
		out.Mode = syscall.S_IFDIR | 0755
		out.Size = uint64(100000) // TODO: 仮
		return chile, 0
	} else {
		object, err := r.Client.GetObject(ctx, key)
		if err != nil {
			return nil, syscall.ENOENT
		}

		body, err := io.ReadAll(object.Body)
		if err != nil {
			return nil, syscall.ENOENT
		}

		chile := r.NewInode(ctx, &fs.MemRegularFile{
			Data: body,
			Attr: fuse.Attr{
				// TODO: 権限は読み込み書き込み専用でいいか
				Mode: 0444,
			},
		}, fs.StableAttr{Mode: syscall.S_IFREG})

		out.Mode = 0444
		out.Size = uint64(object.ContentLengthByte)
		return chile, 0
	}
}

func (r *Root) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	entry := make([]fuse.DirEntry, 0)
	path := r.Path(r.Root())
	var key string
	if !r.IsRoot() {
		if path != "" {
			key = path + "/"
		} else {
			key = path
		}
	}

	iter, err := r.Client.List(ctx, key, "")
	if err != nil {
		return nil, syscall.ENOENT
	}

	hashset := make(map[string]struct{})
	for i := range iter {
		fmt.Println("read", key, iter[i])
		if r.IsRoot() {
			s := strings.Split(iter[i], "/")
			key := s[0]
			if len(s) == 1 {
				// ファイルの場合
				entry = append(entry, fuse.DirEntry{
					Mode: syscall.S_IFREG,
					Name: key,
				})
			} else {
				// ディレクトリの場合
				_, exist := hashset[key]
				if !exist {
					hashset[key] = struct{}{}
					entry = append(entry, fuse.DirEntry{
						Mode: syscall.S_IFDIR,
						Name: key,
					})
				}
			}
		} else {
			// key = child2/child4/
			// child2/child4/grandchild3.txt

			// key = child2/
			// child2/child4/grandchild3.txt
			// child2/grandchild1.txt
			fullPath := iter[i]
			// grandchild3.txt

			// child4/grandchild3.txt
			// grandchild1.txt
			trimedPath := strings.TrimPrefix(fullPath, key)
			// [grandchild3.txt]

			// [child4, grandchild3.txt]
			// [grandchild1.txt]
			splitedPath := strings.Split(trimedPath, "/")
			if len(splitedPath) == 1 {
				// ファイルの場合
				entry = append(entry, fuse.DirEntry{
					Mode: syscall.S_IFREG,
					Name: splitedPath[0],
				})
			} else {
				// ディレクトリの場合
				entry = append(entry, fuse.DirEntry{
					Mode: syscall.S_IFDIR,
					// ディレクトリの名前だけでいい
					// child2/child4/grandchild1.txtの場合は、child4を取得
					Name: splitedPath[0],
				})
			}

		}

	}

	return fs.NewListDirStream(entry), 0
}
