package main

import (
	"errors"
	"os"
	_ "time/tzdata"

	"github.com/gookit/color"
)

func main() {
	if err := run(); err != nil {
		color.Errorf("|-%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if os.Geteuid() != 0 {
		return errors.New("panel must run as root")
	}

	cli, cleanup, err := initCli()
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	return cli.Run()
}
