package memfssrv

import (
	"sigmaos/ctx"
	db "sigmaos/debug"
	"sigmaos/dir"
	"sigmaos/fs"
	"sigmaos/fslib"
	"sigmaos/fslibsrv"
	"sigmaos/lockmap"
	"sigmaos/memfs"
	"sigmaos/proc"
	"sigmaos/procclnt"
	"sigmaos/repl"
	"sigmaos/serr"
	"sigmaos/sesssrv"
)

//
// Servers use memfsssrv to create an in-memory file server.
// memfsssrv uses sesssrv and protsrv to handle client sigmaP
// requests.
//

type MemFs struct {
	*sesssrv.SessSrv
	root     fs.Dir
	ctx      fs.CtxI // server context
	plt      *lockmap.PathLockTable
	fsl      *fslib.FsLib
	procclnt *procclnt.ProcClnt
}

func MakeReplMemFs(addr string, path string, name string, conf repl.Config) (*sesssrv.SessSrv, *serr.Err) {
	root := dir.MkRootDir(ctx.MkCtx("", 0, nil), memfs.MakeInode)
	isInitNamed := false
	// Check if we are one of the initial named replicas
	for _, a := range proc.Named() {
		if a == addr {
			isInitNamed = true
			break
		}
	}
	var srv *sesssrv.SessSrv
	var err error
	if isInitNamed {
		srv, err = fslibsrv.MakeReplServerFsl(root, addr, path, nil, nil, conf)
	} else {
		// If this is not the init named, initialize the fslib & procclnt
		srv, _, _, err = fslibsrv.MakeReplServer(root, addr, path, name, conf)
	}
	if err != nil {
		return nil, serr.MkErrError(err)
	}
	// If this *was* the init named, we now need to init fsl
	if isInitNamed {
		// Server is running, make an fslib for it, mounting itself, to ensure that
		// srv can call checkLock
		fsl, err := fslib.MakeFsLib(name)
		if err != nil {
			return nil, serr.MkErrError(err)
		}
		srv.SetFsl(fsl)
	}
	return srv, nil
}

func MakeReplMemFsFsl(addr string, path string, fsl *fslib.FsLib, pclnt *procclnt.ProcClnt, conf repl.Config) (*sesssrv.SessSrv, *serr.Err) {
	root := dir.MkRootDir(ctx.MkCtx("", 0, nil), memfs.MakeInode)
	srv, err := fslibsrv.MakeReplServerFsl(root, addr, path, fsl, pclnt, conf)
	if err != nil {
		db.DFatalf("Error makeReplMemfsFsl: err")
	}
	return srv, nil
}

func MakeMemFs(pn string, name string) (*MemFs, *fslib.FsLib, *procclnt.ProcClnt, error) {
	fsl, err := fslib.MakeFsLib(name)
	if err != nil {
		return nil, nil, nil, err
	}
	pclnt := procclnt.MakeProcClnt(fsl)
	fs, err := MakeMemFsFsl(pn, fsl, pclnt)
	return fs, fsl, pclnt, err
}

func MakeMemFsFsl(pn string, fsl *fslib.FsLib, pclnt *procclnt.ProcClnt) (*MemFs, error) {
	fs := &MemFs{}
	root := dir.MkRootDir(ctx.MkCtx("", 0, nil), memfs.MakeInode)
	srv, err := fslibsrv.MakeSrv(root, pn, fsl, pclnt)
	if err != nil {
		return nil, err
	}
	fs.SessSrv = srv
	fs.plt = srv.GetPathLockTable()
	fs.fsl = fsl
	fs.procclnt = pclnt
	fs.root = root
	fs.ctx = ctx.MkCtx(pn, 0, nil)
	return fs, err
}
