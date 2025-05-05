package log

import (
	"fmt"
	"io"

	"github.com/optiflow-os/tracelens-cli/pkg/version"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// MaxLogFileSize 是日志文件的最大大小。
	MaxLogFileSize = 25 // 25MB
	// MaxNumberOfBackups 是日志文件备份的最大数量。
	MaxNumberOfBackups = 4
)

// Logger 是日志条目。
type Logger struct {
	entry              *zap.Logger
	atomicLevel        zap.AtomicLevel
	currentOutput      io.Writer
	dynamicWriteSyncer *DynamicWriteSyncer
	metrics            bool
	sendDiagsOnErrors  bool
	verbose            bool
}

// New 创建一个写入到 dest 的新 Logger。
func New(dest io.Writer, opts ...Option) *Logger {
	atom := zap.NewAtomicLevel()
	dynamicWriteSyncer := NewDynamicWriteSyncer(zapcore.AddSync(dest))

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "now"
	encoderCfg.EncodeTime = zapcore.RFC3339TimeEncoder
	encoderCfg.MessageKey = "message"
	encoderCfg.FunctionKey = "func"

	l := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		dynamicWriteSyncer,
		atom,
	),
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zap.FatalLevel),
	)

	l = l.With(
		zap.String("version", version.Version),
		zap.String("os/arch", fmt.Sprintf("%s/%s", version.OS, version.Arch)),
	)

	logger := &Logger{
		entry:              l,
		atomicLevel:        atom,
		currentOutput:      dest,
		dynamicWriteSyncer: dynamicWriteSyncer,
	}

	for _, option := range opts {
		option(logger)
	}

	return logger
}

// IsMetricsEnabled 如果应该收集指标，则返回 true。
func (l *Logger) IsMetricsEnabled() bool {
	return l.metrics
}

// IsVerboseEnabled 如果启用了调试，则返回 true。
func (l *Logger) IsVerboseEnabled() bool {
	return l.verbose
}

// Output 返回当前日志输出。
func (l *Logger) Output() io.Writer {
	return l.currentOutput
}

// SendDiagsOnErrors 如果应该在出错时发送诊断信息，则返回 true。
func (l *Logger) SendDiagsOnErrors() bool {
	return l.sendDiagsOnErrors
}

// SetOutput 定义将日志输出设置为 io.Writer。
func (l *Logger) SetOutput(w io.Writer) {
	l.currentOutput = w
	l.dynamicWriteSyncer.SetWriter(zapcore.AddSync(w))
}

// SetVerbose 如果启用，则将日志级别设置为调试。
func (l *Logger) SetVerbose(verbose bool) {
	l.verbose = verbose

	if verbose {
		l.atomicLevel.SetLevel(zap.DebugLevel)
	} else {
		l.atomicLevel.SetLevel(zap.InfoLevel)
	}
}

// Flush 刷新日志输出并关闭文件。
func (l *Logger) Flush() {
	if err := l.entry.Sync(); err != nil {
		l.Debugf("failed to flush log file: %s", err)
	}

	if closer, ok := l.currentOutput.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			l.Debugf("failed to close log file: %s", err)
		}
	}
}

// Log 在给定级别记录消息。
func (l Logger) Log(level zapcore.Level, msg string) {
	l.entry.Log(level, msg)
}

// Logf 在给定级别记录消息。
func (l Logger) Logf(level zapcore.Level, format string, args ...any) {
	l.entry.Log(level, fmt.Sprintf(format, args...))
}

// Debugf 在 Debug 级别记录消息。
func (l *Logger) Debugf(format string, args ...any) {
	l.entry.Log(zapcore.DebugLevel, fmt.Sprintf(format, args...))
}

// Infof 在 Info 级别记录消息。
func (l *Logger) Infof(format string, args ...any) {
	l.entry.Log(zapcore.InfoLevel, fmt.Sprintf(format, args...))
}

// Warnf 在 Warn 级别记录消息。
func (l *Logger) Warnf(format string, args ...any) {
	l.entry.Log(zapcore.WarnLevel, fmt.Sprintf(format, args...))
}

// Errorf 在 Error 级别记录消息。
func (l *Logger) Errorf(format string, args ...any) {
	l.entry.Log(zapcore.ErrorLevel, fmt.Sprintf(format, args...))
}

// Fatalf 在 Fatal 级别记录消息，然后进程将以状态 1 退出。
func (l *Logger) Fatalf(format string, args ...any) {
	l.entry.Log(zapcore.FatalLevel, fmt.Sprintf(format, args...))
}

// Debugln 在 Debug 级别记录消息。
func (l *Logger) Debugln(msg string) {
	l.entry.Log(zapcore.DebugLevel, msg)
}

// Infoln 在 Info 级别记录消息。
func (l *Logger) Infoln(msg string) {
	l.entry.Log(zapcore.InfoLevel, msg)
}

// Warnln 在 Warn 级别记录消息。
func (l *Logger) Warnln(msg string) {
	l.entry.Log(zapcore.WarnLevel, msg)
}

// Errorln 在 Error 级别记录消息。
func (l *Logger) Errorln(msg string) {
	l.entry.Log(zapcore.ErrorLevel, msg)
}

// Fatalln 在 Fatal 级别记录消息，然后进程将以状态 1 退出。
func (l *Logger) Fatalln(msg string) {
	l.entry.Log(zapcore.FatalLevel, msg)
}

// WithField 向 Logger 添加单个字段。
func (l *Logger) WithField(key string, value any) {
	l.entry = l.entry.With(zap.Any(key, value))
}
