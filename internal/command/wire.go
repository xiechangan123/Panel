//go:build wireinject

package command

import (
	"github.com/libtnb/wire"
	"github.com/urfave/cli/v3"
)

// Module 装配 CLI 命令
var Module = wire.New().
	Provide(Commands).
	Export[[]*cli.Command]()
