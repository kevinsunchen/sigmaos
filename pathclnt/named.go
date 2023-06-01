package pathclnt

import (
	"fmt"
	"time"

	db "sigmaos/debug"
	"sigmaos/etcdclnt"
	"sigmaos/path"
	"sigmaos/serr"
	sp "sigmaos/sigmap"
)

const MAXRETRY = 60

func (pathc *PathClnt) mountNamed(p path.Path) *serr.Err {
	_, rest, err := pathc.mnt.resolve(p, false)
	if err != nil && len(rest) >= 1 && rest[0] == sp.NAMEDV1 {
		pathc.doMountNamed(p)
	}
	return nil
}

func (pathc *PathClnt) doMountNamed(p path.Path) *serr.Err {
	for i := 0; i < MAXRETRY; i++ {
		db.DPrintf(db.NAMEDV1, "mountNamed %d: %v\n", i, p)
		mnt, err := etcdclnt.GetNamed()
		if err == nil {
			if err := pathc.autoMount("", mnt, path.Path{sp.NAMEDV1}); err == nil {
				return nil
			}
			db.DPrintf(db.NAMEDV1, "mountNamed: automount err %v\n", err)
		} else {
			db.DPrintf(db.NAMEDV1, "mountNamed: GetNamed err %v\n", err)
		}
		time.Sleep(1 * time.Second)
	}
	return serr.MkErr(serr.TErrRetry, fmt.Sprintf("%v failure", sp.NAMEDV1))
}