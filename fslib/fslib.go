package fslib

import (
	db "sigmaos/debug"
	"sigmaos/fdclnt"
	"sigmaos/proc"
	sp "sigmaos/sigmap"
)

type FsLib struct {
	pcfg *proc.ProcEnv
	*fdclnt.FdClient
}

// Only to be called by procs.
func NewFsLib(pcfg *proc.ProcEnv) (*FsLib, error) {
	db.DPrintf(db.PORT, "NewFsLib: uname %s lip %s addrs %v\n", pcfg.GetUname(), pcfg.LocalIP, pcfg.EtcdIP)
	fl := &FsLib{
		pcfg:     pcfg,
		FdClient: fdclnt.NewFdClient(pcfg, nil),
	}
	return fl, nil
}

func (fl *FsLib) ProcEnv() *proc.ProcEnv {
	return fl.pcfg
}

func (fl *FsLib) MountTree(addrs sp.Taddrs, tree, mount string) error {
	return fl.FdClient.MountTree(fl.pcfg.GetUname(), addrs, tree, mount)
}

func (fl *FsLib) DetachAll() error {
	return fl.PathClnt.DetachAll()
}
