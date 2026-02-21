package data

import (
	"context"
	"fmt"
	"strings"

	"github.com/moby/moby/client"
	"github.com/spf13/cast"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/acepanel/panel/internal/biz"
)

// getContainerSock 从设置中读取容器 socket 路径
// 如果未配置或读取失败，返回默认值 unix:///var/run/docker.sock
func getContainerSock(settingRepo biz.SettingRepo) string {
	sock, _ := settingRepo.Get(biz.SettingKeyContainerSock)
	if sock == "" {
		sock = "/var/run/docker.sock"
	}
	// 自动补全 scheme
	if !strings.Contains(sock, "://") {
		sock = fmt.Sprintf("unix://%s", sock)
	}
	return sock
}

func getDockerClient(sock string) (*client.Client, error) {
	apiClient, err := client.New(client.WithHost(sock))
	if err != nil {
		return nil, err
	}

	return apiClient, nil
}

// getOperatorID 从 context 中获取操作员ID
// 如果无法获取，返回 0（表示系统操作）
func getOperatorID(ctx context.Context) uint64 {
	if ctx == nil {
		return 0
	}
	userID := ctx.Value("user_id")
	if userID == nil {
		return 0
	}
	return cast.ToUint64(userID)
}

// upsert 分批大小，避免超出 SQLite 变量数限制
const upsertBatchSize = 100

// batchUpsert 通用分批 upsert 辅助函数
func batchUpsert[T any](db *gorm.DB, items []T, conflict clause.OnConflict) error {
	for i := 0; i < len(items); i += upsertBatchSize {
		end := min(i+upsertBatchSize, len(items))
		if err := db.Clauses(conflict).Create(items[i:end]).Error; err != nil {
			return err
		}
	}
	return nil
}
