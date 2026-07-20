package bootstrap

import (
	"errors"
	"log/slog"
	"time"

	"github.com/libtnb/cron"
	"github.com/libtnb/cron/wrap"
	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/job"
	"github.com/acepanel/panel/v3/internal/registry"
)

func NewCron(i do.Injector) (*cron.Cron, error) {
	// 面板任务均为 5 段表达式，不启用 WithSecondsField
	c := cron.New(
		cron.WithLogger(do.MustInvoke[*slog.Logger](i)),
		cron.WithChain(wrap.Recover(), wrap.SkipIfRunning()),
	)

	// 收集全部任务贡献并注册到调度器
	jobs, err := registry.Collect[job.Job](i, job.Prefix)
	if err != nil {
		return nil, err
	}
	for _, j := range jobs {
		id, err := c.Add(j.Spec, j.Task)
		if err != nil {
			return nil, err
		}
		if j.Immediate {
			// 调度器随面板启动(数据库迁移完成)后立即触发一次,未启动前重试等待
			go func(id cron.EntryID) {
				for {
					if err := c.Trigger(id); !errors.Is(err, cron.ErrSchedulerNotRunning) {
						return
					}
					time.Sleep(time.Second)
				}
			}(id)
		}
	}

	return c, nil
}
