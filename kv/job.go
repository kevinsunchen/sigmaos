package kv

import (
	"path"
	"strconv"

	"sigmaos/fslib"
	"sigmaos/group"
	"sigmaos/groupmgr"
	"sigmaos/proc"
	"sigmaos/procclnt"
	"sigmaos/rand"
	"sigmaos/semclnt"
	"sigmaos/sigmaclnt"
	"sigmaos/test"
)

const (
	NKV           = 10
	NSHARD        = 10 * NKV
	NBALANCER     = 3
	KVDIR         = "name/kv/"
	KVCONF        = "config"
	KVBALANCER    = "balancer"
	KVBALANCERCTL = "ctl"
)

func JobDir(job string) string {
	return path.Join(KVDIR, job)
}

func KVConfig(job string) string {
	return path.Join(JobDir(job), KVCONF)
}

func KVBalancer(job string) string {
	return path.Join(JobDir(job), KVBALANCER)
}

func KVBalancerCtl(job string) string {
	return path.Join(KVBalancer(job), KVBALANCERCTL)
}

// TODO make grpdir a subdir of this job.
func kvShardPath(job, kvd string, shard Tshard) string {
	return path.Join(group.GrpPath(JobDir(job), kvd), "shard"+shard.String())
}

type KVFleet struct {
	*sigmaclnt.SigmaClnt
	nkvd     int        // Number of kvd groups to run the test with.
	kvdrepl  int        // kvd replication level
	kvdncore proc.Tcore // Number of exclusive cores allocated to each kvd.
	ck       *KvClerk   // A clerk which can be used for initialization.
	auto     string     // Balancer auto-balancing setting.
	job      string
	ready    chan bool
	sem      *semclnt.SemClnt
	sempath  string
	balgm    *groupmgr.GroupMgr
	kvdgms   []*groupmgr.GroupMgr
	cpids    []proc.Tpid
}

func MakeKvdFleet(sc *sigmaclnt.SigmaClnt, nkvd int, kvdrepl int, kvdncore proc.Tcore, auto string) (*KVFleet, error) {
	kvf := &KVFleet{}
	kvf.SigmaClnt = sc
	kvf.nkvd = nkvd
	kvf.kvdrepl = kvdrepl
	kvf.kvdncore = kvdncore
	kvf.job = rand.String(16)
	kvf.auto = auto
	kvf.ready = make(chan bool)

	// May already exit
	kvf.MkDir(KVDIR, 0777)
	// Should not exist.
	if err := kvf.MkDir(JobDir(kvf.job), 0777); err != nil {
		return nil, err
	}

	kvf.sempath = path.Join(JobDir(kvf.job), "kvclerk-sem")
	kvf.sem = semclnt.MakeSemClnt(kvf.FsLib, kvf.sempath)
	if err := kvf.sem.Init(0); err != nil {
		return nil, err
	}
	kvf.kvdgms = []*groupmgr.GroupMgr{}
	kvf.cpids = []proc.Tpid{}
	return kvf, nil
}

func (kvf *KVFleet) Job() string {
	return kvf.job
}

func (kvf *KVFleet) StartJob() error {
	kvf.balgm = StartBalancers(kvf.FsLib, kvf.ProcClnt, kvf.job, NBALANCER, 0, kvf.kvdncore, "0", kvf.auto)
	// Add an initial kvd group to put keys in.
	return kvf.AddKVDGroup()
}

func (kvf *KVFleet) AddKVDGroup() error {
	// Name group
	grp := group.GRP + strconv.Itoa(len(kvf.kvdgms))
	// Spawn group
	kvf.kvdgms = append(kvf.kvdgms, SpawnGrp(kvf.FsLib, kvf.ProcClnt, kvf.job, grp, kvf.kvdncore, kvf.kvdrepl, 0))
	// Get balancer to add the group
	if err := BalancerOpRetry(kvf.FsLib, kvf.job, "add", grp); err != nil {
		return err
	}
	return nil
}

func (kvf *KVFleet) RemoveKVDGroup() error {
	n := len(kvf.kvdgms) - 1
	// Get group nambe
	grp := group.GRP + strconv.Itoa(n)
	// Get balancer to remove the group
	if err := BalancerOpRetry(kvf.FsLib, kvf.job, "del", grp); err != nil {
		return err
	}
	// Stop kvd group
	if err := kvf.kvdgms[n].Stop(); err != nil {
		return err
	}
	// Remove kvd group
	kvf.kvdgms = kvf.kvdgms[:n]
	return nil
}

func (kvf *KVFleet) Stop() error {
	nkvds := len(kvf.kvdgms)
	for i := 0; i < nkvds-1; i++ {
		kvf.RemoveKVDGroup()
	}
	// Stop the balancers.
	kvf.balgm.Stop()
	// Remove the last kvd group after removing the balancer.
	kvf.kvdgms[0].Stop()
	kvf.kvdgms = nil
	if err := RemoveJob(kvf.FsLib, kvf.job); err != nil {
		return err
	}
	return nil
}

func StartBalancers(fsl *fslib.FsLib, pclnt *procclnt.ProcClnt, jobname string, nbal, crashbal int, kvdncore proc.Tcore, crashhelper, auto string) *groupmgr.GroupMgr {
	kvdnc := strconv.Itoa(int(kvdncore))
	return groupmgr.Start(fsl, pclnt, nbal, "balancer", []string{crashhelper, kvdnc, auto}, jobname, 0, nbal, crashbal, 0, 0)
}

func SpawnGrp(fsl *fslib.FsLib, pclnt *procclnt.ProcClnt, jobname, grp string, ncore proc.Tcore, repl, ncrash int) *groupmgr.GroupMgr {
	return groupmgr.Start(fsl, pclnt, repl, "kvd", []string{grp, strconv.FormatBool(test.Overlays)}, JobDir(jobname), ncore, ncrash, CRASHKVD, 0, 0)
}

func InitKeys(sc *sigmaclnt.SigmaClnt, job string, nkeys int) (*KvClerk, error) {
	// Create keys
	clrk, err := MakeClerkFsl(sc, job)
	if err != nil {
		return nil, err
	}
	for i := uint64(0); i < uint64(nkeys); i++ {
		err := clrk.Put(MkKey(i), []byte{})
		if err != nil {
			return clrk, err
		}
	}
	return clrk, nil
}

func StartClerk(pclnt *procclnt.ProcClnt, job string, args []string, ncore proc.Tcore) (proc.Tpid, error) {
	args = append([]string{job}, args...)
	p := proc.MakeProc("kv-clerk", args)
	p.SetNcore(ncore)
	// SpawnBurst to spread clerks across procds.
	_, errs := pclnt.SpawnBurst([]*proc.Proc{p})
	if len(errs) > 0 {
		return p.GetPid(), errs[0]
	}
	err := pclnt.WaitStart(p.GetPid())
	return p.GetPid(), err
}

func StopClerk(pclnt *procclnt.ProcClnt, pid proc.Tpid) (*proc.Status, error) {
	err := pclnt.Evict(pid)
	if err != nil {
		return nil, err
	}
	status, err := pclnt.WaitExit(pid)
	return status, err
}

func RemoveJob(fsl *fslib.FsLib, job string) error {
	return fsl.RmDir(JobDir(job))
}
