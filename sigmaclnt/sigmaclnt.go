package sigmaclnt

import (
	"path"

	db "sigmaos/debug"
	"sigmaos/fslib"
	"sigmaos/proc"
	"sigmaos/procclnt"
	sp "sigmaos/sigmap"
)

type SigmaClnt struct {
	*fslib.FsLib
	*procclnt.ProcClnt
}

// Create only an FsLib, as a proc.
func MkSigmaClntFsLib(name string) (*SigmaClnt, error) {
	fsl, err := fslib.MakeFsLib(name)
	if err != nil {
		db.DFatalf("MkSigmaClnt: %v", err)
	}
	return &SigmaClnt{fsl, nil}, nil
}

// Only to be called by procs (uses SIGMAREALM env variable, and expects realm
// namespace to be set up for this proc, e.g. procdir).
func MkSigmaClnt(name string) (*SigmaClnt, error) {
	sc, err := MkSigmaClntFsLib(name)
	if err != nil {
		db.DFatalf("MkSigmaClnt: %v", err)
	}
	sc.ProcClnt = procclnt.MakeProcClnt(sc.FsLib)
	return sc, nil
}

// Create only an FsLib, relative to a realm, but with the client being in the root realm
func MkSigmaClntRealmFsLib(rootrealm *fslib.FsLib, name string, rid sp.Trealm) (*SigmaClnt, error) {
	if rid == sp.ROOTREALM {
		return &SigmaClnt{
			rootrealm,
			nil,
		}, nil
	}
	pn := path.Join(sp.REALMS, rid.String())
	target, err := rootrealm.GetFile(pn)
	if err != nil {
		return nil, err
	}
	mnt, r := sp.MkMount(target)
	if r != nil {
		return nil, err
	}
	db.DPrintf(db.SIGMACLNT, "Realm %v NamedAddr %v\n", rid, mnt.Addr)
	realm, err := fslib.MakeFsLibAddrNet(name, rid, rootrealm.GetLocalIP(), mnt.Addr, sp.ROOTREALM.String())
	if err != nil {
		db.DPrintf(db.SIGMACLNT, "Error mkFsLibAddr [%v]: %v", mnt.Addr, err)
		return nil, err
	}
	return &SigmaClnt{realm, nil}, nil
}

// Create a full sigmaclnt relative to a realm (fslib and procclnt)
func MkSigmaClntRealm(rootfsl *fslib.FsLib, name string, rid sp.Trealm) (*SigmaClnt, error) {
	db.DPrintf(db.SIGMACLNT, "MkSigmaClntRealmProc %v\n", rid)
	sc, err := MkSigmaClntRealmFsLib(rootfsl, name, rid)
	if err != nil {
		return nil, err
	}
	sc.ProcClnt = procclnt.MakeProcClntInit(proc.GetPid(), sc.FsLib, name)
	return sc, nil
}

// Only to be used by non-procs (tests, and linux processes), and creates a
// sigmaclnt for the root realm.
func MkSigmaClntRootInit(name string, ip string, namedAddr sp.Taddrs) (*SigmaClnt, error) {
	fsl, err := fslib.MakeFsLibAddrNet(name, sp.ROOTREALM, ip, namedAddr, sp.ROOTREALM.String())
	if err != nil {
		return nil, err
	}
	pclnt := procclnt.MakeProcClntInit(proc.GetPid(), fsl, name)
	return &SigmaClnt{fsl, pclnt}, nil
}
