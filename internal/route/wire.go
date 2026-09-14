//go:build wireinject

package route

import "github.com/libtnb/wire"

// Module 装配路由层
var Module = wire.New().
	Struct[Services]().
	Provide(NewEndpoints).
	Export[[]Endpoints]()
