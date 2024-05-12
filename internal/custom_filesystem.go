package internal

import (
	// standard library
	"context"
	"fmt"
	"io"
	"strings"
	"syscall"

	// external library
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type Node struct {
	fs.Inode

	Client        ClientBase
	name          string // ディレクトリ、ファイルの名前（key name）
	IsDirectory   bool
	DirectoryInfo *DirectoryInfo
}

// これは型アサーションをすることでinterfaceの実装ミスをコンパイル時に防ぐために定義している
// 以下詳細：
// (*Node)(nil)でNode型のnilポインタを返す
// (fs.NodeGetattrer)((*Node)(nil))で↑で作成したHelloRoot型のnilポインタをfs.NodeGetattrer型に型アサーションしようとしている
// こうすることで、HelloRoot構造体が、fs.NodeGetattrer interfaceを実装していない場合にコンパイルエラーが発生するので、コンパイル時に実装ミスに気づける
var _ = (fs.NodeGetattrer)((*Node)(nil))
var _ = (fs.NodeReaddirer)((*Node)(nil))
var _ = (fs.NodeLookuper)((*Node)(nil))
var _ = (fs.NodeCreater)((*Node)(nil))
var _ = (fs.NodeUnlinker)((*Node)(nil))

//var _ = (fs.NodeRmdirer)((*Node)(nil))

// TODO: rm -rした時の挙動を追加
//func (r *Node) Rmdir(ctx context.Context, name string) syscall.Errno {
//

//}

func (r *Node) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	if r.IsDirectory {
		out.Mode = syscall.S_IFDIR | 0777
		if r.DirectoryInfo != nil {
			out.Size = uint64(r.DirectoryInfo.SumContentByte)
			out.Mtime = uint64(r.DirectoryInfo.LastModified)
			out.Atime = out.Mtime
			out.Ctime = out.Atime
		}
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
		info, err := r.Client.GetDirectoryInfo(ctx, name)
		if err != nil {
			return nil, syscall.ENOENT
		}
		chile := r.NewInode(ctx, &Node{name: name, Client: r.Client, IsDirectory: true, DirectoryInfo: info}, fs.StableAttr{Mode: syscall.S_IFDIR})
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
				Mode:  syscall.S_IFREG | 0777,
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
			fullPath := iter[i]
			trimedPath := strings.TrimPrefix(fullPath, path)
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

func (r *Node) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (node *fs.Inode, fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	var key string
	if r.IsRoot() {
		key = name
	} else {
		key = r.Path(r.Root()) + "/" + name
	}
	object, err := r.Client.CreateObject(ctx, key)
	if err != nil {
		return nil, nil, 0, syscall.EACCES
	}
	chile := r.NewInode(ctx, &fs.MemRegularFile{
		Data: nil,
		Attr: fuse.Attr{
			Mode:   mode | 0777,
			Mtime:  uint64(object.LastModified),
			Atime:  uint64(object.LastModified),
			Ctime:  uint64(object.LastModified),
			Flags_: flags,
		},
	}, fs.StableAttr{
		Mode: syscall.S_IFREG,
	})

	return chile, nil, 0, 0

}

func (r *Node) Unlink(ctx context.Context, name string) syscall.Errno {
	var key string
	if r.IsRoot() {
		key = name
	} else {
		key = r.Path(r.Root()) + "/" + name
	}

	err := r.Client.DeleteObject(ctx, key)
	if err != nil {
		fmt.Printf("delete error: %v", err)
		return syscall.ENOENT
	}

	return 0
}
