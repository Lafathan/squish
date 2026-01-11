package main

import (
	"os"
	"runtime"
	"runtime/pprof"
	"squish/internal/cli"
)

func profile() (stop func(), err error) {
	cpuPath := os.Getenv("SQUISH_CPUPROFILE")
	memPath := os.Getenv("SQUISH_MEMPROFILE")
	var cpuFile *os.File
	if cpuPath != "" {
		f, err := os.Create(cpuPath)
		if err != nil {
			return nil, err
		}
		cpuFile = f
		if err := pprof.StartCPUProfile(cpuFile); err != nil {
			_ = cpuFile.Close()
			return nil, err
		}
	}
	stop = func() {
		if cpuFile != nil {
			pprof.StopCPUProfile()
			_ = cpuFile.Close()
		}
		if memPath != "" {
			f, err := os.Create(memPath)
			if err == nil {
				runtime.GC()
				_ = pprof.WriteHeapProfile(f)
				_ = f.Close()
			}
		}
	}
	return stop, nil
}

func main() {
	args := os.Args
	stop, err := profile()
	if err != nil {
	}
	if stop != nil {
		defer stop()
	}
	cli.Run(args[1:])
}
