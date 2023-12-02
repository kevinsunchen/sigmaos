package container

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path"
	"runtime"
	"syscall"
	"time"

	"github.com/opencontainers/runc/libcontainer/apparmor"
	selinux "github.com/opencontainers/selinux/go-selinux"
	"kernel.org/pub/linux/libs/security/libcap/cap"

	db "sigmaos/debug"
	"sigmaos/perf"
	"sigmaos/seccomp"
	sp "sigmaos/sigmap"
)

const (
	OLD_ROOT_MNT  = "old_root"
	APPARMOR_PROF = "docker-default"
)

func isolateUserProc(pid sp.Tpid, program string) (string, error) {
	// Setup and chroot to the process jail.
	s := time.Now()
	if err := jailProcess(pid); err != nil {
		db.DPrintf(db.CONTAINER, "Error jail process %v", err)
		return "", err
	}
	db.DPrintf(db.SPAWN_LAT, "[%v] Uproc create chroot jail %v", pid, time.Since(s))
	s = time.Now()
	pn, err := exec.LookPath(program)
	if err != nil {
		return "", fmt.Errorf("ContainUProc: LookPath: %v", err)
	}
	db.DPrintf(db.SPAWN_LAT, "[%v] Uproc look for exec path %v", pid, time.Since(s))
	// Lock the OS thread, since SE Linux labels are per-thread, and so this
	// thread should disallow the Go runtime from scheduling it on another kernel
	// thread before starting the user proc.
	s = time.Now()
	runtime.LockOSThread()
	db.DPrintf(db.SPAWN_LAT, "[%v] Uproc runtime lock OS thread %v", pid, time.Since(s))
	s = time.Now()
	// Apply SELinux label.
	if err := applySELinuxLabel(pn); err != nil {
		return "", err
	}
	db.DPrintf(db.SPAWN_LAT, "[%v] Uproc apply SELinux label %v", pid, time.Since(s))
	// Apply apparmor profile.
	s = time.Now()
	if err := applyAppArmorProfile(APPARMOR_PROF); err != nil {
		return "", err
	}
	db.DPrintf(db.SPAWN_LAT, "[%v] Uproc apply apparmor prof %v", pid, time.Since(s))
	// Decrease process capabilities.
	s = time.Now()
	if err := setCapabilities(pid); err != nil {
		db.DPrintf(db.CONTAINER, "Error set uproc capabilities: %v", err)
		return "", err
	}
	db.DPrintf(db.SPAWN_LAT, "[%v] Uproc setCapabilities %v", pid, time.Since(s))
	s = time.Now()
	// Seccomp the process.
	if err := seccompProcess(pid); err != nil {
		db.DPrintf(db.CONTAINER, "Error seccomp: %v", err)
		return "", err
	}
	db.DPrintf(db.SPAWN_LAT, "[%v] Uproc seccompProcess %v", pid, time.Since(s))
	return pn, nil
}

func finishIsolation() {
	runtime.UnlockOSThread()
}

func jailProcess(pid sp.Tpid) error {

	u, _ := user.Current()
	c := cap.GetProc()
	db.DPrintf(db.CONTAINER, "Process caps: %v %v\n", c, u)

	newRoot := jailPath(pid)
	// Create directories to use as mount points, as well as the new root directory itself.
	for _, d := range []string{"", OLD_ROOT_MNT, "lib", "usr", "lib64", "etc", "sys", "dev", "proc", "seccomp", "bin", "bin2", "tmp", perf.OUTPUT_PATH, "cgroup"} {
		if err := os.Mkdir(path.Join(newRoot, d), 0700); err != nil {
			db.DPrintf(db.ALWAYS, "failed to mkdir [%v]: %v", d, err)
			return err
		}
	}
	// Mount new file system as a mount point so we can pivot_root to it later
	if err := syscall.Mount(newRoot, newRoot, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		db.DPrintf(db.ALWAYS, "failed to mount new root filesystem: %v", err)
		return err
	}
	// Chdir to new root
	if err := syscall.Chdir(newRoot); err != nil {
		db.DPrintf(db.ALWAYS, "failed to chdir to /: %v", err)
		return err
	}
	// Mount /lib
	if err := syscall.Mount("/lib", "lib", "none", syscall.MS_BIND|syscall.MS_RDONLY, ""); err != nil {
		db.DPrintf(db.ALWAYS, "failed to mount /lib: %v", err)
		return err
	}
	// Mount /lib64
	if err := syscall.Mount("/lib64", "lib64", "none", syscall.MS_BIND|syscall.MS_RDONLY, ""); err != nil {
		db.DPrintf(db.ALWAYS, "failed to mount /lib64: %v", err)
		return err
	}
	// Mount /proc
	if err := syscall.Mount("proc", "proc", "proc", 0, ""); err != nil {
		db.DPrintf(db.ALWAYS, "failed to mount /proc: %v", err)
		return err
	}
	// Mount realm's user bin directory as /bin
	if err := syscall.Mount(path.Join(sp.SIGMAHOME, "bin/user"), "bin", "none", syscall.MS_BIND|syscall.MS_RDONLY, ""); err != nil {
		db.DPrintf(db.ALWAYS, "failed to mount userbin: %v", err)
		return err
	}
	// Mount realm's kernel bin directory as /bin2
	if err := syscall.Mount(path.Join(sp.SIGMAHOME, "bin/kernel"), "bin2", "none", syscall.MS_BIND|syscall.MS_RDONLY, ""); err != nil {
		db.DPrintf(db.ALWAYS, "failed to mount kernelbin: %v", err)
		return err
	}
	// Mount realm's seccomp directory as /seccomp
	if err := syscall.Mount(path.Join(sp.SIGMAHOME, "seccomp"), "seccomp", "none", syscall.MS_BIND|syscall.MS_RDONLY, ""); err != nil {
		db.DPrintf(db.ALWAYS, "failed to mount seccomp: %v", err)
		return err
	}
	// Mount cgroups.
	if err := syscall.Mount("/cgroup", "cgroup", "none", syscall.MS_BIND, ""); err != nil {
		db.DPrintf(db.ALWAYS, "failed to mount cgroup: %v", err)
		return err
	}
	// Mount perf dir (remove starting first slash)
	if err := syscall.Mount(perf.OUTPUT_PATH, perf.OUTPUT_PATH[1:], "none", syscall.MS_BIND, ""); err != nil {
		db.DPrintf(db.ALWAYS, "failed to mount perfoutput: %v", err)
		return err
	}
	// Mount /usr
	if err := syscall.Mount("/usr", "usr", "none", syscall.MS_BIND|syscall.MS_RDONLY, ""); err != nil {
		db.DPrintf(db.ALWAYS, "failed to mount /usr: %v", err)
		return err
	}
	// Mount /sys for /sys/devices/system/cpu/online; XXX exclude
	// /sys/firmware; others?
	if err := syscall.Mount("/sys", "sys", "sysfs", syscall.MS_BIND|syscall.MS_RDONLY, ""); err != nil {
		db.DPrintf(db.ALWAYS, "failed to mount /sys err %v", err)
		return err
	}
	// Mount /dev/urandom
	if err := syscall.Mount("/dev", "dev", "none", syscall.MS_BIND|syscall.MS_RDONLY, ""); err != nil {
		db.DPrintf(db.ALWAYS, "failed to mount /dev: %v", err)
		return err
	}
	// Mount /etc
	if err := syscall.Mount("/etc", "etc", "none", syscall.MS_BIND|syscall.MS_RDONLY, ""); err != nil {
		db.DPrintf(db.ALWAYS, "failed to mount /etc: %v", err)
		return err
	}
	// ========== No more mounts beyond this point ==========
	// pivot_root
	if err := syscall.PivotRoot(".", OLD_ROOT_MNT); err != nil {
		db.DPrintf(db.ALWAYS, "failed to pivot root: %v", err)
		return err
	}
	// Chdir to new root
	if err := syscall.Chdir("/"); err != nil {
		db.DPrintf(db.ALWAYS, "failed to chdir to /: %v", err)
		return err
	}
	// unmount the old root filesystem
	if err := syscall.Unmount(OLD_ROOT_MNT, syscall.MNT_DETACH); err != nil {
		db.DPrintf(db.ALWAYS, "failed to unmount the old root filesystem: %v", err)
		return err
	}
	// Remove the old root filesystem
	if err := os.Remove(OLD_ROOT_MNT); err != nil {
		db.DPrintf(db.ALWAYS, "failed to remove old root filesystem: %v", err)
		return err
	}
	db.DPrintf(db.CONTAINER, "Successfully pivoted to new root %v", newRoot)
	return nil
}

func applySELinuxLabel(pn string) error {
	if selinux.GetEnabled() {
		plabel, flabel := selinux.InitContainerLabels()
		if err := selinux.SetExecLabel(plabel); err != nil {
			db.DPrintf(db.CONTAINER, "Error set SELinux exec label: %v", err)
			return err
		}
		if err := selinux.SetFileLabel(pn, flabel); err != nil {
			db.DPrintf(db.CONTAINER, "Error set SELinux file label: %v", err)
			return err
		}
		db.DPrintf(db.CONTAINER, "Successfully applied SELinux labels plabel:%v flabel:%v", plabel, flabel)
	} else {
		db.DPrintf(db.CONTAINER, "SELinux disabled.")
	}
	return nil
}

func applyAppArmorProfile(prof string) error {
	if sp.Conf.AppArmor.ENABLED {
		// Apply the apparmor profile. Will take effect after the next exec.
		if err := apparmor.ApplyProfile(prof); err != nil {
			db.DPrintf(db.CONTAINER, "Error apply AppArmor profile %v: %v", prof, err)
			// If AppArmor is unloaded, ignore the error.
			if _, err := os.Stat("/sys/kernel/security/apparmor/profiles"); err != nil {
				db.DPrintf(db.CONTAINER, "AppArmor disabled.")
				return nil
			}
			return err
		}
		db.DPrintf(db.CONTAINER, "Successfully applied AppArmor profile %v", prof)
	} else {
		db.DPrintf(db.CONTAINER, "AppArmor disabled.")
	}
	return nil
}

// Capabilities. PID is only for debugging purposes.
func setCapabilities(pid sp.Tpid) error {
	// Taken from https://github.com/moby/moby/blob/master/oci/caps/defaults.go
	dockerDefaults := []cap.Value{
		cap.CHOWN,
		cap.DAC_OVERRIDE,
		cap.FSETID,
		cap.FOWNER,
		cap.NET_RAW,
		cap.SETGID,
		cap.SETUID,
		cap.SETFCAP,
		cap.SETPCAP,
		cap.NET_BIND_SERVICE,
		cap.SYS_CHROOT,
		cap.KILL,
		cap.AUDIT_WRITE,
	}

	// Effective, Permitted, Inheritable.
	caps := cap.NewSet()
	// Bounding.
	capsIAB := cap.NewIAB()
	// Set process Permitted flags.
	if err := caps.SetFlag(cap.Permitted, true, dockerDefaults...); err != nil {
		return err
	}
	// Set process Inheritable flags.
	if err := caps.SetFlag(cap.Inheritable, true, dockerDefaults...); err != nil {
		return err
	}
	// Set process Bounding flags.
	if err := capsIAB.Fill(cap.Bound, caps, cap.Permitted); err != nil {
		return err
	}
	s := time.Now()
	if err := caps.SetProc(); err != nil {
		return err
	}
	db.DPrintf(db.SPAWN_LAT, "[%v] Uproc setCapabilities caps.SetProc %v", pid, time.Since(s))
	s = time.Now()
	if err := capsIAB.SetProc(); err != nil {
		return err
	}
	db.DPrintf(db.SPAWN_LAT, "[%v] Uproc setCapabilities capsIAB.SetProc %v", pid, time.Since(s))
	db.DPrintf(db.CONTAINER, "Successfully set capabilities to:\n%v.\nResulting caps:\n%v", dockerDefaults, cap.GetProc())
	return nil
}

// Seccomp. PID is for debugging purposes.
func seccompProcess(pid sp.Tpid) error {
	// Load the sigmaOS seccomp white list.
	s := time.Now()
	sigmaSCWL, err := seccomp.ReadWhiteList("/seccomp/whitelist.yml")
	if err != nil {
		return err
	}
	db.DPrintf(db.SPAWN_LAT, "[%v] Uproc seccompProcess ReadWhiteList %v", pid, time.Since(s))
	s = time.Now()
	if err := seccomp.LoadFilter(sigmaSCWL); err != nil {
		return err
	}
	db.DPrintf(db.SPAWN_LAT, "[%v] Uproc seccompProcess LoadFilter %v", pid, time.Since(s))
	db.DPrintf(db.CONTAINER, "Successfully seccomped process %v %v", os.Args, sigmaSCWL)
	return nil
}
