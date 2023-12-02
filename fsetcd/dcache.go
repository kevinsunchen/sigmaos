package fsetcd

import (
	"sync"

	"github.com/hashicorp/golang-lru/v2"

	db "sigmaos/debug"
	sp "sigmaos/sigmap"
)

const N = 8192

type dcEntry struct {
	dir  *DirInfo
	v    sp.TQversion
	stat Tstat
}

type Dcache struct {
	sync.Mutex
	c *lru.Cache[sp.Tpath, *dcEntry]
}

func newDcache() *Dcache {
	c, err := lru.New[sp.Tpath, *dcEntry](N)
	if err != nil {
		db.DFatalf("newDcache err %v\n", err)
	}
	return &Dcache{c: c}
}

func (dc *Dcache) lookup(d sp.Tpath) (*dcEntry, bool) {
	de, ok := dc.c.Get(d)
	if ok {
		db.DPrintf(db.FSETCD, "Lookup dcache hit %v %v", d, de)
		return de, ok
	}
	return nil, false
}

// the caller (protsrv) has only a read lock on d, and several threads
// may call insert concurrently; update entry only when v is newer.
func (dc *Dcache) insert(d sp.Tpath, new *dcEntry) {
	dc.Lock()
	defer dc.Unlock()

	de, ok := dc.c.Get(d)
	if ok && de.v >= new.v {
		db.DPrintf(db.FSETCD, "insert: stale insert %v %v", d, new)
		return
	}
	db.DPrintf(db.FSETCD, "insert: insert dcache %v %v", d, new)
	if evict := dc.c.Add(d, new); evict {
		db.DPrintf(db.FSETCD, "Eviction")
	}
}

func (dc *Dcache) remove(d sp.Tpath) {
	db.DPrintf(db.FSETCD, "remove dcache %v", d)
	if present := dc.c.Remove(d); present {
		db.DPrintf(db.FSETCD, "Removed")
	}
}

// d might not be in the cache since it maybe uncacheable. update
// assumes caller (protsrv) has write lock on dirctory d.
func (dc *Dcache) update(d sp.Tpath, dir *DirInfo) bool {
	de, ok := dc.c.Get(d)
	if ok {
		db.DPrintf(db.FSETCD, "Update dcache %v %v %v", d, dir, de.v+1)
		de.dir = dir
		de.v += 1
		return true
	}
	db.DPrintf(db.FSETCD, "Update dcache no entry %v %v", d, dir)
	return false
}
