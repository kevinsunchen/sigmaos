package memfs

import (
	"errors"
	"fmt"
	"log"

	np "ulambda/ninep"
)

type Tinum uint64
type Tversion uint32

const (
	NullInum Tinum = 0
	RootInum Tinum = 1
)

type DataLen interface {
	Len() np.Tlength
}

type Dev interface {
	Write(np.Toffset, []byte) (np.Tsize, error)
	Read(np.Toffset, np.Tsize) ([]byte, error)
	Len() np.Tlength
}

type Inode struct {
	PermT   np.Tperm
	Inum    Tinum
	Version Tversion
	Data    DataLen
}

func makeInode(t np.Tperm, inum Tinum, data DataLen) *Inode {
	i := Inode{}
	i.PermT = t
	i.Inum = inum
	i.Data = data
	return &i
}

func (inode *Inode) String() string {
	str := fmt.Sprintf("Inode %v t 0x%x data %v {}", inode.Inum, inode.PermT>>np.TYPESHIFT,
		inode.Data)
	return str
}

func (inode *Inode) Qid() np.Tqid {
	return np.MakeQid(
		np.Qtype(inode.PermT>>np.QTYPESHIFT),
		np.TQversion(inode.Version),
		np.Tpath(inode.Inum))
}

func (inode *Inode) IsDir() bool {
	return np.IsDir(inode.PermT)
}

func (inode *Inode) IsSymlink() bool {
	return np.IsSymlink(inode.PermT)
}

func (inode *Inode) IsDev() bool {
	return np.IsDevice(inode.PermT)
}

func (inode *Inode) IsPipe() bool {
	return np.IsPipe(inode.PermT)
}

func (inode *Inode) IsDevice() bool {
	return np.IsDevice(inode.PermT)
}

func permToDataLen(t np.Tperm) (DataLen, error) {
	if np.IsDir(t) {
		return makeDir(), nil
	} else if np.IsSymlink(t) {
		return MakeSym(), nil
	} else if np.IsPipe(t) {
		return MakePipe(), nil
	} else if np.IsDevice(t) {
		return nil, nil
	} else if np.IsFile(t) {
		return MakeFile(), nil
	} else {
		return nil, errors.New("Unknown type")
	}
}

func (inode *Inode) Create(root *Root, t np.Tperm, name string) (*Inode, error) {
	if IsCurrentDir(name) {
		return nil, errors.New("Cannot create name")
	}
	if inode.IsDir() {
		d := inode.Data.(*Dir)
		dl, err := permToDataLen(t)
		if err != nil {
			return nil, err
		}
		i := makeInode(t, root.allocInum(), dl)
		if i.IsDir() {
			dir := inode.Data.(*Dir)
			dir.init(i.Inum)
		}
		log.Printf("create %v -> %v\n", name, i)
		return i, d.Create(i, name)
	} else {
		return nil, errors.New("Not a directory")
	}
}

func (inode *Inode) Mode() np.Tperm {
	perm := np.Tperm(0777)
	if inode.IsDir() {
		perm |= np.DMDIR
	}
	return perm
}

func (inode *Inode) Stat() *np.Stat {
	stat := &np.Stat{}
	stat.Type = 0 // XXX
	stat.Qid = inode.Qid()
	stat.Mode = inode.Mode()
	stat.Mtime = 0
	stat.Atime = 0
	stat.Length = inode.Data.Len()
	stat.Name = ""
	stat.Uid = "kaashoek"
	stat.Gid = "kaashoek"
	stat.Muid = ""
	return stat
}

func (inode *Inode) Walk(path []string) ([]*Inode, []string, error) {
	log.Printf("Walk %v at %v\n", path, inode)
	inodes := []*Inode{inode}
	if len(path) == 0 {
		return inodes, nil, nil
	}
	dir, ok := inode.Data.(*Dir)
	if !ok {
		return nil, nil, errors.New("Not a directory")
	}
	inodes, rest, err := dir.Namei(path, inodes)
	if err == nil {
		return inodes, rest, err
		// switch inodes[len(inodes)-1].PermT {
		// case MountT:
		// 	// uf := inode.Data.(*fid.Ufid)
		// 	return nil, rest, err
		// case SymT:
		// 	// s := inode.Data.(*Symlink)
		// 	return nil, rest, err
		// default:
	} else {
		return nil, nil, err
	}
}

// Lookup a directory or file. If file, return parent dir and inode
// for file.  If directory, return it
func (inode *Inode) LookupPath(path []string) (*Dir, *Inode, error) {
	inodes, rest, err := inode.Walk(path)
	if err != nil {
		return nil, nil, err
	}
	if len(rest) != 0 {
		return nil, nil, errors.New("Unknown name")
	}
	i := inodes[len(inodes)-1]
	if i.IsDir() {
		return i.Data.(*Dir), nil, nil
	} else {
		// there must be a parent
		di := inodes[len(inodes)-2]
		dir, ok := di.Data.(*Dir)
		if !ok {
			log.Fatal("Lookup: cast error")
		}
		return dir, inodes[len(inodes)-1], nil
	}
}

func (inode *Inode) Remove(root *Root, path []string) error {
	dir, ino, err := inode.LookupPath(path)
	if err != nil {
		return err
	}
	err = dir.Remove(path[len(path)-1])
	if err != nil {
		log.Fatal("Remove error ", err)
	}
	root.freeInum(ino.Inum)
	return nil
}

func (inode *Inode) Write(offset np.Toffset, data []byte) (np.Tsize, error) {
	log.Print("fs.Writei ", inode)
	if inode.IsDevice() {
		d := inode.Data.(Dev)
		return d.Write(offset, data)
	} else if inode.IsDir() {
		return 0, errors.New("Cannot write directory")
	} else if inode.IsSymlink() {
		s := inode.Data.(*Symlink)
		return s.Write(data)
	} else if inode.IsPipe() {
		p := inode.Data.(*Pipe)
		return p.Write(data)
	} else {
		f := inode.Data.(*File)
		return f.Write(offset, data)
	}
}

func (inode *Inode) Read(offset np.Toffset, n np.Tsize) ([]byte, error) {
	log.Print("fs.Readi ", inode)
	if inode.IsDevice() {
		d := inode.Data.(Dev)
		return d.Read(offset, n)
	} else if inode.IsDir() {
		d := inode.Data.(*Dir)
		return d.Read(offset, n)
	} else if inode.IsSymlink() {
		s := inode.Data.(*Symlink)
		return s.Read(n)
	} else if inode.IsPipe() {
		p := inode.Data.(*Pipe)
		return p.Read(n)
	} else {
		f := inode.Data.(*File)
		return f.Read(offset, n)
	}
}
