package fslib

import (
	db "sigmaos/debug"
	"sigmaos/fdclnt"
	"sigmaos/proc"
	sos "sigmaos/sigmaos"
	sp "sigmaos/sigmap"
)

type FsLib struct {
	pcfg *proc.ProcEnv
	sos.SigmaOS
}

// Only to be called by procs.
func NewFsLib(pcfg *proc.ProcEnv) (*FsLib, error) {
	db.DPrintf(db.PORT, "NewFsLib: uname %s lip %s addrs %v\n", pcfg.GetUname(), pcfg.LocalIP, pcfg.EtcdIP)
	fl := &FsLib{
		pcfg:    pcfg,
		SigmaOS: fdclnt.NewFdClient(pcfg, nil),
	}
	return fl, nil
}

func (fl *FsLib) GetLocalIP() string {
	return fl.pcfg.GetLocalIP()
}

func (fl *FsLib) ProcEnv() *proc.ProcEnv {
	return fl.pcfg
}

func (fl *FsLib) MountTree(addrs sp.Taddrs, tree, mount string) error {
	return fl.SigmaOS.MountTree(addrs, tree, mount)
}

func (fl *FsLib) DetachAll() error {
	return fl.SigmaOS.DetachAll()
}
