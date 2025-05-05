package utils

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/optiflow-os/tracelens-cli/pkg/log"

	"github.com/juju/mutex"
	"github.com/spf13/viper"
	"gopkg.in/ini.v1"
)

const (
	defaultFolder = ".wakatime"
	// defaultFile 是默认 wakatime 配置文件的名称
	defaultFile = ".wakatime.cfg"
	// defaultInternalFile 是默认 wakatime 内部配置文件的名称
	defaultInternalFile = "wakatime-internal.cfg"
	// DateFormat 是配置文件中日期的默认格式
	DateFormat = time.RFC3339
	// defaultTimeout 是获取锁的默认超时时间
	defaultTimeout = time.Second * 5
)

// Writer 定义了写入配置文件的方法
type Writer interface {
	Write(ctx context.Context, section string, keyValue map[string]string) error
}

// WriterConfig 存储写入配置文件所需的配置
type WriterConfig struct {
	ConfigFilepath string
	File           *ini.File
}

// NewWriter 创建一个新的写入器实例
func NewWriter(
	ctx context.Context,
	v *viper.Viper,
	filepathFn func(ctx context.Context, v *viper.Viper) (string, error),
) (*WriterConfig, error) {
	configFilepath, err := filepathFn(ctx, v)
	if err != nil {
		return nil, fmt.Errorf("error getting filepath: %s", err)
	}

	logger := log.Extract(ctx)

	// 检查文件是否存在
	if !fileExists(configFilepath) {
		logger.Debugf("it will create missing config file %q", configFilepath)

		f, err := os.Create(configFilepath) // nolint:gosec
		if err != nil {
			return nil, fmt.Errorf("failed creating file: %s", err)
		}

		if err = f.Close(); err != nil {
			return nil, fmt.Errorf("failed to close file: %s", err)
		}
	}

	ini, err := ini.LoadSources(ini.LoadOptions{
		AllowPythonMultilineValues: true,
		SkipUnrecognizableLines:    true,
	}, configFilepath)
	if err != nil {
		return nil, fmt.Errorf("error loading config file: %s", err)
	}

	return &WriterConfig{
		ConfigFilepath: configFilepath,
		File:           ini,
	}, nil
}

// Write 将键值对持久化到磁盘
func (w *WriterConfig) Write(ctx context.Context, section string, keyValue map[string]string) error {
	logger := log.Extract(ctx)

	if w.File == nil || w.ConfigFilepath == "" {
		return errors.New("got undefined wakatime config file instance")
	}

	for key, value := range keyValue {
		// 防止写入空字符
		key = strings.ReplaceAll(key, "\x00", "")
		value = strings.ReplaceAll(value, "\x00", "")

		w.File.Section(section).Key(key).SetValue(value)
	}

	releaser, err := mutex.Acquire(mutex.Spec{
		Name:    "wakatime-cli-config-mutex",
		Delay:   time.Millisecond,
		Timeout: defaultTimeout,
		Clock:   &mutexClock{delay: time.Millisecond},
	})
	if err != nil {
		logger.Debugf("failed to acquire mutex: %s", err)
	}

	defer func() {
		if releaser != nil {
			releaser.Release()
		}
	}()

	if err := w.File.SaveTo(w.ConfigFilepath); err != nil {
		return fmt.Errorf("error saving wakatime config: %s", err)
	}

	return nil
}

// ReadInConfig 将 wakatime 配置文件读入内存
func ReadInConfig(v *viper.Viper, configFilePath string) error {
	v.SetConfigType("ini")
	v.SetConfigFile(configFilePath)

	if err := v.MergeInConfig(); err != nil {
		return fmt.Errorf("failed to merge config file: %s", err)
	}

	return nil
}

// mutexClock 用于实现 mutex.Clock 接口
type mutexClock struct {
	delay time.Duration
}

func (mc *mutexClock) After(time.Duration) <-chan time.Time {
	return time.After(mc.delay)
}

func (*mutexClock) Now() time.Time {
	return time.Now()
}

// fileExists 检查文件或目录是否存在
func fileExists(fp string) bool {
	_, err := os.Stat(fp)
	return err == nil || os.IsExist(err)
}
