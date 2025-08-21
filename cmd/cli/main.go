package main

import (
	"os"
	_ "time/tzdata"
)

func main() {
	if os.Geteuid() != 0 {
		panic("panel must run as root")
	}

	cli, err := initCli()
	if err != nil {
		panic(err)
	}

	if err = cli.Run(); err != nil {
		panic(err)
	}
}
