package api

import (
	"fmt"

	"github.com/optiflow-os/tracelens-cli/pkg/utils"

	"go.uber.org/zap/zapcore"
)

// Err 表示一个通用的 API 错误。
type Err struct {
	Err error
}

var _ utils.Error = Err{}

// Error 方法实现错误接口。
func (e Err) Error() string {
	return e.Err.Error()
}

// ExitCode 方法实现 wakaerror.Error 接口。
func (Err) ExitCode() int {
	return utils.ErrAPI
}

// Message 方法实现 wakaerror.Error 接口。
func (e Err) Message() string {
	return fmt.Sprintf("api error: %s", e.Err)
}

// SendDiagsOnErrors 方法实现 wakaerror.SendDiagsOnErrors 接口。
func (Err) SendDiagsOnErrors() bool {
	return false
}

// ShouldLogError 方法实现 wakaerror.ShouldLogError 接口。
func (Err) ShouldLogError() bool {
	return true
}

// ErrAuth 表示认证错误。
type ErrAuth struct {
	Err error
}

var _ utils.Error = ErrAuth{}

// Error 方法实现错误接口。
func (e ErrAuth) Error() string {
	return e.Err.Error()
}

// ExitCode 方法实现 wakaerror.Error 接口。
func (ErrAuth) ExitCode() int {
	return utils.ErrAuth
}

// Message 方法实现 wakaerror.Error 接口。
func (e ErrAuth) Message() string {
	return fmt.Sprintf("invalid api key... find yours at wakatime.com/api-key. %s", e.Err.Error())
}

// SendDiagsOnErrors 方法实现 wakaerror.SendDiagsOnErrors 接口。
func (ErrAuth) SendDiagsOnErrors() bool {
	return false
}

// ShouldLogError 方法实现 wakaerror.ShouldLogError 接口。
func (ErrAuth) ShouldLogError() bool {
	return true
}

// ErrBadRequest 表示来自 API 的 400 响应。
type ErrBadRequest struct {
	Err error
}

var _ utils.Error = ErrBadRequest{}

// Error 方法实现错误接口。
func (e ErrBadRequest) Error() string {
	return e.Err.Error()
}

// ExitCode 方法实现 wakaerror.Error 接口。
func (ErrBadRequest) ExitCode() int {
	return utils.ErrGeneric
}

// Message 方法实现 wakaerror.Error 接口。
func (e ErrBadRequest) Message() string {
	return fmt.Sprintf("bad request: %s", e.Err)
}

// SendDiagsOnErrors 方法实现 wakaerror.SendDiagsOnErrors 接口。
func (ErrBadRequest) SendDiagsOnErrors() bool {
	return false
}

// ShouldLogError 方法实现 wakaerror.ShouldLogError 接口。
func (ErrBadRequest) ShouldLogError() bool {
	return true
}

// ErrBackoff 表示因为当前被限速而稍后发送。
type ErrBackoff struct {
	Err error
}

var _ utils.Error = ErrBackoff{}

// Error 方法实现错误接口。
func (e ErrBackoff) Error() string {
	return e.Err.Error()
}

// ExitCode 方法实现 wakaerror.Error 接口。
func (ErrBackoff) ExitCode() int {
	return utils.ErrBackoff
}

// LogLevel 方法实现 wakaerror.LogLevel 接口。
func (ErrBackoff) LogLevel() int8 {
	return int8(zapcore.DebugLevel)
}

// Message 方法实现 wakaerror.Error 接口。
func (e ErrBackoff) Message() string {
	return fmt.Sprintf("rate limited: %s", e.Err)
}

// SendDiagsOnErrors 方法实现 wakaerror.SendDiagsOnErrors 接口。
func (ErrBackoff) SendDiagsOnErrors() bool {
	return false
}

// ShouldLogError 方法实现 wakaerror.ShouldLogError 接口。
func (ErrBackoff) ShouldLogError() bool {
	return false
}

// ErrTimeout 表示超时错误。
type ErrTimeout struct {
	Err error
}

var _ utils.Error = ErrTimeout{}

// Error 方法实现错误接口。
func (e ErrTimeout) Error() string {
	return e.Err.Error()
}

// ExitCode 方法实现 wakaerror.Error 接口。
func (ErrTimeout) ExitCode() int {
	return utils.ErrGeneric
}

// LogLevel 方法实现 wakaerror.LogLevel 接口。
func (ErrTimeout) LogLevel() int8 {
	return int8(zapcore.DebugLevel)
}

// Message 方法实现 wakaerror.Error 接口。
func (e ErrTimeout) Message() string {
	return fmt.Sprintf("timeout: %s", e.Err)
}

// SendDiagsOnErrors 方法实现 wakaerror.SendDiagsOnErrors 接口。
func (ErrTimeout) SendDiagsOnErrors() bool {
	return false
}

// ShouldLogError 方法实现 wakaerror.ShouldLogError 接口。
func (ErrTimeout) ShouldLogError() bool {
	return false
}
