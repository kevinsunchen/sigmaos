package rpcclnt

import (
	"path"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/proto"

	db "sigmaos/debug"
	"sigmaos/fslib"
	"sigmaos/protdev"
	rpcproto "sigmaos/protdev/proto"
	"sigmaos/serr"
	"sigmaos/sessdevclnt"
	"sigmaos/sessp"
	sp "sigmaos/sigmap"
)

type RPCClnt struct {
	fsls []*fslib.FsLib
	fds  []int
	si   *protdev.StatInfo
	sdc  *sessdevclnt.SessDevClnt
	pn   string
	idx  int32
}

func MkRPCClnt(fsls []*fslib.FsLib, pn string) (*RPCClnt, error) {
	rpcc := &RPCClnt{
		fsls: make([]*fslib.FsLib, 0, len(fsls)),
		fds:  make([]int, 0, len(fsls)),
		si:   protdev.MakeStatInfo(),
		pn:   pn,
	}
	sdc, err := sessdevclnt.MkSessDevClnt(fsls[0], path.Join(pn, protdev.RPC))
	if err != nil {
		return nil, err
	}
	for _, fsl := range fsls {
		rpcc.fsls = append(rpcc.fsls, fsl)
		n, err := fsl.Open(sdc.DataPn(), sp.ORDWR)
		if err != nil {
			return nil, err
		}
		rpcc.fds = append(rpcc.fds, n)
	}
	return rpcc, nil
}

func (rpcc *RPCClnt) rpc(method string, a []byte, f *sessp.Tfence) (*rpcproto.Reply, error) {
	req := rpcproto.Request{}
	req.Method = method
	req.Args = a

	b, err := proto.Marshal(&req)
	if err != nil {
		return nil, serr.MkErrError(err)
	}

	start := time.Now()
	idx := int(atomic.AddInt32(&rpcc.idx, 1))
	b, err = rpcc.fsls[idx%len(rpcc.fsls)].WriteRead(rpcc.fds[idx%len(rpcc.fds)], b, f)
	if err != nil {
		return nil, err
	}
	// Record stats
	rpcc.si.Stat(method, time.Since(start).Microseconds())

	rep := &rpcproto.Reply{}
	if err := proto.Unmarshal(b, rep); err != nil {
		return nil, serr.MkErrError(err)
	}

	return rep, nil
}

func (rpcc *RPCClnt) RPCFence(method string, arg proto.Message, res proto.Message, f *sessp.Tfence) error {
	b, err := proto.Marshal(arg)
	if err != nil {
		return err
	}
	rep, err := rpcc.rpc(method, b, f)
	if err != nil {
		return err
	}
	if rep.Err.ErrCode != 0 {
		return sp.MkErr(rep.Err)
	}
	if err := proto.Unmarshal(rep.Res, res); err != nil {
		return err
	}
	return nil
}

func (rpcc *RPCClnt) RPC(method string, arg proto.Message, res proto.Message) error {
	return rpcc.RPCFence(method, arg, res, sessp.NewFence())
}

func (rpcc *RPCClnt) StatsClnt() map[string]*protdev.MethodStat {
	return rpcc.si.Stats()
}

func (rpcc *RPCClnt) StatsSrv() (*protdev.SigmaRPCStats, error) {
	stats := &protdev.SigmaRPCStats{}
	if err := rpcc.fsls[0].GetFileJson(path.Join(rpcc.pn, protdev.RPC, protdev.STATS), stats); err != nil {
		db.DFatalf("Error getting stats")
		return nil, err
	}
	return stats, nil
}
