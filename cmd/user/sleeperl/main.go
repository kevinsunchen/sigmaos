package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"ulambda/perf"
	"ulambda/test_lambdas"
)

func main() {
	start := time.Now()
	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, "Usage: %v pid sleep_length out <native> <pprof_path>\n", os.Args[0])
		os.Exit(1)
	}
	var p *perf.Perf
	if len(os.Args) > 4 {
		prof := false
		pprofPath := ""
		if os.Args[4] != "native" || len(os.Args) > 5 {
			prof = true
			if os.Args[4] == "native" {
				pprofPath = os.Args[5]
			} else {
				pprofPath = os.Args[4]
			}
		}
		if prof {
			// If we're benchmarking, make a flame graph
			p = perf.MakePerf()
			p.SetupPprof(pprofPath)
			defer p.Teardown()
		}
	}
	l, err := test_lambdas.MakeSleeperl(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v: error %v", os.Args[0], err)
		os.Exit(1)
	}
	l.Work()
	l.Exit()
	end := time.Now()
	log.Printf("E2E time: %v usec", end.Sub(start).Microseconds())
}
