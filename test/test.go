package test

import (
	"flag"
	"fmt"
	"testing"

	"sigmaos/bootkernelclnt"
	"sigmaos/container"
	db "sigmaos/debug"
	"sigmaos/proc"
	"sigmaos/realmclnt"
	"sigmaos/sigmaclnt"
	sp "sigmaos/sigmap"
)

//
// If running test with --start, test will start sigmaos kernel.
// Without --start, it will test create a kernelclnt without starting
// kernel.
//

const (
	BOOT_REALM       = "realm"
	BOOT_ALL         = "all"
	BOOT_NAMED       = "named"
	BOOT_NODE        = "node"
	BOOT_LCSCHEDNODE = "lcschednode"
)

var Start bool
var noShutdown bool
var tag string
var EtcdIP string
var Overlays bool

func init() {
	flag.StringVar(&EtcdIP, "etcdIP", "127.0.0.1", "Etcd IP")
	flag.StringVar(&tag, "tag", "local-build", "Docker image tag")
	flag.BoolVar(&Start, "start", false, "Start system")
	flag.BoolVar(&noShutdown, "no-shutdown", false, "Don't shut down the system")
	flag.BoolVar(&Overlays, "overlays", false, "Overlays")
}

func Mbyte(sz sp.Tlength) float64 {
	return float64(sz) / float64(sp.MBYTE)
}

func TputStr(sz sp.Tlength, ms int64) string {
	s := float64(ms) / 1000
	return fmt.Sprintf("%.2fMB/s", Mbyte(sz)/s)
}

func Tput(sz sp.Tlength, ms int64) float64 {
	t := float64(ms) / 1000
	return Mbyte(sz) / t
}

type Tstate struct {
	*sigmaclnt.SigmaClnt
	rc      *realmclnt.RealmClnt
	kclnts  []*bootkernelclnt.Kernel
	killidx int
	T       *testing.T
	proc    *proc.Proc
}

func NewTstatePath(t *testing.T, path string) *Tstate {
	ts, err := newSysClntPath(t, path)
	if err != nil {
		db.DFatalf("NewTstatePath: %v\n", err)
	}
	return ts
}

func NewTstate(t *testing.T) *Tstate {
	return NewTstatePath(t, sp.NAMED)
}

func NewTstateAll(t *testing.T) *Tstate {
	return NewTstatePath(t, "all")
}

func NewTstateAllWithProvider(t *testing.T, provider sp.Tprovider) *Tstate {
	ts, err := newSysClntWithProvider(t, BOOT_ALL, provider)
	if err != nil {
		db.DFatalf("NewTstatePath: %v\n", err)
	}
	return ts
}

func NewTstateWithRealms(t *testing.T) *Tstate {
	ts, err := newSysClnt(t, BOOT_REALM)
	if err != nil {
		db.DFatalf("NewTstateRealm: %v\n", err)
	}
	rc, err := realmclnt.NewRealmClnt(ts.FsLib)
	if err != nil {
		db.DFatalf("NewRealmClnt make realmclnt: %v\n", err)
	}
	ts.rc = rc
	return ts
}

func newSysClntPath(t *testing.T, path string) (*Tstate, error) {
	if path == sp.NAMED {
		return newSysClnt(t, BOOT_NAMED)
	} else {
		return newSysClnt(t, BOOT_ALL)
	}
}

func newSysClnt(t *testing.T, srvs string) (*Tstate, error) {
	return newSysClntWithProvider(t, srvs, sp.DEFAULT_PRVDR)
}

func newSysClntWithProvider(t *testing.T, srvs string, provider sp.Tprovider) (*Tstate, error) {
	localIP, err1 := container.LocalIP()
	if err1 != nil {
		db.DFatalf("Error local IP: %v", err1)
	}
	pcfg := proc.NewTestProcEnv(sp.ROOTREALM, EtcdIP, localIP, tag, Overlays)
	proc.SetSigmaDebugPid(pcfg.GetPID().String())
	var kernelid string
	var err error
	var k *bootkernelclnt.Kernel
	if Start {
		kernelid = bootkernelclnt.GenKernelId()
		_, err := bootkernelclnt.StartWithProvider(kernelid, pcfg, srvs, Overlays, provider)
		if err != nil {
			db.DPrintf(db.ALWAYS, "Error start kernel")
			return nil, err
		}
	}
	if err != nil {
		db.DPrintf(db.ALWAYS, "Error set named ip")
		return nil, err
	}
	k, err = bootkernelclnt.NewKernelClnt(kernelid, pcfg)
	if err != nil {
		db.DPrintf(db.ALWAYS, "Error make kernel clnt")
		return nil, err
	}
	return &Tstate{
		SigmaClnt: k.SigmaClnt,
		kclnts:    []*bootkernelclnt.Kernel{k},
		killidx:   0,
		T:         t,
	}, nil
}

func (ts *Tstate) BootNode(n int) error {
	return ts.BootNodeWithProvider(n, sp.DEFAULT_PRVDR)
}

func (ts *Tstate) BootNodeWithProvider(n int, provider sp.Tprovider) error {
	for i := 0; i < n; i++ {
		kclnt, err := bootkernelclnt.NewKernelClntStartWithProvider(ts.ProcEnv(), BOOT_NODE, Overlays, provider)
		if err != nil {
			return err
		}
		ts.kclnts = append(ts.kclnts, kclnt)
	}
	return nil
}

func (ts *Tstate) BootLcschedNodeWithProvider(provider sp.Tprovider) error {
	kclnt, err := bootkernelclnt.NewKernelClntStartWithProvider(ts.ProcEnv(), BOOT_LCSCHEDNODE, Overlays, provider)
	if err != nil {
		return err
	}
	ts.kclnts = append(ts.kclnts, kclnt)

	return nil
}

func (ts *Tstate) Boot(s string) error {
	return ts.kclnts[0].Boot(s)
}

func (ts *Tstate) BootFss3d() error {
	return ts.Boot(sp.S3REL)
}

func (ts *Tstate) KillOne(s string) error {
	idx := ts.killidx
	ts.killidx++
	return ts.kclnts[idx].Kill(s)
}

func (ts *Tstate) NewClnt(idx int, pcfg *proc.ProcEnv) (*sigmaclnt.SigmaClnt, error) {
	return ts.kclnts[idx].NewSigmaClnt(pcfg)
}

func (ts *Tstate) Shutdown() error {
	db.DPrintf(db.TEST, "Shutdown")
	defer db.DPrintf(db.TEST, "Done Shutdown")
	if noShutdown {
		db.DPrintf(db.ALWAYS, "Skipping shutdown")
		db.DPrintf(db.TEST, "Skipping shutdown")
	} else {
		db.DPrintf(db.SYSTEM, "Shutdown")
		if err := ts.RmDir(ts.ProcEnv().ProcDir); err != nil {
			db.DPrintf(db.ALWAYS, "Failed to clean up %v err %v", ts.ProcEnv().ProcDir, err)
		}
		// Shut down other kernel; the one running named last
		for i := len(ts.kclnts) - 1; i >= 0; i-- {
			if err := ts.kclnts[i].Shutdown(); err != nil {
				db.DPrintf(db.ALWAYS, "Shutdown %v err %v", ts.kclnts[i].KernelId, err)
			}
		}
	}
	return nil
}
