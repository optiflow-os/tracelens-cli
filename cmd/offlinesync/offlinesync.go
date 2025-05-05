package offlinesync

import (
	"context"
	"fmt"

	"github.com/optiflow-os/tracelens-cli/pkg/exitcode"
	"github.com/optiflow-os/tracelens-cli/pkg/log"
	"github.com/optiflow-os/tracelens-cli/pkg/offline"

	"github.com/spf13/viper"
)

// RunWithoutRateLimiting 运行离线同步命令，不考虑速率限制。
func RunWithoutRateLimiting(ctx context.Context, v *viper.Viper) (int, error) {
	logger := log.Extract(ctx)
	logger.Debugln("运行离线同步命令 (无速率限制)")

	// 获取要同步的心跳数量
	syncCount := v.GetInt("sync-offline-activity")

	// 在真实实现中，这将从离线数据库中获取心跳并同步到 API
	// 这里我们只是记录同步行为
	logger.Infof("将同步 %d 个离线心跳（最大值）", syncCount)
	fmt.Printf("同步离线活动：%d 个心跳（最大值）\n", syncCount)

	return exitcode.Success, nil
}

// RunWithRateLimiting 运行受速率限制的离线同步命令。
func RunWithRateLimiting(ctx context.Context, v *viper.Viper) (int, error) {
	logger := log.Extract(ctx)
	logger.Debugln("运行离线同步命令 (有速率限制)")

	// 获取速率限制秒数
	rateLimitSecs := v.GetInt("heartbeat-rate-limit-seconds")
	if rateLimitSecs == 0 {
		// 如果速率限制为 0，则使用无速率限制的版本
		return RunWithoutRateLimiting(ctx, v)
	}

	// 检查同步数量，默认使用 SyncMaxDefault
	syncCount := offline.SyncMaxDefault
	if v.IsSet("sync-offline-activity") {
		syncCount = v.GetInt("sync-offline-activity")
	}

	// 在真实实现中，这将以受控速率从离线数据库同步心跳
	// 这里我们只是记录同步行为
	logger.Infof("将同步 %d 个离线心跳（受 %d 秒的速率限制）", syncCount, rateLimitSecs)
	fmt.Printf("同步离线活动：%d 个心跳（受 %d 秒的速率限制）\n", syncCount, rateLimitSecs)

	return exitcode.Success, nil
}
