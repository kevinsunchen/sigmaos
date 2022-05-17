package mr

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strconv"
	"time"

	// "github.com/klauspost/readahead"

	"ulambda/awriter"
	"ulambda/crash"
	db "ulambda/debug"
	"ulambda/fslib"
	np "ulambda/ninep"
	"ulambda/proc"
	"ulambda/procclnt"
	"ulambda/rand"
	"ulambda/writer"
)

type wrt struct {
	wrt  *writer.Writer
	awrt *awriter.Writer
	bwrt *bufio.Writer
}

type Mapper struct {
	*fslib.FsLib
	*procclnt.ProcClnt
	mapf        MapT
	nreducetask int
	input       string
	file        string
	wrts        []*wrt
	rand        string
}

func makeMapper(mapf MapT, args []string) (*Mapper, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("MakeMapper: too few arguments %v", args)
	}
	m := &Mapper{}
	m.mapf = mapf
	n, err := strconv.Atoi(args[0])
	if err != nil {
		return nil, fmt.Errorf("MakeMapper: nreducetask %v isn't int", args[0])
	}
	m.nreducetask = n
	m.input = args[1]
	m.file = path.Base(m.input)
	m.rand = rand.String(16)
	m.wrts = make([]*wrt, m.nreducetask)

	m.FsLib = fslib.MakeFsLib("mapper-" + proc.GetPid().String() + " " + m.input)
	m.ProcClnt = procclnt.MakeProcClnt(m.FsLib)
	if err := m.Started(); err != nil {
		return nil, fmt.Errorf("MakeMapper couldn't start %v", args)
	}
	crash.Crasher(m.FsLib)
	return m, nil
}

func (m *Mapper) initMapper() error {
	// Make a directory for holding the output files of a map task.  Ignore
	// error in case it already exits.  XXX who cleans up?
	m.MkDir(Moutdir(m.file), 0777)

	// Create the output files
	for r := 0; r < m.nreducetask; r++ {
		// create temp output shard for reducer r
		oname := mshardfile(m.file, r) + m.rand
		w, err := m.CreateWriter(oname, 0777, np.OWRITE)
		if err != nil {
			m.closewrts()
			return fmt.Errorf("%v: create %v err %v\n", proc.GetName(), oname, err)
		}
		aw := awriter.NewWriterSize(w, BUFSZ)
		bw := bufio.NewWriterSize(aw, BUFSZ)
		m.wrts[r] = &wrt{w, aw, bw}
	}
	return nil
}

// XXX use writercloser
func (m *Mapper) closewrts() error {
	for r := 0; r < m.nreducetask; r++ {
		if m.wrts[r] != nil {
			if err := m.wrts[r].awrt.Close(); err != nil {
				return fmt.Errorf("%v: aclose %v err %v\n", proc.GetName(), m.wrts[r], err)
			}
			if err := m.wrts[r].wrt.Close(); err != nil {
				return fmt.Errorf("%v: close %v err %v\n", proc.GetName(), m.wrts[r], err)
			}
		}
	}
	return nil
}

func (m *Mapper) flushwrts() (np.Tlength, error) {
	n := np.Tlength(0)
	for r := 0; r < m.nreducetask; r++ {
		if err := m.wrts[r].bwrt.Flush(); err != nil {
			return 0, fmt.Errorf("%v: flush %v err %v\n", proc.GetName(), m.wrts[r], err)
		}
		n += m.wrts[r].wrt.Nbytes()
	}
	return n, nil
}

// Inform reducer where to find map output
func (m *Mapper) informReducer() error {
	st, err := m.Stat(MLOCALSRV)
	if err != nil {
		return fmt.Errorf("%v: stat %v err %v\n", proc.GetName(), MLOCALSRV, err)
	}
	for r := 0; r < m.nreducetask; r++ {
		fn := mshardfile(m.file, r)
		err = m.Rename(fn+m.rand, fn)
		if err != nil {
			return fmt.Errorf("%v: rename %v -> %v err %v\n", proc.GetName(), fn+m.rand, fn, err)
		}

		name := symname(strconv.Itoa(r), m.file)

		// Remove name in case an earlier mapper created the
		// symlink.  A reducer may have opened and is reading
		// the old target, open the new input file and read
		// the new target, or fail because there is no
		// symlink. Failing is fine because the coodinator
		// will start a new reducer once this map completes.
		// We could use rename to atomically remove and create
		// the symlink if we want to avoid the failing case.
		m.Remove(name)

		target := shardtarget(st.Name, m.file, r)
		err = m.Symlink([]byte(target), name, 0777)
		if err != nil {
			db.DFatalf("%v: FATAL symlink %v err %v\n", proc.GetName(), name, err)
		}
	}
	return nil
}

func (m *Mapper) emit(kv *KeyValue) error {
	r := Khash(kv.Key) % m.nreducetask
	if err := fslib.WriteJsonRecord(m.wrts[r].bwrt, kv); err != nil {
		return fmt.Errorf("%v: mapper %v err %v", proc.GetName(), r, err)
	}
	return nil
}

func (m *Mapper) doMap() (np.Tlength, np.Tlength, error) {
	rdr, err := m.OpenReader(m.input)
	if err != nil {
		db.DFatalf("%v: read %v err %v", proc.GetName(), m.input, err)
	}
	defer rdr.Close()

	brdr := bufio.NewReaderSize(rdr, BUFSZ)
	//ardr, err := readahead.NewReaderSize(rdr, 4, BUFSZ)
	//if err != nil {
	//db.DFatalf("%v: readahead.NewReaderSize err %v", proc.GetName(), err)
	//}
	if err := m.mapf(m.input, brdr, m.emit); err != nil {
		return 0, 0, err
	}
	nout, err := m.flushwrts()
	if err != nil {
		return 0, 0, err
	}
	if err := m.closewrts(); err != nil {
		return 0, 0, err
	}
	if err := m.informReducer(); err != nil {
		return 0, 0, err
	}
	return rdr.Nbytes(), nout, nil
}

func RunMapper(mapf MapT, args []string) {
	m, err := makeMapper(mapf, args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v: error %v", os.Args[0], err)
		os.Exit(1)
	}
	if err = m.initMapper(); err != nil {
		m.Exited(proc.MakeStatusErr(err.Error(), nil))
		return
	}
	start := time.Now()
	nin, nout, err := m.doMap()
	if err == nil {
		m.Exited(proc.MakeStatusInfo(proc.StatusOK, m.input, Result{nin, nout,
			time.Since(start).Milliseconds()}))
	} else {
		m.Exited(proc.MakeStatusErr(err.Error(), nil))
	}
}
