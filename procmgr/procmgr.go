package procmgr

import (
	"sync"

	db "sigmaos/debug"
	"sigmaos/memfssrv"
	"sigmaos/proc"
	"sigmaos/procclnt"
	"sigmaos/sigmaclnt"
	sp "sigmaos/sigmap"
	"sigmaos/uprocclnt"
)

type ProcMgr struct {
	sync.Mutex
	mfs      *memfssrv.MemFs
	scheddIp string
	rootsc   *sigmaclnt.SigmaClnt
	updm     *uprocclnt.UprocdMgr
	sclnts   map[sp.Trealm]*sigmaclnt.SigmaClnt
	running  map[proc.Tpid]*proc.Proc
}

// Manages the state and lifecycle of a proc.
func MakeProcMgr(mfs *memfssrv.MemFs) *ProcMgr {
	return &ProcMgr{
		mfs:      mfs,
		scheddIp: mfs.MyAddr(),
		rootsc:   mfs.SigmaClnt(),
		updm:     uprocclnt.MakeUprocdMgr(mfs.SigmaClnt().FsLib),
		sclnts:   make(map[sp.Trealm]*sigmaclnt.SigmaClnt),
		running:  make(map[proc.Tpid]*proc.Proc),
	}
}

// Proc has been spawned.
func (mgr *ProcMgr) Spawn(p *proc.Proc) {
	mgr.postProcInQueue(p)
}

func (mgr *ProcMgr) RunProc(p *proc.Proc) {
	p.Finalize(mgr.scheddIp)
	mgr.setupProcState(p)
	mgr.downloadProc(p)
	mgr.runProc(p)
	mgr.teardownProcState(p)
}

// Try to steal a proc from another schedd. Must be callled after RPCing the
// victim schedd.
func (mgr *ProcMgr) TryStealProc(p *proc.Proc) {
	// Remove the proc from the ws queue. This can only be done *after* RPCing
	// schedd. Otherwise, if this proc crashes after removing the stealable proc
	// but before claiming it from the victim schedd, the proc will not be added
	// back to the WS queue, and other schedds will not have the opportunity to
	// steal it.
	//
	// It is safe, however, to remove the proc regardless of whether or not the
	// steal is actually successful. If the steal is unsuccessful, that means
	// another schedd was granted the proc by the victim, and will remove it
	// anyway. Eagerly removing it here stops additional schedds from trying to
	// steal it in the intervening time.
	mgr.removeWSLink(p)
}

func (mgr *ProcMgr) OfferStealableProc(p *proc.Proc) {
	mgr.createWSLink(p)
}

func (mgr *ProcMgr) getSigmaClnt(realm sp.Trealm) *sigmaclnt.SigmaClnt {
	mgr.Lock()
	defer mgr.Unlock()

	var clnt *sigmaclnt.SigmaClnt
	var ok bool
	if clnt, ok = mgr.sclnts[realm]; !ok {
		// No need to make a new client for the root realm.
		if realm == sp.Trealm(proc.GetRealm()) {
			clnt = &sigmaclnt.SigmaClnt{mgr.rootsc.FsLib, nil}
		} else {
			var err error
			if clnt, err = sigmaclnt.MkSigmaClntRealm(mgr.rootsc.FsLib, sp.SCHEDDREL, realm); err != nil {
				db.DFatalf("Err MkSigmaClntRealm: %v", err)
			}
			// Mount KPIDS.
			procclnt.MountPids(clnt.FsLib, clnt.FsLib.NamedAddr())
		}
		mgr.sclnts[realm] = clnt
	}
	return clnt
}