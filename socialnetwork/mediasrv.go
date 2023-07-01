package socialnetwork

import (
	sp "sigmaos/sigmap"
	dbg "sigmaos/debug"
	"sigmaos/protdevsrv"
	"sigmaos/mongoclnt"
	"sigmaos/cacheclnt"
	"sigmaos/fs"
	"sigmaos/socialnetwork/proto"
	"strconv"
	"fmt"
	"sync"
	"math/rand"
	"gopkg.in/mgo.v2/bson"
)

// YH:
// Media Storage service for social network

const (
	MEDIA_QUERY_OK = "OK"
	MEDIA_CACHE_PREFIX = "media_"
)

type MediaSrv struct {
	mongoc *mongoclnt.MongoClnt
	cachec *cacheclnt.CacheClnt
	sid    int32
	ucount int32
	mu     sync.Mutex
}

func RunMediaSrv(public bool, jobname string) error {
	dbg.DPrintf(dbg.SOCIAL_NETWORK_MEDIA, "Creating media service\n")
	msrv := &MediaSrv{}
	msrv.sid = rand.Int31n(536870912) // 2^29
	pds, err := protdevsrv.MakeProtDevSrvPublic(sp.SOCIAL_NETWORK_MEDIA, msrv, public)
	if err != nil {
		return err
	}
	mongoc, err := mongoclnt.MkMongoClnt(pds.MemFs.SigmaClnt().FsLib)
	if err != nil {
		return err
	}
	mongoc.EnsureIndex(SN_DB, MEDIA_COL, []string{"mediaid"})
	msrv.mongoc = mongoc
	fsls := MakeFsLibs(sp.SOCIAL_NETWORK_MEDIA)
	cachec, err := cacheclnt.MkCacheClnt(fsls, jobname)
	if err != nil {
		return err
	}
	msrv.cachec = cachec
	dbg.DPrintf(dbg.SOCIAL_NETWORK_MEDIA, "Starting media service\n")
	return pds.RunServer()
}

func (msrv *MediaSrv) StoreMedia(ctx fs.CtxI, req proto.StoreMediaRequest, res *proto.StoreMediaResponse) error {
	res.Ok = "No"
	mId := msrv.getNextMediaId()
	newMedia := Media{mId, req.Mediatype, req.Mediadata}
	if err := msrv.mongoc.Insert(SN_DB, MEDIA_COL, newMedia); err != nil {
		dbg.DFatalf("Mongo Error: %v", err)
		return err
	}
	res.Ok = POST_QUERY_OK
	res.Mediaid = mId
	return nil
}

func (msrv *MediaSrv) ReadMedia(ctx fs.CtxI, req proto.ReadMediaRequest, res *proto.ReadMediaResponse) error {
	res.Ok = "No."
	mediatypes := make([]string, len(req.Mediaids))
	mediadatas := make([][]byte, len(req.Mediaids))
	missing := false
	for idx, mediaid := range req.Mediaids {
		media, err := msrv.getMedia(mediaid)
		if err != nil {
			return err
		}
		if media == nil {
			missing = true
			res.Ok = res.Ok + fmt.Sprintf(" Missing %v.", mediaid)
		} else {
			mediatypes[idx] = media.Type
			mediadatas[idx] = media.Data
		}
	}
	res.Mediatypes = mediatypes
	res.Mediadatas = mediadatas
	if !missing {
		res.Ok = MEDIA_QUERY_OK
	}
	return nil
}

func (msrv *MediaSrv) getMedia(mediaid int64) (*Media, error) {
	key := MEDIA_CACHE_PREFIX + strconv.FormatInt(mediaid, 10)
	media := &Media{}
	cacheItem := &proto.CacheItem{}
	if err := msrv.cachec.Get(key, cacheItem); err != nil {
		if !msrv.cachec.IsMiss(err) {
			return nil, err
		}
		dbg.DPrintf(dbg.SOCIAL_NETWORK_MEDIA, "Media %v cache miss\n", key)
		found, err := msrv.mongoc.FindOne(SN_DB, MEDIA_COL, bson.M{"mediaid": mediaid}, media)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, nil
		}
		encoded, _ := bson.Marshal(media)
		msrv.cachec.Put(key, &proto.CacheItem{Key: key, Val: encoded})
		dbg.DPrintf(dbg.SOCIAL_NETWORK_MEDIA, "Found media for %v in DB: %v", mediaid, media)
	} else {
		bson.Unmarshal(cacheItem.Val, media)
		dbg.DPrintf(dbg.SOCIAL_NETWORK_MEDIA, "Found media %v in cache!\n", mediaid)
	}
	return media, nil
}

func (msrv *MediaSrv) incCountSafe() int32 {
	msrv.mu.Lock()
	defer msrv.mu.Unlock()
	msrv.ucount++
	return msrv.ucount
}

func (msrv *MediaSrv) getNextMediaId() int64 {
	return int64(msrv.sid)*1e10 + int64(msrv.incCountSafe())
}

type Media struct {
	Mediaid int64  `bson:mediaid`
	Type    string `bson:type`
	Data    []byte `bson:data`
}