package pathclnt

import (
	db "sigmaos/debug"
	"sigmaos/reader"
	"sigmaos/serr"
	sp "sigmaos/sigmap"
	"sigmaos/union"
)

func (pathc *PathClnt) unionScan(fid sp.Tfid, name, q string) (sp.Tfid, *serr.Err) {
	fid1, _, err := pathc.FidClnt.Walk(fid, []string{name})
	if err != nil {
		db.DPrintf(db.WALK, "unionScan: error walk: %v", err)
		return sp.NoFid, err
	}
	defer pathc.FidClnt.Clunk(fid1)

	target, err := pathc.readlink(fid1)
	if err != nil {
		db.DPrintf(db.WALK, "unionScan: Err readlink %v\n", err)
		return sp.NoFid, err
	}
	db.DPrintf(db.WALK, "unionScan: %v target: %v\n", name, string(target))
	mnt, err := sp.NewMount(target)
	if err != nil {
		return sp.NoFid, nil
	}
	db.DPrintf(db.WALK, "unionScan: %v mnt: %v\n", name, mnt)
	if union.UnionMatch(pathc.pcfg.LocalIP, q, mnt) {
		fid2, _, err := pathc.FidClnt.Walk(fid, []string{name})
		if err != nil {
			return sp.NoFid, err
		}
		return fid2, nil
	}
	return sp.NoFid, nil
}

// Caller is responsible for clunking fid
func (pathc *PathClnt) unionLookup(fid sp.Tfid, q string) (sp.Tfid, *serr.Err) {
	_, err := pathc.FidClnt.Open(fid, sp.OREAD)
	if err != nil {
		return sp.NoFid, err
	}
	rdr := reader.NewReader(pathc.FidClnt, "", fid, pathc.chunkSz)
	drdr := reader.MkDirReader(rdr)
	rfid := sp.NoFid
	_, error := reader.ReadDir(drdr, func(st *sp.Stat) (bool, error) {
		fid1, err := pathc.unionScan(fid, st.Name, q)
		if err != nil {
			db.DPrintf(db.WALK, "unionScan %v err %v\n", st.Name, err)
			// ignore error; keep going
			return false, nil
		}
		if fid1 != sp.NoFid { // success
			rfid = fid1
			return true, nil
		}
		return false, nil
	})
	if error == nil && rfid != sp.NoFid {
		return rfid, nil
	}
	return rfid, serr.NewErr(serr.TErrNotfound, q)
}
