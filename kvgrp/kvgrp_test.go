package kvgrp_test

import (
	"path"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	db "sigmaos/debug"
	"sigmaos/groupmgr"
	"sigmaos/kvgrp"
	sp "sigmaos/sigmap"
	"sigmaos/test"
)

const (
	GRP       = "grp-0"
	CRASH_KVD = 5000
	N_REPL    = 3
	N_KEYS    = 10000
	JOBDIR    = "name/group"
)

type Tstate struct {
	*test.Tstate
	grp string
	gm  *groupmgr.GroupMgr
}

func makeTstate(t *testing.T, nrepl, ncrash int) *Tstate {
	ts := &Tstate{grp: GRP}
	ts.Tstate = test.MakeTstateAll(t)
	ts.RmDir(JOBDIR)
	ts.MkDir(JOBDIR, 0777)
	ts.gm = groupmgr.Start(ts.SigmaClnt, nrepl, "kvd", []string{ts.grp, strconv.FormatBool(test.Overlays)}, JOBDIR, 0, ncrash, CRASH_KVD, 0, 0)
	cfg, err := kvgrp.WaitStarted(ts.SigmaClnt.FsLib, JOBDIR, ts.grp)
	assert.Nil(t, err)
	db.DPrintf(db.TEST, "cfg %v\n", cfg)
	return ts
}

func (ts *Tstate) Shutdown() {
	ts.Tstate.Shutdown()
}

func (ts *Tstate) setupKeys(nkeys int) {
	db.DPrintf(db.TEST, "setupKeys")
	for i := 0; i < nkeys; i++ {
		i_str := strconv.Itoa(i)
		fname := path.Join(kvgrp.GrpPath(JOBDIR, ts.grp), i_str)
		_, err := ts.PutFile(fname, 0777, sp.OWRITE|sp.OREAD, []byte(i_str))
		assert.Nil(ts.T, err, "Put %v", err)
	}
	db.DPrintf(db.TEST, "done setupKeys")
}

func (ts *Tstate) testGetPutSet(nkeys int) {
	db.DPrintf(db.TEST, "testGetPutSet")
	for i := 0; i < nkeys; i++ {
		i_str := strconv.Itoa(i)
		fname := path.Join(kvgrp.GrpPath(JOBDIR, ts.grp), i_str)
		b, err := ts.GetFile(fname)
		assert.Nil(ts.T, err, "Get %v", err)
		assert.Equal(ts.T, i_str, string(b), "Didn't read expected")
		_, err = ts.PutFile(fname, 0777, sp.OWRITE|sp.OREAD|sp.OEXCL, []byte(i_str))
		assert.NotNil(ts.T, err, "Put nil")
		_, err = ts.SetFile(fname, []byte(i_str+i_str), sp.OWRITE|sp.OREAD, 0)
		assert.Nil(ts.T, err, "Set %v", err)
	}
	db.DPrintf(db.TEST, "done testGetPutSet")
}

func TestStartStopRepl0(t *testing.T) {
	ts := makeTstate(t, 0, 0)

	sts, _, err := ts.ReadDir(kvgrp.GrpPath(JOBDIR, ts.grp) + "/")
	db.DPrintf(db.TEST, "Stat: %v %v\n", sp.Names(sts), err)
	assert.Nil(t, err, "stat")

	err = ts.gm.Stop()
	assert.Nil(ts.T, err, "Stop")
	ts.Shutdown()
}

func TestStartStopRepl1(t *testing.T) {
	ts := makeTstate(t, 1, 0)

	st, err := ts.Stat(kvgrp.GrpPath(JOBDIR, ts.grp) + "/")
	db.DPrintf(db.TEST, "Stat: %v %v\n", st, err)
	assert.Nil(t, err, "stat")

	sts, _, err := ts.ReadDir(kvgrp.GrpPath(JOBDIR, ts.grp) + "/")
	db.DPrintf(db.TEST, "Stat: %v %v\n", sp.Names(sts), err)
	assert.Nil(t, err, "stat")

	err = ts.gm.Stop()
	assert.Nil(ts.T, err, "Stop")
	ts.Shutdown()
}

func TestStartStopReplN(t *testing.T) {
	ts := makeTstate(t, N_REPL, 0)
	err := ts.gm.Stop()
	assert.Nil(ts.T, err, "Stop")
	ts.Shutdown()
}

func TestGetPutSetReplOK(t *testing.T) {
	ts := makeTstate(t, N_REPL, 0)
	ts.setupKeys(N_KEYS)
	ts.testGetPutSet(N_KEYS)
	err := ts.gm.Stop()
	assert.Nil(ts.T, err, "Stop")
	ts.Shutdown()
}

func TestGetPutSetFail1(t *testing.T) {
	ts := makeTstate(t, N_REPL, 1)
	ts.setupKeys(N_KEYS)
	ts.testGetPutSet(N_KEYS)
	db.DPrintf(db.TEST, "Pre stop")
	err := ts.gm.Stop()
	assert.Nil(ts.T, err, "Stop")
	db.DPrintf(db.TEST, "Post stop")
	ts.Shutdown()
}