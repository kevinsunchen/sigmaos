package uprocclnt

import (
	"path"
	"sync"

	"github.com/docker/docker/client"

	"sigmaos/container"
	db "sigmaos/debug"
	"sigmaos/fslib"
	"sigmaos/proc"
	"sigmaos/procclnt"
	"sigmaos/protdevclnt"
	sp "sigmaos/sigmap"
	"sigmaos/uprocsrv/proto"
)

type UprocClnt struct {
	pdc       *protdevclnt.ProtDevClnt
	cli       *client.Client
	container string
}

type UprocdMgr struct {
	mu    sync.Mutex
	fsl   *fslib.FsLib
	pclnt *procclnt.ProcClnt
	upc   *UprocClnt // XXX one per realm
}

func MakeUprocdMgr(fsl *fslib.FsLib, pclnt *procclnt.ProcClnt) *UprocdMgr {
	updm := &UprocdMgr{fsl: fsl, pclnt: pclnt}
	return updm
}

func (updm *UprocdMgr) lookupClnt(realm string) (*UprocClnt, error) {
	updm.mu.Lock()
	defer updm.mu.Unlock()
	if updm.upc == nil {
		u, err := updm.StartUprocd(realm)
		if err != nil {
			return nil, err
		}
		updm.upc = u
	}
	return updm.upc, nil
}

func (updm *UprocdMgr) MakeUProc(uproc *proc.Proc, realm string) error {
	upc, err := updm.lookupClnt(realm)
	if err != nil {
		return err
	}
	req := &proto.RunRequest{
		ProcProto: uproc.GetProto(),
	}
	res := &proto.RunResponse{}
	err = upc.pdc.RPC("UprocSrv.Run", req, res)
	if err != nil {
		return err
	}
	return nil
}

// Spawn uprocd in side of docker container
func (updm *UprocdMgr) StartUprocd(realm string) (*UprocClnt, error) {
	u := &UprocClnt{}
	program := "uprocd"
	args := []string{realm}
	pid := proc.Tpid(program + "-" + proc.GenPid().String())
	p := proc.MakePrivProcPid(pid, program, args, true)

	if err := updm.pclnt.SpawnContainer(p, fslib.Named(), realm); err != nil {
		return nil, err
	}

	// XXX don't hard code
	p.AppendEnv("PATH", "/home/sigmaos/bin/user:/home/sigmaos/bin/kernel")
	p.FinalizeEnv("NONE")

	cli, c, err := container.MkContainer(p, realm)
	if err != nil {
		return nil, err
	}

	u.cli = cli
	u.container = c
	db.DPrintf(db.CONTAINER, "container %s\n", u.container[:10])
	updm.pclnt.WaitStart(p.GetPid())
	db.DPrintf(db.CONTAINER, "container started %s\n", u.container[:10])
	pn := path.Join(sp.PROCD, "~local", sp.UPROCDREL)
	pdc, err := protdevclnt.MkProtDevClnt(updm.fsl, pn)
	if err != nil {
		return nil, err
	}
	u.pdc = pdc
	return u, nil
}
