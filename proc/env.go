package proc

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"runtime/debug"
	"strconv"

	sp "sigmaos/sigmap"
)

const (
	SIGMACONFIG = "SIGMACONFIG"
	NOT_SET     = "NOT_SET" // Catch cases where we fail to set a variable.
)

type ProcEnv struct {
	*ProcEnvProto
	// PID        sp.Tpid   `json:pid,omitempty`
	// Realm      sp.Trealm `json:realm,omitempty`
	// Uname      sp.Tuname `json:uname,omitempty`
	// KernelID   string    `json:kernelid,omitempty`
	// UprocdPID  sp.Tpid   `json:uprocdpid,omitempty`
	// Net        string    `json:net,omitempty`
	// Privileged bool      `json:privileged,omitempty` // XXX phase out?
	// Program    string    `json:program,omitempty`
	// ProcDir    string    `json:procdir,omitempty`   // XXX phase out?
	// ParentDir  string    `json:parentdir,omitempty` // XXX phase out?
	// Perf       string    `json:perf,omitempty`
	// Debug      string    `json:debug,omitempty`
	// EtcdIP     string    `json:etcdip,omitempty`
	// LocalIP    string    `json:localip,omitempty`
	// BuildTag   string    `json:buildtag,omitempty`
	// // For testing purposes
	// Crash     bool `json:crash,omitempty`
	// Partition bool `json:partition,omitempty`
}

func GetProcEnv() *ProcEnv {
	pestr := os.Getenv(SIGMACONFIG)
	if pestr == "" {
		stack := debug.Stack()
		log.Fatalf("%s\nError: No Sigma Config", stack)
	}
	return Unmarshal(pestr)
}

func NewProcEnv(program string, pid sp.Tpid, realm sp.Trealm, uname sp.Tuname, procDir string, parentDir string, priv bool) *ProcEnv {
	// Load Perf & Debug from the environment for convenience.
	return &ProcEnv{
		ProcEnvProto: &ProcEnvProto{
			PidStr:       string(pid),
			RealmStr:     string(realm),
			UnameStr:     string(uname),
			ProcDir:      procDir,
			ParentDir:    parentDir,
			Program:      program,
			LocalIP:      NOT_SET,
			KernelID:     NOT_SET,
			BuildTag:     NOT_SET,
			Net:          NOT_SET,
			Perf:         os.Getenv(SIGMAPERF),
			Debug:        os.Getenv(SIGMADEBUG),
			UprocdPIDStr: NOT_SET,
			Privileged:   priv,
		},
	}
}

func NewProcEnvUnset(priv bool) *ProcEnv {
	// Load Perf & Debug from the environment for convenience.
	return NewProcEnv(NOT_SET, sp.Tpid(NOT_SET), sp.Trealm(NOT_SET), sp.Tuname(NOT_SET), NOT_SET, NOT_SET, priv)
}

func NewProcEnvFromProto(p *ProcEnvProto) *ProcEnv {
	return &ProcEnv{p}
}

func NewBootProcEnv(uname sp.Tuname, etcdIP, localIP string) *ProcEnv {
	pe := NewProcEnvUnset(true)
	pe.SetUname(uname)
	pe.Program = "kernel"
	pe.SetPID(sp.GenPid(string(uname)))
	pe.EtcdIP = etcdIP
	pe.LocalIP = localIP
	pe.SetRealm(sp.ROOTREALM)
	pe.ProcDir = path.Join(sp.KPIDS, pe.GetPID().String())
	return pe
}

func NewTestProcEnv(realm sp.Trealm, etcdIP, localIP, buildTag string) *ProcEnv {
	pe := NewProcEnvUnset(true)
	pe.SetUname("test")
	pe.SetPID(sp.GenPid("test"))
	pe.SetRealm(realm)
	pe.EtcdIP = etcdIP
	pe.LocalIP = localIP
	pe.BuildTag = buildTag
	pe.Program = "test"
	pe.ProcDir = path.Join(sp.KPIDS, pe.GetPID().String())
	return pe
}

// Create a new sigma config which is a derivative of an existing sigma config.
func NewAddedProcEnv(pe *ProcEnv, idx int) *ProcEnv {
	pe2 := NewProcEnvUnset(pe.Privileged)
	*(pe2.ProcEnvProto) = *(pe.ProcEnvProto)
	pe2.SetUname(sp.Tuname(string(pe.GetUname()) + "-clnt-" + strconv.Itoa(idx)))
	return pe2
}

func NewDifferentRealmProcEnv(pe *ProcEnv, realm sp.Trealm) *ProcEnv {
	pe2 := NewProcEnvUnset(pe.Privileged)
	*(pe2.ProcEnvProto) = *(pe.ProcEnvProto)
	pe2.SetRealm(realm)
	pe2.SetUname(sp.Tuname(string(pe.GetUname()) + "-realm-" + realm.String()))
	return pe2
}

func (pe *ProcEnvProto) GetPID() sp.Tpid {
	return sp.Tpid(pe.PidStr)
}

func (pe *ProcEnvProto) SetPID(pid sp.Tpid) {
	pe.PidStr = string(pid)
}

func (pe *ProcEnvProto) GetRealm() sp.Trealm {
	return sp.Trealm(pe.RealmStr)
}

func (pe *ProcEnvProto) SetRealm(realm sp.Trealm) {
	pe.RealmStr = string(realm)
}

func (pe *ProcEnvProto) GetUname() sp.Tuname {
	return sp.Tuname(pe.UnameStr)
}

func (pe *ProcEnvProto) SetUname(uname sp.Tuname) {
	pe.UnameStr = string(uname)
}

func (pe *ProcEnvProto) SetUprocdPID(pid sp.Tpid) {
	pe.UprocdPIDStr = string(pid)
}

func (pe *ProcEnvProto) GetUprocdPID() sp.Tpid {
	return sp.Tpid(pe.UprocdPIDStr)
}

func (pe *ProcEnv) GetProto() *ProcEnvProto {
	return pe.ProcEnvProto
}

func (pe *ProcEnv) Marshal() string {
	b, err := json.Marshal(pe)
	if err != nil {
		log.Fatalf("FATAL Error marshal sigmaconfig: %v")
	}
	return string(b)
}

func Unmarshal(pestr string) *ProcEnv {
	pe := &ProcEnv{}
	err := json.Unmarshal([]byte(pestr), pe)
	if err != nil {
		log.Fatalf("FATAL Error unmarshal ProcEnv %v", err)
	}
	return pe
}

// TODO: cleanup
func (pe *ProcEnv) String() string {
	return fmt.Sprintf("&{ Program: %v Pid:%v Realm:%v Uname:%v KernelID:%v UprocdPID:%v Net:%v ProcDir:%v ParentDir:%v Perf:%v Debug:%v EtcdIP:%v LocalIP:%v BuildTag:%v Privileged:%v Crash:%v Partition:%v }", pe.Program, pe.GetPID(), pe.GetRealm(), pe.GetUname(), pe.KernelID, nil /*pe.UprocdPID*/, pe.Net, nil /*pe.Privileged*/, pe.ProcDir, pe.ParentDir, pe.Perf, pe.Debug, pe.EtcdIP, pe.LocalIP, pe.BuildTag, pe.Privileged, nil, nil /*pe.Crash, pe.Partition*/)
}
