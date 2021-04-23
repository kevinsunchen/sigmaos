package locald

import (
	//	"github.com/sasha-s/go-deadlock"
	"encoding/json"
	"io"
	"log"
	"net"
	"sync"
	"time"

	db "ulambda/debug"
	"ulambda/fsclnt"
	"ulambda/fslib"
	np "ulambda/ninep"
	npo "ulambda/npobjsrv"
	"ulambda/npsrv"
)

const (
	NO_OP_LAMBDA = "no-op-lambda"
)

type LocalD struct {
	//	mu deadlock.Mutex
	mu   sync.Mutex
	cond *sync.Cond
	load int // XXX bogus
	bin  string
	nid  uint64
	root *Dir
	done bool
	ip   string
	ls   map[string]*Lambda
	srv  *npsrv.NpServer
	*fslib.FsLib
	group sync.WaitGroup
}

func MakeLocalD(bin string) *LocalD {
	ld := &LocalD{}
	ld.cond = sync.NewCond(&ld.mu)
	ld.load = 0
	ld.nid = 0
	ld.bin = bin
	db.Name("locald")
	ld.root = ld.makeDir([]string{}, np.DMDIR, nil)
	ld.root.time = time.Now().Unix()
	ld.ls = map[string]*Lambda{}
	ip, err := fsclnt.LocalIP()
	ld.ip = ip
	if err != nil {
		log.Fatalf("LocalIP %v\n", err)
	}
	ld.srv = npsrv.MakeNpServer(ld, ld.ip+":0")
	fsl := fslib.MakeFsLib("locald")
	fsl.Mkdir(fslib.LOCALD_ROOT, 0777)
	ld.FsLib = fsl
	err = fsl.PostServiceUnion(ld.srv.MyAddr(), fslib.LOCALD_ROOT, ld.srv.MyAddr())
	if err != nil {
		log.Fatalf("PostServiceUnion failed %v %v\n", ld.srv.MyAddr(), err)
	}
	// Try to make scheduling directories if they don't already exist
	fsl.Mkdir(fslib.RUNQ, 0777)
	fsl.Mkdir(fslib.WAITQ, 0777)
	fsl.Mkdir(fslib.CLAIMED, 0777)
	fsl.Mkdir(fslib.CLAIMED_EPH, 0777)
	fsl.Mkdir(fslib.LOCKS, 0777)
	return ld
}

func (ld *LocalD) spawn(a []byte) (*Lambda, error) {
	ld.mu.Lock()
	defer ld.mu.Unlock()
	l := &Lambda{}
	l.ld = ld
	err := l.init(a)
	if err != nil {
		return nil, err
	}
	ld.ls[l.Pid] = l
	return l, nil
}

func (ld *LocalD) Connect(conn net.Conn) npsrv.NpAPI {
	return npo.MakeNpConn(ld, conn)
}

func (ld *LocalD) Done() {
	ld.mu.Lock()
	defer ld.mu.Unlock()

	ld.done = true
	ld.SignalNewJob()
}

func (ld *LocalD) WatchTable() *npo.WatchTable {
	return nil
}

func (ld *LocalD) ConnTable() *npo.ConnTable {
	return nil
}

func (ld *LocalD) readDone() bool {
	ld.mu.Lock()
	defer ld.mu.Unlock()
	return ld.done
}

func (ld *LocalD) RootAttach(uname string) (npo.NpObj, npo.CtxI) {
	return ld.root, nil
}

// Tries to claim a job from the runq. If none are available, return.
func (ld *LocalD) getLambda() ([]byte, error) {
	err := ld.WaitForJob()
	if err != nil {
		return []byte{}, err
	}
	jobs, err := ld.ReadRunQ()
	if err != nil {
		return []byte{}, err
	}
	for _, j := range jobs {
		b, claimed := ld.ClaimRunQJob(j.Name)
		if err != nil {
			return []byte{}, err
		}
		if claimed {
			return b, nil
		}
	}
	return []byte{}, nil
}

// Scan through the waitq, and try to move jobs to the runq.
func (ld *LocalD) checkWaitingLambdas() {
	jobs, err := ld.ReadWaitQ()
	if err != nil {
		log.Fatalf("Error reading WaitQ: %v", err)
	}
	for _, j := range jobs {
		b, err := ld.ReadWaitQJob(j.Name)
		// Ignore errors: they may be frequent under high concurrency
		if err != nil || len(b) == 0 {
			continue
		}
		if ld.jobIsRunnable(j, b) {
			// Ignore errors: they may be frequent under high concurrency
			ld.MarkJobRunnable(j.Name)
		}
	}
}

/*
 * 1. Timer-based lambdas are runnable after Mtime + attr.Timer > time.Now()
 * 2. PairDep-based lambdas are runnable only if they are the producer (whoever
 *    claims and runs the producer will also start the consumer, so we disallow
 *    unilaterally claiming the consumer for now), and only once all of their
 *    consumers have been spawned. For now we assume that
 *    consumers only have one producer, and the roles of producer and consumer
 *    are mutually exclusive. We also expect (though not strictly necessary)
 *    that producers only have one consumer each. If this is no longer the case,
 *    we should handle oversubscription more carefully.
 * 3. ExitDep-based lambdas are runnable after all entries in the ExitDep map
 *    are true, whether that be because the dependencies explicitly exited or
 *    because they did not exist at spawn time (and were pruned).
 *
 * ***For now, we assume the three "types" described above are mutually
 *    exclusive***
 */
func (ld *LocalD) jobIsRunnable(j *np.Stat, a []byte) bool {
	var attr fslib.Attr
	err := json.Unmarshal(a, &attr)
	if err != nil {
		log.Printf("Couldn't unmarshal job to check if runnable %v: %v", a, err)
		return false
	}
	// If this is a timer-based lambda
	if attr.Timer != 0 {
		// If the timer has expired
		if uint32(time.Now().Unix()) > j.Mtime+attr.Timer {
			return true
		} else {
			// XXX Factor this out & do it in a monitor lambda
			// For now, just make sure *some* locald eventually wakes up to mark this
			// lambda as runnable. Otherwise, if there are only timer lambdas, localds
			// may never wake up to scan them.
			go func(timer uint32) {
				dur := time.Duration(uint64(timer) * 1000000000)
				time.Sleep(dur)
				ld.SignalNewJob()
			}(attr.Timer)
			return false
		}
	}

	// If this is a PairDep-based labmda
	allConsSpawned := true
	for _, pair := range attr.PairDep {
		if attr.Pid == pair.Consumer {
			return false
		} else if attr.Pid == pair.Producer {
			allConsSpawned = allConsSpawned && ld.HasBeenSpawned(pair.Consumer)
		} else {
			log.Fatalf("Locald got PairDep-based lambda with lambda not in pair: %v, %v", attr.Pid, pair)
		}
	}

	// Bail out if a consumer hasn't been spawned yet.
	if !allConsSpawned {
		return false
	}

	// If this is an ExitDep-based lambda
	for _, b := range attr.ExitDep {
		if !b {
			return false
		}
	}
	return true
}

func (ld *LocalD) claimConsumers(consumers []string) [][]byte {
	bs := [][]byte{}
	for _, c := range consumers {
		if b, ok := ld.ClaimWaitQJob(c); ok {
			bs = append(bs, b)
		} else {
			runq, _ := ld.ReadRunQ()
			waitq, _ := ld.ReadWaitQ()
			log.Fatalf("Couldn't claim consumer job: %v, runq:%v, waitq:%v", c, runq, waitq)
		}
	}
	return bs
}

func (ld *LocalD) spawnConsumers(bs [][]byte) []*Lambda {
	ls := []*Lambda{}
	for _, b := range bs {
		l, err := ld.spawn(b)
		if err != nil {
			log.Fatalf("Couldn't spawn consumer job: %v", string(b))
		}
		ls = append(ls, l)
	}
	return ls
}

func (ld *LocalD) runAll(ls []*Lambda) {
	var wg sync.WaitGroup
	for _, l := range ls {
		wg.Add(1)
		go func(l *Lambda) {
			defer wg.Done()
			l.run()
		}(l)
	}
	wg.Wait()
}

// Worker runs one lambda at a time
func (ld *LocalD) Worker(workerId uint) {
	ld.SignalNewJob()

	// TODO pin to a core
	for !ld.readDone() {
		b, err := ld.getLambda()
		// If no job was on the runq, try and move some from waitq -> runq
		if err == nil && len(b) == 0 {
			ld.checkWaitingLambdas()
			continue
		}
		if err == io.EOF {
			continue
		}
		if err != nil {
			log.Fatalf("Locald GetLambda error %v, %v\n", err, b)
		}
		// XXX return err from spawn
		l, err := ld.spawn(b)
		if err != nil {
			log.Fatalf("Locald spawn error %v\n", err)
		}
		// Try to claim, spawn, and run consumers if this lamba is a producer
		consumers := l.getConsumers()
		bs := ld.claimConsumers(consumers)
		consumerLs := ld.spawnConsumers(bs)
		ls := []*Lambda{l}
		ls = append(ls, consumerLs...)
		ld.runAll(ls)
	}
	ld.SignalNewJob()
	ld.group.Done()
}

func (ld *LocalD) Work() {
	var NWorkers uint
	if NCores < 20 {
		NWorkers = 20
	} else {
		NWorkers = NCores
	}
	for i := uint(0); i < NWorkers; i++ {
		ld.group.Add(1)
		go ld.Worker(i)
	}
	ld.group.Wait()

}
