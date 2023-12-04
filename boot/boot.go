package boot

import (
	"os"

	db "sigmaos/debug"
	"sigmaos/kernel"
	"sigmaos/kernelsrv"
	"sigmaos/proc"
)

type Boot struct {
	k *kernel.Kernel
}

// The boot processes enters here
func BootUp(param *kernel.Param, pcfg *proc.ProcEnv) error {
	db.DPrintf(db.KERNEL, "Boot param %v ProcEnv %v env %v", param, pcfg, os.Environ())
	k, err := kernel.NewKernel(param, pcfg)
	if err != nil {
		return err
	}

	db.DPrintf(db.BOOT, "container %s booted %v\n", os.Args[1], k.Ip())

	if err := kernelsrv.RunKernelSrv(k); err != nil {
		return err
	}

	return nil
}
