package procqprvdrsrv

import (
	"fmt"
	"sync"
	"time"

	"sigmaos/proc"
	sp "sigmaos/sigmap"
)

const (
	DEF_Q_SZ = 10
)

type Qitem struct {
	p     *proc.Proc
	kidch chan string
	enqTS time.Time
}

func newQitem(p *proc.Proc) *Qitem {
	return &Qitem{
		p:     p,
		kidch: make(chan string),
		enqTS: time.Now(),
	}
}

type QueuePrvdr struct {
	sync.Mutex
	procsbyprvdr map[proc.Tprovider][]*Qitem
	pmap         map[sp.Tpid]*proc.Proc
}

func newQueuePrvdr() *QueuePrvdr {
	return &QueuePrvdr{
		procsbyprvdr: make(map[proc.Tprovider][]*Qitem, 0),
		pmap:         make(map[sp.Tpid]*proc.Proc, 0),
	}
}

func (q *QueuePrvdr) Enqueue(p *proc.Proc) chan string {
	q.Lock()
	defer q.Unlock()

	prvdr := p.GetProvider()
	q.pmap[p.GetPid()] = p
	qi := newQitem(p)

	// For now, default to AWS for procs labeled for any provider. In the future,
	// could add intelligent scheduling here (e.g. SkyPilot optimizer)
	if prvdr == proc.T_ANY {
		prvdr = proc.T_AWS
	}

	if qforprvdr, ok := q.procsbyprvdr[prvdr]; ok {
		q.procsbyprvdr[prvdr] = append(qforprvdr, qi)
	} else {
		q.procsbyprvdr[prvdr] = make([]*Qitem, 0, DEF_Q_SZ)
	}

	return qi.kidch
}

func (q *QueuePrvdr) Dequeue(prvdr proc.Tprovider) (*proc.Proc, chan string, time.Time, bool) {
	q.Lock()
	defer q.Unlock()

	var p *proc.Proc
	var ok bool
	var kidch chan string = nil
	var enqTS time.Time
	if len(q.procsbyprvdr) > 0 {
		var qi *Qitem
		qi, q.procsbyprvdr[prvdr] = q.procsbyprvdr[prvdr][0], q.procsbyprvdr[prvdr][1:]
		p = qi.p
		kidch = qi.kidch
		enqTS = qi.enqTS
		ok = true
		delete(q.pmap, qi.p.GetPid())
	}
	return p, kidch, enqTS, ok
}

func (q *QueuePrvdr) String() string {
	q.Lock()
	defer q.Unlock()

	return fmt.Sprintf("{ procs:%v }", q.procsbyprvdr)
}
