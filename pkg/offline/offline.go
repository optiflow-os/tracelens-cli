package offline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"path/filepath"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/optiflow-os/tracelens-cli/pkg/api"
	"github.com/optiflow-os/tracelens-cli/pkg/heartbeat"
	"github.com/optiflow-os/tracelens-cli/pkg/log"
	"github.com/optiflow-os/tracelens-cli/pkg/utils"
	"github.com/spf13/viper"

	bolt "go.etcd.io/bbolt"
)

const (
	// dbFilename 是默认的 bolt 数据库文件名。
	dbFilename = "offline_heartbeats.bdb"
	// dbBucket 是标准 bolt 数据库桶名称。
	dbBucket = "heartbeats"
	// maxRequeueAttempts 定义了重新排队心跳的最大尝试次数，
	// 这些心跳无法成功发送到 WakaTime API。
	maxRequeueAttempts = 3
	// PrintMaxDefault 是默认情况下要打印的离线心跳的最大数量
	PrintMaxDefault = 10
	// RateLimitDefaultSeconds 是向 API 发送心跳之间的默认秒数
	// 如果没有经过足够的时间，心跳将保存到离线队列中。
	RateLimitDefaultSeconds = 120
	// SendLimit 是一次向 WakaTime API 发送的最大心跳数。
	SendLimit = 25
	// SyncMaxDefault 是从离线队列中同步的心跳数量的默认最大值，
	// 这些心跳将在向 API 发送心跳时被同步。
	SyncMaxDefault = 1000
)

// Noop 是一个无操作 API 客户端，由 offline.SaveHeartbeats 使用。
type Noop struct{}

// SendHeartbeats 总是返回一个错误。
func (Noop) SendHeartbeats(_ context.Context, _ []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
	return nil, api.Err{Err: errors.New("skip sending heartbeats and only save to offline db")}
}

// WithQueue 初始化并返回一个心跳处理选项，可以
// 在心跳处理管道中用于自动处理发送到 API 的心跳
// 失败情况。由于缺少或连接 API 失败、发送失败或 API 返回错误，
// 心跳将暂时存储在数据库中，并在下次使用 wakatime cli 时重试发送。
func WithQueue(filepath string) heartbeat.HandleOption {
	return func(next heartbeat.Handle) heartbeat.Handle {
		return func(ctx context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
			logger := log.Extract(ctx)
			logger.Debugf("execute offline queue with file %s", filepath)

			if len(hh) == 0 {
				logger.Debugln("abort execution, as there are no heartbeats ready for sending")

				return nil, nil
			}

			results, err := next(ctx, hh)
			if err != nil {
				logger.Debugf("pushing %d heartbeat(s) to queue after error: %s", len(hh), err)

				requeueErr := pushHeartbeatsWithRetry(ctx, filepath, hh)
				if requeueErr != nil {
					return nil, fmt.Errorf(
						"failed to push heartbeats to queue: %s",
						requeueErr,
					)
				}

				return nil, err
			}

			err = handleResults(ctx, filepath, results, hh)
			if err != nil {
				return nil, fmt.Errorf("failed to handle results: %s", err)
			}

			return results, nil
		}
	}
}

// QueueFilepath 返回离线队列数据库文件的路径。如果
// 无法检测到资源目录，默认为当前目录。
func QueueFilepath(ctx context.Context, v *viper.Viper) (string, error) {
	paramFile := utils.GetString(v, "offline-queue-file")
	if paramFile != "" {
		p, err := homedir.Expand(paramFile)
		if err != nil {
			return "", fmt.Errorf("failed expanding offline-queue-file param: %s", err)
		}

		return p, nil
	}

	folder, err := utils.TLResourcesDir(ctx)
	if err != nil {
		return dbFilename, fmt.Errorf("failed getting resource directory, defaulting to current directory: %s", err)
	}

	return filepath.Join(folder, dbFilename), nil
}

// WithSync 初始化并返回一个心跳处理选项，可以
// 用于心跳处理管道中，从离线队列中弹出心跳
// 并将心跳发送到 WakaTime API。
func WithSync(filepath string, syncLimit int) heartbeat.HandleOption {
	return func(next heartbeat.Handle) heartbeat.Handle {
		return func(ctx context.Context, _ []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
			logger := log.Extract(ctx)
			logger.Debugf("execute offline sync with file %s", filepath)

			err := Sync(ctx, filepath, syncLimit)(next)
			if err != nil {
				return nil, fmt.Errorf("failed to sync offline heartbeats: %s", err)
			}

			return nil, nil
		}
	}
}

// Sync 返回一个函数，用于将队列中的心跳发送到 WakaTime API。
func Sync(ctx context.Context, filepath string, syncLimit int) func(next heartbeat.Handle) error {
	return func(next heartbeat.Handle) error {
		var (
			alreadySent int
			run         int
		)

		if syncLimit == 0 {
			syncLimit = math.MaxInt32
		}

		logger := log.Extract(ctx)

		for {
			run++

			if alreadySent >= syncLimit {
				break
			}

			var num = SendLimit

			if alreadySent+SendLimit > syncLimit {
				num = syncLimit - alreadySent
				alreadySent += num
			}

			hh, err := popHeartbeats(ctx, filepath, num)
			if err != nil {
				return fmt.Errorf("failed to fetch heartbeat from offline queue: %s", err)
			}

			if len(hh) == 0 {
				logger.Debugln("no queued heartbeats ready for sending")

				break
			}

			logger.Debugf("send %d heartbeats on sync run %d", len(hh), run)

			results, err := next(ctx, hh)
			if err != nil {
				requeueErr := pushHeartbeatsWithRetry(ctx, filepath, hh)
				if requeueErr != nil {
					logger.Warnf("failed to push heartbeats to queue after api error: %s", requeueErr)
				}

				return err
			}

			err = handleResults(ctx, filepath, results, hh)
			if err != nil {
				return fmt.Errorf("failed to handle heartbeats api results: %s", err)
			}
		}

		return nil
	}
}

func handleResults(ctx context.Context, filepath string, results []heartbeat.Result, hh []heartbeat.Heartbeat) error {
	var (
		err               error
		withInvalidStatus []heartbeat.Heartbeat
	)

	logger := log.Extract(ctx)

	// 将无效结果状态码的心跳推送到队列
	for n, result := range results {
		if n >= len(hh) {
			logger.Warnln("results from api not matching heartbeats sent")
			break
		}

		if result.Status == http.StatusBadRequest {
			serialized, jsonErr := json.Marshal(result.Heartbeat)
			if jsonErr != nil {
				logger.Warnf(
					"failed to json marshal heartbeat: %s. heartbeat: %#v",
					jsonErr,
					result.Heartbeat,
				)
			}

			logger.Debugf("heartbeat result status bad request: %s", string(serialized))

			continue
		}

		if result.Status < http.StatusOK || result.Status > 299 {
			withInvalidStatus = append(withInvalidStatus, hh[n])
		}
	}

	if len(withInvalidStatus) > 0 {
		logger.Debugf("pushing %d heartbeat(s) with invalid result to queue", len(withInvalidStatus))

		err = pushHeartbeatsWithRetry(ctx, filepath, withInvalidStatus)
		if err != nil {
			logger.Warnf("failed to push heartbeats with invalid status to queue: %s", err)
		}
	}

	// 处理剩余的心跳
	leftovers := len(hh) - len(results)
	if leftovers > 0 {
		logger.Warnf("missing %d results from api.", leftovers)

		start := len(hh) - leftovers

		err = pushHeartbeatsWithRetry(ctx, filepath, hh[start:])
		if err != nil {
			logger.Warnf("failed to push leftover heartbeats to queue: %s", err)
		}
	}

	return err
}

func popHeartbeats(ctx context.Context, filepath string, limit int) ([]heartbeat.Heartbeat, error) {
	db, close, err := openDB(ctx, filepath)
	if err != nil {
		return nil, err
	}

	defer close()

	tx, err := db.Begin(true)
	if err != nil {
		return nil, fmt.Errorf("failed to start db transaction: %s", err)
	}

	queue := NewQueue(tx)
	logger := log.Extract(ctx)

	queued, err := queue.PopMany(limit)
	if err != nil {
		errrb := tx.Rollback()
		if errrb != nil {
			logger.Errorf("failed to rollback transaction: %s", errrb)
		}

		return nil, fmt.Errorf("failed to pop heartbeat(s) from queue: %s", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit db transaction: %s", err)
	}

	return queued, nil
}

func pushHeartbeatsWithRetry(ctx context.Context, filepath string, hh []heartbeat.Heartbeat) error {
	var (
		count int
		err   error
	)

	logger := log.Extract(ctx)

	for {
		if count >= maxRequeueAttempts {
			serialized, jsonErr := json.Marshal(hh)
			if jsonErr != nil {
				logger.Warnf("failed to json marshal heartbeats: %s. heartbeats: %#v", jsonErr, hh)
			}

			return fmt.Errorf(
				"abort requeuing after %d unsuccessful attempts: %s. heartbeats: %s",
				count,
				err,
				string(serialized),
			)
		}

		err = pushHeartbeats(ctx, filepath, hh)
		if err != nil {
			count++

			sleepSeconds := math.Pow(2, float64(count))

			time.Sleep(time.Duration(sleepSeconds) * time.Second)

			continue
		}

		break
	}

	return nil
}

func pushHeartbeats(ctx context.Context, filepath string, hh []heartbeat.Heartbeat) error {
	db, close, err := openDB(ctx, filepath)
	if err != nil {
		return err
	}

	defer close()

	tx, err := db.Begin(true)
	if err != nil {
		return fmt.Errorf("failed to start db transaction: %s", err)
	}

	queue := NewQueue(tx)

	err = queue.PushMany(hh)
	if err != nil {
		return fmt.Errorf("failed to push heartbeat(s) to queue: %s", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit db transaction: %s", err)
	}

	return nil
}

// CountHeartbeats 返回离线数据库中心跳的总数。
func CountHeartbeats(ctx context.Context, filepath string) (int, error) {
	db, close, err := openDB(ctx, filepath)
	if err != nil {
		return 0, err
	}

	defer close()

	tx, err := db.Begin(true)
	if err != nil {
		return 0, fmt.Errorf("failed to start db transaction: %s", err)
	}

	logger := log.Extract(ctx)

	defer func() {
		err := tx.Rollback()
		if err != nil {
			logger.Errorf("failed to rollback transaction: %s", err)
		}
	}()

	queue := NewQueue(tx)

	count, err := queue.Count()
	if err != nil {
		return 0, fmt.Errorf("failed to count heartbeats: %s", err)
	}

	return count, nil
}

// ReadHeartbeats 读取离线数据库中的指定心跳。
func ReadHeartbeats(ctx context.Context, filepath string, limit int) ([]heartbeat.Heartbeat, error) {
	db, close, err := openDB(ctx, filepath)
	if err != nil {
		return nil, err
	}

	defer close()

	tx, err := db.Begin(true)
	if err != nil {
		return nil, fmt.Errorf("failed to start db transaction: %s", err)
	}

	queue := NewQueue(tx)
	logger := log.Extract(ctx)

	hh, err := queue.ReadMany(limit)
	if err != nil {
		logger.Errorf("failed to read offline heartbeats: %s", err)

		_ = tx.Rollback()

		return nil, err
	}

	err = tx.Rollback()
	if err != nil {
		logger.Warnf("failed to rollback transaction: %s", err)
	}

	return hh, nil
}

// openDB 打开与离线数据库的连接。
// 它返回指向 bolt.DB 的指针、关闭连接的函数和一个错误。
// 尽管应该避免使用命名参数，但这个函数使用它们来在延迟函数内部访问并设置错误。
func openDB(ctx context.Context, filepath string) (db *bolt.DB, _ func(), err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ErrOpenDB{Err: fmt.Errorf("panicked: %v", r)}
		}
	}()

	db, err = bolt.Open(filepath, 0644, &bolt.Options{Timeout: 30 * time.Second})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open db file: %s", err)
	}

	return db, func() {
		logger := log.Extract(ctx)

		// 从关闭数据库时的 panic 中恢复
		defer func() {
			if r := recover(); r != nil {
				logger.Warnf("panicked: failed to close db file: %v", r)
			}
		}()

		if err := db.Close(); err != nil {
			logger.Debugf("failed to close db file: %s", err)
		}
	}, err
}

// Queue 是一个数据库客户端，用于在 bolt 数据库中临时存储心跳，以防无法
// 将心跳发送到 wakatime api。事务处理留给用户
// 通过传入的事务。
type Queue struct {
	Bucket string
	tx     *bolt.Tx
}

// NewQueue 创建一个新的 Queue 实例。
func NewQueue(tx *bolt.Tx) *Queue {
	return &Queue{
		Bucket: dbBucket,
		tx:     tx,
	}
}

// Count 返回离线数据库中心跳的总数。
func (q *Queue) Count() (int, error) {
	b, err := q.tx.CreateBucketIfNotExists([]byte(q.Bucket))
	if err != nil {
		return 0, fmt.Errorf("failed to create/load bucket: %s", err)
	}

	return b.Stats().KeyN, nil
}

// PopMany 从数据库中检索具有指定 ID 的心跳。
func (q *Queue) PopMany(limit int) ([]heartbeat.Heartbeat, error) {
	b, err := q.tx.CreateBucketIfNotExists([]byte(q.Bucket))
	if err != nil {
		return nil, fmt.Errorf("failed to create/load bucket: %s", err)
	}

	var (
		heartbeats []heartbeat.Heartbeat
		ids        []string
	)

	// 加载值
	c := b.Cursor()

	for key, value := c.First(); key != nil; key, value = c.Next() {
		if len(heartbeats) >= limit {
			break
		}

		var h heartbeat.Heartbeat

		err := json.Unmarshal(value, &h)
		if err != nil {
			return nil, fmt.Errorf("failed to json unmarshal heartbeat data: %s", err)
		}

		heartbeats = append(heartbeats, h)
		ids = append(ids, string(key))
	}

	for _, id := range ids {
		if err := b.Delete([]byte(id)); err != nil {
			return nil, fmt.Errorf("failed to delete key %q: %s", id, err)
		}
	}

	return heartbeats, nil
}

// PushMany 将提供的心跳存储在数据库中。
func (q *Queue) PushMany(hh []heartbeat.Heartbeat) error {
	b, err := q.tx.CreateBucketIfNotExists([]byte(q.Bucket))
	if err != nil {
		return fmt.Errorf("failed to create/load bucket: %s", err)
	}

	for _, h := range hh {
		data, err := json.Marshal(h)
		if err != nil {
			return fmt.Errorf("failed to json marshal heartbeat: %s", err)
		}

		err = b.Put([]byte(h.ID()), data)
		if err != nil {
			return fmt.Errorf("failed to store heartbeat with id %q: %s", h.ID(), err)
		}
	}

	return nil
}

// ReadMany 从数据库中读取心跳而不删除它们。
func (q *Queue) ReadMany(limit int) ([]heartbeat.Heartbeat, error) {
	b, err := q.tx.CreateBucketIfNotExists([]byte(q.Bucket))
	if err != nil {
		return nil, fmt.Errorf("failed to create/load bucket: %s", err)
	}

	var heartbeats = make([]heartbeat.Heartbeat, 0)

	// 加载值
	c := b.Cursor()

	for key, value := c.First(); key != nil; key, value = c.Next() {
		if len(heartbeats) >= limit {
			break
		}

		var h heartbeat.Heartbeat

		err := json.Unmarshal(value, &h)
		if err != nil {
			return nil, fmt.Errorf("failed to json unmarshal heartbeat data: %s", err)
		}

		heartbeats = append(heartbeats, h)
	}

	return heartbeats, nil
}
