package types

// EnvironmentDetail 环境详情
type EnvironmentDetail struct {
	Type             string `json:"type"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	Slug             string `json:"slug"`
	Version          string `json:"version"`
	InstalledVersion string `json:"installed_version"`
	Installed        bool   `json:"installed"`
	HasUpdate        bool   `json:"has_update"`
}
