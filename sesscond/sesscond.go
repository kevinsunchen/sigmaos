package sesscond

import (
	"fmt"
	"log"
	"sync"
	// "errors"

	//	"github.com/sasha-s/go-deadlock"

	// db "ulambda/debug"
	np "ulambda/ninep"
	"ulambda/session"
	"ulambda/threadmgr"
)

//
// sesscond wraps cond vars so that if a session terminates, it can
// wakeup threads that are associated with that session.  Each cond
// var is represented as several cond vars, one per goroutine using it.
//

type cond struct {
	isClosed  bool
	threadmgr *threadmgr.ThreadMgr
	c         *sync.Cond
}

// Sess cond has one cond per session.  The lock is, for example, a
// pipe lock or watch lock, which SessCond releases in Wait() and
// re-acquires before returning out of Wait().
type SessCond struct {
	lock  sync.Locker
	st    *session.SessionTable
	nref  int // under sct lock
	conds map[np.Tsession][]*cond
}

func makeSessCond(st *session.SessionTable, lock sync.Locker) *SessCond {
	sc := &SessCond{}
	sc.lock = lock
	sc.st = st
	sc.conds = make(map[np.Tsession][]*cond)
	return sc
}

// A session has been closed: wake up threads associated with this
// session. Grab c lock to ensure that wakeup isn't missed while a
// thread is about enter wait (and releasing sess and sc lock).
func (sc *SessCond) closed(sessid np.Tsession) {
	sc.lock.Lock()
	defer sc.lock.Unlock()

	// log.Printf("cond %p: close %v %v\n", sc, sessid, sc.conds)
	if condlist, ok := sc.conds[sessid]; ok {
		// log.Printf("%p: sess %v closed\n", sc, sessid)
		for _, c := range condlist {
			c.isClosed = true
			c.threadmgr.Wake(c.c)
		}
	}
	delete(sc.conds, sessid)
}

func (sc *SessCond) alloc(sessid np.Tsession) *cond {
	if _, ok := sc.conds[sessid]; !ok {
		sc.conds[sessid] = []*cond{}
	}
	c := &cond{}
	c.threadmgr = sc.st.SessThread(sessid)
	c.c = sync.NewCond(sc.lock)
	sc.conds[sessid] = append(sc.conds[sessid], c)
	return c
}

// Caller should hold sc lock and will receive it back on return. Wait releases
// sess lock, so that other threads on the session can run. sc.lock ensures
// atomicity of releasing sc lock and going to sleep.
func (sc *SessCond) Wait(sessid np.Tsession) error {
	c := sc.alloc(sessid)

	c.threadmgr.Sleep(c.c)

	closed := c.isClosed

	if closed {
		log.Printf("wait sess closed %v\n", sessid)
		return fmt.Errorf("session closed %v", sessid)
	}
	return nil
}

// Caller should hold sc lock.
func (sc *SessCond) Signal() {
	for sid, condlist := range sc.conds {
		// acquire c.lock() to ensure signal doesn't happen
		// between releasing sc or sess lock and going to
		// sleep.
		for _, c := range condlist {
			c.threadmgr.Wake(c.c)
		}
		delete(sc.conds, sid)
	}
}

// Caller should hold sc lock.
func (sc *SessCond) Broadcast() {
	for sid, condlist := range sc.conds {
		for _, c := range condlist {
			c.threadmgr.Wake(c.c)
		}
		delete(sc.conds, sid)
	}
}

type SessCondTable struct {
	//	deadlock.Mutex
	sync.Mutex
	conds  map[*SessCond]bool
	st     *session.SessionTable
	closed bool
}

func MakeSessCondTable(st *session.SessionTable) *SessCondTable {
	t := &SessCondTable{}
	t.conds = make(map[*SessCond]bool)
	t.st = st
	return t
}

func (sct *SessCondTable) MakeSessCond(lock sync.Locker) *SessCond {
	sct.Lock()
	defer sct.Unlock()

	sc := makeSessCond(sct.st, lock)
	sct.conds[sc] = true
	sc.nref++
	return sc
}

func (sct *SessCondTable) FreeSessCond(sc *SessCond) {
	sct.Lock()
	defer sct.Unlock()
	sc.nref--
	if sc.nref != 0 {
		log.Fatalf("freesesscond %v\n", sc)
	}
	delete(sct.conds, sc)
}

func (sct *SessCondTable) toSlice() []*SessCond {
	sct.Lock()
	defer sct.Unlock()

	sct.closed = true
	t := make([]*SessCond, 0, len(sct.conds))
	for sc, _ := range sct.conds {
		t = append(t, sc)
	}
	return t
}

// Close all sess conds for sessid, which wakes up waiting threads.  A
// thread may delete a sess cond from sct, if the thread is the last
// user.  So we need, a lock around sct.conds.  But, DeleteSess
// violates lock order, which is lock sc.lock first (e.g., watch on
// directory), then acquire sct.lock (if file watch must create sess
// cond in sct).  To avoid order violation, DeleteSess makes copy
// first, then close() sess conds.  Threads many add new sess conds to
// sct while bailing out (e.g., to remove an emphemeral file), but
// threads shouldn't wait on these sess conds, so we don't have to
// close those.
func (sct *SessCondTable) DeleteSess(sessid np.Tsession) {
	t := sct.toSlice()
	//log.Printf("%v: delete sess %v\n", sessid, t)
	for _, sc := range t {
		sc.closed(sessid)
	}
}
