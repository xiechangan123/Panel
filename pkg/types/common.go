package types

import (
	"strings"

	"github.com/samber/lo"
)

type NV struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type KV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type LV struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type LVInt struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

// KVToMap 将 key-value 切片转换为 map
func KVToMap(kvs []KV) map[string]string {
	return lo.SliceToMap(kvs, func(item KV) (string, string) {
		return item.Key, item.Value
	})
}

// MapToKV 将 map 转换为 key-value 切片
func MapToKV(m map[string]string) []KV {
	return lo.MapToSlice(m, func(k, v string) KV {
		return KV{Key: k, Value: v}
	})
}

// KVToSlice 将 key-value 切片转换为 key=value 切片
func KVToSlice(kvs []KV) []string {
	return lo.Map(kvs, func(item KV, _ int) string {
		return item.Key + "=" + item.Value
	})
}

// SliceToKV 将 key=value 切片转换为 key-value 切片
func SliceToKV(s []string) []KV {
	return lo.FilterMap(s, func(item string, _ int) (KV, bool) {
		kv := strings.SplitN(item, "=", 2)
		if len(kv) != 2 {
			return KV{}, false
		}
		return KV{Key: kv[0], Value: kv[1]}, true
	})
}
