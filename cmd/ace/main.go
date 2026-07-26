package main

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	_ "time/tzdata"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run() error {
	if os.Geteuid() != 0 {
		return errors.New("panel must run as root")
	}

	debug.SetGCPercent(10)

	ace, cleanup, err := initAce()
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	return ace.Run()
}
