package procqsrv

import (
	"path"
	"sync"

	db "sigmaos/debug"
	"sigmaos/fs"
	"sigmaos/memfssrv"
	"sigmaos/perf"
	"sigmaos/proc"
	proto "sigmaos/procqsrv/proto"
	sp "sigmaos/sigmap"
	"sigmaos/sigmasrv"
)

type ProcQ struct {
	mu   sync.Mutex
	cond *sync.Cond
	mfs  *memfssrv.MemFs
	qs   map[sp.Trealm]*Queue
}

func NewProcQ(mfs *memfssrv.MemFs) *ProcQ {
	pq := &ProcQ{
		mfs: mfs,
		qs:  make(map[sp.Trealm]*Queue),
	}
	pq.cond = sync.NewCond(&pq.mu)
	return pq
}

func (pq *ProcQ) Enqueue(ctx fs.CtxI, req proto.EnqueueRequest, res *proto.EnqueueResponse) error {
	p := proc.NewProcFromProto(req.ProcProto)
	db.DPrintf(db.PROCQ, "[%v] Enqueued %v", p.GetRealm(), p)

	ch := pq.addProc(p)
	res.KernelID = <-ch
	return nil
}

func (pq *ProcQ) addProc(p *proc.Proc) chan string {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	q, ok := pq.qs[p.GetRealm()]
	if !ok {
		q = pq.addRealmQueueL(p.GetRealm())
	}
	// Enqueue the proc according to its realm
	ch := q.Enqueue(p)
	// Signal that a new proc may be runnable.
	pq.cond.Signal()
	return ch
}

func (pq *ProcQ) GetProc(ctx fs.CtxI, req proto.GetProcRequest, res *proto.GetProcResponse) error {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	// XXX seems fishy to loop forever...
	for {
		// XXX Should probably do this more efficiently (just select a realm).
		// Iterate through the realms round-robin.
		for r, q := range pq.qs {
			p, ch, ok := q.Dequeue()
			if ok {
				db.DPrintf(db.PROCQ, "[%v] Dequeued %v", r, p)
				res.ProcProto = p.GetProto()
				ch <- req.KernelID
				return nil
			}
		}
		// If unable to schedule a proc from any realm, wait.
		db.DPrintf(db.PROCQ, "No procs schedulable qs:%v", pq.qs)
		pq.cond.Wait()
	}
	return nil
}

// Caller must hold lock.
func (pq *ProcQ) addRealmQueueL(realm sp.Trealm) *Queue {
	q := newQueue()
	pq.qs[realm] = q
	return q
}

// Run a ProcQ
func Run() {
	pcfg := proc.GetProcEnv()
	mfs, err := memfssrv.NewMemFs(path.Join(sp.PROCQ, pcfg.GetPID().String()), pcfg)
	if err != nil {
		db.DFatalf("Error NewMemFs: %v", err)
	}
	pq := NewProcQ(mfs)
	ssrv, err := sigmasrv.NewSigmaSrvMemFs(mfs, pq)
	if err != nil {
		db.DFatalf("Error PDS: %v", err)
	}
	setupMemFsSrv(ssrv.MemFs)
	setupFs(ssrv.MemFs)
	// Perf monitoring
	p, err := perf.NewPerf(pcfg, perf.PROCQ)
	if err != nil {
		db.DFatalf("Error NewPerf: %v", err)
	}
	defer p.Done()

	ssrv.RunServer()
}
