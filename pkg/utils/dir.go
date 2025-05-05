package utils

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/optiflow-os/tracelens-cli/pkg/log"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

// TLHomeType 是 TraceLens 主目录类型
type TLHomeType int

const (
	// TLHomeTypeUnknown 是未知的 WakaTime 主目录类型
	TLHomeTypeUnknown TLHomeType = iota
	// TLHomeTypeEnvVar 是来自环境变量的 WakaTime 主目录类型
	TLHomeTypeEnvVar
	// TLHomeTypeOSDir 是来自操作系统目录的 WakaTime 主目录类型
	TLHomeTypeOSDir
)

// TLHomeDir 返回当前用户的主目录
func TLHomeDir(ctx context.Context) (string, TLHomeType, error) {
	logger := log.Extract(ctx)

	home, exists := os.LookupEnv("TRACELENS_HOME")
	if exists && home != "" {
		home, err := homedir.Expand(home)
		if err == nil {
			return home, TLHomeTypeEnvVar, nil
		}

		logger.Warnf("failed to expand TRACELENS_HOME filepath: %s. It will try to get user home dir.", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		logger.Warnf("failed to get user home dir: %s", err)
	}

	if home != "" {
		return home, TLHomeTypeOSDir, nil
	}

	u, err := user.LookupId(strconv.Itoa(os.Getuid()))
	if err != nil {
		logger.Warnf("failed to user info by userid: %s", err)
	}

	if u.HomeDir != "" {
		return u.HomeDir, TLHomeTypeOSDir, nil
	}

	return "", TLHomeTypeUnknown, fmt.Errorf("could not determine tracelens home dir")
}

// TLResourcesDir 返回 ~/.tracelens/ 文件夹
func TLResourcesDir(ctx context.Context) (string, error) {
	home, hometype, err := TLHomeDir(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get user's home directory: %s", err)
	}

	switch hometype {
	case TLHomeTypeEnvVar:
		return home, nil
	default:
		return filepath.Join(home, defaultFolder), nil
	}
}

// FilePath 返回 wakatime 配置文件的路径
func FilePath(ctx context.Context, v *viper.Viper) (string, error) {
	configFilepath := GetString(v, "config")
	if configFilepath != "" {
		p, err := homedir.Expand(configFilepath)
		if err != nil {
			return "", fmt.Errorf("failed to expand config param: %s", err)
		}

		return p, nil
	}

	home, _, err := TLHomeDir(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get user's home directory: %s", err)
	}

	return filepath.Join(home, defaultFile), nil
}

// ImportFilePath 返回自定义 wakatime 配置文件的路径
// 它用于将 API 密钥保持在主文件夹之外，通常是为了避免备份敏感的 wakatime 配置文件
// https://github.com/optiflow-os/tracelens-cli/issues/464
func ImportFilePath(_ context.Context, v *viper.Viper) (string, error) {
	configFilepath := GetString(v, "settings.import_cfg")
	if configFilepath != "" {
		p, err := homedir.Expand(configFilepath)
		if err != nil {
			return "", fmt.Errorf("failed to expand settings.import_cfg param: %s", err)
		}

		return p, nil
	}

	return "", nil
}

// InternalFilePath 返回 wakatime 内部配置文件的路径，该文件包含
// 最后心跳时间戳和退避时间
func InternalFilePath(ctx context.Context, v *viper.Viper) (string, error) {
	configFilepath := GetString(v, "internal-config")
	if configFilepath != "" {
		p, err := homedir.Expand(configFilepath)
		if err != nil {
			return "", fmt.Errorf("failed to expand internal-config param: %s", err)
		}

		return p, nil
	}

	folder, err := TLResourcesDir(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get user's home directory: %s", err)
	}

	return filepath.Join(folder, defaultInternalFile), nil
}
