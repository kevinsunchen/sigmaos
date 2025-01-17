package semclnt_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"sigmaos/delay"
	"sigmaos/fslib"
	"sigmaos/proc"
	"sigmaos/semclnt"
	"sigmaos/test"
)

const (
	WAIT_PATH = "name/wait"
)

func TestCompile(t *testing.T) {
}

func TestSemClntSimple(t *testing.T) {
	ts := test.NewTstate(t)

	err := ts.MkDir(WAIT_PATH, 0777)
	assert.Nil(ts.T, err, "Mkdir")
	pcfg := proc.NewAddedProcEnv(ts.ProcEnv(), 1)
	fsl0, err := fslib.NewFsLib(pcfg)
	assert.Nil(ts.T, err, "fsl0")

	sem := semclnt.NewSemClnt(ts.FsLib, WAIT_PATH+"/x")
	sem.Init(0)

	ch := make(chan bool)
	go func(ch chan bool) {
		sem := semclnt.NewSemClnt(fsl0, WAIT_PATH+"/x")
		sem.Down()
		ch <- true
	}(ch)

	time.Sleep(100 * time.Millisecond)

	select {
	case ok := <-ch:
		assert.False(ts.T, ok, "down should be blocked")
	default:
	}

	sem.Up()

	ok := <-ch
	assert.True(ts.T, ok, "down")

	err = ts.RmDir(WAIT_PATH)
	assert.Nil(t, err, "RmDir: %v", err)

	ts.Shutdown()
}

func TestSemClntConcur(t *testing.T) {
	ts := test.NewTstate(t)

	err := ts.MkDir(WAIT_PATH, 0777)
	assert.Nil(ts.T, err, "Mkdir")
	pcfg1 := proc.NewAddedProcEnv(ts.ProcEnv(), 1)
	fsl0, err := fslib.NewFsLib(pcfg1)
	assert.Nil(ts.T, err, "fsl0")
	pcfg2 := proc.NewAddedProcEnv(ts.ProcEnv(), 2)
	fsl1, err := fslib.NewFsLib(pcfg2)
	assert.Nil(ts.T, err, "fsl1")

	for i := 0; i < 100; i++ {
		sem := semclnt.NewSemClnt(ts.FsLib, WAIT_PATH+"/x")
		sem.Init(0)

		ch := make(chan bool)

		go func(ch chan bool) {
			delay.Delay(200)
			sem := semclnt.NewSemClnt(fsl0, WAIT_PATH+"/x")
			sem.Down()
			ch <- true
		}(ch)
		go func(ch chan bool) {
			delay.Delay(200)
			sem := semclnt.NewSemClnt(fsl1, WAIT_PATH+"/x")
			sem.Up()
			ch <- true
		}(ch)

		for i := 0; i < 2; i++ {
			<-ch
		}
	}
	err = ts.RmDir(WAIT_PATH)
	assert.Nil(t, err, "RmDir: %v", err)
	ts.Shutdown()
}
