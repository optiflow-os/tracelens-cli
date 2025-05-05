package today

import (
	"context"
	"fmt"
	"time"

	"github.com/optiflow-os/tracelens-cli/pkg/exitcode"
	"github.com/optiflow-os/tracelens-cli/pkg/log"

	"github.com/spf13/viper"
)

// Run 执行今日统计命令。
func Run(ctx context.Context, v *viper.Viper) (int, error) {
	logger := log.Extract(ctx)
	logger.Debugln("执行今日统计命令")

	// 获取是否隐藏类别
	hideCategories := v.GetBool("today-hide-categories")

	// 在真实实现中，这将从 API 获取今天的编码统计信息
	// 这里我们使用模拟数据进行演示
	totalSeconds := 12345 // 模拟值：今天的总编码时间（秒）

	// 格式化为小时和分钟
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60

	// 输出今天的统计信息
	currentDate := time.Now().Format("2006年01月02日")
	fmt.Printf("%s 的编码时间统计\n", currentDate)
	fmt.Printf("总计: %d 小时 %d 分钟\n", hours, minutes)

	// 如果不隐藏类别，则显示按类别划分的时间
	if !hideCategories {
		fmt.Println("\n按类别划分:")
		fmt.Printf("编码:     %d 小时 %d 分钟\n", hours/2, minutes/2)
		fmt.Printf("调试:     %d 小时 %d 分钟\n", hours/4, minutes/4)
		fmt.Printf("文档:     %d 小时 %d 分钟\n", hours/8, minutes/8)
		fmt.Printf("其他:     %d 小时 %d 分钟\n", hours/8, minutes/8)
	}

	return exitcode.Success, nil
}
