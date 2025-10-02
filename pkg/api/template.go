package api

import (
	"fmt"
	"time"
)

type Template struct {
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Slug         string    `json:"slug"`
	Icon         string    `json:"icon"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Categories   []string  `json:"categories"`
	Version      string    `json:"version"`
	Compose      string    `json:"compose"`
	Environments []struct {
		Name    string            `json:"name"`              // 变量名
		Type    string            `json:"type"`              // 变量类型， text, password, number, port, select
		Options map[string]string `json:"options,omitempty"` // 下拉框选项，key -> value
		Default string            `json:"default"`           // 默认值
	} `json:"environments"`
}

type Templates []*Template

// Templates 返回所有模版
func (r *API) Templates() (*Templates, error) {
	resp, err := r.client.R().SetResult(&Response{}).Get("/templates")
	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get templates: %s", resp.String())
	}

	templates, err := getResponseData[Templates](resp)
	if err != nil {
		return nil, err
	}

	return templates, nil
}

// TemplateBySlug 根据slug返回模版
func (r *API) TemplateBySlug(slug string) (*Template, error) {
	resp, err := r.client.R().SetResult(&Response{}).Get(fmt.Sprintf("/templates/%s", slug))
	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("failed to get template: %s", resp.String())
	}

	template, err := getResponseData[Template](resp)
	if err != nil {
		return nil, err
	}

	return template, nil
}

// TemplateCallback 模版下载回调
func (r *API) TemplateCallback(slug string) error {
	resp, err := r.client.R().SetResult(&Response{}).Post(fmt.Sprintf("/templates/%s/callback", slug))
	if err != nil {
		return err
	}
	if !resp.IsSuccess() {
		return fmt.Errorf("failed to callback template: %s", resp.String())
	}

	return nil
}
