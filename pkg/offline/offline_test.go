package offline_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/optiflow-os/tracelens-cli/pkg/heartbeat"
	"github.com/optiflow-os/tracelens-cli/pkg/ini"
	"github.com/optiflow-os/tracelens-cli/pkg/offline"
	"github.com/spf13/viper"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func TestQueueFilepath(t *testing.T) {
	ctx := context.Background()

	tests := map[string]struct {
		EnvVar string
	}{
		"default": {},
		"env_trailing_slash": {
			EnvVar: "~/path2/",
		},
		"env_without_trailing_slash": {
			EnvVar: "~/path2",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv("WAKATIME_HOME", test.EnvVar)

			folder, err := ini.WakaResourcesDir(ctx)
			require.NoError(t, err)

			v := viper.New()
			queueFilepath, err := offline.QueueFilepath(ctx, v)
			require.NoError(t, err)

			expected := filepath.Join(folder, "offline_heartbeats.bdb")

			assert.Equal(t, expected, queueFilepath)
		})
	}
}

func TestWithQueue(t *testing.T) {
	// setup
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	defer f.Close()

	db, err := bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	dataJs, err := os.ReadFile("testdata/heartbeat_js.json")
	require.NoError(t, err)

	insertHeartbeatRecords(t, db, "heartbeats", []heartbeatRecord{
		{
			ID:        "1592868394.084354-file-building-wakatime-todaygoal-/tmp/main.js-false",
			Heartbeat: string(dataJs),
		},
	})

	err = db.Close()
	require.NoError(t, err)

	opt := offline.WithQueue(f.Name())

	handle := opt(func(_ context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
		assert.Len(t, hh, 2)
		assert.Contains(t, hh, testHeartbeats()[0])
		assert.Contains(t, hh, testHeartbeats()[1])

		return []heartbeat.Result{
			{
				Status:    http.StatusCreated,
				Heartbeat: testHeartbeats()[0],
			},
			{
				Status:    http.StatusCreated,
				Heartbeat: testHeartbeats()[1],
			},
		}, nil
	})

	// run
	results, err := handle(context.Background(), []heartbeat.Heartbeat{
		testHeartbeats()[0],
		testHeartbeats()[1],
	})
	require.NoError(t, err)

	// check
	assert.Equal(t, []heartbeat.Result{
		{
			Status:    http.StatusCreated,
			Heartbeat: testHeartbeats()[0],
		},
		{
			Status:    http.StatusCreated,
			Heartbeat: testHeartbeats()[1],
		},
	}, results)

	db, err = bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	var stored []heartbeatRecord

	err = db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("heartbeats")).Cursor()

		for key, value := c.First(); key != nil; key, value = c.Next() {
			stored = append(stored, heartbeatRecord{
				ID:        string(key),
				Heartbeat: string(value),
			})
		}

		return nil
	})
	require.NoError(t, err)

	err = db.Close()
	require.NoError(t, err)

	require.Len(t, stored, 1)

	assert.Equal(t, "1592868394.084354-file-building-wakatime-todaygoal-/tmp/main.js-false", stored[0].ID)
	assert.JSONEq(t, string(dataJs), stored[0].Heartbeat)
}

func TestWithQueue_NoHeartbeats(t *testing.T) {
	// setup
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	defer f.Close()

	opt := offline.WithQueue(f.Name())

	handle := opt(func(_ context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
		assert.Len(t, hh, 0)

		return []heartbeat.Result{}, nil
	})

	// run
	results, err := handle(context.Background(), []heartbeat.Heartbeat{})
	require.NoError(t, err)

	// check
	assert.Nil(t, results)
}

func TestWithQueue_ApiError(t *testing.T) {
	// setup
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	defer f.Close()

	opt := offline.WithQueue(f.Name())

	handle := opt(func(_ context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
		assert.Equal(t, hh, []heartbeat.Heartbeat{
			testHeartbeats()[0],
			testHeartbeats()[1],
		})

		return []heartbeat.Result{}, errors.New("error")
	})

	// run
	_, err = handle(context.Background(), []heartbeat.Heartbeat{
		testHeartbeats()[0],
		testHeartbeats()[1],
	})
	require.Error(t, err)

	// check
	db, err := bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	var stored []heartbeatRecord

	err = db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("heartbeats")).Cursor()

		for key, value := c.First(); key != nil; key, value = c.Next() {
			stored = append(stored, heartbeatRecord{
				ID:        string(key),
				Heartbeat: string(value),
			})
		}

		return nil
	})
	require.NoError(t, err)

	err = db.Close()
	require.NoError(t, err)

	dataGo, err := os.ReadFile("testdata/heartbeat_go.json")
	require.NoError(t, err)

	dataPy, err := os.ReadFile("testdata/heartbeat_py.json")
	require.NoError(t, err)

	require.Len(t, stored, 2)

	assert.Equal(t, "1592868367.219124-12-file-coding-wakatime-cli-heartbeat-/tmp/main.go-true", stored[0].ID)
	assert.JSONEq(t, string(dataGo), stored[0].Heartbeat)

	assert.Equal(t, "1592868386.079084-13-file-debugging-wakatime-summary-/tmp/main.py-false", stored[1].ID)
	assert.JSONEq(t, string(dataPy), stored[1].Heartbeat)
}

func TestWithQueue_InvalidResults(t *testing.T) {
	// setup
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	defer f.Close()

	opt := offline.WithQueue(f.Name())

	handle := opt(func(_ context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
		assert.Equal(t, hh, testHeartbeats())

		return []heartbeat.Result{
			{
				Status:    201,
				Heartbeat: testHeartbeats()[0],
			},
			{
				Status:    500,
				Heartbeat: testHeartbeats()[1],
			},
			{
				Status: 429,
				Errors: []string{"Too many heartbeats"},
			},
		}, nil
	})

	// run
	results, err := handle(context.Background(), testHeartbeats())
	require.NoError(t, err)

	// check
	assert.Equal(t, []heartbeat.Result{
		{
			Status:    201,
			Heartbeat: testHeartbeats()[0],
		},
		{
			Status:    500,
			Heartbeat: testHeartbeats()[1],
		},
		{
			Status: 429,
			Errors: []string{"Too many heartbeats"},
		},
	}, results)

	// check db
	db, err := bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	var stored []heartbeatRecord

	err = db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("heartbeats")).Cursor()

		for key, value := c.First(); key != nil; key, value = c.Next() {
			stored = append(stored, heartbeatRecord{
				ID:        string(key),
				Heartbeat: string(value),
			})
		}

		return nil
	})
	require.NoError(t, err)

	err = db.Close()
	require.NoError(t, err)

	dataPy, err := os.ReadFile("testdata/heartbeat_py.json")
	require.NoError(t, err)

	dataJs, err := os.ReadFile("testdata/heartbeat_js.json")
	require.NoError(t, err)

	assert.Len(t, stored, 2)

	assert.Equal(t, "1592868386.079084-13-file-debugging-wakatime-summary-/tmp/main.py-false", stored[0].ID)
	assert.JSONEq(t, string(dataPy), stored[0].Heartbeat)

	assert.Equal(t, "1592868394.084354-14-file-building-wakatime-todaygoal-/tmp/main.js-false", stored[1].ID)
	assert.JSONEq(t, string(dataJs), stored[1].Heartbeat)
}

func TestWithQueue_HandleLeftovers(t *testing.T) {
	// setup
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	defer f.Close()

	opt := offline.WithQueue(f.Name())

	handle := opt(func(_ context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
		assert.Equal(t, hh, testHeartbeats())

		return []heartbeat.Result{
			{
				Status:    201,
				Heartbeat: testHeartbeats()[0],
			},
		}, nil
	})

	// run
	results, err := handle(context.Background(), testHeartbeats())
	require.NoError(t, err)

	// check
	assert.Equal(t, []heartbeat.Result{
		{
			Status:    201,
			Heartbeat: testHeartbeats()[0],
		},
	}, results)

	db, err := bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	var stored []heartbeatRecord

	err = db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("heartbeats")).Cursor()

		for key, value := c.First(); key != nil; key, value = c.Next() {
			stored = append(stored, heartbeatRecord{
				ID:        string(key),
				Heartbeat: string(value),
			})
		}

		return nil
	})
	require.NoError(t, err)

	err = db.Close()
	require.NoError(t, err)

	dataPy, err := os.ReadFile("testdata/heartbeat_py.json")
	require.NoError(t, err)

	dataJs, err := os.ReadFile("testdata/heartbeat_js.json")
	require.NoError(t, err)

	require.Len(t, stored, 2)

	assert.Equal(t, "1592868386.079084-13-file-debugging-wakatime-summary-/tmp/main.py-false", stored[0].ID)
	assert.JSONEq(t, string(dataPy), stored[0].Heartbeat)

	assert.Equal(t, "1592868394.084354-14-file-building-wakatime-todaygoal-/tmp/main.js-false", stored[1].ID)
	assert.JSONEq(t, string(dataJs), stored[1].Heartbeat)
}

func TestWithSync(t *testing.T) {
	// setup
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	defer f.Close()

	db, err := bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	dataGo, err := os.ReadFile("testdata/heartbeat_go.json")
	require.NoError(t, err)

	dataPy, err := os.ReadFile("testdata/heartbeat_py.json")
	require.NoError(t, err)

	insertHeartbeatRecords(t, db, "heartbeats", []heartbeatRecord{
		{
			ID:        "1592868367.219124-12-file-coding-wakatime-cli-heartbeat-/tmp/main.go-true",
			Heartbeat: string(dataGo),
		},
		{
			ID:        "1592868386.079084-13-file-debugging-wakatime-summary-/tmp/main.py-false",
			Heartbeat: string(dataPy),
		},
	})

	err = db.Close()
	require.NoError(t, err)

	opt := offline.WithSync(f.Name(), offline.SyncMaxDefault)

	handle := opt(func(_ context.Context, _ []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
		return []heartbeat.Result{
			{
				Status:    http.StatusCreated,
				Heartbeat: testHeartbeats()[0],
			},
			{
				Status:    http.StatusCreated,
				Heartbeat: testHeartbeats()[1],
			},
		}, nil
	})

	// run
	results, err := handle(context.Background(), nil)
	require.NoError(t, err)

	// check
	assert.Nil(t, results)

	// check db
	db, err = bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	var stored []heartbeatRecord

	err = db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("heartbeats")).Cursor()

		for key, value := c.First(); key != nil; key, value = c.Next() {
			stored = append(stored, heartbeatRecord{
				ID:        string(key),
				Heartbeat: string(value),
			})
		}

		return nil
	})
	require.NoError(t, err)

	err = db.Close()
	require.NoError(t, err)

	require.Len(t, stored, 0)
}

func TestSync_MultipleRequests(t *testing.T) {
	// setup
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	defer f.Close()

	db, err := bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	dataGo, err := os.ReadFile("testdata/heartbeat_go.json")
	require.NoError(t, err)

	for i := 0; i < 26; i++ {
		insertHeartbeatRecord(t, db, "heartbeats", heartbeatRecord{
			ID:        strconv.Itoa(i) + "1592868367.219124-file-coding-wakatime-cli-heartbeat-/tmp/main.go-true",
			Heartbeat: string(dataGo),
		})
	}

	err = db.Close()
	require.NoError(t, err)

	syncFn := offline.Sync(context.Background(), f.Name(), 1000)

	var numCalls int

	// run
	err = syncFn(func(_ context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
		numCalls++

		// first request
		if numCalls == 1 {
			assert.Len(t, hh, 25)

			result := heartbeat.Result{
				Status:    http.StatusCreated,
				Heartbeat: testHeartbeats()[0],
			}

			return []heartbeat.Result{
				result, result, result, result, result,
				result, result, result, result, result,
				result, result, result, result, result,
				result, result, result, result, result,
				result, result, result, result, result,
				result, result,
			}, nil
		}

		// second request
		assert.Len(t, hh, 1)

		results := []heartbeat.Result{
			{
				Status:    http.StatusCreated,
				Heartbeat: testHeartbeats()[0],
			},
		}

		return results, nil
	})
	require.NoError(t, err)

	// check db
	db, err = bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	var stored []heartbeatRecord

	err = db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("heartbeats")).Cursor()

		for key, value := c.First(); key != nil; key, value = c.Next() {
			stored = append(stored, heartbeatRecord{
				ID:        string(key),
				Heartbeat: string(value),
			})
		}

		return nil
	})
	require.NoError(t, err)

	err = db.Close()
	require.NoError(t, err)

	require.Len(t, stored, 0)

	assert.Eventually(t, func() bool { return numCalls == 2 }, time.Second, 50*time.Millisecond)
}

func TestSync_APIError(t *testing.T) {
	// setup
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	defer f.Close()

	db, err := bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	dataGo, err := os.ReadFile("testdata/heartbeat_go.json")
	require.NoError(t, err)

	dataPy, err := os.ReadFile("testdata/heartbeat_py.json")
	require.NoError(t, err)

	insertHeartbeatRecords(t, db, "heartbeats", []heartbeatRecord{
		{
			ID:        "1592868367.219124-12-file-coding-wakatime-cli-heartbeat-/tmp/main.go-true",
			Heartbeat: string(dataGo),
		},
		{
			ID:        "1592868386.079084-13-file-debugging-wakatime-summary-/tmp/main.py-false",
			Heartbeat: string(dataPy),
		},
	})

	err = db.Close()
	require.NoError(t, err)

	syncFn := offline.Sync(context.Background(), f.Name(), 10)

	var numCalls int

	// run
	err = syncFn(func(_ context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
		numCalls++

		assert.Equal(t, []heartbeat.Heartbeat{
			testHeartbeats()[0],
			testHeartbeats()[1],
		}, hh)

		return nil, errors.New("failed")
	})
	require.Error(t, err)

	// check db
	db, err = bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	var stored []heartbeatRecord

	err = db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("heartbeats")).Cursor()

		for key, value := c.First(); key != nil; key, value = c.Next() {
			stored = append(stored, heartbeatRecord{
				ID:        string(key),
				Heartbeat: string(value),
			})
		}

		return nil
	})
	require.NoError(t, err)

	err = db.Close()
	require.NoError(t, err)

	require.Len(t, stored, 2)

	assert.Equal(t, "1592868367.219124-12-file-coding-wakatime-cli-heartbeat-/tmp/main.go-true", stored[0].ID)
	assert.JSONEq(t, string(dataGo), stored[0].Heartbeat)

	assert.Equal(t, "1592868386.079084-13-file-debugging-wakatime-summary-/tmp/main.py-false", stored[1].ID)
	assert.JSONEq(t, string(dataPy), stored[1].Heartbeat)

	assert.Eventually(t, func() bool { return numCalls == 1 }, time.Second, 50*time.Millisecond)
}

func TestSync_InvalidResults(t *testing.T) {
	// setup
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	defer f.Close()

	db, err := bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	dataGo, err := os.ReadFile("testdata/heartbeat_go.json")
	require.NoError(t, err)

	dataPy, err := os.ReadFile("testdata/heartbeat_py.json")
	require.NoError(t, err)

	dataJs, err := os.ReadFile("testdata/heartbeat_js.json")
	require.NoError(t, err)

	insertHeartbeatRecords(t, db, "heartbeats", []heartbeatRecord{
		{
			ID:        "1592868367.219124-file-coding-wakatime-cli-heartbeat-/tmp/main.go-true",
			Heartbeat: string(dataGo),
		},
		{
			ID:        "1592868386.079084-file-debugging-wakatime-summary-/tmp/main.py-false",
			Heartbeat: string(dataPy),
		},
		{
			ID:        "1592868394.084354-file-building-wakatime-todaygoal-/tmp/main.js-false",
			Heartbeat: string(dataJs),
		},
	})

	err = db.Close()
	require.NoError(t, err)

	syncFn := offline.Sync(context.Background(), f.Name(), 1000)

	var numCalls int

	// run
	err = syncFn(func(_ context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
		numCalls++

		// first request
		if numCalls == 1 {
			require.Len(t, hh, 3)
			assert.Equal(t, []heartbeat.Heartbeat{
				testHeartbeats()[0],
				testHeartbeats()[1],
				testHeartbeats()[2],
			}, hh)

			return []heartbeat.Result{
				{
					Status:    201,
					Heartbeat: testHeartbeats()[0],
				},
				// any non 201/202/400 status results will be retried.
				{
					Status:    429,
					Errors:    []string{"Too many heartbeats"},
					Heartbeat: testHeartbeats()[1],
				},
				// 400 status results will be discarded
				{
					Status:    400,
					Heartbeat: testHeartbeats()[2],
				},
			}, nil
		}

		// second request: assert retry of 429 result
		require.Len(t, hh, 1)
		assert.Equal(t, []heartbeat.Heartbeat{
			testHeartbeats()[1],
		}, hh)

		return []heartbeat.Result{
			{
				Status:    201,
				Heartbeat: testHeartbeats()[1],
			},
		}, nil
	})
	require.NoError(t, err)

	// check db
	db, err = bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	var stored []heartbeatRecord

	err = db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("heartbeats")).Cursor()

		for key, value := c.First(); key != nil; key, value = c.Next() {
			stored = append(stored, heartbeatRecord{
				ID:        string(key),
				Heartbeat: string(value),
			})
		}

		return nil
	})
	require.NoError(t, err)

	err = db.Close()
	require.NoError(t, err)

	require.Len(t, stored, 0)

	assert.Eventually(t, func() bool { return numCalls == 2 }, time.Second, 50*time.Millisecond)
}

func TestSync_SyncLimit(t *testing.T) {
	// setup
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	defer f.Close()

	db, err := bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	dataGo, err := os.ReadFile("testdata/heartbeat_go.json")
	require.NoError(t, err)

	dataPy, err := os.ReadFile("testdata/heartbeat_py.json")
	require.NoError(t, err)

	insertHeartbeatRecords(t, db, "heartbeats", []heartbeatRecord{
		{
			ID:        "1592868367.219124-file-coding-wakatime-cli-heartbeat-/tmp/main.go-true",
			Heartbeat: string(dataGo),
		},
		{
			ID:        "1592868386.079084-file-debugging-wakatime-summary-/tmp/main.py-false",
			Heartbeat: string(dataPy),
		},
	})

	err = db.Close()
	require.NoError(t, err)

	syncFn := offline.Sync(context.Background(), f.Name(), 1)

	var numCalls int

	// run
	err = syncFn(func(_ context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
		numCalls++

		assert.Len(t, hh, 1)

		return []heartbeat.Result{
			{
				Status:    201,
				Heartbeat: testHeartbeats()[0],
			},
		}, nil
	})
	require.NoError(t, err)

	// check db
	db, err = bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	var stored []heartbeatRecord

	err = db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("heartbeats")).Cursor()

		for key, value := c.First(); key != nil; key, value = c.Next() {
			stored = append(stored, heartbeatRecord{
				ID:        string(key),
				Heartbeat: string(value),
			})
		}

		return nil
	})
	require.NoError(t, err)

	err = db.Close()
	require.NoError(t, err)

	require.Len(t, stored, 1)

	assert.Equal(t, "1592868386.079084-file-debugging-wakatime-summary-/tmp/main.py-false", stored[0].ID)
	assert.JSONEq(t, string(dataPy), stored[0].Heartbeat)

	assert.Eventually(t, func() bool { return numCalls == 1 }, time.Second, 50*time.Millisecond)
}

func TestSync_SyncUnlimited(t *testing.T) {
	// setup
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	defer f.Close()

	db, err := bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	dataGo, err := os.ReadFile("testdata/heartbeat_go.json")
	require.NoError(t, err)

	dataPy, err := os.ReadFile("testdata/heartbeat_py.json")
	require.NoError(t, err)

	insertHeartbeatRecords(t, db, "heartbeats", []heartbeatRecord{
		{
			ID:        "1592868367.219124-file-coding-wakatime-cli-heartbeat-/tmp/main.go-true",
			Heartbeat: string(dataGo),
		},
		{
			ID:        "1592868386.079084-file-debugging-wakatime-summary-/tmp/main.py-false",
			Heartbeat: string(dataPy),
		},
	})

	err = db.Close()
	require.NoError(t, err)

	syncFn := offline.Sync(context.Background(), f.Name(), 0)

	var numCalls int

	// run
	err = syncFn(func(_ context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
		numCalls++

		assert.Len(t, hh, 2)

		return []heartbeat.Result{
			{
				Status:    201,
				Heartbeat: testHeartbeats()[0],
			},
			{
				Status:    201,
				Heartbeat: testHeartbeats()[1],
			},
		}, nil
	})
	require.NoError(t, err)

	// check db
	db, err = bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	var stored []heartbeatRecord

	err = db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("heartbeats")).Cursor()

		for key, value := c.First(); key != nil; key, value = c.Next() {
			stored = append(stored, heartbeatRecord{
				ID:        string(key),
				Heartbeat: string(value),
			})
		}

		return nil
	})
	require.NoError(t, err)

	err = db.Close()
	require.NoError(t, err)

	require.Len(t, stored, 0)

	assert.Eventually(t, func() bool { return numCalls == 1 }, time.Second, 50*time.Millisecond)
}

func TestCountHeartbeats(t *testing.T) {
	// setup
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	defer f.Close()

	db, err := bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	insertHeartbeatRecords(t, db, "heartbeats", []heartbeatRecord{
		{
			ID:        "1592868367.219124-file-coding-wakatime-cli-heartbeat-/tmp/main.go-true",
			Heartbeat: "heartbeat_go",
		},
		{
			ID:        "1592868386.079084-file-debugging-wakatime-summary-/tmp/main.py-false",
			Heartbeat: "heartbeat_py",
		},
		{
			ID:        "1592868394.084354-file-building-wakatime-todaygoal-/tmp/main.js-false",
			Heartbeat: "heartbeat_js",
		},
	})

	err = db.Close()
	require.NoError(t, err)

	count, err := offline.CountHeartbeats(context.Background(), f.Name())
	require.NoError(t, err)

	assert.Equal(t, count, 3)
}

func TestCountHeartbeats_Empty(t *testing.T) {
	// setup
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	defer f.Close()

	count, err := offline.CountHeartbeats(context.Background(), f.Name())
	require.NoError(t, err)

	assert.Equal(t, count, 0)
}

func TestReadHeartbeats(t *testing.T) {
	// setup
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	defer f.Close()

	db, err := bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	dataGo, err := os.ReadFile("testdata/heartbeat_go.json")
	require.NoError(t, err)

	dataPy, err := os.ReadFile("testdata/heartbeat_py.json")
	require.NoError(t, err)

	insertHeartbeatRecords(t, db, "heartbeats", []heartbeatRecord{
		{
			ID:        "1592868367.219124-file-coding-wakatime-cli-heartbeat-/tmp/main.go-true",
			Heartbeat: string(dataGo),
		},
		{
			ID:        "1592868386.079084-file-debugging-wakatime-summary-/tmp/main.py-false",
			Heartbeat: string(dataPy),
		},
	})

	err = db.Close()
	require.NoError(t, err)

	hh, err := offline.ReadHeartbeats(context.Background(), f.Name(), offline.PrintMaxDefault)
	require.NoError(t, err)

	assert.Len(t, hh, 2)
}

func TestReadHeartbeats_WithLimit(t *testing.T) {
	// setup
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	defer f.Close()

	db, err := bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	dataGo, err := os.ReadFile("testdata/heartbeat_go.json")
	require.NoError(t, err)

	dataPy, err := os.ReadFile("testdata/heartbeat_py.json")
	require.NoError(t, err)

	insertHeartbeatRecords(t, db, "heartbeats", []heartbeatRecord{
		{
			ID:        "1592868367.219124-file-coding-wakatime-cli-heartbeat-/tmp/main.go-true",
			Heartbeat: string(dataGo),
		},
		{
			ID:        "1592868386.079084-file-debugging-wakatime-summary-/tmp/main.py-false",
			Heartbeat: string(dataPy),
		},
	})

	err = db.Close()
	require.NoError(t, err)

	hh, err := offline.ReadHeartbeats(context.Background(), f.Name(), 1)
	require.NoError(t, err)

	assert.Len(t, hh, 1)
}

func TestReadHeartbeats_Empty(t *testing.T) {
	// setup
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	defer f.Close()

	hh, err := offline.ReadHeartbeats(context.Background(), f.Name(), offline.PrintMaxDefault)
	require.NoError(t, err)

	assert.Len(t, hh, 0)
}

func TestQueue_Count(t *testing.T) {
	// setup
	db, cleanup := initDB(t)
	defer cleanup()

	tx, err := db.Begin(true)
	require.NoError(t, err)

	q := offline.NewQueue(tx)
	q.Bucket = "test_bucket"

	count, err := q.Count()
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	err = tx.Rollback()
	require.NoError(t, err)

	var heartbeatPy heartbeat.Heartbeat

	dataPy, err := os.ReadFile("testdata/heartbeat_py.json")
	require.NoError(t, err)

	err = json.Unmarshal(dataPy, &heartbeatPy)
	require.NoError(t, err)

	var heartbeatJs heartbeat.Heartbeat

	dataJs, err := os.ReadFile("testdata/heartbeat_js.json")
	require.NoError(t, err)

	err = json.Unmarshal(dataJs, &heartbeatJs)
	require.NoError(t, err)

	tx, err = db.Begin(true)
	require.NoError(t, err)

	// run
	q = offline.NewQueue(tx)
	q.Bucket = "test_bucket"
	err = q.PushMany([]heartbeat.Heartbeat{heartbeatPy, heartbeatJs})
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	tx, err = db.Begin(true)
	require.NoError(t, err)

	q = offline.NewQueue(tx)
	q.Bucket = "test_bucket"

	count, err = q.Count()
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	err = tx.Rollback()
	require.NoError(t, err)
}

func TestQueue_PopMany(t *testing.T) {
	// setup
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	defer f.Close()

	db, err := bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	defer func() {
		err = db.Close()
		require.NoError(t, err)
	}()

	dataGo, err := os.ReadFile("testdata/heartbeat_go.json")
	require.NoError(t, err)

	dataPy, err := os.ReadFile("testdata/heartbeat_py.json")
	require.NoError(t, err)

	dataJs, err := os.ReadFile("testdata/heartbeat_js.json")
	require.NoError(t, err)

	insertHeartbeatRecords(t, db, "test_bucket", []heartbeatRecord{
		{
			ID:        "1592868367.219124-file-coding-wakatime-cli-heartbeat-/tmp/main.go-true",
			Heartbeat: string(dataGo),
		},
		{
			ID:        "1592868386.079084-file-debugging-wakatime-summary-/tmp/main.py-false",
			Heartbeat: string(dataPy),
		},
		{
			ID:        "1592868394.084354-file-building-wakatime-todaygoal-/tmp/main.js-false",
			Heartbeat: string(dataJs),
		},
	})

	tx, err := db.Begin(true)
	require.NoError(t, err)

	// run
	q := offline.NewQueue(tx)
	q.Bucket = "test_bucket"
	hh, err := q.PopMany(2)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// check
	assert.Len(t, hh, 2)
	assert.Contains(t, hh, testHeartbeats()[0])
	assert.Contains(t, hh, testHeartbeats()[1])

	var stored []heartbeatRecord

	err = db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("test_bucket")).Cursor()

		for key, value := c.First(); key != nil; key, value = c.Next() {
			stored = append(stored, heartbeatRecord{
				ID:        string(key),
				Heartbeat: string(value),
			})
		}

		return nil
	})
	require.NoError(t, err)

	assert.Len(t, stored, 1)
	assert.Equal(t, "1592868394.084354-file-building-wakatime-todaygoal-/tmp/main.js-false", stored[0].ID)
	assert.JSONEq(t, string(dataJs), stored[0].Heartbeat)
}

func TestQueue_PushMany(t *testing.T) {
	// setup
	db, cleanup := initDB(t)
	defer cleanup()

	dataGo, err := os.ReadFile("testdata/heartbeat_go.json")
	require.NoError(t, err)

	insertHeartbeatRecord(t, db, "test_bucket", heartbeatRecord{
		ID:        "1592868367.219124-1-file-coding-wakatime-cli-heartbeat-/tmp/main.go-true",
		Heartbeat: string(dataGo),
	})

	var heartbeatPy heartbeat.Heartbeat

	dataPy, err := os.ReadFile("testdata/heartbeat_py.json")
	require.NoError(t, err)

	err = json.Unmarshal(dataPy, &heartbeatPy)
	require.NoError(t, err)

	var heartbeatJs heartbeat.Heartbeat

	dataJs, err := os.ReadFile("testdata/heartbeat_js.json")
	require.NoError(t, err)

	err = json.Unmarshal(dataJs, &heartbeatJs)
	require.NoError(t, err)

	tx, err := db.Begin(true)
	require.NoError(t, err)

	// run
	q := offline.NewQueue(tx)
	q.Bucket = "test_bucket"
	err = q.PushMany([]heartbeat.Heartbeat{heartbeatPy, heartbeatJs})
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// check
	var stored []heartbeatRecord

	err = db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("test_bucket")).Cursor()

		for key, value := c.First(); key != nil; key, value = c.Next() {
			stored = append(stored, heartbeatRecord{
				ID:        string(key),
				Heartbeat: string(value),
			})
		}

		return nil
	})
	require.NoError(t, err)

	assert.Len(t, stored, 3)

	assert.Equal(t, "1592868367.219124-1-file-coding-wakatime-cli-heartbeat-/tmp/main.go-true", stored[0].ID)
	assert.JSONEq(t, string(dataGo), stored[0].Heartbeat)

	assert.Equal(t, "1592868386.079084-13-file-debugging-wakatime-summary-/tmp/main.py-false", stored[1].ID)
	assert.JSONEq(t, string(dataPy), stored[1].Heartbeat)

	assert.Equal(t, "1592868394.084354-14-file-building-wakatime-todaygoal-/tmp/main.js-false", stored[2].ID)
	assert.JSONEq(t, string(dataJs), stored[2].Heartbeat)
}

func TestQueue_ReadMany(t *testing.T) {
	// setup
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	defer f.Close()

	db, err := bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	defer func() {
		err = db.Close()
		require.NoError(t, err)
	}()

	dataGo, err := os.ReadFile("testdata/heartbeat_go.json")
	require.NoError(t, err)

	dataPy, err := os.ReadFile("testdata/heartbeat_py.json")
	require.NoError(t, err)

	dataJs, err := os.ReadFile("testdata/heartbeat_js.json")
	require.NoError(t, err)

	insertHeartbeatRecords(t, db, "test_bucket", []heartbeatRecord{
		{
			ID:        "1592868367.219124-file-coding-wakatime-cli-heartbeat-/tmp/main.go-true",
			Heartbeat: string(dataGo),
		},
		{
			ID:        "1592868386.079084-file-debugging-wakatime-summary-/tmp/main.py-false",
			Heartbeat: string(dataPy),
		},
		{
			ID:        "1592868394.084354-file-building-wakatime-todaygoal-/tmp/main.js-false",
			Heartbeat: string(dataJs),
		},
	})

	tx, err := db.Begin(true)
	require.NoError(t, err)

	// run
	q := offline.NewQueue(tx)
	q.Bucket = "test_bucket"
	hh, err := q.ReadMany(2)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// check
	assert.Len(t, hh, 2)
	assert.Contains(t, hh, testHeartbeats()[0])
	assert.Contains(t, hh, testHeartbeats()[1])

	var stored []heartbeatRecord

	err = db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("test_bucket")).Cursor()

		for key, value := c.First(); key != nil; key, value = c.Next() {
			stored = append(stored, heartbeatRecord{
				ID:        string(key),
				Heartbeat: string(value),
			})
		}

		return nil
	})
	require.NoError(t, err)

	assert.Len(t, stored, 3)

	assert.Equal(t, "1592868367.219124-file-coding-wakatime-cli-heartbeat-/tmp/main.go-true", stored[0].ID)
	assert.Equal(t, "1592868386.079084-file-debugging-wakatime-summary-/tmp/main.py-false", stored[1].ID)
	assert.Equal(t, "1592868394.084354-file-building-wakatime-todaygoal-/tmp/main.js-false", stored[2].ID)

	assert.JSONEq(t, string(dataGo), stored[0].Heartbeat)
	assert.JSONEq(t, string(dataPy), stored[1].Heartbeat)
	assert.JSONEq(t, string(dataJs), stored[2].Heartbeat)
}

func TestQueue_ReadMany_Empty(t *testing.T) {
	// setup
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	defer f.Close()

	db, err := bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	defer func() {
		err = db.Close()
		require.NoError(t, err)
	}()

	tx, err := db.Begin(true)
	require.NoError(t, err)

	// run
	q := offline.NewQueue(tx)
	q.Bucket = "test_bucket"
	hh, err := q.ReadMany(10)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// check
	assert.Len(t, hh, 0)
}

func initDB(t *testing.T) (*bolt.DB, func()) {
	// create tmp file
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	// init db
	db, err := bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	return db, func() {
		defer f.Close()
		defer func() {
			err = db.Close()
			require.NoError(t, err)
		}()
	}
}

func testHeartbeats() []heartbeat.Heartbeat {
	return []heartbeat.Heartbeat{
		{
			Branch:         heartbeat.PointerTo("heartbeat"),
			Category:       heartbeat.CodingCategory,
			CursorPosition: heartbeat.PointerTo(12),
			Dependencies:   []string{"dep1", "dep2"},
			Entity:         "/tmp/main.go",
			EntityType:     heartbeat.FileType,
			IsWrite:        heartbeat.PointerTo(true),
			Language:       heartbeat.PointerTo("Go"),
			LineNumber:     heartbeat.PointerTo(42),
			Lines:          heartbeat.PointerTo(100),
			Project:        heartbeat.PointerTo("wakatime-cli"),
			Time:           1592868367.219124,
			UserAgent:      "wakatime/13.0.6",
		},
		{
			Branch:         heartbeat.PointerTo("summary"),
			Category:       heartbeat.DebuggingCategory,
			CursorPosition: heartbeat.PointerTo(13),
			Dependencies:   []string{"dep3", "dep4"},
			Entity:         "/tmp/main.py",
			EntityType:     heartbeat.FileType,
			IsWrite:        heartbeat.PointerTo(false),
			Language:       heartbeat.PointerTo("Python"),
			LineNumber:     heartbeat.PointerTo(43),
			Lines:          heartbeat.PointerTo(101),
			Project:        heartbeat.PointerTo("wakatime"),
			Time:           1592868386.079084,
			UserAgent:      "wakatime/13.0.7",
		},
		{
			Branch:         heartbeat.PointerTo("todaygoal"),
			Category:       heartbeat.BuildingCategory,
			CursorPosition: heartbeat.PointerTo(14),
			Dependencies:   []string{"dep5", "dep6"},
			Entity:         "/tmp/main.js",
			EntityType:     heartbeat.FileType,
			IsWrite:        heartbeat.PointerTo(false),
			Language:       heartbeat.PointerTo("JavaScript"),
			LineNumber:     heartbeat.PointerTo(44),
			Lines:          heartbeat.PointerTo(102),
			Project:        heartbeat.PointerTo("wakatime"),
			Time:           1592868394.084354,
			UserAgent:      "wakatime/13.0.8",
		},
	}
}

type heartbeatRecord struct {
	ID        string
	Heartbeat string
}

func insertHeartbeatRecords(t *testing.T, db *bolt.DB, bucket string, hh []heartbeatRecord) {
	for _, h := range hh {
		insertHeartbeatRecord(t, db, bucket, h)
	}
}

func insertHeartbeatRecord(t *testing.T, db *bolt.DB, bucket string, h heartbeatRecord) {
	t.Helper()

	err := db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return fmt.Errorf("failed to create bucket: %s", err)
		}

		err = b.Put([]byte(h.ID), []byte(h.Heartbeat))
		if err != nil {
			return fmt.Errorf("failed put heartbeat: %s", err)
		}

		return nil
	})
	require.NoError(t, err)
}
