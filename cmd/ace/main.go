package main

import (
	"os"
	"runtime/debug"
	_ "time/tzdata"
)

func main() {
	if os.Geteuid() != 0 {
		panic("panel must run as root")
	}

	debug.SetGCPercent(10)
	debug.SetMemoryLimit(256 << 20)

	ace, err := initAce()
	if err != nil {
		panic(err)
	}

	if err = ace.Run(); err != nil {
		panic(err)
	}
}
