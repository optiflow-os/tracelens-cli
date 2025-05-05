package utils

import (
	"strings"

	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

// FirstNonEmptyBool 接受多个键，并通过这些键从 viper.Viper 返回第一个非空布尔值。
// 非空意味着未设置的键将不被接受。
func FirstNonEmptyBool(v *viper.Viper, keys ...string) bool {
	if v == nil {
		return false
	}

	for _, key := range keys {
		if !v.IsSet(key) {
			continue
		}

		value := v.Get(key)

		parsed, err := cast.ToBoolE(value)
		if err != nil {
			continue
		}

		return parsed
	}

	return false
}

// FirstNonEmptyInt 接受多个键，并通过这些键从 viper.Viper 返回第一个非空整数值。
// 非空意味着未设置的键将不被接受。
// 如果无法获取非空整数值，将返回 false 作为第二个参数。
func FirstNonEmptyInt(v *viper.Viper, keys ...string) (int, bool) {
	if v == nil {
		return 0, false
	}

	for _, key := range keys {
		if !v.IsSet(key) {
			continue
		}

		// 当设置时，零表示有效值，因此需要使用通用函数，然后将其转换为整数
		value := v.Get(key)

		// 如果值不是整数，将继续查找下一个非空键
		parsed, err := cast.ToIntE(value)
		if err != nil {
			continue
		}

		return parsed, true
	}

	return 0, false
}

// FirstNonEmptyString 接受多个键，并通过这些键从 viper.Viper 返回第一个非空字符串值。
// 如果找不到值，则默认返回空字符串。
func FirstNonEmptyString(v *viper.Viper, keys ...string) string {
	if v == nil {
		return ""
	}

	for _, key := range keys {
		if !v.IsSet(key) {
			continue
		}

		value := v.Get(key)

		parsed, err := cast.ToStringE(value)
		if err != nil {
			continue
		}

		return strings.Trim(parsed, `"'`)
		//	if value := GetString(v, key); value != "" {
		//		return value
		//	}
	}

	return ""
}

// GetString 通过键获取参数/设置并去除任何引号。
func GetString(v *viper.Viper, key string) string {
	return strings.Trim(v.GetString(key), `"'`)
}

// GetStringMapString 通过键前缀获取参数/设置并去除任何引号。
func GetStringMapString(v *viper.Viper, prefix string) map[string]string {
	m := map[string]string{}

	for _, k := range v.AllKeys() {
		if !strings.HasPrefix(k, prefix+".") {
			continue
		}

		m[strings.TrimPrefix(k, prefix+".")] = GetString(v, k)
	}

	return m
}
