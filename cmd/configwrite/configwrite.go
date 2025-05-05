package configwrite

import (
	"context"
	"fmt"

	"github.com/optiflow-os/tracelens-cli/pkg/exitcode"
	"github.com/optiflow-os/tracelens-cli/pkg/ini"
	"github.com/optiflow-os/tracelens-cli/pkg/log"
	"github.com/optiflow-os/tracelens-cli/pkg/vipertools"

	"github.com/spf13/viper"
)

// Run 执行配置写入命令。
func Run(ctx context.Context, v *viper.Viper) (int, error) {
	logger := log.Extract(ctx)
	logger.Debugln("执行配置写入命令")

	// 获取要写入的键值对
	keyValues := vipertools.GetStringMapString(v, "config-write")
	if len(keyValues) == 0 {
		logger.Errorf("没有提供配置键值对")
		return exitcode.ErrConfigFileWrite, fmt.Errorf("没有提供配置键值对")
	}

	// 获取配置节
	section := v.GetString("config-section")

	// 创建配置文件写入器
	writer, err := ini.NewWriter(ctx, v, ini.FilePath)
	if err != nil {
		logger.Errorf("创建配置写入器失败: %s", err)
		return exitcode.ErrConfigFileWrite, fmt.Errorf("创建配置写入器失败: %s", err)
	}

	// 写入配置
	if err := writer.Write(ctx, section, keyValues); err != nil {
		logger.Errorf("写入配置失败: %s", err)
		return exitcode.ErrConfigFileWrite, fmt.Errorf("写入配置失败: %s", err)
	}

	// 输出写入的键值对信息
	logger.Debugf("已写入配置到节 [%s]: %v", section, keyValues)
	for key, value := range keyValues {
		fmt.Printf("已写入 %s = %s 到节 [%s]\n", key, value, section)
	}

	return exitcode.Success, nil
}
