package kv

import (
	"fmt"
	"log"
	"strconv"
	"sync"

	"ulambda/crash"
	"ulambda/fenceclnt"
	"ulambda/fslib"
	"ulambda/proc"
	"ulambda/procclnt"
)

// XXX cmd line utility rmdir

type Deleter struct {
	mu sync.Mutex
	*fslib.FsLib
	*procclnt.ProcClnt
	fclnt  *fenceclnt.FenceClnt
	blConf Config
}

func MakeDeleter(N string) (*Deleter, error) {
	dl := &Deleter{}
	dl.FsLib = fslib.MakeFsLib("deleter-" + proc.GetPid())
	dl.ProcClnt = procclnt.MakeProcClnt(dl.FsLib)
	crash.Crasher(dl.FsLib)
	err := dl.Started(proc.GetPid())
	dl.fclnt = fenceclnt.MakeFenceClnt(dl.FsLib, KVCONFIG, 0)
	err = dl.fclnt.AcquireConfig(&dl.blConf)
	if err != nil {
		log.Printf("%v: fence %v err %v\n", proc.GetProgram(), dl.fclnt.Name(), err)
		return nil, err
	}
	// log.Printf("%v: bal config %v\n", proc.GetProgram(), dl.blConf.N)
	if N != strconv.Itoa(dl.blConf.N) {
		log.Printf("%v: wrong config %v\n", proc.GetProgram(), N)
		return nil, fmt.Errorf("wrong config %v\n", N)
	}
	return dl, err
}

func (dl *Deleter) Delete(sharddir string) {
	// log.Printf("%v: conf %v delete %v\n", proc.GetProgram(), dl.blConf.N, sharddir)
	err := dl.RmDir(sharddir)
	if err != nil {
		log.Printf("%v: conf %v rmdir %v err %v\n", proc.GetProgram(), dl.blConf.N, sharddir, err)
		dl.Exited(proc.GetPid(), proc.MakeStatusErr(err.Error()))
	} else {
		dl.Exited(proc.GetPid(), proc.MakeStatus(proc.StatusOK))
	}
}
