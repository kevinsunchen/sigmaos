package main

import (
	"fmt"
	"math/rand"
	"time"

	"gonum.org/v1/gonum/stat/distuv"
)

const (
	NNODE   = 1
	NTENANT = 2
	NTRIAL  = 1 // 10

	NTICK                    = 100
	AVG_ARRIVAL_RATE float64 = 0.1 // per tick
	MAX_SERVICE_TIME         = 5   // in ticks
	MAX_BID          float64 = 1.0 // per tick
)

func zipf(r *rand.Rand) uint64 {
	z := rand.NewZipf(r, 2.0, 1.0, MAX_SERVICE_TIME-1)
	return z.Uint64() + 1
}

func uniform(r *rand.Rand) uint64 {
	return (rand.Uint64() % MAX_SERVICE_TIME) + 1
}

//
// Tenants runs procs.  At each tick, each tenant creates new procs
// based AVG_ARRIVAL_RATE.  Each proc runs for nTick, following either
// uniform or zipfian distribution.
//

type Proc struct {
	nTick uint64
}

func (p *Proc) String() string {
	return fmt.Sprintf("n %d", p.nTick)
}

//
// Computing nodes that the manager allocates to tenants.  Each node
// runs one proc or is idle.
//

type Node struct {
	proc   *Proc
	price  float64
	tenant *Tenant
}

func (n *Node) String() string {
	return fmt.Sprintf("{proc %v price %.2f %p}", n.proc, n.price, n.tenant)
}

func (n *Node) reallocate(to *Tenant, b float64) {
	fmt.Printf("reallocate %v(%p) to %p\n", n, n, to)
	n.tenant.evict(n)
	n.tenant = to
	n.price = b
}

//
// Tenants run procs on the available nodes to them. If they have more
// procs to run than available nodes, tenant bids for more nodes up
// till its maxbid.
//

type Tenant struct {
	maxbid  float64 // per tick
	procs   []*Proc
	nodes   []*Node
	sim     *Sim
	nproc   int
	nnode   int
	maxnode int
	nwork   int
	nwait   int
	nevict  int
}

func (t Tenant) String() string {
	s := fmt.Sprintf("\n{nproc %d nnode %d procq: [", t.nproc, t.nnode)
	for _, p := range t.procs {
		s += fmt.Sprintf("{%v} ", p)
	}
	s += fmt.Sprintf("] nodes: [")
	for _, n := range t.nodes {
		s += fmt.Sprintf("%v ", n)
	}
	return s + "]}"
}

func (t *Tenant) tick() {
	nproc := int(t.sim.poisson.Rand())
	for i := 0; i < nproc; i++ {
		t.procs = append(t.procs, t.sim.mkProc())
	}
	t.nproc += nproc
	t.schedule()

	// if we still have procs queued for execution, bid for a new
	// node, increasing the bid until mgr accepts or bid reaches max
	// bid.
	bid := float64(0.0)
	for len(t.procs) > 0 && bid <= t.maxbid {
		if n := t.sim.mgr.bidNode(t, bid); n != nil {
			fmt.Printf("%p: bid accepted at %.2f\n", t, bid)
			t.nodes = append(t.nodes, n)
			t.schedule()
		} else {
			bid += 0.1
		}
	}

	t.freeIdle()

	t.nnode += len(t.nodes)
	if len(t.nodes) > t.maxnode {
		t.maxnode = len(t.nodes)
	}
	t.nwait += len(t.procs)
}

func (t *Tenant) freeIdle() {
	for i, _ := range t.nodes {
		n := t.nodes[i]
		if n.proc == nil {
			t.nodes = append(t.nodes[0:i], t.nodes[i+1:]...)
			t.sim.mgr.yield(n)
		}
	}
}

func (t *Tenant) schedule() {
	for _, n := range t.nodes {
		if len(t.procs) == 0 {
			return
		}
		if n.proc == nil {
			n.proc = t.procs[0]
			t.procs = t.procs[1:]
		}
	}
}

// Manager is taking away a node
func (t *Tenant) evict(n *Node) {
	if n.proc != nil {
		t.nevict++
		n.proc = nil
	}
	for i, _ := range t.nodes {
		if t.nodes[i] == n {
			t.nodes = append(t.nodes[0:i], t.nodes[i+1:]...)
			return
		}
	}
	panic("evict")
}

func (t *Tenant) stats() {
	n := float64(NTICK)
	fmt.Printf("%p: lambda %.2f avg nnode %.2f max node %d nwork %d load %.2f nwait %d nevict %d\n", t, float64(t.nproc)/n, float64(t.nnode)/n, t.maxnode, t.nwork, float64(t.nwork)/float64(t.nnode), t.nwait, t.nevict)
}

//
// Manager assigns nodes to tenants
//

type Mgr struct {
	price float64
	nodes *[NNODE]Node
	index int
}

func mkMgr(nodes *[NNODE]Node) *Mgr {
	m := &Mgr{}
	m.nodes = nodes
	return m
}

func (m *Mgr) String() string {
	s := fmt.Sprintf("{mgr price %.2f nodes:", m.price)
	for i, _ := range m.nodes {
		s += fmt.Sprintf("{%v} ", m.nodes[i])
	}
	return s + "}"
}

func (m *Mgr) findFree(t *Tenant, b float64) *Node {
	for i, _ := range m.nodes {
		n := &m.nodes[i]
		if n.tenant == nil {
			n.tenant = t
			n.price = b
			return n
		}
	}
	return nil
}

func (m *Mgr) yield(n *Node) {
	n.tenant = nil
}

func (m *Mgr) bidNode(t *Tenant, b float64) *Node {
	fmt.Printf("bidNode %p %.2f\n", t, b)
	if n := m.findFree(t, b); n != nil {
		fmt.Printf("bidNode -> unused %v\n", n)
		return n
	}
	// no unused nodes; look for a node with price lower than b
	// re-allocate it to tenant t, after given the old tenant a chance
	// to evict its proc from the node.
	s := m.index
	for {
		n := &m.nodes[m.index%len(m.nodes)]
		m.index = (m.index + 1) % len(m.nodes)
		if b > n.price && n.tenant != t {
			n.reallocate(t, b)
			return n
		}
		if m.index == s { // looped around; no lower priced node exists
			break
		}
	}
	return nil
}

//
// Run simulation
//

type Sim struct {
	time    uint64
	nodes   [NNODE]Node
	tenants [NTENANT]Tenant
	rand    *rand.Rand
	mgr     *Mgr
	poisson *distuv.Poisson
}

func mkSim() *Sim {
	sim := &Sim{}
	sim.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	sim.mgr = mkMgr(&sim.nodes)
	sim.poisson = &distuv.Poisson{Lambda: AVG_ARRIVAL_RATE}
	for i := 0; i < NTENANT; i++ {
		t := &sim.tenants[i]
		t.procs = make([]*Proc, 0)
		t.sim = sim
		t.maxbid = MAX_BID
	}
	return sim
}

func (sim *Sim) Print() {
	fmt.Printf("%v: nodes %v\n", sim.time, sim.nodes)
}

func (sim *Sim) mkProc() *Proc {
	p := &Proc{}
	// p.nTick = zipf(sim.rand)
	p.nTick = uniform(sim.rand)
	return p
}

func (sim *Sim) tickTenants() {
	for i, _ := range sim.tenants {
		sim.tenants[i].tick()
	}
}

func (sim *Sim) tick() {
	sim.tickTenants()
	fmt.Printf("tick tenants %v\n", sim.tenants)
	for i, _ := range sim.nodes {
		n := &sim.nodes[i]
		if n.proc != nil {
			n.proc.nTick--
			n.tenant.nwork++
			if n.proc.nTick == 0 {
				n.proc = nil
			}
		}
	}
}

func main() {
	for i := 0; i < NTRIAL; i++ {
		sim := mkSim()
		for ; sim.time < NTICK; sim.time++ {
			sim.tick()
		}
		for i, _ := range sim.tenants {
			sim.tenants[i].stats()
		}
	}
}
