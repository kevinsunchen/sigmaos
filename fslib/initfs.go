package fslib

import (
	"fmt"
	"log"

	db "ulambda/debug"
	"ulambda/memfs"
	"ulambda/memfsd"
	"ulambda/npsrv"
)

type FsLibSrv struct {
	*FsLib
	*memfsd.Fsd
	srv *npsrv.NpServer
}

func (fsl *FsLib) PostService(srvaddr, srvname string) error {
	err := fsl.Remove(srvname)
	if err != nil {
		db.DPrintf("Remove failed %v %v\n", srvname, err)
	}
	err = fsl.Symlink(srvaddr+":pubkey:"+srvname, srvname, 0777)
	return err
}

func InitFsMemFsD(name string, memfs *memfs.Root, memfsd *memfsd.Fsd, dev memfs.Dev) (*FsLibSrv, error) {
	srv := npsrv.MakeNpServer(memfsd, ":0")
	fsl := &FsLibSrv{MakeFsLib(name), memfsd, srv}
	fs := memfsd.Root()
	if dev != nil {
		_, err := fs.MkNod(fsl.Uname(), fs.RootInode(),
			"dev", dev)
		if err != nil {
			log.Fatal("Create error: dev: ", err)
		}
	}
	err := fsl.PostService(fsl.srv.MyAddr(), name)
	if err != nil {
		return nil, fmt.Errorf("PostService %v error: %v\n", name, err)
	}
	return fsl, nil
}

func InitFsMemFs(name string, memfs *memfs.Root, dev memfs.Dev) (*FsLibSrv, error) {
	memfsd := memfsd.MakeFsd(memfs, nil)
	return InitFsMemFsD(name, memfs, memfsd, dev)
}

func InitFs(name string, dev memfs.Dev) (*FsLibSrv, error) {
	fs := memfs.MakeRoot()
	fsd := memfsd.MakeFsd(fs, nil)
	return InitFsMemFsD(name, fs, fsd, dev)
}

func (fsl *FsLib) ExitFs(name string) {
	err := fsl.Remove(name)
	if err != nil {
		db.DPrintf("Remove failed %v %v\n", name, err)
	}
}
