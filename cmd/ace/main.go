package main

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	_ "time/tzdata"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/injector"
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

	inj := injector.New()
	defer func() { _ = inj.Shutdown() }()

	ace, err := do.Invoke[*app.Ace](inj)
	if err != nil {
		return err
	}

	return ace.Run()
}
