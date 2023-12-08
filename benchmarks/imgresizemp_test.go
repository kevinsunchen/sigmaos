package benchmarks_test

import (
	"fmt"
	"path"
	"time"

	"github.com/stretchr/testify/assert"

	db "sigmaos/debug"
	"sigmaos/groupmgr"
	"sigmaos/imgresizedmp"
	"sigmaos/perf"
	"sigmaos/proc"
	rd "sigmaos/rand"
	sp "sigmaos/sigmap"
	"sigmaos/test"
)

type ImgResizeJobParams struct {
	provider sp.Tprovider
	input    string
	ntasks   int
}

func (i ImgResizeJobParams) String() string {
	return fmt.Sprintf("{input: %v; provider: %v, ntasks: %v}", i.input, i.provider, i.ntasks)
}

type ImgResizeMultiProviderJobInstance struct {
	sigmaos      bool
	job          string
	mcpu         proc.Tmcpu
	mem          proc.Tmem
	ready        chan bool
	inputs       []*ImgResizeJobParams
	imgd         *groupmgr.GroupMgr
	p            *perf.Perf
	nrounds      int
	initProvider sp.Tprovider
	*test.RealmTstate
}

func NewImgResizeJobParams(provider sp.Tprovider, input string, ntasks int) *ImgResizeJobParams {
	jp := &ImgResizeJobParams{}
	jp.provider = provider
	jp.input = input
	jp.ntasks = ntasks

	return jp
}

func NewImgResizeMultiProviderJob(ts *test.RealmTstate, p *perf.Perf, sigmaos bool, inputs []*ImgResizeJobParams, mcpu proc.Tmcpu, mem proc.Tmem, nrounds int, initProvider sp.Tprovider) *ImgResizeMultiProviderJobInstance {
	ji := &ImgResizeMultiProviderJobInstance{}
	ji.sigmaos = sigmaos
	ji.job = "imgresize-" + ts.GetRealm().String() + "-" + rd.String(4)
	ji.inputs = inputs
	ji.ready = make(chan bool)
	ji.RealmTstate = ts
	ji.p = p
	ji.mcpu = mcpu
	ji.mem = mem
	ji.nrounds = nrounds
	ji.initProvider = initProvider

	err := imgresizedmp.MkDirs(ji.FsLib, ji.job)
	assert.Nil(ts.Ts.T, err, "Error MkDirs: %v", err)

	return ji
}

func (ji *ImgResizeMultiProviderJobInstance) StartImgResizeMultiProviderJob() {
	db.DPrintf(db.ALWAYS, "StartImgResizeJob input %v mcpu %v", ji.inputs, ji.mcpu)
	ji.imgd = imgresizedmp.StartImgdMultiProvider(ji.SigmaClnt, ji.job, ji.mcpu, ji.mem, false, ji.nrounds, ji.initProvider)
	// fn := ji.input
	// fns := make([]string, 0, ji.ninputs)
	// for i := 0; i < ji.ninputs; i++ {
	// 	fns = append(fns, fn)
	// }
	for _, params := range ji.inputs {
		db.DPrintf(db.ALWAYS, "Submitting task: job %v params %v", ji.job, params)
		for i := 0; i < params.ntasks; i++ {
			err := imgresizedmp.SubmitTask(ji.SigmaClnt.FsLib, ji.job, params.provider, params.input)
			assert.Nil(ji.Ts.T, err, "Error SubmitTask: %v", err)
		}
	}
	db.DPrintf(db.ALWAYS, "Done starting ImgResizeJob")
}

func (ji *ImgResizeMultiProviderJobInstance) Wait() {
	totaltasks := 0
	for _, params := range ji.inputs {
		totaltasks += params.ntasks
	}
	db.DPrintf(db.TEST, "Waiting for ImgResizeJob to finish; %v total tasks", totaltasks)
	for {
		n, err := imgresizedmp.NTaskDone(ji.SigmaClnt.FsLib, ji.job)
		assert.Nil(ji.Ts.T, err, "Error NTaskDone: %v", err)
		db.DPrintf(db.TEST, "ImgResizeJob NTaskDone: %v", n)
		if n == totaltasks {
			break
		}
		time.Sleep(1 * time.Second)
	}
	db.DPrintf(db.TEST, "Done waiting for ImgResizeJob to finish")
	ji.imgd.StopGroup()
	db.DPrintf(db.TEST, "Imgd shutdown")
}

func (ji *ImgResizeMultiProviderJobInstance) Cleanup() {
	for _, params := range ji.inputs {
		dir := path.Join(sp.UX, "~local", path.Dir(params.input))
		db.DPrintf(db.TEST, "Cleaning up dir %v", dir)
		imgresizedmp.Cleanup(ji.FsLib, dir)
	}
}
