package hotel

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/harlow/go-micro-services/data"

	"sigmaos/dbclnt"
	db "sigmaos/debug"
	"sigmaos/fs"
	"sigmaos/hotel/proto"
	"sigmaos/proc"
	"sigmaos/protdevsrv"
	sp "sigmaos/sigmap"
	"sigmaos/tracing"
)

const (
	NHOTEL = 80
)

type ProfSrv struct {
	dbc    *dbclnt.DbClnt
	cachec CacheClnt
	tracer *tracing.Tracer
}

func RunProfSrv(job string, public bool, cache string) error {
	ps := &ProfSrv{}
	pds, err := protdevsrv.MakeProtDevSrvPublic(sp.HOTELPROF, ps, public)
	if err != nil {
		return err
	}
	dbc, err := dbclnt.MkDbClnt(pds.MemFs.SigmaClnt().FsLib, sp.DBD)
	if err != nil {
		return err
	}
	ps.dbc = dbc
	cachec, err := MkCacheClnt(cache, pds.MemFs.SigmaClnt().FsLib, job)
	if err != nil {
		return err
	}
	ps.cachec = cachec
	file := data.MustAsset("data/hotels.json")
	profs := []*Profile{}
	if err := json.Unmarshal(file, &profs); err != nil {
		return err
	}
	ps.initDB(profs)
	ps.tracer = tracing.Init("prof", proc.GetSigmaJaegerIP())
	defer ps.tracer.Flush()
	return pds.RunServer()
}

// Inserts a flatten profile into db
func (ps *ProfSrv) insertProf(p *Profile) error {
	q := fmt.Sprintf("INSERT INTO profile (hotelid, name, phone, description, streetnumber, streetname, city, state, country, postal, lat, lon) VALUES ('%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%f', '%f');", p.Id, p.Name, p.PhoneNumber, p.Description, p.Address.StreetNumber, p.Address.StreetName, p.Address.City, p.Address.State, p.Address.Country, p.Address.PostalCode, p.Address.Lat, p.Address.Lon)
	if err := ps.dbc.Exec(q); err != nil {
		return err
	}
	return nil
}

func (ps *ProfSrv) getProf(sctx context.Context, id string) (*proto.ProfileFlat, error) {
	q := fmt.Sprintf("SELECT * from profile where hotelid='%s';", id)
	var profs []proto.ProfileFlat

	dbspan := ps.tracer.StartContextSpan(sctx, "db.Query")
	error := ps.dbc.Query(q, &profs)
	dbspan.End()
	if error != nil {
		return nil, error
	}
	if len(profs) == 0 {
		return nil, fmt.Errorf("unknown hotel %s", id)
	}
	return &profs[0], nil
}

func (ps *ProfSrv) initDB(profs []*Profile) error {
	q := fmt.Sprintf("truncate profile;")
	if err := ps.dbc.Exec(q); err != nil {
		return err
	}
	for _, p := range profs {
		if err := ps.insertProf(p); err != nil {
			return err
		}
	}

	for i := 7; i <= NHOTEL; i++ {
		p := Profile{
			strconv.Itoa(i),
			"St. Regis San Francisco",
			"(415) 284-40" + strconv.Itoa(i),
			"St. Regis Museum Tower is a 42-story, 484 ft skyscraper in the South of Market district of San Francisco, California, adjacent to Yerba Buena Gardens, Moscone Center, PacBell Building and the San Francisco Museum of Modern Art.",
			&Address{
				"125",
				"3rd St",
				"San Francisco",
				"CA",
				"United States",
				"94109",
				37.7835 + float32(i)/500.0*3,
				-122.41 + float32(i)/500.0*4,
			},
			nil,
		}
		if err := ps.insertProf(&p); err != nil {
			return err
		}
	}

	return nil
}

func (ps *ProfSrv) GetProfiles(ctx fs.CtxI, req proto.ProfRequest, res *proto.ProfResult) error {
	sctx, span := ps.tracer.StartRPCSpan(&req, "GetProfiles")
	defer span.End()

	db.DPrintf(db.HOTEL_PROF, "Req %v\n", req)
	for _, id := range req.HotelIds {
		p := &proto.ProfileFlat{}
		key := id + "_prof"
		span2 := ps.tracer.StartContextSpan(sctx, "Cache.Get")
		err := ps.cachec.Get(key, p)
		span2.End()
		if err != nil {
			if !ps.cachec.IsMiss(err) {
				return err
			}
			db.DPrintf(db.HOTEL_PROF, "Cache miss: key %v\n", id)
			p, err = ps.getProf(sctx, id)
			if err != nil {
				return err
			}
			if err := ps.cachec.Put(key, p); err != nil {
				return err
			}
		}
		if p != nil && p.HotelId != "" {
			res.Hotels = append(res.Hotels, p)
		}
	}
	return nil
}
