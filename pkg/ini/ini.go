package ini

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/optiflow-os/tracelens-cli/pkg/log"
	"github.com/optiflow-os/tracelens-cli/pkg/vipertools"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"gopkg.in/ini.v1"
)

// FilePath 返回 tracelens 配置文件的路径。
func FilePath(ctx context.Context, v *viper.Viper) (string, error) {
	logger := log.Extract(ctx)

	configFile := vipertools.GetString(v, "config")
	if configFile != "" {
		p, err := homedir.Expand(configFile)
		if err != nil {
			return "", fmt.Errorf("展开配置文件路径失败: %s", err)
		}

		logger.Debugf("使用配置文件路径: %s", p)

		return p, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %s", err)
	}

	return filepath.Join(home, ".tracelens.cfg"), nil
}

// ImportFilePath 返回自定义 tracelens 配置文件的路径。
// 它用于将 API 密钥保存在主文件夹之外，通常是为了避免备份敏感的 tracelens 配置文件。
func ImportFilePath(_ context.Context, v *viper.Viper) (string, error) {
	configFilepath := vipertools.GetString(v, "settings.import_cfg")
	if configFilepath != "" {
		p, err := homedir.Expand(configFilepath)
		if err != nil {
			return "", fmt.Errorf("展开 settings.import_cfg 参数失败: %s", err)
		}

		return p, nil
	}

	return "", nil
}

// InternalFilePath 返回 tracelens 内部配置文件的路径，该文件包含
// 最后一次心跳时间戳和退避时间。
func InternalFilePath(ctx context.Context, v *viper.Viper) (string, error) {
	logger := log.Extract(ctx)

	internalConfigFile := vipertools.GetString(v, "internal-config")
	if internalConfigFile != "" {
		p, err := homedir.Expand(internalConfigFile)
		if err != nil {
			return "", fmt.Errorf("展开内部配置文件路径失败: %s", err)
		}

		logger.Debugf("使用内部配置文件路径: %s", p)

		return p, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %s", err)
	}

	return filepath.Join(home, ".tracelens", "tracelens-internal.cfg"), nil
}

// ReadInConfig 将 tracelens 配置文件读入内存。
func ReadInConfig(v *viper.Viper, configFilePath string) error {
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return fmt.Errorf("文件不存在: %s", configFilePath)
	}

	v.SetConfigFile(configFilePath)

	if err := v.MergeInConfig(); err != nil {
		return fmt.Errorf("合并配置失败: %s", err)
	}

	return nil
}

// NewWriter 创建一个新的写入器实例。
func NewWriter(
	ctx context.Context,
	v *viper.Viper,
	filepathFn func(context.Context, *viper.Viper) (string, error),
) (*WriterConfig, error) {
	configFilepath, err := filepathFn(ctx, v)
	if err != nil {
		return nil, fmt.Errorf("获取文件路径时出错: %s", err)
	}

	logger := log.Extract(ctx)

	// 检查文件是否存在
	if !fileExists(configFilepath) {
		logger.Debugf("将创建缺失的配置文件 %q", configFilepath)

		dir := filepath.Dir(configFilepath)
		if err := os.MkdirAll(dir, 0750); err != nil {
			return nil, fmt.Errorf("创建目录失败: %s", err)
		}

		f, err := os.Create(configFilepath) // nolint:gosec
		if err != nil {
			return nil, fmt.Errorf("创建文件失败: %s", err)
		}

		if err = f.Close(); err != nil {
			return nil, fmt.Errorf("关闭文件失败: %s", err)
		}
	}

	iniFile, err := ini.LoadSources(ini.LoadOptions{
		AllowPythonMultilineValues: true,
	}, configFilepath)
	if err != nil {
		return nil, fmt.Errorf("加载配置文件失败: %s", err)
	}

	return &WriterConfig{
		ConfigFilepath: configFilepath,
		ini:            iniFile,
	}, nil
}

// WriterConfig 包含配置写入所需的数据。
type WriterConfig struct {
	ConfigFilepath string
	ini            *ini.File
}

// Write 保存配置到文件。
func (w *WriterConfig) Write(ctx context.Context, section string, keyValue map[string]string) error {
	logger := log.Extract(ctx)

	// 获取指定的段落，如果不存在则创建
	sec, err := w.ini.GetSection(section)
	if err != nil {
		// 创建一个新的段落
		sec, err = w.ini.NewSection(section)
		if err != nil {
			return fmt.Errorf("创建新段落失败: %s", err)
		}
	}

	// 应用键值对更改
	for k, v := range keyValue {
		sec.Key(k).SetValue(v)
	}

	// 保存文件
	if err := w.ini.SaveTo(w.ConfigFilepath); err != nil {
		return fmt.Errorf("保存配置文件失败: %s", err)
	}

	logger.Debugf("配置已写入 %s 段落: %v", section, keyValue)

	return nil
}

// fileExists 返回指定的文件是否存在并且可以访问
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}
