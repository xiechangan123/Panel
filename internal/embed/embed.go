package embed

import "embed"

// TODO 移到pkg目录下

//go:embed all:frontend/*
var PublicFS embed.FS

//go:embed all:website/*
var WebsiteFS embed.FS
