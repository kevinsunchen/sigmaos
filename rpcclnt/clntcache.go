package rpcclnt

import (
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	db "sigmaos/debug"
	"sigmaos/fslib"
	"sigmaos/pathclnt"
	"sigmaos/protdev"
	"sigmaos/serr"
)

type ClntCache struct {
	sync.Mutex
	fsl   *fslib.FsLib
	rpccs map[string]*RPCClnt
}

func NewRPCClntCache(fsl *fslib.FsLib) *ClntCache {
	return &ClntCache{fsl: fsl, rpccs: make(map[string]*RPCClnt)}
}

// Note: several threads may call MkRPCClnt for same pn,
// overwriting the rpcc of the last thread that called NewClnt.
func (cc *ClntCache) Lookup(pn string) (*RPCClnt, error) {
	cc.Lock()
	defer cc.Unlock()
	rpcc, ok := cc.rpccs[pn]
	if ok {
		return rpcc, nil
	}
	cc.Unlock()
	rpcc, err := MkRPCClnt([]*fslib.FsLib{cc.fsl}, pn)
	cc.Lock()
	if err != nil {
		return nil, err
	}
	cc.rpccs[pn] = rpcc
	return rpcc, nil
}

func (cc *ClntCache) Delete(pn string) {
	cc.Lock()
	defer cc.Unlock()
	delete(cc.rpccs, pn)
}

func (cc *ClntCache) RPC(pn string, method string, arg proto.Message, res proto.Message) error {
	for i := 0; i < pathclnt.MAXRETRY; i++ {
		rpcc, err := cc.Lookup(pn)
		if err != nil {
			return err
		}
		if err := rpcc.RPC(method, arg, res); err == nil {
			return nil
		} else {
			cc.Delete(pn)
			if serr.IsErrCode(err, serr.TErrUnreachable) {
				time.Sleep(pathclnt.TIMEOUT * time.Millisecond)
				db.DPrintf(db.ALWAYS, "RPC: retry %v %v\n", pn, method)
				continue
			}
			return err
		}
	}
	return serr.MkErr(serr.TErrUnreachable, pn)
}

func (cc *ClntCache) StatsSrv() ([]*protdev.SigmaRPCStats, error) {
	stats := make([]*protdev.SigmaRPCStats, 0, len(cc.rpccs))
	for _, rpcc := range cc.rpccs {
		st, err := rpcc.StatsSrv()
		if err != nil {
			return nil, err
		}
		stats = append(stats, st)
	}
	return stats, nil
}

func (cc *ClntCache) StatsClnt() []map[string]*protdev.MethodStat {
	stats := make([]map[string]*protdev.MethodStat, 0, len(cc.rpccs))
	for _, rpcc := range cc.rpccs {
		stats = append(stats, rpcc.StatsClnt())
	}
	return stats
}