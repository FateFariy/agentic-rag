package utils

import (
	"fmt"
	"strconv"
)

func Int64ToString(i int64) string {
	return strconv.FormatInt(i, 10)
}

func StringToInt64(s string) int64 {
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

// MustConvert 将任意类型转换为目标类型 T，转换失败返回 error
func MustConvert[T any](v any) (T, error) {
	var zero T

	if v == nil {
		return zero, fmt.Errorf("value is nil")
	}

	// 1. 类型完全匹配，直接返回
	if val, ok := v.(T); ok {
		return val, nil
	}

	// 2. 根据目标类型做自动转换
	switch any(zero).(type) {
	case int:
		val, err := toInt(v)
		return any(val).(T), err
	case int32:
		val, err := toInt32(v)
		return any(val).(T), err
	case int64:
		val, err := toInt64(v)
		return any(val).(T), err
	case float32:
		val, err := toFloat32(v)
		return any(val).(T), err
	case float64:
		val, err := toFloat64(v)
		return any(val).(T), err
	case string:
		return any(fmt.Sprintf("%v", v)).(T), nil
	case bool:
		val, err := toBool(v)
		return any(val).(T), err
	}

	return zero, fmt.Errorf("unsupported conversion from %T to %T", v, zero)
}

// Convert 将任意类型转换为目标类型 T，转换失败返回零值（无 error）
func Convert[T any](v any) T {
	val, _ := MustConvert[T](v)
	return val
}

// CastSliceTo 将 any 转换为 []T；失败返回空切片（零值兜底）
// 支持：[]T、[]any（逐元素断言）、单个 T（自动包装为切片）
func CastSliceTo[T any](v any) []T {
	result, _ := MustCastSliceTo[T](v)
	return result
}

// MustCastSliceTo 将 any 转换为 []T；失败返回错误
func MustCastSliceTo[T any](v any) ([]T, error) {
	if v == nil {
		return nil, fmt.Errorf("value is nil")
	}

	// 1. 直接就是 []T
	if slice, ok := v.([]T); ok {
		return slice, nil
	}

	// 2. 是 []any，逐元素断言
	if slice, ok := v.([]any); ok {
		result := make([]T, 0, len(slice))
		for i, item := range slice {
			if val, ok := item.(T); ok {
				result = append(result, val)
			} else {
				var zero T
				return nil, fmt.Errorf("index %d: cannot cast %T to %T", i, item, zero)
			}
		}
		return result, nil
	}

	// 3. 单个元素，自动包装成切片
	var zero T
	if val, ok := v.(T); ok {
		return []T{val}, nil
	}

	return nil, fmt.Errorf("cannot cast %T to []%T", v, zero)
}

func toInt(v any) (int, error) {
	switch val := v.(type) {
	case int:
		return val, nil
	case int32:
		return int(val), nil
	case int64:
		return int(val), nil
	case float32:
		return int(val), nil
	case float64:
		return int(val), nil
	case string:
		var i int
		_, err := fmt.Sscanf(val, "%d", &i)
		return i, err
	}
	return 0, fmt.Errorf("cannot convert %T to int", v)
}

func toInt32(v any) (int32, error) {
	switch val := v.(type) {
	case int32:
		return val, nil
	case int:
		return int32(val), nil
	case int64:
		return int32(val), nil
	case float32:
		return int32(val), nil
	case float64:
		return int32(val), nil
	case string:
		var i int32
		_, err := fmt.Sscanf(val, "%d", &i)
		return i, err
	}
	return 0, fmt.Errorf("cannot convert %T to int32", v)
}

func toInt64(v any) (int64, error) {
	switch val := v.(type) {
	case int64:
		return val, nil
	case int:
		return int64(val), nil
	case int32:
		return int64(val), nil
	case float32:
		return int64(val), nil
	case float64:
		return int64(val), nil
	case string:
		var i int64
		_, err := fmt.Sscanf(val, "%d", &i)
		return i, err
	}
	return 0, fmt.Errorf("cannot convert %T to int64", v)
}

func toFloat32(v any) (float32, error) {
	switch val := v.(type) {
	case float32:
		return val, nil
	case float64:
		return float32(val), nil
	case int:
		return float32(val), nil
	case int32:
		return float32(val), nil
	case int64:
		return float32(val), nil
	case string:
		var f float32
		_, err := fmt.Sscanf(val, "%f", &f)
		return f, err
	}
	return 0, fmt.Errorf("cannot convert %T to float32", v)
}

func toFloat64(v any) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case string:
		var f float64
		_, err := fmt.Sscanf(val, "%f", &f)
		return f, err
	}
	return 0, fmt.Errorf("cannot convert %T to float64", v)
}

func toBool(v any) (bool, error) {
	switch val := v.(type) {
	case bool:
		return val, nil
	case string:
		return val == "true" || val == "1", nil
	case int:
		return val != 0, nil
	case int32:
		return val != 0, nil
	case int64:
		return val != 0, nil
	}
	return false, fmt.Errorf("cannot convert %T to bool", v)
}
