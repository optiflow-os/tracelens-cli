package todaygoal

import (
	"context"
	"fmt"
	"time"

	"github.com/optiflow-os/tracelens-cli/pkg/exitcode"
	"github.com/optiflow-os/tracelens-cli/pkg/log"

	"github.com/spf13/viper"
)

// Run 执行今日目标命令。
func Run(ctx context.Context, v *viper.Viper) (int, error) {
	logger := log.Extract(ctx)
	logger.Debugln("执行今日目标命令")

	// 获取目标 ID
	goalID := v.GetString("today-goal")
	if goalID == "" {
		logger.Errorf("没有提供目标 ID")
		return exitcode.ErrGeneric, fmt.Errorf("没有提供目标 ID")
	}

	// 在真实实现中，这将从 API 获取目标进度
	// 这里我们使用模拟数据进行演示
	totalSecondsToday := 10800 // 模拟值：今天的编码时间（3小时）
	goalSeconds := 14400       // 模拟值：目标编码时间（4小时）

	// 计算完成百分比
	percentComplete := float64(totalSecondsToday) / float64(goalSeconds) * 100

	// 计算剩余时间
	remainingSeconds := goalSeconds - totalSecondsToday
	if remainingSeconds < 0 {
		remainingSeconds = 0
	}

	// 格式化为小时和分钟
	todayHours := totalSecondsToday / 3600
	todayMinutes := (totalSecondsToday % 3600) / 60

	remainingHours := remainingSeconds / 3600
	remainingMinutes := (remainingSeconds % 3600) / 60

	goalHours := goalSeconds / 3600
	goalMinutes := (goalSeconds % 3600) / 60

	// 输出目标进度信息
	currentDate := time.Now().Format("2006年01月02日")
	fmt.Printf("%s 的目标进度 (ID: %s)\n", currentDate, goalID)
	fmt.Printf("目标: %d 小时 %d 分钟\n", goalHours, goalMinutes)
	fmt.Printf("已完成: %d 小时 %d 分钟 (%.1f%%)\n", todayHours, todayMinutes, percentComplete)

	if remainingSeconds > 0 {
		fmt.Printf("剩余: %d 小时 %d 分钟\n", remainingHours, remainingMinutes)
	} else {
		fmt.Println("目标已完成！")
	}

	return exitcode.Success, nil
}
