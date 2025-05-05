package diagnostic

import "fmt"

// Type 是诊断的类型。
type Type int

const (
	// TypeUnknown 表示未知类型的诊断。
	TypeUnknown Type = iota
	// TypeError 表示错误类型的诊断。
	TypeError
	// TypeLogs 表示日志类型的诊断。
	TypeLogs
	// TypeStack 表示堆栈跟踪类型的诊断。
	TypeStack
)

// Diagnostic 包含诊断信息。
type Diagnostic struct {
	Type  Type
	Value string
}

// Error 创建一个 TypeError 类型的诊断实例。
func Error(err any) Diagnostic {
	return Diagnostic{
		Type:  TypeError,
		Value: fmt.Sprintf("%v", err),
	}
}

// Logs 创建一个 TypeLogs 类型的诊断实例。
func Logs(logs string) Diagnostic {
	return Diagnostic{
		Type:  TypeLogs,
		Value: logs,
	}
}

// Stack 创建一个 TypeStack 类型的诊断实例。
func Stack(stack string) Diagnostic {
	return Diagnostic{
		Type:  TypeStack,
		Value: stack,
	}
}
