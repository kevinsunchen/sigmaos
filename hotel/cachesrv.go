package hotel

import (
	"encoding/json"
	"errors"
	"sync"

	db "sigmaos/debug"
	"sigmaos/fs"
	"sigmaos/inode"
	"sigmaos/memfssrv"
	np "sigmaos/ninep"
	"sigmaos/proc"
	"sigmaos/protdevsrv"
	"sigmaos/sessdev"
)

const (
	DUMP = "dump"
)

var (
	ErrMiss = errors.New("cache miss")
)

type CacheRequest struct {
	Key   string
	Value []byte
}

type CacheResult struct {
	Value []byte
}

type cache struct {
	sync.Mutex
	cache map[string][]byte
}

type CacheSrv struct {
	c   *cache
	grp string
}

func RunCacheSrv(args []string) error {
	s := &CacheSrv{}
	if len(args) > 2 {
		s.grp = args[2]
	}
	s.c = &cache{}
	s.c.cache = make(map[string][]byte)
	db.DPrintf("HOTELCACHE", "%v: Run %v\n", proc.GetName(), s.grp)
	pds, err := protdevsrv.MakeProtDevSrv(np.HOTELCACHE+s.grp, s)
	if err != nil {
		return err
	}
	if err := sessdev.MkSessDev(pds.MemFs, DUMP, s.mkSession); err != nil {
		return err
	}
	return pds.RunServer()
}

// XXX support timeout
func (s *CacheSrv) Set(req CacheRequest, rep *CacheResult) error {
	db.DPrintf("HOTELCACHE", "%v: Set %v\n", proc.GetName(), req)
	s.c.Lock()
	defer s.c.Unlock()
	s.c.cache[req.Key] = req.Value
	return nil
}

func (s *CacheSrv) Get(req CacheRequest, rep *CacheResult) error {
	db.DPrintf("HOTELCACHE", "%v: Get %v\n", proc.GetName(), req)
	s.c.Lock()
	defer s.c.Unlock()

	b, ok := s.c.cache[req.Key]
	if ok {
		rep.Value = b
		return nil
	}
	return ErrMiss
}

type cacheSession struct {
	*inode.Inode
	c *cache
}

func (s *CacheSrv) mkSession(mfs *memfssrv.MemFs, sid np.Tsession) (fs.Inode, *np.Err) {
	cs := &cacheSession{mfs.MakeDevInode(), s.c}
	db.DPrintf("HOTELCACHE", "mkSession %v %p\n", cs.c, cs)
	return cs, nil
}

// XXX incremental read
func (cs *cacheSession) Read(ctx fs.CtxI, off np.Toffset, cnt np.Tsize, v np.TQversion) ([]byte, *np.Err) {
	if off > 0 {
		return nil, nil
	}
	db.DPrintf("HOTELCACHE", "Dump cache %p %v\n", cs, cs.c)
	b, err := json.Marshal(cs.c.cache)
	if err != nil {
		return nil, np.MkErrError(err)
	}
	return b, nil
}

func (cs *cacheSession) Write(ctx fs.CtxI, off np.Toffset, b []byte, v np.TQversion) (np.Tsize, *np.Err) {
	return 0, np.MkErr(np.TErrNotSupported, nil)
}
