package biz

import (
	"context"

	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/types"
)

// ToolboxMigrationSourceRepo 封装不同来源面板的 API 差异
type ToolboxMigrationSourceRepo interface {
	Probe(ctx context.Context, conn *request.ToolboxMigrationConnection) (*types.MigrationSourceInfo, error)
	Items(ctx context.Context, conn *request.ToolboxMigrationConnection) ([]types.MigrationSourceItem, error)
	Detail(ctx context.Context, conn *request.ToolboxMigrationConnection, item types.MigrationSourceItem) (*types.MigrationSourceDetail, error)
	SetRunning(ctx context.Context, conn *request.ToolboxMigrationConnection, detail *types.MigrationSourceDetail, running bool) error
	Prepare(ctx context.Context, conn *request.ToolboxMigrationConnection, detail *types.MigrationSourceDetail) (*types.MigrationArtifact, error)
	Download(ctx context.Context, conn *request.ToolboxMigrationConnection, artifact *types.MigrationArtifact, target string) error
}
