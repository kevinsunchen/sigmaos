package mr

import (
	"fmt"
	"hash/fnv"
	"io"

	"github.com/dustin/go-humanize"
	"github.com/mitchellh/mapstructure"

	"ulambda/fslib"
	np "ulambda/ninep"
)

// Map and reduce functions produce and consume KeyValue pairs
type KeyValue struct {
	K string
	V string
}

type EmitT func(*KeyValue) error

// The mr library calls the reduce function once for each key
// generated by the map tasks, with a list of all the values created
// for that key by any map task.
type ReduceT func(string, []string, EmitT) error

// The mr library calls the map function for each line of input, which
// is passed in as an io.Reader.  The map function outputs its values
// by calling an emit function and passing it a KeyValue.
type MapT func(string, io.Reader, EmitT) error

// for sorting by key.
type ByKey []*KeyValue

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].K < a[j].K }

// Use Khash(key) % NReduce to choose the reduce task number for each
// KeyValue emitted by Map.
func Khash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// An input split
type Split struct {
	File   string     `json:"File"`
	Offset np.Toffset `json:"Offset"`
	Length np.Tlength `json:"Length"`
}

func (s Split) String() string {
	return fmt.Sprintf("{f %s o %v l %v}", s.File, humanize.Bytes(uint64(s.Offset)), humanize.Bytes(uint64(s.Length)))
}

type Bin []Split

func (b Bin) String() string {
	r := fmt.Sprintf("bins (%d): [ %v, ", len(b), b[0])
	for i, s := range b[1:] {
		if s.File == b[i].File {
			r += fmt.Sprintf("_ o %v l %v,", humanize.Bytes(uint64(s.Offset)), humanize.Bytes(uint64(s.Length)))
		} else {
			r += fmt.Sprintf("[ %v, ", s)
		}
	}
	return r + "]\n"
}

// Result of mapper or reducer
type Result struct {
	IsM  bool       `json:"IsM"`
	Task string     `json:"Task"`
	In   np.Tlength `json:"In"`
	Out  np.Tlength `json:"Out"`
	Ms   int64      `json:"Ms"`
}

func mkResult(data interface{}) *Result {
	r := &Result{}
	mapstructure.Decode(data, r)
	return r
}

// Each bin has a slice of splits.  Assign splits of files to a bin
// until the bin is full
func MkBins(fsl *fslib.FsLib, dir string, maxbinsz np.Tlength) ([]Bin, error) {
	bins := make([]Bin, 0)
	binsz := np.Tlength(0)
	bin := Bin{}
	splitsz := maxbinsz >> 3

	sts, err := fsl.GetDir(dir)
	if err != nil {
		return nil, err
	}
	for _, st := range sts {
		for i := np.Tlength(0); ; {
			n := splitsz
			if i+n > st.Length {
				n = st.Length - i
			}
			split := Split{dir + "/" + st.Name, np.Toffset(i), n}
			bin = append(bin, split)
			binsz += n
			if binsz+splitsz > maxbinsz { // bin full?
				bins = append(bins, bin)
				bin = Bin{}
				binsz = np.Tlength(0)
			}
			if n < splitsz { // next file
				break
			}
			i += n
		}
	}
	if binsz > 0 {
		bins = append(bins, bin)
	}
	return bins, nil
}
