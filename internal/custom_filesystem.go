package internal

import (
	// standard library
	"context"
	"io"
	"log/slog"
	"strings"
	"syscall"

	// external library
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type Node struct {
	fs.Inode

	Client           ClientBase
	Name             string // the Name of directory or file
	IsDirectory      bool
	DirectoryInfo    *DirectoryInfo
	isEmptyDirectory bool
}

var _ = (fs.NodeGetattrer)((*Node)(nil))
var _ = (fs.NodeReaddirer)((*Node)(nil))
var _ = (fs.NodeLookuper)((*Node)(nil))
var _ = (fs.NodeCreater)((*Node)(nil))
var _ = (fs.NodeUnlinker)((*Node)(nil))
var _ = (fs.NodeRmdirer)((*Node)(nil))
var _ = (fs.NodeMkdirer)((*Node)(nil))
var _ = (fs.NodeRenamer)((*Node)(nil))

const FilePermission = 0644
const DirectoryPermission = 0777
const notFoundFileCount = 10

var notFoundFileCountMap = make(map[string]int)
var notFoundFileHashSet = make(map[string]struct{})

// NOTE: fileの場合は、MemRegularFileを利用。directoryの場合はNodeを利用
func (r *Node) fullPath(name string) string {
	path := r.Path(r.Root())
	var key string
	if !r.IsRoot() {
		key = path + "/" + name
	} else {
		key = name
	}
	return key
}

func (r *Node) createNewFullPath(name string) string {
	var key string
	if r.IsRoot() {
		key = name
	} else {
		key = r.Path(r.Root()) + "/" + name
	}

	return key
}

func isEmptyFile(filesUnderDirectory []string, directoryName string) bool {
	return len(filesUnderDirectory) == 1 && filesUnderDirectory[0] == directoryName
}

func (r *Node) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	key := r.fullPath(name)
	object, err := r.Client.CreateObject(ctx, key+"/")
	if err != nil {
		return nil, syscall.ENOENT
	}

	dirInfo := DirectoryInfo{
		LastModified: object.LastModified,
	}
	newNode := r.NewInode(ctx, &Node{Name: name, Client: r.Client, IsDirectory: true, DirectoryInfo: &dirInfo}, fs.StableAttr{Mode: syscall.S_IFDIR})

	return newNode, 0
}

func (r *Node) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (node *fs.Inode, fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	key := r.createNewFullPath(name)

	object, err := r.Client.CreateObject(ctx, key)
	if err != nil {
		return nil, nil, 0, syscall.EACCES
	}
	chile := r.NewInode(ctx, &fs.MemRegularFile{
		Data: nil,
		Attr: fuse.Attr{
			Mode:  mode | FilePermission,
			Mtime: uint64(object.LastModified),
			Atime: uint64(object.LastModified),
			Ctime: uint64(object.LastModified),
			// NOTE: A file is created with the uchg flag by default. If so, you cannot remove that file without confirmation.
			Flags_: 0,
		},
	}, fs.StableAttr{
		Mode: syscall.S_IFREG,
	})

	return chile, nil, 0, 0

}

func (r *Node) Rmdir(ctx context.Context, name string) syscall.Errno {
	// NOTE: このメソッドが呼ばれる前に、対象ディレクトリ配下のオブジェクトのunlinkが呼ばれてすでに削除されている
	//  なので、ここではディレクトリ自体を削除する
	// treeはunlinkで削除されているので、ここでの処理は不要
	key := r.fullPath(name)
	list, err := r.Client.List(ctx, key)

	if err != nil {
		return syscall.ENOENT
	}

	directory := key + "/"
	if isEmptyFile(list, directory) {
		err = r.Client.DeleteObject(ctx, directory)
		if err != nil {
			return syscall.ENOENT
		}
	}

	return 0
}

func (r *Node) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	if r.IsDirectory {
		out.Mode = syscall.S_IFDIR | DirectoryPermission
		if r.DirectoryInfo != nil {
			out.Size = uint64(r.DirectoryInfo.SumContentByte)
			out.Mtime = uint64(r.DirectoryInfo.LastModified)
			out.Atime = out.Mtime
			out.Ctime = out.Atime
		}
	} else {
		out.Mode = syscall.S_IFREG | FilePermission
	}
	return 0
}

func isNotFoundFile(name string) bool {
	_, exists := notFoundFileHashSet[name]
	if exists {
		return true
	}
	notFoundFileCountMap[name] += 1
	if notFoundFileCountMap[name] > notFoundFileCount {
		notFoundFileHashSet[name] = struct{}{}
		return true
	}
	return false
}

func resetNotFoundFileCount(name string) {
	notFoundFileCountMap[name] = 0
}

// NOTE: 対象ディレクトリの中身を探索する、一回処理がきてinodeを返しているとそのnameのlookupはそれ以上呼ばれない
// ただ、touchやrmなど対象のファイル作成。削除が行われるとその都度このメソッドが呼ばれている
func (r *Node) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {

	// TODO: renameした際にtreeにnodeを追加しても、その後のlookupを呼ばれた時点でchildrenから消えている。。なぜかわからん

	// TODO: ↓lookupしても存在しないファイルはキャッシュに入れてすぐにnot foundを返すようにする
	//isNotFoundFile := isNotFoundFile(Name)
	//if isNotFoundFile {
	//	return nil, syscall.ENOENT
	//}
	//fmt.Println("GetChild: ", Name, r.GetChild(Name), r.Name, notFoundFileCountMap)
	// NOTE: すでにtree上に存在している場合は、そのinodeを返す
	c := r.GetChild(name)
	if c != nil {
		//resetNotFoundFileCount(Name)
		return c, 0
	} else {
		key := r.fullPath(name)
		isDirectory, err := r.Client.IsDirectory(ctx, key)
		if err != nil {
			return nil, syscall.ENOENT
		}
		if isDirectory {
			info, err := r.Client.GetDirectoryInfo(ctx, key)
			if err != nil {
				return nil, syscall.ENOENT
			}
			chile := r.NewInode(ctx, &Node{Name: name, Client: r.Client, IsDirectory: true, DirectoryInfo: info}, fs.StableAttr{Mode: syscall.S_IFDIR})
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

			child := r.NewInode(ctx, &fs.MemRegularFile{
				Data: body,
				Attr: fuse.Attr{
					// In the case of file, this attributes is shown when executing ls command.
					// However, the size attribute seems to be calculated automatically in the go-fuse.
					Mode:  syscall.S_IFREG | FilePermission,
					Mtime: uint64(object.LastModified),
					Atime: uint64(object.LastModified),
					Ctime: uint64(object.LastModified),
				},
			}, fs.StableAttr{
				Mode: syscall.S_IFREG,
			})
			return child, 0
		}
	}
}

func (r *Node) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	if r.isEmptyDirectory {
		return fs.NewListDirStream(nil), 0
	}
	childrenCount := len(r.Children())
	// すでにバックエンドのファイルシステムからnodeを取得している場合
	if childrenCount > 0 {
		entries := make([]fuse.DirEntry, childrenCount)
		var i int
		for k, _ := range r.Children() {
			entries[i] = fuse.DirEntry{
				Mode: syscall.S_IFREG,
				Name: k,
			}
			i += 1
		}
		return fs.NewListDirStream(entries), 0
	} else {
		path := r.Path(r.Root())
		if !r.IsRoot() {
			path = path + "/"
		}

		iter, err := r.Client.List(ctx, path)
		if err != nil {
			slog.Error(err.Error())
			return nil, syscall.ENOENT
		}

		// NOTE:
		// mkdirで作成したdirectoryで、中身が何も入っていない場合は
		// s3は空のオブジェクトを作成して、ディレクトリをエミュレートしている
		// この空のオブジェクトがあった場合にはディレクトリ自体が空であると表現する
		//[git][* feature/mkdir]:~/work_space/go-filesystem/ aws --endpoint-url=http://localhost:4566 s3 ls s3://my-bucket/sss-dir/                                                                                                                                                                                                  [/Users/jinzhengpengye/work_space/go-filesystem]
		//2024-05-31 10:53:44          0
		if isEmptyFile(iter, path) {
			// nilを返すと、5回ほどReaddirが呼ばれるので、isEmptyDirectoryにしてReaddirの最上段でreturnする
			r.isEmptyDirectory = true
			return fs.NewListDirStream(nil), 0
		}

		hashset := make(map[string]struct{})
		var entry []fuse.DirEntry
		for i := range iter {
			// NOTE: In the case of empty object, not display
			if path == iter[i] {
				continue
			}
			if r.IsRoot() {
				s := strings.Split(iter[i], "/")
				key := s[0]
				if len(s) == 1 {
					// file
					entry = append(entry, fuse.DirEntry{
						Mode: syscall.S_IFREG,
						Name: key,
					})
				} else {
					// directory
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
					// file
					entry = append(entry, fuse.DirEntry{
						Mode: syscall.S_IFREG,
						Name: splitedPath[0],
					})
				} else {
					// directory
					entry = append(entry, fuse.DirEntry{
						Mode: syscall.S_IFDIR,
						Name: splitedPath[0],
					})
				}

			}

		}

		return fs.NewListDirStream(entry), 0
	}
}

func (r *Node) Unlink(ctx context.Context, name string) syscall.Errno {
	key := r.createNewFullPath(name)

	err := r.Client.DeleteObject(ctx, key)
	if err != nil {
		return syscall.ENOENT
	}

	success, _ := r.RmChild(name)
	if !success {
		return syscall.ENOENT
	}
	return 0
}

func (r *Node) Rename(ctx context.Context, name string, newParent fs.InodeEmbedder, newName string, flags uint32) syscall.Errno {
	// TODO: ディレクトリを移動しないmvは完了したので、ディレクトリをまたいだmvを次回実装
	n := newParent.(*Node)
	isMoved := r.MvChild(name, &n.Inode, newName, true)
	if !isMoved {
		return syscall.ENOENT
	}

	// NOTE: renameした後に、nameとnewNameのinodeに対してlookupが走るので
	//  newNameの方をlocalstackに新規で作成し、nameの方を削除する
	// nameを削除
	key := r.createNewFullPath(name)
	err := r.Client.DeleteObject(ctx, key)
	if err != nil {
		return syscall.ENOENT
	}

	// newNameのobjectを作成
	key = r.createNewFullPath(newName)
	_, errs := r.Client.CreateObject(ctx, key)
	if errs != nil {
		return syscall.ENOENT
	}
	return fs.OK
}
