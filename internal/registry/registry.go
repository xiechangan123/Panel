package registry

import (
	"fmt"
	"sort"
	"strings"

	"github.com/samber/do/v2"
)

// Collect 解析所有名称以 prefix 开头的服务
func Collect[T any](i do.Injector, prefix string) ([]T, error) {
	var names []string
	for _, desc := range i.ListProvidedServices() {
		if strings.HasPrefix(desc.Service, prefix) {
			names = append(names, desc.Service)
		}
	}
	sort.Strings(names)

	out := make([]T, 0, len(names))
	for _, name := range names {
		svc, err := do.InvokeNamed[T](i, name)
		if err != nil {
			return nil, err
		}
		out = append(out, svc)
	}

	return out, nil
}

// Verify 对带命名前缀却不匹配任何已知前缀的贡献报错
func Verify(i do.Injector, prefixes ...string) error {
	for _, desc := range i.ListProvidedServices() {
		name := desc.Service
		if !strings.Contains(name, ":") {
			continue
		}
		known := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(name, prefix) {
				known = true
				break
			}
		}
		if !known {
			return fmt.Errorf("contribution %q matches no known prefix %v", name, prefixes)
		}
	}

	return nil
}

// Lazy 将单依赖的纯构造函数适配为惰性 provider，使构造函数无需感知容器
func Lazy[T, D any](ctor func(D) T) func(do.Injector) {
	return do.Lazy(func(i do.Injector) (T, error) {
		return ctor(do.MustInvoke[D](i)), nil
	})
}

// Lazy2 是 Lazy 的双依赖版本
func Lazy2[T, D1, D2 any](ctor func(D1, D2) T) func(do.Injector) {
	return do.Lazy(func(i do.Injector) (T, error) {
		return ctor(do.MustInvoke[D1](i), do.MustInvoke[D2](i)), nil
	})
}
