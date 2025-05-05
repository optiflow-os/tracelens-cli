package fileexperts

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/optiflow-os/tracelens-cli/pkg/exitcode"
	"github.com/optiflow-os/tracelens-cli/pkg/log"

	"github.com/spf13/viper"
)

// FileExpert 表示文件专家信息。
type FileExpert struct {
	Name       string  `json:"name"`
	Email      string  `json:"email"`
	TotalTime  int     `json:"total_time"`
	Percentage float64 `json:"percentage"`
}

// Run 执行文件专家命令。
func Run(ctx context.Context, v *viper.Viper) (int, error) {
	logger := log.Extract(ctx)
	logger.Debugln("执行文件专家命令")

	// 获取实体路径
	entity := v.GetString("entity")
	if entity == "" {
		logger.Errorf("没有提供文件路径")
		return exitcode.ErrGeneric, fmt.Errorf("没有提供文件路径")
	}

	// 在真实实现中，这将从 API 获取文件专家信息
	// 这里我们使用模拟数据进行演示
	experts := []FileExpert{
		{
			Name:       "张三",
			Email:      "zhangsan@example.com",
			TotalTime:  45000, // 12.5 小时
			Percentage: 65.0,
		},
		{
			Name:       "李四",
			Email:      "lisi@example.com",
			TotalTime:  18000, // 5 小时
			Percentage: 25.0,
		},
		{
			Name:       "王五",
			Email:      "wangwu@example.com",
			TotalTime:  7200, // 2 小时
			Percentage: 10.0,
		},
	}

	// 确定输出格式
	outputFormat := v.GetString("output")

	// 根据输出格式输出结果
	switch outputFormat {
	case "json", "raw-json":
		data, err := json.MarshalIndent(experts, "", "  ")
		if err != nil {
			logger.Errorf("序列化专家数据失败: %s", err)
			return exitcode.ErrGeneric, fmt.Errorf("序列化专家数据失败: %s", err)
		}
		fmt.Println(string(data))
	default:
		// 以文本形式输出
		fmt.Printf("文件: %s 的专家\n\n", entity)
		fmt.Println("姓名\t\t邮箱\t\t\t\t时间\t\t百分比")
		fmt.Println("------------------------------------------------------------")

		for _, expert := range experts {
			hours := expert.TotalTime / 3600
			minutes := (expert.TotalTime % 3600) / 60

			fmt.Printf("%s\t\t%s\t\t%d小时%d分钟\t%.1f%%\n",
				expert.Name,
				expert.Email,
				hours,
				minutes,
				expert.Percentage)
		}
	}

	return exitcode.Success, nil
}
