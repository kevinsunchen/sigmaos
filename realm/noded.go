package realm

import (
	"path"
	"time"

	"sigmaos/config"
	"sigmaos/container"
	db "sigmaos/debug"
	"sigmaos/electclnt"
	"sigmaos/fslib"
	"sigmaos/kernel"
	"sigmaos/machine"
	"sigmaos/memfssrv"
	"sigmaos/proc"
	"sigmaos/procclnt"
	"sigmaos/protdevclnt"
	"sigmaos/protdevsrv"
	"sigmaos/realm/proto"
	"sigmaos/semclnt"
	sp "sigmaos/sigmap"
)

type Noded struct {
	*fslib.FsLib
	*procclnt.ProcClnt
	id        string
	machineId string
	localIP   string
	cfgPath   string
	done      chan bool
	cfg       *NodedConfig
	s         *kernel.Kernel
	ec        *electclnt.ElectClnt
	pds       *protdevsrv.ProtDevSrv
	sclnt     *protdevclnt.ProtDevClnt
	*config.ConfigClnt
}

func MakeNoded(machineId string) *Noded {
	nd := &Noded{}
	nd.id = proc.GetPid().String()
	nd.machineId = machineId
	nd.cfgPath = NodedConfPath(nd.id)
	nd.done = make(chan bool)
	fsl, err := fslib.MakeFsLib(nd.id)
	if err != nil {
		db.DFatalf("Error MakeFsLib: %v", err)
	}
	nd.FsLib = fsl
	nd.ProcClnt = procclnt.MakeProcClnt(nd.FsLib)
	nd.ConfigClnt = config.MakeConfigClnt(nd.FsLib)
	mfs, err := memfssrv.MakeMemFsFsl(path.Join(machine.MACHINES, machineId, machine.NODEDS)+"/", nd.FsLib, nd.ProcClnt)
	if err != nil {
		db.DFatalf("Error MakeMemFsFsl: %v", err)
	}

	nd.pds, err = protdevsrv.MakeProtDevSrvMemFs(mfs, nd)
	if err != nil {
		db.DFatalf("Error MakeMemFs: %v", err)
	}
	nd.sclnt, err = protdevclnt.MkProtDevClnt(nd.pds.FsLib(), sp.SIGMAMGR)
	if err != nil {
		db.DFatalf("Error MkProtDevClnt: %v", err)
	}

	// Mount the KPIDS dir.
	if err := procclnt.MountPids(nd.FsLib, proc.Named()); err != nil {
		db.DFatalf("Error mountpids: %v", err)
	}

	ip, err := container.LocalIP()
	if err != nil {
		db.DFatalf("Error LocalIP: %v", err)
	}
	nd.localIP = ip

	// Set the noded id so that child kernel procs inherit it.
	proc.SetNodedId(nd.id)

	// Set up the noded config
	nd.cfg = MakeNodedConfig()
	db.DPrintf(db.NODED, "Boot on machine %v", machineId)
	nd.cfg.MachineId = machineId

	return nd
}

func (nd *Noded) GrantCores(req proto.NodedRequest, res *proto.NodedResponse) error {
	db.DPrintf(db.NODED, "Noded %v granted cores %v", nd.id, req.Cores)
	//	msg := resource.MakeResourceMsg(resource.Tgrant, resource.Tcore, req.Cores.Marshal(), int(req.Cores.Size()))
	//	nd.forwardResourceMsgToProcd(msg)
	//
	//	nd.cfg.Cores = append(nd.cfg.Cores, req.Cores)
	//	nd.WriteConfig(nd.cfgPath, nd.cfg)
	//
	//	lockRealm(nd.ec, nd.cfg.RealmId)
	//	defer unlockRealm(nd.ec, nd.cfg.RealmId)
	//
	//	realmCfg := GetRealmConfig(nd.FsLib, nd.cfg.RealmId)
	//	realmCfg.NCores += proc.Tcore(req.Cores.Size())
	//	nd.WriteConfig(RealmConfPath(nd.cfg.RealmId), realmCfg)
	//	res.OK = true
	return nil
}

func (nd *Noded) RevokeCores(req proto.NodedRequest, res *proto.NodedResponse) error {
	db.DPrintf(db.NODED, "Noded %v lost cores %v", nd.id, req.Cores)

	// If all cores were requested, shut down.
	if req.AllCores || len(nd.cfg.Cores) == 1 {
		db.DPrintf(db.NODED, "Noded %v evicted from Realm %v", nd.id, nd.cfg.RealmId)
		// Leave the realm and prepare to shut down.
		nd.leaveRealm()
		nd.done <- true
		close(nd.done)
	} else {
		//		msg := resource.MakeResourceMsg(resource.Trequest, resource.Tcore, req.Cores.Marshal(), int(req.Cores.Size()))
		//		nd.forwardResourceMsgToProcd(msg)
		//
		//		cores := nd.cfg.Cores[len(nd.cfg.Cores)-1]
		//
		//		// Sanity check: should be at least 2 core groups when removing one.
		//		// Otherwise, we should have shut down.
		//		if len(nd.cfg.Cores) < 2 {
		//			db.DFatalf("Requesting cores form a noded with <2 core groups: %v", nd.cfg)
		//		}
		//		// Sanity check: we always take the last cores allocated.
		//		if cores.Start != req.Cores.Start || cores.End != req.Cores.End {
		//			db.DFatalf("Removed unexpected core group: %v from %v", req.Cores.Marshal(), nd.cfg)
		//		}
		//
		//		// Update the core allocations for this noded.
		//		var rmCores *sessp.Tinterval
		//		nd.cfg.Cores, rmCores = nd.cfg.Cores[:len(nd.cfg.Cores)-1], nd.cfg.Cores[len(nd.cfg.Cores)-1]
		//		nd.WriteConfig(nd.cfgPath, nd.cfg)
		//
		//		// Update the realm's total core count. The Realmmgr holds the realm
		//		// lock.
		//		realmCfg := GetRealmConfig(nd.FsLib, nd.cfg.RealmId)
		//		realmCfg.NCores -= proc.Tcore(rmCores.Size())
		//		nd.WriteConfig(RealmConfPath(nd.cfg.RealmId), realmCfg)
		//
		//		machine.PostCores(nd.sclnt, nd.machineId, cores)
	}
	res.OK = true
	return nil
}

//func (nd *Noded) forwardResourceMsgToProcd(msg *resource.ResourceMsg) {
//	procdIp := nd.s.GetProcdIp()
//	// Pass the resource message on to this noded's procd.
//	resource.SendMsg(nd.FsLib, path.Join(RealmPath(nd.cfg.RealmId), sp.PROCDREL, procdIp, sp.RESOURCE_CTL), msg)
//}

// Update configuration.
func (nd *Noded) getNextConfig() {
	for {
		nd.ReadConfig(nd.cfgPath, nd.cfg)
		// Make sure we've been assigned to a realm
		if nd.cfg.RealmId != kernel.NO_REALM {
			nd.ec = electclnt.MakeElectClnt(nd.FsLib, realmFencePath(nd.cfg.RealmId), 0777)
			break
		}
	}
}

func (nd *Noded) countNCores() proc.Tcore {
	ncores := proc.Tcore(0)
	for _, c := range nd.cfg.Cores {
		ncores += proc.Tcore(c.Size())
	}
	return ncores
}

// If we need more named replicas, help initialize a realm by starting another
// named replica for it. Return true when all named replicas have been
// initialized.
func (nd *Noded) tryAddNamedReplicaL() bool {
	// Get config
	realmCfg := GetRealmConfig(nd.FsLib, nd.cfg.RealmId)

	initDone := false
	// If this is the last required noded replica...
	if len(realmCfg.NodedsActive) == nReplicas()-1 {
		initDone = true
	}

	// If we need to add a named replica, do so
	if len(realmCfg.NodedsActive) < nReplicas() {
		namedAddrs := genNamedAddrs(1, nd.localIP)

		realmCfg.NamedAddrs = append(realmCfg.NamedAddrs, namedAddrs...)

		// Start a named instance.
		_, pid, err := kernel.BootNamed(nd.ProcClnt, namedAddrs[0], nReplicas() > 1, len(realmCfg.NamedAddrs), realmCfg.NamedAddrs, nd.cfg.RealmId)
		if err != nil {
			db.DFatalf("Error BootNamed in Noded.tryInitRealmL: %v", err)
		}
		// Update config
		realmCfg.NamedPids = append(realmCfg.NamedPids, pid.String())
		nd.WriteConfig(RealmConfPath(realmCfg.Rid), realmCfg)
		db.DPrintf(db.NODED, "Added named replica: %v", realmCfg)
	}
	return initDone
}

// Register this noded as part of a realm.
func (nd *Noded) register(cfg *RealmConfig) {
	cfg.NodedsActive = append(cfg.NodedsActive, nd.id)
	cfg.NCores += nd.countNCores()
	nd.WriteConfig(RealmConfPath(cfg.Rid), cfg)
	// Symlink into realmmgr's fs.
	mnt := sp.MkMountServer(nd.pds.MyAddr())
	if err := nd.MkMountSymlink(nodedPath(cfg.Rid, nd.id), mnt); err != nil {
		db.DFatalf("Error symlink: %v", err)
	}
}

func (nd *Noded) boot(realmCfg *RealmConfig) {
	sys, err := kernel.MakeSystem("realm", realmCfg.Rid, realmCfg.NamedAddrs, nd.cfg.Cores[0])
	if err != nil {
		db.DFatalf("Error MakeSystem in Noded.boot: %v", err)
	}
	nd.s = sys
	if err := nd.s.BootSubs(); err != nil {
		db.DFatalf("Error Boot in Noded.boot: %v", err)
	}
	// Update the config with the procd IP.
	nd.cfg.ProcdIp = nd.s.GetProcdIp()
	nd.WriteConfig(nd.cfgPath, nd.cfg)
}

// Join a realm
func (nd *Noded) joinRealm() {
	lockRealm(nd.ec, nd.cfg.RealmId)
	defer unlockRealm(nd.ec, nd.cfg.RealmId)

	// Try to initalize this realm if it hasn't been initialized already.
	initDone := nd.tryAddNamedReplicaL()
	// Get the realm config
	realmCfg := GetRealmConfig(nd.FsLib, nd.cfg.RealmId)
	// Register this noded
	nd.register(realmCfg)
	// Boot this noded's system services
	nd.boot(realmCfg)
	// Signal that the realm has been initialized
	if initDone {
		rStartSem := semclnt.MakeSemClnt(nd.FsLib, path.Join(sp.BOOT, nd.cfg.RealmId))
		rStartSem.Up()
	}
	db.DPrintf(db.NODED, "Noded %v joined Realm %v", nd.id, nd.cfg.RealmId)
}

func (nd *Noded) teardown() {
	// Tear down realm resources
	//	nd.s.Shutdown()
}

func (nd *Noded) deregister(cfg *RealmConfig) {
	for i := range cfg.NodedsActive {
		if cfg.NodedsActive[i] == nd.id {
			cfg.NodedsActive = append(cfg.NodedsActive[:i], cfg.NodedsActive[i+1:]...)
			break
		}
	}

	for i := range cfg.NodedsAssigned {
		if cfg.NodedsAssigned[i] == nd.id {
			cfg.NodedsAssigned = append(cfg.NodedsAssigned[:i], cfg.NodedsAssigned[i+1:]...)
			break
		}
	}

	cfg.NCores -= nd.countNCores()
	cfg.LastResize = time.Now()

	nd.WriteConfig(RealmConfPath(cfg.Rid), cfg)

	// Remove the symlink to this noded from the realmmgr dir.
	nd.Remove(nodedPath(cfg.Rid, nd.id))

	for _, c := range nd.cfg.Cores {
		machine.PostCores(nd.sclnt, nd.machineId, c)
	}
}

func (nd *Noded) tryDestroyRealmL(realmCfg *RealmConfig) {
	// If this is the last noded, destroy the noded's named
	if len(realmCfg.NodedsActive) == 0 {
		db.DPrintf(db.NODED, "Destroy realm %v", realmCfg.Rid)

		ShutdownNamedReplicas(nd.ProcClnt, realmCfg.NamedPids)

		// Remove the realm config file
		if err := nd.Remove(RealmConfPath(realmCfg.Rid)); err != nil {
			db.DFatalf("Error Remove in REALM_CONFIG Noded.tryDestroyRealmL: %v", err)
		}

		// Remove the realm's named directory
		if err := nd.Remove(RealmPath(realmCfg.Rid)); err != nil {
			db.DPrintf(db.NODED_ERR, "Error Remove REALM_NAMEDS in Noded.tryDestroyRealmL: %v", err)
		}

		// Signal that the realm has been destroyed
		rExitSem := semclnt.MakeSemClnt(nd.FsLib, path.Join(sp.BOOT, realmCfg.Rid))
		rExitSem.Up()
	}
}

// Leave a realm. Expects realmmgr to hold the realm lock.
func (nd *Noded) leaveRealm() {
	db.DPrintf(db.NODED, "Noded %v leaving Realm %v", nd.id, nd.cfg.RealmId)

	// Tear down resources
	nd.teardown()

	db.DPrintf(db.NODED, "Noded %v done with teardown", nd.id)

	// Get the realm config
	realmCfg := GetRealmConfig(nd.FsLib, nd.cfg.RealmId)
	// Deregister this noded
	nd.deregister(realmCfg)
	// Try to destroy a realm (if this is the last noded remaining)
	nd.tryDestroyRealmL(realmCfg)
}

func (nd *Noded) Work() {
	// Get the next realm assignment.
	db.DPrintf(db.NODED, "Noded %v started, waiting for config", nd.id)
	nd.getNextConfig()
	db.DPrintf(db.NODED, "Noded %v got config %v", nd.id, nd.cfg)

	// Join a realm
	nd.joinRealm()

	if err := nd.Started(); err != nil {
		db.DFatalf("Error Started: %v", err)
	}
	db.DPrintf(db.NODED, "Noded %v started", nd.id)

	<-nd.done

	nd.Exited(proc.MakeStatus(proc.StatusOK))
}
