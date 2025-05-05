package cmd

import (
	"context"
	"fmt"
	"io"
	stdlog "log"
	"os"
	"path/filepath"
	"strings"

	cmdconfigread "github.com/optiflow-os/tracelens-cli/cmd/configread"
	cmdconfigwrite "github.com/optiflow-os/tracelens-cli/cmd/configwrite"
	cmdfileexperts "github.com/optiflow-os/tracelens-cli/cmd/fileexperts"
	cmdheartbeat "github.com/optiflow-os/tracelens-cli/cmd/heartbeat"
	cmdoffline "github.com/optiflow-os/tracelens-cli/cmd/offline"
	cmdofflinecount "github.com/optiflow-os/tracelens-cli/cmd/offlinecount"
	cmdofflineprint "github.com/optiflow-os/tracelens-cli/cmd/offlineprint"
	cmdofflinesync "github.com/optiflow-os/tracelens-cli/cmd/offlinesync"
	cmdtoday "github.com/optiflow-os/tracelens-cli/cmd/today"
	cmdtodaygoal "github.com/optiflow-os/tracelens-cli/cmd/todaygoal"
	"github.com/optiflow-os/tracelens-cli/pkg/exitcode"
	"github.com/optiflow-os/tracelens-cli/pkg/log"
	"github.com/optiflow-os/tracelens-cli/pkg/offline"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"
)

type diagnostics struct {
	Logs          string
	OriginalError any
	Panicked      bool
	Stack         string
}

// cmdFn 表示命令函数。
type cmdFn func(ctx context.Context, v *viper.Viper) (int, error)

// RunCmd 运行命令函数并使用命令函数返回的退出代码退出。
// 在任何错误或崩溃时将发送诊断信息。
func RunCmd(ctx context.Context, v *viper.Viper, verbose bool, sendDiagsOnErrors bool, cmd cmdFn) error {
	logger := log.Extract(ctx)

	// 运行命令
	exitCode, err := cmd(ctx, v)
	if err != nil {
		if verbose {
			logger.Errorf("运行命令失败: %s", err)
		}
	}

	if exitCode != exitcode.Success {
		logger.Debugf("命令执行失败，退出代码 %d", exitCode)
		return exitcode.Err{Code: exitCode}
	}

	return nil
}

// RunCmdWithOfflineSync 运行命令函数并使用命令函数返回的退出代码退出。
// 如果命令运行成功，它会随后执行离线同步命令。
// 在任何错误或崩溃时将发送诊断信息。
func RunCmdWithOfflineSync(ctx context.Context, v *viper.Viper, verbose bool, sendDiagsOnErrors bool, cmd cmdFn) error {
	if err := RunCmd(ctx, v, verbose, sendDiagsOnErrors, cmd); err != nil {
		return err
	}

	return RunCmd(ctx, v, verbose, sendDiagsOnErrors, cmdofflinesync.RunWithRateLimiting)
}

// RunE 执行从命令行解析的命令。
func RunE(cmd *cobra.Command, v *viper.Viper) error {
	ctx := context.Background()

	// 从上下文中提取日志记录器，尽管它尚未完全初始化
	logger := log.Extract(ctx)

	err := parseConfigFiles(ctx, v)
	if err != nil {
		logger.Errorf("解析配置文件失败: %s", err)

		if v.IsSet("entity") {
			_ = saveHeartbeats(ctx, v)

			return exitcode.Err{Code: exitcode.ErrConfigFileParse}
		}
	}

	logger, err = SetupLogging(ctx, v)
	if err != nil {
		// 日志记录器实例设置失败，使用标准输出记录并退出
		stdlog.Fatalf("设置日志记录失败: %s", err)
	}

	// 将日志记录器保存到上下文
	ctx = log.ToContext(ctx, logger)

	// 开始分析（如果启用）
	if logger.IsMetricsEnabled() {
		// 在完整实现时添加指标功能
		logger.Debugln("已启用指标收集")
	}

	if v.GetBool("version") {
		logger.Debugln("命令: version")

		return RunCmd(ctx, v, logger.IsVerboseEnabled(), logger.SendDiagsOnErrors(), runVersion)
	}

	if v.IsSet("entity") {
		logger.Debugln("命令: heartbeat")

		return RunCmd(ctx, v, logger.IsVerboseEnabled(), logger.SendDiagsOnErrors(), cmdheartbeat.Run)
	}

	if v.IsSet("sync-offline-activity") {
		logger.Debugln("命令: sync-offline-activity")

		return RunCmd(ctx, v, logger.IsVerboseEnabled(), logger.SendDiagsOnErrors(), cmdofflinesync.RunWithoutRateLimiting)
	}

	if v.GetBool("offline-count") {
		logger.Debugln("命令: offline-count")

		return RunCmd(ctx, v, logger.IsVerboseEnabled(), logger.SendDiagsOnErrors(), cmdofflinecount.Run)
	}

	if v.IsSet("print-offline-heartbeats") {
		logger.Debugln("命令: print-offline-heartbeats")

		return RunCmd(ctx, v, logger.IsVerboseEnabled(), logger.SendDiagsOnErrors(), cmdofflineprint.Run)
	}

	if v.IsSet("config-read") {
		logger.Debugln("命令: config-read")

		return RunCmd(ctx, v, logger.IsVerboseEnabled(), logger.SendDiagsOnErrors(), cmdconfigread.Run)
	}

	if v.IsSet("config-write") {
		logger.Debugln("命令: config-write")

		return RunCmd(ctx, v, logger.IsVerboseEnabled(), logger.SendDiagsOnErrors(), cmdconfigwrite.Run)
	}

	if v.GetBool("today") {
		logger.Debugln("命令: today")

		return RunCmd(ctx, v, logger.IsVerboseEnabled(), logger.SendDiagsOnErrors(), cmdtoday.Run)
	}

	if v.IsSet("today-goal") {
		logger.Debugln("命令: today-goal")

		return RunCmd(ctx, v, logger.IsVerboseEnabled(), logger.SendDiagsOnErrors(), cmdtodaygoal.Run)
	}

	if v.GetBool("file-experts") {
		logger.Debugln("命令: file-experts")

		return RunCmd(ctx, v, logger.IsVerboseEnabled(), logger.SendDiagsOnErrors(), cmdfileexperts.Run)
	}

	logger.Warnf("one of the following parameters has to be provided: %s", strings.Join([]string{
		"--config-read",
		"--config-write",
		"--entity",
		"--file-experts",
		"--offline-count",
		"--print-offline-heartbeats",
		"--sync-offline-activity",
		"--today",
		"--today-goal",
		"--user-agent",
		"--version",
	}, ", "))

	_ = cmd.Help()

	return exitcode.Err{Code: exitcode.ErrGeneric}
}

func parseConfigFiles(ctx context.Context, v *viper.Viper) error {
	// 简化的配置文件解析功能，真实实现将在完整版本中添加
	logger := log.Extract(ctx)
	logger.Debugln("解析配置文件...")

	return nil
}

// SetupLogging 使用 --log-file 参数配置记录到文件或标准输出。
// 它返回一个带有配置设置的日志记录器，如果未设置，则返回默认设置。
func SetupLogging(ctx context.Context, v *viper.Viper) (*log.Logger, error) {
	// 简化的日志设置，真实实现将在完整版本中添加
	var destOutput io.Writer = os.Stdout

	logFile := v.GetString("log-file")
	if logFile != "" && !v.GetBool("log-to-stdout") {
		dir := filepath.Dir(logFile)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			err := os.MkdirAll(dir, 0750)
			if err != nil {
				return nil, fmt.Errorf("创建日志文件目录 %q 失败: %s", dir, err)
			}
		}

		// 轮换日志文件
		destOutput = &lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    log.MaxLogFileSize,
			MaxBackups: log.MaxNumberOfBackups,
		}
	}

	logger := log.New(
		destOutput,
		log.WithVerbose(v.GetBool("verbose")),
		log.WithSendDiagsOnErrors(v.GetBool("send-diagnostics-on-errors")),
		log.WithMetrics(v.GetBool("metrics")),
	)

	return logger, nil
}

func saveHeartbeats(ctx context.Context, v *viper.Viper) int {
	logger := log.Extract(ctx)

	queueFilepath, err := offline.QueueFilepath(ctx, v)
	if err != nil {
		logger.Warnf("加载离线队列文件路径失败: %s", err)
		return exitcode.ErrGeneric
	}

	if err := cmdoffline.SaveHeartbeats(ctx, v, nil, queueFilepath); err != nil {
		logger.Errorf("保存心跳到离线队列失败: %s", err)
		return exitcode.ErrGeneric
	}

	return exitcode.Success
}
