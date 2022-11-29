package kv

import (
	"fmt"

	"sigmaos/fslib"
	np "sigmaos/sigmap"
)

type Move struct {
	Src string
	Dst string
}

type Moves []*Move

func (mvs Moves) String() string {
	s := "["
	for _, m := range mvs {
		if m == nil {
			s += fmt.Sprintf("nil,")
		} else {
			s += fmt.Sprintf("%v -> %v,", m.Src, m.Dst)
		}
	}
	s += "]"
	return s
}

type Config struct {
	Epoch  np.Tepoch
	Shards []string // slice mapping shard # to server
	Moves  Moves    // shards to be deleted because they moved
}

func (cf *Config) String() string {
	return fmt.Sprintf("{Epoch %v, Shards %v, Moves %v}", cf.Epoch, cf.Shards, cf.Moves)
}

func MakeConfig(e np.Tepoch) *Config {
	cf := &Config{e, make([]string, NSHARD), Moves{}}
	return cf
}

func (cf *Config) Present(n string) bool {
	for _, s := range cf.Shards {
		if s == n {
			return true
		}
	}
	return false
}

func readConfig(fsl *fslib.FsLib, conffile string) (*Config, error) {
	conf := Config{}
	err := fsl.GetFileJson(conffile, &conf)
	if err != nil {
		return nil, err
	}
	return &conf, nil
}

type KvSet struct {
	Set map[string]int
}

func MakeKvs(shards []string) *KvSet {
	ks := &KvSet{}
	ks.Set = make(map[string]int)
	for _, kv := range shards {
		if _, ok := ks.Set[kv]; !ok && kv != "" {
			ks.Set[kv] = 0
		}
		if kv != "" {
			ks.Set[kv] += 1
		}
	}
	return ks
}

func (ks *KvSet) present(kv string) bool {
	_, ok := ks.Set[kv]
	return ok
}

func (ks *KvSet) mkKvs() []string {
	kvs := make([]string, 0, len(ks.Set))
	for kv, _ := range ks.Set {
		kvs = append(kvs, kv)
	}
	return kvs
}

func (ks *KvSet) add(new []string) {
	for _, kv := range new {
		ks.Set[kv] = 0
	}
}

func (ks *KvSet) del(old []string) {
	for _, kv := range old {
		delete(ks.Set, kv)
	}
}

func (ks *KvSet) high(notkv string) (string, int) {
	h := ""
	n := 0
	for k, v := range ks.Set {
		if v > n && k != notkv {
			h = k
			n = v
		}
	}
	return h, n
}

func (ks *KvSet) low() (string, int) {
	l := ""
	n := NSHARD
	for k, v := range ks.Set {
		if v < n && k != "" {
			l = k
			n = v
		}
	}
	return l, n
}

func (ks *KvSet) nshards(kv string) int {
	return ks.Set[kv]
}

// assign to t shards from hkv to newkv
func assign(conf *Config, nextShards []string, hkv string, t int, newkv string) []string {
	for i, kv := range nextShards {
		if kv == hkv {
			nextShards[i] = newkv
			t -= 1
			if t <= 0 {
				break
			}
		}
	}
	return nextShards
}

func AddKv(conf *Config, newkv string) []string {
	nextShards := make([]string, NSHARD)
	kvs := MakeKvs(conf.Shards)
	l := len(kvs.mkKvs()) + 1

	if l == 1 { // newkv is first shard
		for i, _ := range conf.Shards {
			nextShards[i] = newkv
		}
		return nextShards
	}
	for i, kv := range conf.Shards {
		nextShards[i] = kv
	}
	kvs.Set[newkv] = 0
	n := (NSHARD + l - 1) / l
	// log.Printf("add: n = %v\n", n)
	for i := 0; i < n; {
		hkv, h := kvs.high(newkv)
		t := 1
		if h-n >= 1 {
			t = h - n
		}
		// log.Printf("n = %v h = %v t = %v %v->%v\n", n, h, t, hkv, newkv)
		nextShards = assign(conf, nextShards, hkv, t, newkv)
		kvs.Set[hkv] -= t
		kvs.Set[newkv] += t
		i += t
	}
	return nextShards
}

func DelKv(conf *Config, delkv string) []string {
	nextShards := make([]string, NSHARD)
	kvs := MakeKvs(conf.Shards)
	n := kvs.nshards(delkv)
	kvs.del([]string{delkv})

	l := len(kvs.mkKvs())
	p := n / l
	n1 := (NSHARD + l - 1) / l
	// log.Printf("del: n = %v p = %v n1 = %v\n", n, p, n1)
	for i, kv := range conf.Shards {
		nextShards[i] = kv
	}
	for i := n; i > 0; {
		lkv, l := kvs.low()
		t := p
		if i < p {
			t = i

		}
		if l+t > n1 {
			t = 1
		}
		// log.Printf("i = %v l = %v t = %v %v->%v\n", i, l, t, delkv, lkv)
		nextShards = assign(conf, nextShards, delkv, t, lkv)
		kvs.Set[lkv] += t
		i -= t
	}
	return nextShards
}
