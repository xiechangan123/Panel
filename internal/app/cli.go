package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"

	"github.com/acepanel/panel/v3/pkg/tui"
)

// CliBuilder 构建一棵全新的命令树，交互模式每次执行都要新树，避免复用已解析过的标志状态
type CliBuilder func() *cli.Command

type Cli struct {
	build    CliBuilder
	t        *gotext.Locale
	migrator *gormigrate.Gormigrate
}

func NewCli(build CliBuilder, t *gotext.Locale, migrator *gormigrate.Gormigrate) *Cli {
	IsCli = true
	return &Cli{
		build:    build,
		t:        t,
		migrator: migrator,
	}
}

func (r *Cli) Run() error {
	// migrate database
	// 这里不处理错误，这么做是为了在异常时用户可以用 fix 命令尝试修复
	_ = r.migrator.Migrate()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	context.AfterFunc(ctx, stop)

	// 裸执行且处于终端时进入交互模式，其余情况按普通命令行处理
	if len(os.Args) == 1 && term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())) {
		return tui.Run(ctx, r.t, r.build(), func(ctx context.Context, args []string) error {
			return r.build().Run(ctx, args)
		})
	}

	// 错误必须向上返回，调用方据此以非零码退出，后台任务才能正确判定失败
	return r.build().Run(ctx, os.Args)
}
