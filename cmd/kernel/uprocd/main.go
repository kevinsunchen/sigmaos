package main

import (
	"os"

	db "sigmaos/debug"
	"sigmaos/uprocsrv"
)

func main() {
	if len(os.Args) != 3 {
		db.DFatalf("Usage: %v kernelId port", os.Args[0])
	}
	// ignore scheddIp
	if err := uprocsrv.RunUprocSrv(os.Args[1], os.Args[2]); err != nil {
		db.DFatalf("Fatal start: %v %v\n", os.Args[0], err)
	}
}
