package offline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/optiflow-os/tracelens-cli/pkg/log"
	"github.com/spf13/viper"
)

// Heartbeat 表示一个心跳数据结构。
type Heartbeat struct {
	Entity        string  `json:"entity"`
	Type          string  `json:"type"`
	Time          float64 `json:"time"`
	Project       string  `json:"project,omitempty"`
	Language      string  `json:"language,omitempty"`
	IsWrite       bool    `json:"is_write,omitempty"`
	LineAdditions int     `json:"line_additions,omitempty"`
	LineDeletions int     `json:"line_deletions,omitempty"`
}

// SaveHeartbeats 将心跳保存到离线队列。
func SaveHeartbeats(ctx context.Context, v *viper.Viper, heartbeats []Heartbeat, queueFilepath string) error {
	logger := log.Extract(ctx)
	logger.Debugf("保存心跳到离线队列: %s", queueFilepath)

	// 确保目录存在
	dir := filepath.Dir(queueFilepath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("创建目录失败: %s", err)
	}

	// 在实际实现中，这里会将心跳存储到 bolt 数据库
	// 为简化起见，我们只将其写入 JSON 文件
	if len(heartbeats) > 0 {
		data, err := json.Marshal(heartbeats)
		if err != nil {
			return fmt.Errorf("序列化心跳失败: %s", err)
		}

		// 以追加模式打开文件
		f, err := os.OpenFile(queueFilepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("打开队列文件失败: %s", err)
		}
		defer f.Close()

		// 写入数据
		if _, err := f.Write(data); err != nil {
			return fmt.Errorf("写入心跳到队列文件失败: %s", err)
		}

		logger.Debugf("成功保存 %d 个心跳到离线队列", len(heartbeats))
	} else {
		logger.Debugln("没有心跳需要保存")
	}

	return nil
}
