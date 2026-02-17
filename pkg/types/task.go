package types

import "context"

// TaskRunner 任务运行器接口
type TaskRunner interface {
	Run(ctx context.Context)
	Notify()
}
