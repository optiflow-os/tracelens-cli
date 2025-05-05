// 本文件提供了所有错误代码的定义

package utils

import "strconv"

const (
	// Success 用于标识心跳发送成功时使用。
	Success = 0
	// ErrGeneric 用于一般错误。
	ErrGeneric = 1
	// ErrAPI 当 TraceLens API 返回错误时使用。
	ErrAPI = 102
	// ErrAuth 当 API 密钥无效时使用。
	ErrAuth = 104
	// ErrConfigFileParse 当无法解析 ~/.tracelens.cfg 配置文件时使用。
	ErrConfigFileParse = 103
	// ErrConfigFileRead 用于配置读取命令的错误。
	ErrConfigFileRead = 110
	// ErrConfigFileWrite 用于配置写入命令的错误。
	ErrConfigFileWrite = 111
	// ErrBackoff 当由于速率限制而推迟发送心跳时使用。
	ErrBackoff = 112
)

// Err 表示退出代码错误的类型响应。成功响应也包装在此类型中。
type Err struct {
	Code int
}

// Error 方法实现错误接口。
func (e Err) Error() string {
	return strconv.Itoa(e.Code)
}
