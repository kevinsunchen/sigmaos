package overlay

import (
	"sync"

	db "sigmaos/debug"
	"sigmaos/fs"
	"sigmaos/inode"
	"sigmaos/path"
	"sigmaos/serr"
	sp "sigmaos/sigmap"
)

//
// Overlay a directory with another directory transparently.  Servers
// can mount other file systems in the overlay directory (e.g.,
// statsd, fsfence, etc.).  This allows a server to export information
// to clients through sigmaP.
//

type DirOverlay struct {
	fs.Inode
	underlay fs.Dir
	mu       sync.Mutex
	entries  map[string]fs.Inode // XXX use sortedmap?
}

func MkDirOverlay(dir fs.Dir) *DirOverlay {
	d := &DirOverlay{}
	d.Inode = inode.NewInode(nil, sp.DMDIR, nil)
	d.underlay = dir
	d.entries = make(map[string]fs.Inode)
	return d
}

func (dir *DirOverlay) Mount(name string, i fs.Inode) {
	dir.mu.Lock()
	defer dir.mu.Unlock()

	db.DPrintf(db.OVERLAYDIR, "Mount i %v as %v\n", i, name)

	dir.entries[name] = i
}

func (dir *DirOverlay) lookupMount(name string) fs.FsObj {
	dir.mu.Lock()
	defer dir.mu.Unlock()

	db.DPrintf(db.OVERLAYDIR, "lookupMount %v %v\n", name, dir.entries)

	if i, ok := dir.entries[name]; ok {
		return i
	}
	return nil
}

func (dir *DirOverlay) removeMount(name string) bool {
	dir.mu.Lock()
	defer dir.mu.Unlock()

	db.DPrintf(db.OVERLAYDIR, "removeMount %v %v\n", name, dir.entries)

	if _, ok := dir.entries[name]; ok {
		delete(dir.entries, name)
		return true
	}
	return false
}

func (dir *DirOverlay) ls() []*sp.Stat {
	dir.mu.Lock()
	defer dir.mu.Unlock()

	entries := make([]*sp.Stat, 0, len(dir.entries))
	for k, i := range dir.entries {
		st, _ := i.Stat(nil)
		st.Name = k
		entries = append(entries, st)
	}
	return entries
}

// lookup in overlay
func (dir *DirOverlay) Lookup(ctx fs.CtxI, name string) (fs.FsObj, *serr.Err) {
	if i := dir.lookupMount(name); i != nil {
		return i, nil
	}
	return nil, serr.NewErr(serr.TErrNotfound, name)
}

func (dir *DirOverlay) LookupPath(ctx fs.CtxI, path path.Path) ([]fs.FsObj, fs.FsObj, path.Path, *serr.Err) {
	if i := dir.lookupMount(path[0]); i != nil {
		return []fs.FsObj{i}, i, path[1:], nil
	} else {
		os, o, rest, err := dir.underlay.LookupPath(ctx, path)
		if o == dir.underlay {
			o = dir
		}
		return os, o, rest, err
	}
}

func (dir *DirOverlay) Create(ctx fs.CtxI, name string, perm sp.Tperm, m sp.Tmode, lid sp.TleaseId, f sp.Tfence) (fs.FsObj, *serr.Err) {
	if i := dir.lookupMount(name); i != nil {
		return i, serr.NewErr(serr.TErrExists, name)
	}
	return dir.underlay.Create(ctx, name, perm, m, lid, f)
}

// XXX account for extra entries in cursor, and sort
// XXX ignores size
func (dir *DirOverlay) ReadDir(ctx fs.CtxI, cursor int, n sp.Tsize) ([]*sp.Stat, *serr.Err) {
	sts, err := dir.underlay.ReadDir(ctx, cursor, n)
	if err != nil {
		return nil, err
	}
	db.DPrintf(db.OVERLAYDIR, "ReadDir underlay %v cursor %d %v\n", dir, cursor, sp.Names(sts))
	if cursor > 0 { // did we already sent the extra ones
		return sts, err
	}
	// prepend the extra ones
	sts = append(dir.ls(), sts...)
	return sts, nil
}

func (dir *DirOverlay) Rename(ctx fs.CtxI, from, to string, f sp.Tfence) *serr.Err {
	if i := dir.lookupMount(from); i != nil {
		return serr.NewErr(serr.TErrNotSupported, from)
	}
	return dir.underlay.Rename(ctx, from, to, f)
}

func (dir *DirOverlay) Renameat(ctx fs.CtxI, old string, nd fs.Dir, new string, f sp.Tfence) *serr.Err {
	if i := dir.lookupMount(old); i != nil {
		return serr.NewErr(serr.TErrNotSupported, old)
	}
	return dir.underlay.Renameat(ctx, old, nd, new, f)
}

func (dir *DirOverlay) Remove(ctx fs.CtxI, n string, f sp.Tfence) *serr.Err {
	if dir.removeMount(n) {
		return nil
	}
	return dir.underlay.Remove(ctx, n, f)
}
