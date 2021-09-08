package realm

import (
	"encoding/json"
	"fmt"
	"log"
	"path"

	"ulambda/atomic"
	"ulambda/config"
	"ulambda/fsclnt"
	"ulambda/fslib"
	"ulambda/kernel"
	"ulambda/named"
	"ulambda/sync"
)

const (
	DEFAULT_REALMD_PRIORITY = "0"
	REALM_LOCK              = "realm-lock."
)

type RealmdConfig struct {
	Id      string
	RealmId string
}

type Realmd struct {
	id          string
	bin         string
	cfgPath     string
	cfg         *RealmdConfig
	s           *kernel.System
	freeRealmds *sync.FilePriorityBag
	realmLock   *sync.Lock
	*config.ConfigClnt
	*fslib.FsLib
}

func MakeRealmd(bin string) *Realmd {
	// XXX Get id somehow
	id, err := fsclnt.LocalIP()
	if err != nil {
		log.Fatalf("Error LocalIP in MakeRealmd: %v", err)
	}
	r := &Realmd{}
	r.id = id
	r.bin = bin
	r.cfgPath = path.Join(REALMD_CONFIG, id)
	r.FsLib = fslib.MakeFsLib(fmt.Sprintf("realmd-%v", id))
	r.ConfigClnt = config.MakeConfigClnt(r.FsLib)

	// Set up the realmd config
	r.cfg = &RealmdConfig{}
	r.cfg.Id = id
	r.cfg.RealmId = NO_REALM

	// Write the initial config file
	r.WriteConfig(r.cfgPath, r.cfg)

	r.freeRealmds = sync.MakeFilePriorityBag(r.FsLib, FREE_REALMDS)

	// Mark self as available for allocation
	r.markFree()

	return r
}

// Mark self as available for allocation to a realm.
func (r *Realmd) markFree() {
	cfg := &RealmdConfig{}
	cfg.Id = r.id
	cfg.RealmId = NO_REALM

	b, err := json.Marshal(cfg)
	if err != nil {
		log.Fatalf("Error Marshal in MakeRealm: %v", err)
	}

	if err := r.freeRealmds.Put(DEFAULT_REALMD_PRIORITY, r.id, b); err != nil {
		log.Fatalf("Error Put in MakeRealmd: %v", err)
	}
}

// Update configuration.
func (r *Realmd) getNextConfig() {
	// XXX Does it matter that we spin?
	for {
		r.ReadConfig(r.cfgPath, r.cfg)
		// Make sure we've been assigned to a realm
		if r.cfg.RealmId != NO_REALM {
			break
		}
	}
	// Update the realm lock
	r.realmLock = sync.MakeLock(r.FsLib, named.LOCKS, REALM_LOCK+r.cfg.RealmId, true)
}

// If this is the first realmd assigned to a realm, initialize the realm by
// starting a named for it.
func (r *Realmd) tryInitRealmL() bool {
	rds, err := r.ReadDir(path.Join(REALMS, r.cfg.RealmId))
	if err != nil {
		log.Fatalf("Error ReadDir in Realmd.tryInitRealmL: %v", err)
	}

	// If this is the first realmd, start the realm's named.
	if len(rds) == 0 {
		ip, err := fsclnt.LocalIP()
		if err != nil {
			log.Fatalf("Error LocalIP in Realmd.tryInitRealmL: %v", err)
		}
		namedAddr := genNamedAddr(ip)

		// Start a named instance.
		if _, err := BootNamed(r.bin, namedAddr); err != nil {
			log.Fatalf("Error BootNamed in Realmd.tryInitRealmL: %v", err)
		}

		realmCfg := GetRealmConfig(r.FsLib, r.cfg.RealmId)
		realmCfg.NamedAddr = namedAddr
		r.WriteConfig(path.Join(REALM_CONFIG, realmCfg.Rid), realmCfg)

		return true
	}
	return false
}

// Register this realmd as part of a realm.
func (r *Realmd) register() {
	// Register this realmd as belonging to this realm.
	if err := atomic.MakeFileAtomic(r.FsLib, path.Join(REALMS, r.cfg.RealmId, r.id), 0777, []byte{}); err != nil {
		log.Fatalf("Error MakeFileAtomic in Realmd.register: %v", err)
	}
}

func (r *Realmd) boot(realmCfg *RealmConfig) {
	r.s = kernel.MakeSystemNamedAddr(r.bin, realmCfg.NamedAddr)
	if err := r.s.Boot(); err != nil {
		log.Fatalf("Error Boot in Realmd.boot: %v", err)
	}
}

// Join a realm
func (r *Realmd) joinRealm() chan bool {
	r.realmLock.Lock()
	defer r.realmLock.Unlock()

	// Try to initalize this realm if it hasn't been initialized already.
	first := r.tryInitRealmL()
	// Get the realm config
	realmCfg := GetRealmConfig(r.FsLib, r.cfg.RealmId)
	// Register this realmd
	r.register()
	// Boot this realmd's system services
	r.boot(realmCfg)
	// Signal that the realm has been initialized
	if first {
		rStartCond := sync.MakeCond(r.FsLib, path.Join(named.BOOT, r.cfg.RealmId), nil)
		rStartCond.Destroy()
	}
	// Watch for changes to the config
	return r.WatchConfig(r.cfgPath)
}

func (r *Realmd) teardown() {
	// TODO: evict procs gracefully
	// Tear down realm resources
	r.s.Shutdown()
}

func (r *Realmd) deregister() {
	// Register this realmd as belonging to this realm
	if err := r.Remove(path.Join(REALMS, r.cfg.RealmId, r.id)); err != nil {
		log.Fatalf("Error Remove in Realmd.deregister: %v", err)
	}
}

func (r *Realmd) tryDestroyRealmL() {
	rds, err := r.ReadDir(path.Join(REALMS, r.cfg.RealmId))
	if err != nil {
		log.Fatalf("Error ReadDir in Realmd.tryDestroyRealmL: %v", err)
	}

	// If this is the last realmd, destroy the realmd's named
	if len(rds) == 0 {
		realmCfg := GetRealmConfig(r.FsLib, r.cfg.RealmId)
		ShutdownNamed(realmCfg.NamedAddr)

		// Remove the realm config file
		if err := r.Remove(path.Join(REALM_CONFIG, r.cfg.RealmId)); err != nil {
			log.Fatalf("Error Remove in REALM_CONFIG Realmd.tryDestroyRealmL: %v", err)
		}

		// Remove the realm directory
		if err := r.Remove(path.Join(REALMS, r.cfg.RealmId)); err != nil {
			log.Fatalf("Error Remove REALMS in Realmd.tryDestroyRealmL: %v", err)
		}

		// Signal that the realm has been initialized
		rExitCond := sync.MakeCond(r.FsLib, path.Join(named.BOOT, r.cfg.RealmId), nil)
		rExitCond.Destroy()
	}
}

// Leave a realm
func (r *Realmd) leaveRealm() {
	r.realmLock.Lock()
	defer r.realmLock.Unlock()

	// Tear down resources
	r.teardown()
	// Deregister this realmd
	r.deregister()
	// Try to destroy a realm (if this is the last realmd remaining)
	r.tryDestroyRealmL()
}

func (r *Realmd) Work() {
	for {
		// Get the next realm assignment.
		r.getNextConfig()

		// Join a realm
		done := r.joinRealm()
		// Wait for the watch to trigger
		<-done

		// Leave a realm
		r.leaveRealm()

		// Mark self as available for allocation.
		r.markFree()
	}
}
