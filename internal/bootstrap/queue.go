package bootstrap

import (
	"github.com/acepanel/panel/pkg/queue"
)

func NewQueue() *queue.Queue {
	return queue.New(100)
}
