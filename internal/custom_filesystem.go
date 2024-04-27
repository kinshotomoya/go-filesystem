package internal

import (
	"context"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"io"
	"strings"
	"syscall"
)

type Node struct {
	fs.Inode

	Client      ClientBase
	name        string // ディレクトリ、ファイルの名前（key name）
	IsDirectory bool
}

// これは型アサーションをすることでinterfaceの実装ミスをコンパイル時に防ぐために定義している
// 以下詳細：
// (*Node)(nil)でHelloRoot型のnilポインタを返す
// (fs.NodeGetattrer)((*Node)(nil))で↑で作成したHelloRoot型のnilポインタをfs.NodeGetattrer型に型アサーションしようとしている
// こうすることで、HelloRoot構造体が、fs.NodeGetattrer interfaceを実装していない場合にコンパイルエラーが発生するので、コンパイル時に実装ミスに気づける
var _ = (fs.NodeGetattrer)((*Node)(nil))
var _ = (fs.NodeReaddirer)((*Node)(nil))
var _ = (fs.NodeLookuper)((*Node)(nil))

func (r *Node) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	if r.IsDirectory {
		out.Mode = syscall.S_IFDIR | 0755
		// TODO: ディレクトリの場合は、子供のファイルのサイズを合算する
		out.Size = 1024 * 1024
	} else {
		out.Mode = syscall.S_IFREG | 0777
	}
	return 0
}

func (r *Node) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	path := r.Path(r.Root())
	var key string
	if !r.IsRoot() {
		key = path + "/" + name
	} else {
		key = name
	}
	isDirectory, err := r.Client.IsDirectory(ctx, key)
	if err != nil {
		return nil, syscall.ENOENT
	}
	if isDirectory {
		chile := r.NewInode(ctx, &Node{name: name, Client: r.Client, IsDirectory: true}, fs.StableAttr{Mode: syscall.S_IFDIR})
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
				// ファイルの場合はここでの設定がlsで表示されている
				// ただsizeはgo-fuse内部で自動計算されていそう
				Mode:  syscall.S_IFREG | 0755,
				Mtime: uint64(object.LastModified),
				Atime: uint64(object.LastModified),
				Ctime: uint64(object.LastModified),
			},
		}, fs.StableAttr{
			Mode: syscall.S_IFREG,
		})
		return chile, 0
	}
}

func (r *Node) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	path := r.Path(r.Root())
	if !r.IsRoot() {
		path = path + "/"
	}

	iter, err := r.Client.List(ctx, path)
	if err != nil {
		return nil, syscall.ENOENT
	}

	hashset := make(map[string]struct{})
	entry := make([]fuse.DirEntry, 0)
	for i := range iter {
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
			// child2/child4ディレクトリでls打った場合
			// key = child2/child4/
			// child2/child4/grandchild3.txt

			// child2ディレクトリでls打った場合
			// key = child2/
			// child2/child4/grandchild3.txt
			// child2/grandchild1.txt
			fullPath := iter[i]
			// grandchild3.txt

			// child4/grandchild3.txt
			// grandchild1.txt
			trimedPath := strings.TrimPrefix(fullPath, path)
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
