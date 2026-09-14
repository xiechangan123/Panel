//go:build wireinject

package job

import "github.com/libtnb/wire"

// Module 装配定时任务
var Module = wire.New().
	Struct[Dependencies]().
	Provide(NewJobs).
	Export[[]Job]()
