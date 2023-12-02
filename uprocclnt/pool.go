package uprocclnt

import (
	"sync"

	db "sigmaos/debug"
	sp "sigmaos/sigmap"
)

const (
	POOL_SZ = 1 // Size of running-but-unused pool uprocds to be maintained at all times
)

// A pool of booted, but unused, uprocds.
type pool struct {
	sync.Mutex
	cond        *sync.Cond
	startUprocd startUprocdFn
	clnts       []*UprocdClnt
	pids        []sp.Tpid
}

func newPool(fn startUprocdFn) *pool {
	p := &pool{
		startUprocd: fn,
		clnts:       make([]*UprocdClnt, 0, POOL_SZ),
		pids:        make([]sp.Tpid, 0, POOL_SZ),
	}
	p.cond = sync.NewCond(&p.Mutex)
	return p
}

// Fill the pool.
func (p *pool) fill() {
	p.Lock()
	defer p.Unlock()

	db.DPrintf(db.UPROCDMGR, "Fill uprocd pool len %v target %v", len(p.clnts), POOL_SZ)
	for len(p.clnts) < POOL_SZ {
		pid, clnt := p.startUprocd()
		p.pids = append(p.pids, pid)
		p.clnts = append(p.clnts, clnt)
	}
	db.DPrintf(db.UPROCDMGR, "Done Fill uprocd pool len %v target %v", len(p.clnts), POOL_SZ)
	p.cond.Broadcast()
}

func (p *pool) get() (sp.Tpid, *UprocdClnt) {
	p.Lock()
	defer p.Unlock()

	// Wait for there to be available uprocds in the pool.
	for len(p.clnts) == 0 {
		db.DPrintf(db.UPROCDMGR, "Wait for uprocd pool to be filled len %v", len(p.clnts))
		p.cond.Wait()
	}
	db.DPrintf(db.UPROCDMGR, "Pop from uprocd pool")

	var pid sp.Tpid
	var clnt *UprocdClnt

	// Pop from the pool of uprocds.
	pid, p.pids = p.pids[0], p.pids[1:]
	clnt, p.clnts = p.clnts[0], p.clnts[1:]

	// Refill asynchronously.
	go p.fill()

	return pid, clnt
}
