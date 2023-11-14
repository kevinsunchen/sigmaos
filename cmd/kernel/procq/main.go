package main

import (
	"os"

	db "sigmaos/debug"
	"sigmaos/procqsrv"
	sp "sigmaos/sigmap"
)

func main() {
	if len(os.Args) != 2 {
		db.DFatalf("Usage: provider %v", os.Args[0])
	}
	// TODO_PRVDR use actual OS args
	provider := sp.ParseTprovider(os.Args[1])
	procqsrv.Run(provider)
}
