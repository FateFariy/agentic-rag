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
	case int8:
		val, err := toInt8(v)
		return any(val).(T), err
	case int16:
		val, err := toInt16(v)
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

func toInt(v any) (int, error) {
	switch val := v.(type) {
	case int:
		return val, nil
	case int8:
		return int(val), nil
	case int16:
		return int(val), nil
	case int32:
		return int(val), nil
	case int64:
		return int(val), nil
	case float32:
		return int(val), nil
	case float64:
		return int(val), nil
	case string:
		return strconv.Atoi(val)
	}
	return 0, fmt.Errorf("cannot convert %T to int", v)
}

func toInt8(v any) (int8, error) {
	val, err := toInt64(v)
	return int8(val), err
}

func toInt16(v any) (int16, error) {
	val, err := toInt64(v)
	return int16(val), err
}

func toInt32(v any) (int32, error) {
	switch val := v.(type) {
	case int32:
		return val, nil
	case int:
		return int32(val), nil
	case int8:
		return int32(val), nil
	case int16:
		return int32(val), nil
	case int64:
		return int32(val), nil
	case float32:
		return int32(val), nil
	case float64:
		return int32(val), nil
	case string:
		i, err := strconv.ParseInt(val, 10, 32)
		return int32(i), err
	}
	return 0, fmt.Errorf("cannot convert %T to int32", v)
}

func toInt64(v any) (int64, error) {
	switch val := v.(type) {
	case int64:
		return val, nil
	case int:
		return int64(val), nil
	case int8:
		return int64(val), nil
	case int16:
		return int64(val), nil
	case int32:
		return int64(val), nil
	case float32:
		return int64(val), nil
	case float64:
		return int64(val), nil
	case string:
		return strconv.ParseInt(val, 10, 64)
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
		f, err := strconv.ParseFloat(val, 32)
		return float32(f), err
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
		return strconv.ParseFloat(val, 64)
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
