package procd

import (
	"encoding/json"
	"fmt"
	"path"

	db "ulambda/debug"
	"ulambda/fs"
	"ulambda/inode"
	np "ulambda/ninep"
	"ulambda/proc"
	"ulambda/semclnt"
)

type SpawnFile struct {
	pd *Procd
	fs.Inode
}

func makeSpawnFile(pd *Procd, ctx fs.CtxI, parent fs.Dir) *SpawnFile {
	i := inode.MakeInode(ctx, 0, parent)
	return &SpawnFile{pd, i}
}

func (ctl *SpawnFile) Read(ctx fs.CtxI, off np.Toffset, cnt np.Tsize, v np.TQversion) ([]byte, *np.Err) {
	return nil, np.MkErr(np.TErrNotSupported, "Read")
}

func (ctl *SpawnFile) Write(ctx fs.CtxI, off np.Toffset, b []byte, v np.TQversion) (np.Tsize, *np.Err) {
	p := proc.MakeEmptyProc()
	err := json.Unmarshal(b, p)
	if err != nil {
		np.MkErr(np.TErrInval, fmt.Sprintf("Unmarshal %v", err))
	}

	db.DPrintf("PROCD", "Control file write: %v", p)

	// Create an ephemeral semaphore to indicate a proc has started.
	semStart := semclnt.MakeSemClnt(ctl.pd.FsLib, path.Join(p.ParentDir, proc.START_SEM))
	semStart.Init(np.DMTMP)

	db.DPrintf("PROCD", "Sem init done: %v", p)

	ctl.pd.fs.spawn(p, b)

	db.DPrintf("PROCD", "fs spawn done: %v", p)

	return np.Tsize(len(b)), nil
}
