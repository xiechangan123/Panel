package request

type App struct {
	Slug    string `json:"slug" form:"slug" validate:"required && not_exists:apps,slug"`
	Channel string `json:"channel" form:"channel" validate:"required"`
}

type AppSlug struct {
	Slug string `json:"slug" form:"slug" validate:"required"`
}

type AppSlugs struct {
	Slugs string `json:"slugs" form:"slugs" validate:"required"`
}

// AppCustomSave 保存自定义编译参数
type AppCustomSave struct {
	Slug      string `json:"slug" form:"slug" validate:"required"`
	PreScript string `json:"pre_script" form:"pre_script"`
	Args      string `json:"args" form:"args"`
}

type AppUpdateShow struct {
	Slug string `json:"slug" form:"slug" validate:"required && exists:apps,slug"`
	Show bool   `json:"show" form:"show"`
}

type AppUpdateOrder struct {
	Slugs []string `json:"slugs" form:"slugs" validate:"required && unique"`
}
