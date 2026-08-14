package data

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/lo"
	"github.com/spf13/cast"
	"resty.dev/v3"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/types"
)

// migrationAdapter 单个来源面板的 API 适配
type migrationAdapter interface {
	Probe(ctx context.Context) (*types.MigrationSource, error)
	Items(ctx context.Context) ([]types.MigrationItem, error)
	Detail(ctx context.Context, item types.MigrationItem) (*types.MigrationDetail, error)
	SetRunning(ctx context.Context, item types.MigrationItem, running bool) error
	Backup(ctx context.Context, detail *types.MigrationDetail) (string, error)
	Download(ctx context.Context, remote, local string) error
}

type migrationSourceRepo struct {
	t *gotext.Locale
}

func NewMigrationSourceRepo(t *gotext.Locale) (biz.MigrationSourceRepo, error) {
	return &migrationSourceRepo{t: t}, nil
}

func (r *migrationSourceRepo) adapter(conn *request.ToolboxMigrationConnection) (migrationAdapter, error) {
	switch conn.SourcePanel {
	case "baota":
		return &baotaAdapter{t: r.t, migrationClient: newBaotaClient(conn)}, nil
	case "onepanel":
		return &onePanelAdapter{t: r.t, migrationClient: newOnePanelClient(conn)}, nil
	default:
		return nil, errors.New(r.t.Get("unsupported source panel: %s", conn.SourcePanel))
	}
}

func (r *migrationSourceRepo) Probe(ctx context.Context, conn *request.ToolboxMigrationConnection) (*types.MigrationSource, error) {
	adapter, err := r.adapter(conn)
	if err != nil {
		return nil, err
	}
	return adapter.Probe(ctx)
}

func (r *migrationSourceRepo) Items(ctx context.Context, conn *request.ToolboxMigrationConnection) ([]types.MigrationItem, error) {
	adapter, err := r.adapter(conn)
	if err != nil {
		return nil, err
	}
	return adapter.Items(ctx)
}

func (r *migrationSourceRepo) Detail(
	ctx context.Context,
	conn *request.ToolboxMigrationConnection,
	item types.MigrationItem,
) (*types.MigrationDetail, error) {
	adapter, err := r.adapter(conn)
	if err != nil {
		return nil, err
	}
	return adapter.Detail(ctx, item)
}

func (r *migrationSourceRepo) SetRunning(
	ctx context.Context,
	conn *request.ToolboxMigrationConnection,
	item types.MigrationItem,
	running bool,
) error {
	adapter, err := r.adapter(conn)
	if err != nil {
		return err
	}
	return adapter.SetRunning(ctx, item, running)
}

func (r *migrationSourceRepo) Backup(
	ctx context.Context,
	conn *request.ToolboxMigrationConnection,
	detail *types.MigrationDetail,
) (string, error) {
	adapter, err := r.adapter(conn)
	if err != nil {
		return "", err
	}
	return adapter.Backup(ctx, detail)
}

func (r *migrationSourceRepo) Download(ctx context.Context, conn *request.ToolboxMigrationConnection, remote, local string) error {
	adapter, err := r.adapter(conn)
	if err != nil {
		return err
	}
	return adapter.Download(ctx, remote, local)
}

// migrationClient 来源面板 HTTP 客户端，屏蔽两家面板的认证与响应格式差异
type migrationClient struct {
	url string
	// sign 为每个请求注入认证信息
	sign func(*resty.Request)
	// form 为真时以表单提交，否则提交 JSON
	form bool
	// unwrap 从响应体中取出业务数据
	unwrap func([]byte) (any, error)
	// downloadPath 与 downloadParam 描述文件下载接口
	downloadPath  string
	downloadParam string
}

func (c *migrationClient) client(timeout time.Duration) *resty.Client {
	return resty.New().
		SetBaseURL(strings.TrimRight(c.url, "/")).
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}).
		SetTimeout(timeout)
}

// call 调用来源面板 API 并返回业务数据
func (c *migrationClient) call(ctx context.Context, method, path string, body map[string]any) (any, error) {
	client := c.client(45 * time.Second)
	defer func() { _ = client.Close() }()

	req := client.R().SetContext(ctx)
	c.sign(req)
	switch {
	case method == http.MethodGet:
		req.SetQueryParams(cast.ToStringMapString(body))
	case c.form:
		req.SetFormData(cast.ToStringMapString(body))
	case body != nil:
		req.SetBody(body)
	}

	resp, err := req.Execute(method, path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode(), strings.TrimSpace(resp.String()))
	}
	return c.unwrap(resp.Bytes())
}

// download 下载来源面板上的文件
func (c *migrationClient) download(ctx context.Context, remote, local string) error {
	client := c.client(6 * time.Hour)
	defer func() { _ = client.Close() }()

	req := client.R().SetContext(ctx).SetResponseDoNotParse(true).SetQueryParam(c.downloadParam, remote)
	c.sign(req)
	resp, err := req.Get(c.downloadPath)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode() != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("status %d: %s", resp.StatusCode(), strings.TrimSpace(string(body)))
	}

	if err = os.MkdirAll(filepath.Dir(local), 0755); err != nil {
		return err
	}
	file, err := os.Create(local)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	_, err = io.Copy(file, resp.Body)
	return err
}

// rows 把列表响应转换为若干行
func (c *migrationClient) rows(value any) []map[string]any {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	return lo.FilterMap(values, func(value any, _ int) (map[string]any, bool) {
		row := cast.ToStringMap(value)
		return row, len(row) > 0
	})
}

// phpVersion 把 8.3 之类的版本号转为面板使用的 83
func (c *migrationClient) phpVersion(version string) uint {
	parts := strings.FieldsFunc(strings.TrimPrefix(strings.ToLower(strings.TrimSpace(version)), "php"), func(char rune) bool {
		return char == '.' || char == '-' || char == '_'
	})
	if len(parts) < 2 {
		return 0
	}
	major, minor := cast.ToUint(parts[0]), cast.ToUint(parts[1])
	if major == 0 {
		return 0
	}
	return major*10 + minor
}
