package main

import (
	"context"
	"fmt"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"log"
	"syscall"
)

type HelloRoot struct {
	fs.Inode
}
type Directory struct {
	fs.Inode

	name string
}

// マウント時に呼ばれる
// 初期化に必要な場合に実装する
//func (r *HelloRoot) OnAdd(ctx context.Context) {
//	fmt.Println("adddd")
//	ch := r.NewPersistentInode(
//		ctx, &fs.MemRegularFile{
//			Data: []byte("file.txt"),
//			Attr: fuse.Attr{
//				Mode: 0644,
//			},
//		}, fs.StableAttr{Ino: 2})
//	r.AddChild("files.txt", ch, true)
//
//}

// TODO: ここで指定したデータとlookupなどの各メソッドで定義した情報どちらが使われるのか調べる
func (r *HelloRoot) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = 0755
	return 0
}

func (r *HelloRoot) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	fmt.Println(name)
	// TODO: readdirの後に、ファイルやディレクトリそれぞれで呼ばれる
	//  それぞれ個々の処理が来たら、localstackにnameで検索かけに行き、ファイルならファイルのフラグで返す
	//  ディレクトリならディレクトリのフラグで処理を返してあげる
	//  以下は例として仮想的なファイルとディレクトリを返している

	// ファイルの場合
	if name == "hoge.txt" {
		chile := r.NewInode(ctx, &fs.MemRegularFile{
			Data: []byte("hogeeee"),
			Attr: fuse.Attr{
				Mode: 0444,
			},
		}, fs.StableAttr{Mode: syscall.S_IFREG})

		out.Mode = 0444
		out.Size = 1
		return chile, 0
		// ディレクトリの場合
	} else if name == "hoge" {
		chile := r.NewInode(ctx, &Directory{name: "hoge"}, fs.StableAttr{Mode: syscall.S_IFDIR})

		// ディレクトリであるフラグとファイルパーミッションをORでビット演算している
		out.Mode = syscall.S_IFDIR | 0755
		return chile, 0
	}

	return nil, syscall.ENOENT
}

func (r *HelloRoot) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	// TODO: ここでは、localstackのルートディレクトリ配下のファイル、ディレクトリを返す
	//  以下は例として仮想的なファイルとディレクトリを返している
	fmt.Println("readdir")
	entry := []fuse.DirEntry{
		{
			Mode: syscall.S_IFREG,
			Name: "hoge.txt",
		},
		{
			Mode: syscall.S_IFDIR,
			Name: "hoge",
		},
	}
	return fs.NewListDirStream(entry), 0
}

func (d *Directory) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	if name == "children.txt" {
		chile := d.NewInode(ctx, &fs.MemRegularFile{
			Data: []byte("childrennnnn"),
			Attr: fuse.Attr{
				Mode: 0444,
			},
		}, fs.StableAttr{Mode: syscall.S_IFREG})

		out.Mode = 0444
		out.Size = 1
		return chile, 0
	}

	return nil, syscall.ENOENT
}

func (d *Directory) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	// TODO: Directoryが保持しているnameでlocalstackを検索してあげる
	//  以下は例として明示的にディレクトリ名とファイル名を指定している
	if d.name == "hoge" {
		entry := []fuse.DirEntry{
			{
				Mode: syscall.S_IFREG,
				Name: "children.txt",
			},
		}
		return fs.NewListDirStream(entry), 0
	}
	return nil, syscall.ENOENT
}

// これは型アサーションをすることでinterfaceの実装ミスをコンパイル時に防ぐために定義している
// 以下詳細：
// (*HelloRoot)(nil)でHelloRoot型のnilポインタを返す
// (fs.NodeGetattrer)((*HelloRoot)(nil))で↑で作成したHelloRoot型のnilポインタをfs.NodeGetattrer型に型アサーションしようとしている
// こうすることで、HelloRoot構造体が、fs.NodeGetattrer interfaceを実装していない場合にコンパイルエラーが発生するので、コンパイル時に実装ミスに気づける
var _ = (fs.NodeGetattrer)((*HelloRoot)(nil))
var _ = (fs.NodeReaddirer)((*HelloRoot)(nil))
var _ = (fs.NodeLookuper)((*HelloRoot)(nil))

func main() {
	// fs.api.goに定義されているそれぞれのinterfaceを実装することで、ファイルシステムに対するシステムコールをハンドリングできるようになる
	// 例えば、Readdirメソッドを実装すると、lsコマンドで発行されるシステムコールをgoプロセス内でハンドリングできる
	opts := &fs.Options{}
	// ルートディレクトリにマウントしている
	server, err := fs.Mount("/tmp/myown-filesystem", &HelloRoot{}, opts)
	if err != nil {
		log.Fatal("fatal mount")
	}
	server.Wait()
}
