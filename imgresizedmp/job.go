package imgresizedmp

import (
	"strconv"

	"sigmaos/groupmgr"
	"sigmaos/proc"
	"sigmaos/sigmaclnt"
	sp "sigmaos/sigmap"
)

func StartImgdMultiProvider(sc *sigmaclnt.SigmaClnt, job string, workerMcpu proc.Tmcpu, workerMem proc.Tmem, persist bool, nrounds int, groupProvider sp.Tprovider) *groupmgr.GroupMgr {
	cfg := groupmgr.NewGroupConfigWithProvider(1, "imgresizedmp", []string{strconv.Itoa(0), strconv.Itoa(int(workerMcpu)), strconv.Itoa(int(workerMem)), strconv.Itoa(nrounds)}, 0, job, groupProvider)
	if persist {
		cfg.Persist(sc.FsLib)
	}
	return cfg.StartGrpMgr(sc, 0)
}
