package main

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

	"gonum.org/v1/gonum/stat/distuv"
)

const (
	NNODE   = 20
	NTENANT = 100
	NTRIAL  = 1 // 10

	NTICK                    = 100
	AVG_ARRIVAL_RATE float64 = 0.1 // per tick
	MAX_SERVICE_TIME         = 5   // in ticks
	// DURATION                 = NTICK
	PRICE_ONDEMAND Price = 0.00000001155555555555 // per ms for 1h on AWS
	PRICE_SPOT     Price = 0.00000000347222222222 // per ms
	BIT_INCREMENT        = 0.000000000001
	MAX_BID        Price = 3 * PRICE_SPOT
)

func zipf(r *rand.Rand) uint64 {
	z := rand.NewZipf(r, 2.0, 1.0, MAX_SERVICE_TIME-1)
	return z.Uint64() + 1
}

func uniform(r *rand.Rand) uint64 {
	return (rand.Uint64() % MAX_SERVICE_TIME) + 1
}

//
// Price
//

type Price float64

func (p Price) String() string {
	return fmt.Sprintf("$%.12f", p)
}

//
// Bid
//

type Bid struct {
	tenant *Tenant
	bids   []Price // one bid per node
}

func (b *Bid) String() string {
	return fmt.Sprintf("{t: %p %v %d}", b.tenant, b.bids, len(b.bids))
}

func mkBid(t *Tenant, bs []Price) *Bid {
	return &Bid{t, bs}
}

type Bids []*Bid

func (bs *Bids) PopHighest(rand *rand.Rand) (*Tenant, Price) {
	bid := Price(0.0)

	if len(*bs) == 0 {
		return nil, bid
	}

	// find highest bid
	for _, b := range *bs {
		if b.bids[0] > bid {
			bid = b.bids[0]
		}
	}

	// find bidders for highest
	bidders := make([]int, 0)
	for i, b := range *bs {
		if b.bids[0] == bid {
			bidders = append(bidders, i)
		}
	}

	// pick a random higest
	n := bidders[int(rand.Uint64()%uint64(len(bidders)))]
	t := (*bs)[n].tenant

	// remove n's bid
	(*bs)[n].bids = (*bs)[n].bids[1:]
	if len((*bs)[n].bids) == 0 {
		*bs = append((*bs)[:n], (*bs)[n+1:]...)
	}
	return t, bid
}

//
// Tenants runs procs.  At each tick, each tenant creates new procs
// based AVG_ARRIVAL_RATE.  Each proc runs for nTick, following either
// uniform or zipfian distribution.
//

type Proc struct {
	nLength uint64 // in ticks
	nTick   uint64 // #ticks remaining
	cost    Price  // cost for this proc
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
	price  Price // the price for a tick
	tenant *Tenant
}

func (n *Node) String() string {
	return fmt.Sprintf("{%p: proc %v price %v t %p}", n, n.proc, n.price, n.tenant)
}

type Nodes []*Node

func (ns *Nodes) remove(n1 *Node) *Node {
	for i, n := range *ns {
		if n == n1 {
			*ns = append((*ns)[:i], (*ns)[i+1:]...)
			return n
		}
	}
	return nil
}

func (ns *Nodes) findFree() *Node {
	for i, n := range *ns {
		if n.tenant == nil {
			*ns = append((*ns)[:i], (*ns)[i+1:]...)
			return n
		}
	}
	return nil
}

func (ns *Nodes) findVictim(t *Tenant, bid Price) *Node {
	for _, n := range *ns {
		if n.tenant != t && bid > n.price {
			return n
		}
	}
	return nil
}

//
// Tenants run procs on the nodes allocated to them by the mgr. If
// they have more procs to run than available nodes, tenant bids for
// more nodes up till its maxbid.
//

type Tenant struct {
	poisson  *distuv.Poisson
	maxbid   Price
	procs    []*Proc
	nodes    Nodes
	sim      *Sim
	nproc    int // sum of proc ticks
	nnode    int // sum of node ticks
	maxnode  int
	nwork    uint64 // sum of # ticks running a proc
	cost     Price  // cost for nwork ticks
	nwait    uint64 // sum of # ticks waiting to be run
	nevict   int    // # evictions
	nwasted  uint64 // sum # ticks wasted because of eviction
	sunkCost Price  // the cost of the wasted ticks
}

func (t *Tenant) String() string {
	s := fmt.Sprintf("{nproc %d nnode %d procq: [", t.nproc, t.nnode)
	for _, p := range t.procs {
		s += fmt.Sprintf("{%v} ", p)
	}
	s += fmt.Sprintf("] nodes: [")
	for _, n := range t.nodes {
		s += fmt.Sprintf("%v ", n)
	}
	return s + "]}"
}

// New procs "arrive" based on Poisson distribution. Schedule queued
// procs on the available nodes, and release nodes we don't use.
func (t *Tenant) genProcs() {
	nproc := int(t.poisson.Rand())
	for i := 0; i < nproc; i++ {
		t.procs = append(t.procs, t.sim.mkProc())
	}
	t.nproc += nproc
	t.schedule()
	t.yieldIdle()
}

// Bid for new nodes if we have queued procs.
func (t *Tenant) bid() *Bid {
	if len(t.procs) > 0 {
		bids := make([]Price, 0)
		if t == &t.sim.tenants[0] && len(t.nodes) == 0 {
			bids = append(bids, PRICE_ONDEMAND)
			for i := 0; i < len(t.procs)-1; i++ {
				bids = append(bids, t.maxbid)
			}

		} else {
			for i := 0; i < len(t.procs); i++ {
				bids = append(bids, t.maxbid)
			}
		}
		return mkBid(t, bids)
	}
	return nil
}

// After bidding, we may have received new nodes; use them.
func (t *Tenant) scheduleNodes() {
	t.schedule()
	t.nnode += len(t.nodes)
	if len(t.nodes) > t.maxnode {
		t.maxnode = len(t.nodes)
	}
	t.nwait += uint64(len(t.procs))
	t.charge()
}

func (t *Tenant) yieldIdle() {
	for i := 0; i < len(t.nodes); i++ {
		n := t.nodes[i]
		if n.proc == nil {
			t.nodes = append(t.nodes[0:i], t.nodes[i+1:]...)
			i--
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
		c := n.proc.nLength - n.proc.nTick // wasted ticks
		t.nevict++
		t.nwasted += c
		if c > 0 {
			t.sunkCost += n.proc.cost
		}
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

func (t *Tenant) charge() {
	c := Price(0)
	for _, n := range t.nodes {
		if n.proc == nil {
			panic("charge")
		}
		n.proc.cost += n.price
		c += n.price
	}
	t.cost += c
	t.sim.mgr.revenue += c
}

func (t *Tenant) stats() {
	n := float64(NTICK)
	fmt.Printf("%p: l %v P/T %.2f maxN %d work %dT util %.2f nwait %dT #evict %dP (waste %dT) charge %v sunk %v tick %v\n", t, float64(t.nproc)/n, float64(t.nnode)/n, t.maxnode, t.nwork, float64(t.nwork)/float64(t.nnode), t.nwait, t.nevict, t.nwasted, t.cost, t.sunkCost, Price(float64(t.cost)/float64(t.nwork)))
}

//
// Manager assigns nodes to tenants
//

type Mgr struct {
	sim     *Sim
	price   Price
	free    Nodes
	cur     Nodes
	index   int
	revenue Price
	nwork   int
}

func mkMgr(sim *Sim) *Mgr {
	m := &Mgr{}
	m.sim = sim
	ns := make(Nodes, NNODE, NNODE)
	for i, _ := range ns {
		ns[i] = &Node{}
	}
	m.free = ns
	return m
}

func (m *Mgr) String() string {
	s := fmt.Sprintf("{mgr price %v nodes:", m.price)
	for _, n := range m.cur {
		s += fmt.Sprintf("{%v} ", n)
	}
	return s + "}"
}

func (m *Mgr) stats() {
	n := NTICK * NNODE
	fmt.Printf("Mgr revenue %.2f avg rev/tick %.2f util %.2f\n", m.revenue, float64(m.revenue)/float64(m.nwork), float64(m.nwork)/float64(n))
}

func (m *Mgr) yield(n *Node) {
	fmt.Printf("yield %v\n", n)
	n.tenant = nil
	m.free = append(m.free, n)
	m.cur.remove(n)
}

func (m *Mgr) collectBids() (int, Bids) {
	bids := make([]*Bid, 0)
	n := 0
	for i, _ := range m.sim.tenants {
		if b := m.sim.tenants[i].bid(); b != nil {
			// sort the bids in b
			sort.Slice(b.bids, func(i, j int) bool {
				return b.bids[i] > b.bids[j]
			})
			n += len(b.bids)
			bids = append(bids, b)
		}
	}
	return n, bids
}

func (m *Mgr) assignNodes() Nodes {
	bnn, bids := m.collectBids()
	fmt.Printf("bids %v %d %v\n", bids, bnn, len(m.free))
	new := make(Nodes, 0)
	for {
		t, bid := bids.PopHighest(m.sim.rand)
		if t == nil {
			break
		}
		fmt.Printf("assignNodes: %p bid highest %v\n", t, bid)
		if n := m.free.findFree(); n != nil {
			n.tenant = t
			n.price = bid
			//fmt.Printf("assignNodes: allocate %p to %p at %v\n", n, t, bid)
			t.nodes = append(t.nodes, n)
			new = append(new, n)
		} else if n := m.cur.findVictim(t, bid); n != nil {
			fmt.Printf("assignNodes: reallocate %v to %p at %v\n", n, t, bid)
			n.tenant.evict(n)
			n.tenant = t
			n.price = bid
		} else {
			fmt.Printf("assignNodes: no nodes left\n")
			break
		}
	}
	// 		bid += BIT_INCREMENT
	m.cur = append(m.cur, new...)
	return m.cur
}

//
// Run simulation
//

type Sim struct {
	time    uint64
	tenants [NTENANT]Tenant
	rand    *rand.Rand
	mgr     *Mgr
}

func mkSim() *Sim {
	sim := &Sim{}
	sim.rand = rand.New(rand.NewSource(time.Now().UnixNano()))

	sim.mgr = mkMgr(sim)
	for i := 0; i < NTENANT; i++ {
		t := &sim.tenants[i]
		t.procs = make([]*Proc, 0)
		t.sim = sim
		if i == 0 {
			t.poisson = &distuv.Poisson{Lambda: 10 * AVG_ARRIVAL_RATE}
			t.maxbid = MAX_BID
		} else {
			t.poisson = &distuv.Poisson{Lambda: AVG_ARRIVAL_RATE}
			t.maxbid = MAX_BID / 2
		}
	}
	return sim
}

func (sim *Sim) mkProc() *Proc {
	p := &Proc{}
	// p.nTick = zipf(sim.rand)
	p.nTick = uniform(sim.rand)
	p.nLength = p.nTick
	return p
}

// At each tick, a tenants generates load in the form of procs that
// need to be run.
func (sim *Sim) genLoad() {
	for i, _ := range sim.tenants {
		sim.tenants[i].genProcs()
	}
}

func (sim *Sim) scheduleNodes() {
	for i, _ := range sim.tenants {
		sim.tenants[i].scheduleNodes()
	}
}

func (sim *Sim) printTenants(tick uint64, nn int) {
	fmt.Printf("tick %d nodes %d:", tick, nn)
	for i, _ := range sim.tenants {
		t := &sim.tenants[i]
		if len(t.procs) > 0 || len(t.nodes) > 0 {
			fmt.Printf("\n%p: %v", t, t)
		}
	}
	fmt.Printf("\n")
}

func (sim *Sim) runProcs(ns Nodes) {
	for _, n := range ns {
		if n.proc != nil {
			n.proc.nTick--
			n.tenant.nwork++
			sim.mgr.nwork++
			if n.proc.nTick == 0 {
				n.proc = nil
			}
		}
	}
}

func (sim *Sim) tick(tick uint64) {
	sim.genLoad()
	ns := sim.mgr.assignNodes()
	fmt.Printf("assignment %d nodes: %v\n", len(ns), ns)
	sim.scheduleNodes()
	sim.printTenants(tick, len(ns))
	sim.runProcs(ns)
}

func main() {
	for i := 0; i < NTRIAL; i++ {
		sim := mkSim()
		for ; sim.time < NTICK; sim.time++ {
			sim.tick(sim.time)
		}
		for i, _ := range sim.tenants {
			sim.tenants[i].stats()
		}
		sim.mgr.stats()
	}
}
