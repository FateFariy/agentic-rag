package convert

import (
	"encoding/json"
	"strconv"

	"github.com/duke-git/lancet/v2/strutil"
)

// NormalizeString 去除字符串首尾空白
func NormalizeString(s string) string {
	return strutil.Trim(s)
}

// Int64ToString int64 转字符串
func Int64ToString(i int64) string {
	return strconv.FormatInt(i, 10)
}

// StringToInt64 字符串转 int64，解析失败返回 0
func StringToInt64(s string) int64 {
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

// RawMessageToString 将 json.RawMessage 转为字符串
func RawMessageToString(src json.RawMessage) string {
	return string(src)
}
