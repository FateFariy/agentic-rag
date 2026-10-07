package utils

import (
	"cmp"
	"slices"

	"github.com/duke-git/lancet/v2/maputil"
)

// MapValues 获取 map 的所有值
func MapValues[K comparable, V any](m map[K]V) []V {
	return maputil.Values(m)
}

// MapKeys 获取 map 的所有键
func MapKeys[K comparable, V any](m map[K]V) []K {
	return maputil.Keys(m)
}

// MapValue 获取 map 中指定键的值
func MapValue[T any](m map[string]any, key string) T {
	if v, ok := m[key]; ok && v != nil {
		if val, ok := v.(T); ok {
			return val
		}
	}
	var zero T
	return zero
}

// SortedKeys 返回 map 按键升序排序后的键列表（确定性遍历，空 map 返回 nil）
func SortedKeys[K cmp.Ordered, V any](m map[K]V) []K {
	if len(m) == 0 {
		return nil
	}
	keys := maputil.Keys(m)
	slices.Sort(keys)
	return keys
}
