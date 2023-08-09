package kv

import (
	"errors"
	"hash/fnv"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	"sigmaos/cache"
	"sigmaos/cacheclnt"
	db "sigmaos/debug"
	"sigmaos/fslib"
	"sigmaos/kvgrp"
	"sigmaos/rand"
	"sigmaos/serr"
	sp "sigmaos/sigmap"
	tproto "sigmaos/tracing/proto"
)

//
// Clerk for sharded kv service, which repeatedly reads/writes keys.
//

const (
	NKEYS  = 100
	WAITMS = 100
)

func key2shard(key cache.Tkey) cache.Tshard {
	h := fnv.New32a()
	h.Write([]byte(key))
	shard := cache.Tshard(h.Sum32() % NSHARD)
	return shard
}

type KvClerk struct {
	*fslib.FsLib
	conf  *Config
	job   string
	cc    *cacheclnt.CacheClnt
	cid   sp.TclntId
	seqno sp.Tseqno
}

func MakeClerkFsl(fsl *fslib.FsLib, job string) (*KvClerk, error) {
	return makeClerkStart(fsl, job)
}

func MakeClerkFslOnly(fsl *fslib.FsLib, job string) *KvClerk {
	return makeClerk(fsl, job)
}

func MakeClerk(uname sp.Tuname, job string) (*KvClerk, error) {
	fsl, err := fslib.MakeFsLib(uname)
	if err != nil {
		return nil, err
	}
	return makeClerkStart(fsl, job)
}

func makeClerk(fsl *fslib.FsLib, job string) *KvClerk {
	kc := &KvClerk{
		FsLib: fsl,
		conf:  &Config{},
		job:   job,
		cc:    cacheclnt.NewCacheClnt([]*fslib.FsLib{fsl}, job, NSHARD),
		cid:   sp.TclntId(rand.Uint64()),
	}
	return kc
}

func makeClerkStart(fsl *fslib.FsLib, job string) (*KvClerk, error) {
	kc := makeClerk(fsl, job)
	return kc, kc.StartClerk()
}

func (kc *KvClerk) nextSeqno() sp.Tseqno {
	seq := &kc.seqno
	return seq.Next()
}

func (kc *KvClerk) StartClerk() error {
	if err := kc.switchConfig(); err != nil {
		return err
	}
	return nil
}

// Detach servers not in kvs
func (kc *KvClerk) DetachKVs(kvs *KvSet) {
	mnts := kc.Mounts()
	for _, mnt := range mnts {
		db.DPrintf(db.KVCLERK, "mnt kv %v", mnt)
		if strings.HasPrefix(mnt, JobDir(kc.job)+"/grp") {
			kvd := strings.TrimPrefix(mnt, JobDir(kc.job)+"/")
			if !kvs.present(kvd) {
				db.DPrintf(db.KVCLERK, "Detach kv %v", kvd)
				kc.Detach(kvGrpPath(kc.job, kvd))
			}
		}
	}
}

func paths(job string, kvset *KvSet) []string {
	kvs := kvset.mkKvs()
	dirs := make([]string, 0, len(kvs)+1)
	for _, kvd := range kvs {
		dirs = append(dirs, kvgrp.GrpPath(JobDir(job), kvd))
	}
	return dirs
}

// Read config, and retry if we have a stale group fence
func (kc *KvClerk) switchConfig() error {
	for {
		err := kc.GetFileJsonWatch(KVConfig(kc.job), kc.conf)
		if err != nil {
			db.DPrintf(db.KVCLERK_ERR, "GetFileJsonWatch %v err %v", KVConfig(kc.job), err)
			return err
		}
		db.DPrintf(db.KVCLERK, "Conf %v", kc.conf)
		kvset := MakeKvs(kc.conf.Shards)
		// detach groups not in use; diff between new and mount table?
		kc.DetachKVs(kvset)
		break
	}
	return nil
}

// Try to fix err; if return is nil, retry.
func (kc *KvClerk) fixRetry(err error) error {
	var sr *serr.Err
	if !errors.As(err, &sr) {
		return err
	}
	if sr.IsErrRetry() {
		// Shard hasn't been created yet (config 0) or isn't ready
		// yet, so wait a bit, and retry.  XXX make sleep time
		// dynamic?
		db.DPrintf(db.KVCLERK_ERR, "Wait for shard %v", err)
		time.Sleep(WAITMS * time.Millisecond)
		return nil
	}
	if sr.IsErrStale() ||
		(sr.IsErrNotfound() && (strings.HasPrefix(sr.ErrPath(), "grp-") ||
			strings.HasPrefix(sr.ErrPath(), "shard"))) {
		db.DPrintf(db.KVCLERK_ERR, "fixRetry %v", err)
		return kc.switchConfig()
	}
	return err
}

// Do an operation. If an error, try to fix the error (e.g., rereading
// config), and on success, retry.
func (kc *KvClerk) doop(o *op) {
	s := key2shard(o.k)
	for {
		db.DPrintf(db.KVCLERK, "o %v conf %v", o.kind, kc.conf)
		kc.do(o, kvGrpPath(kc.job, kc.conf.Shards[s]), s)
		if o.err == nil { // success?
			return
		}
		o.err = kc.fixRetry(o.err)
		if o.err != nil {
			return
		}
	}
}

type Top string

const (
	GET     Top = "Get"
	PUT     Top = "Put"
	GETVALS Top = "GetVals"
)

type op struct {
	kind  Top
	val   proto.Message
	k     cache.Tkey
	m     sp.Tmode
	cid   sp.TclntId
	seqno sp.Tseqno
	err   error
	vals  []proto.Message
}

func newOp(o Top, val proto.Message, k cache.Tkey, m sp.Tmode, cid sp.TclntId, s sp.Tseqno) *op {
	return &op{kind: o, val: val, k: k, m: m, cid: cid, seqno: s}
}

func (kc *KvClerk) do(o *op, srv string, s cache.Tshard) {
	switch o.kind {
	case GET:
		o.err = kc.cc.GetSrv(srv, string(o.k), o.val, o.cid, o.seqno)
	case GETVALS:
		o.vals, o.err = kc.cc.GetVals(srv, string(o.k), o.val, o.cid, o.seqno, &kc.conf.Fence)
	case PUT:
		if o.m == sp.OAPPEND {
			o.err = kc.cc.AppendFence(srv, string(o.k), o.val, o.cid, o.seqno, &kc.conf.Fence)
		} else {
			o.err = kc.cc.PutSrv(srv, string(o.k), o.val, o.cid, o.seqno)
		}
	}
	db.DPrintf(db.KVCLERK, "op %v(%v) f %v srv %v %v err %v", o.kind, o.m == sp.OAPPEND, kc.conf.Fence, srv, s, o.err)
}

func (kc *KvClerk) Get(key string, val proto.Message) error {
	op := newOp(GET, val, cache.Tkey(key), sp.OREAD, kc.cid, kc.nextSeqno())
	kc.doop(op)
	return op.err
}

func (kc *KvClerk) GetTraced(sctx *tproto.SpanContextConfig, key string, val proto.Message) error {
	return kc.Get(key, val)
}

func (kc *KvClerk) GetVals(k cache.Tkey, val proto.Message) ([]proto.Message, error) {
	op := newOp(GETVALS, val, k, sp.OREAD, kc.cid, kc.nextSeqno())
	kc.doop(op)
	return op.vals, op.err
}

func (kc *KvClerk) Append(k cache.Tkey, val proto.Message) error {
	op := newOp(PUT, val, k, sp.OAPPEND, kc.cid, kc.nextSeqno())
	kc.doop(op)
	return op.err
}

func (kc *KvClerk) PutTraced(sctx *tproto.SpanContextConfig, key string, val proto.Message) error {
	return kc.Put(key, val)
}

func (kc *KvClerk) Put(k string, val proto.Message) error {
	op := newOp(PUT, val, cache.Tkey(k), sp.OWRITE, kc.cid, kc.nextSeqno())
	kc.doop(op)
	return op.err
}

func (kc *KvClerk) DeleteTraced(sctx *tproto.SpanContextConfig, key string) error {
	return kc.Delete(key)
}

func (kc *KvClerk) Delete(k string) error {
	db.DFatalf("Unimplemented")
	return nil
}

// Count the number of keys stored at each group.
func (kc *KvClerk) GetKeyCountsPerGroup(keys []string) map[string]int {
	if err := kc.switchConfig(); err != nil {
		db.DFatalf("Error switching KV config: %v", err)
	}
	cnts := make(map[string]int)
	for _, k := range keys {
		s := key2shard(cache.Tkey(k))
		grp := kc.conf.Shards[s]
		if _, ok := cnts[grp]; !ok {
			cnts[grp] = 0
		}
		cnts[grp]++
	}
	return cnts
}
