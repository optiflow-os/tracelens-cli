package configread

import (
	"context"
	"fmt"

	"github.com/optiflow-os/tracelens-cli/pkg/exitcode"
	"github.com/optiflow-os/tracelens-cli/pkg/log"
	"github.com/optiflow-os/tracelens-cli/pkg/vipertools"

	"github.com/spf13/viper"
)

// Run 执行配置读取命令。
func Run(ctx context.Context, v *viper.Viper) (int, error) {
	logger := log.Extract(ctx)
	logger.Debugln("执行配置读取命令")

	// 获取要读取的配置键
	key := v.GetString("config-read")
	if key == "" {
		logger.Errorf("没有提供配置键")
		return exitcode.ErrConfigFileRead, fmt.Errorf("没有提供配置键")
	}

	// 获取配置节
	section := v.GetString("config-section")
	if section != "" && section != "settings" {
		// 添加节前缀
		key = fmt.Sprintf("%s.%s", section, key)
	}

	// 读取配置值
	value := vipertools.GetString(v, key)

	// 输出配置值
	fmt.Println(value)
	logger.Debugf("已读取配置键 %s 的值：%s", key, value)

	return exitcode.Success, nil
}
