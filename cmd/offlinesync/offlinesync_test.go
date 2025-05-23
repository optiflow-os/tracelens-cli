package offlinesync_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/optiflow-os/tracelens-cli/cmd/offlinesync"
	cmdparams "github.com/optiflow-os/tracelens-cli/cmd/params"
	"github.com/optiflow-os/tracelens-cli/pkg/exitcode"
	"github.com/optiflow-os/tracelens-cli/pkg/heartbeat"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func TestRunWithRateLimiting(t *testing.T) {
	resetSingleton(t)

	testServerURL, router, tearDown := setupTestServer()
	defer tearDown()

	var (
		plugin   = "plugin/0.0.1"
		numCalls int
	)

	router.HandleFunc("/users/current/heartbeats.bulk", func(w http.ResponseWriter, req *http.Request) {
		numCalls++

		// check request
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, []string{"application/json"}, req.Header["Accept"])
		assert.Equal(t, []string{"application/json"}, req.Header["Content-Type"])
		assert.Equal(t, []string{"Basic MDAwMDAwMDAtMDAwMC00MDAwLTgwMDAtMDAwMDAwMDAwMDAw"}, req.Header["Authorization"])
		assert.True(t, strings.HasSuffix(req.Header["User-Agent"][0], plugin), fmt.Sprintf(
			"%q should have suffix %q",
			req.Header["User-Agent"][0],
			plugin,
		))

		expectedBody, err := os.ReadFile("testdata/api_heartbeats_request_template.json")
		require.NoError(t, err)

		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)

		assert.JSONEq(t, string(expectedBody), string(body))

		// send response
		w.WriteHeader(http.StatusCreated)

		f, err := os.Open("testdata/api_heartbeats_response.json")
		require.NoError(t, err)
		defer f.Close()

		_, err = io.Copy(w, f)
		require.NoError(t, err)
	})

	// setup offline queue
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

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

	v := viper.New()
	v.Set("api-url", testServerURL)
	v.Set("key", "00000000-0000-4000-8000-000000000000")
	v.Set("offline-queue-file", f.Name())
	v.Set("sync-offline-activity", 100)
	v.Set("plugin", plugin)

	code, err := offlinesync.RunWithRateLimiting(context.Background(), v)
	require.NoError(t, err)

	assert.Equal(t, exitcode.Success, code)
	assert.Eventually(t, func() bool { return numCalls == 1 }, time.Second, 50*time.Millisecond)
}

func TestRunWithoutRateLimiting(t *testing.T) {
	resetSingleton(t)

	testServerURL, router, tearDown := setupTestServer()
	defer tearDown()

	var (
		plugin   = "plugin/0.0.1"
		numCalls int
	)

	router.HandleFunc("/users/current/heartbeats.bulk", func(w http.ResponseWriter, req *http.Request) {
		numCalls++

		// check request
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, []string{"application/json"}, req.Header["Accept"])
		assert.Equal(t, []string{"application/json"}, req.Header["Content-Type"])
		assert.Equal(t, []string{"Basic MDAwMDAwMDAtMDAwMC00MDAwLTgwMDAtMDAwMDAwMDAwMDAw"}, req.Header["Authorization"])
		assert.True(t, strings.HasSuffix(req.Header["User-Agent"][0], plugin), fmt.Sprintf(
			"%q should have suffix %q",
			req.Header["User-Agent"][0],
			plugin,
		))

		expectedBody, err := os.ReadFile("testdata/api_heartbeats_request_template.json")
		require.NoError(t, err)

		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)

		assert.JSONEq(t, string(expectedBody), string(body))

		// send response
		w.WriteHeader(http.StatusCreated)

		f, err := os.Open("testdata/api_heartbeats_response.json")
		require.NoError(t, err)
		defer f.Close()

		_, err = io.Copy(w, f)
		require.NoError(t, err)
	})

	// setup offline queue
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

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

	v := viper.New()
	v.Set("api-url", testServerURL)
	v.Set("key", "00000000-0000-4000-8000-000000000000")
	v.Set("offline-queue-file", f.Name())
	v.Set("sync-offline-activity", 100)
	v.Set("plugin", plugin)

	code, err := offlinesync.RunWithoutRateLimiting(context.Background(), v)
	require.NoError(t, err)

	assert.Equal(t, exitcode.Success, code)
	assert.Eventually(t, func() bool { return numCalls == 1 }, time.Second, 50*time.Millisecond)
}

func TestRunWithRateLimiting_RateLimited(t *testing.T) {
	resetSingleton(t)

	v := viper.New()
	v.Set("key", "00000000-0000-4000-8000-000000000000")
	v.Set("heartbeat-rate-limit-seconds", 500)
	v.Set("internal.heartbeats_last_sent_at", time.Now().Add(-time.Minute).Format(time.RFC3339))

	code, err := offlinesync.RunWithRateLimiting(context.Background(), v)
	require.NoError(t, err)

	assert.Equal(t, exitcode.Success, code)
}

func TestSyncOfflineActivity(t *testing.T) {
	resetSingleton(t)

	testServerURL, router, tearDown := setupTestServer()
	defer tearDown()

	var (
		plugin   = "plugin/0.0.1"
		numCalls int
	)

	router.HandleFunc("/users/current/heartbeats.bulk", func(w http.ResponseWriter, req *http.Request) {
		numCalls++

		// check request
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, []string{"application/json"}, req.Header["Accept"])
		assert.Equal(t, []string{"application/json"}, req.Header["Content-Type"])
		assert.Equal(t, []string{"Basic MDAwMDAwMDAtMDAwMC00MDAwLTgwMDAtMDAwMDAwMDAwMDAw"}, req.Header["Authorization"])
		assert.True(t, strings.HasSuffix(req.Header["User-Agent"][0], plugin), fmt.Sprintf(
			"%q should have suffix %q",
			req.Header["User-Agent"][0],
			plugin,
		))

		expectedBody, err := os.ReadFile("testdata/api_heartbeats_request_template.json")
		require.NoError(t, err)

		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)

		assert.JSONEq(t, string(expectedBody), string(body))

		// send response
		w.WriteHeader(http.StatusCreated)

		f, err := os.Open("testdata/api_heartbeats_response.json")
		require.NoError(t, err)
		defer f.Close()

		_, err = io.Copy(w, f)
		require.NoError(t, err)
	})

	// setup offline queue
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

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

	v := viper.New()
	v.Set("api-url", testServerURL)
	v.Set("key", "00000000-0000-4000-8000-000000000000")
	v.Set("sync-offline-activity", 100)
	v.Set("plugin", plugin)

	err = offlinesync.SyncOfflineActivity(context.Background(), v, f.Name())
	require.NoError(t, err)

	assert.Eventually(t, func() bool { return numCalls == 1 }, time.Second, 50*time.Millisecond)
}

func TestSyncOfflineActivity_MultipleApiKey(t *testing.T) {
	resetSingleton(t)

	testServerURL, router, tearDown := setupTestServer()
	defer tearDown()

	var (
		plugin   = "plugin/0.0.1"
		numCalls int
	)

	router.HandleFunc("/users/current/heartbeats.bulk", func(w http.ResponseWriter, req *http.Request) {
		numCalls++

		// check auth header
		switch numCalls {
		case 1:
			assert.Equal(t, []string{"Basic MDAwMDAwMDAtMDAwMC00MDAwLTgwMDAtMDAwMDAwMDAwMDAw"}, req.Header["Authorization"])
		case 2:
			assert.Equal(t, []string{"Basic MDAwMDAwMDAtMDAwMC00MDAwLTgwMDAtMDAwMDAwMDAwMDAx"}, req.Header["Authorization"])
		}

		// send response
		f, err := os.Open("testdata/api_heartbeats_response.json")
		require.NoError(t, err)
		defer f.Close()

		w.WriteHeader(http.StatusCreated)
		_, err = io.Copy(w, f)
		require.NoError(t, err)
	})

	// setup offline queue
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	db, err := bolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)

	dataGo, err := os.ReadFile("testdata/heartbeat_go.json")
	require.NoError(t, err)

	var hgo heartbeat.Heartbeat

	err = json.Unmarshal(dataGo, &hgo)
	require.NoError(t, err)

	hgo.APIKey = "00000000-0000-4000-8000-000000000000"

	dataGoChanged, err := json.Marshal(hgo)
	require.NoError(t, err)

	dataPy, err := os.ReadFile("testdata/heartbeat_py.json")
	require.NoError(t, err)

	var hpy heartbeat.Heartbeat

	err = json.Unmarshal(dataPy, &hpy)
	require.NoError(t, err)

	hpy.APIKey = "00000000-0000-4000-8000-000000000001"

	dataPyChanged, err := json.Marshal(hpy)
	require.NoError(t, err)

	insertHeartbeatRecords(t, db, "heartbeats", []heartbeatRecord{
		{
			ID:        "1592868367.219124-file-coding-wakatime-cli-heartbeat-/tmp/main.go-true",
			Heartbeat: string(dataGoChanged),
		},
		{
			ID:        "1592868386.079084-file-debugging-wakatime-summary-/tmp/main.py-false",
			Heartbeat: string(dataPyChanged),
		},
	})

	err = db.Close()
	require.NoError(t, err)

	v := viper.New()
	v.Set("api-url", testServerURL)
	v.Set("key", "00000000-0000-4000-8000-000000000000")
	v.Set("sync-offline-activity", 100)
	v.Set("plugin", plugin)

	err = offlinesync.SyncOfflineActivity(context.Background(), v, f.Name())
	require.NoError(t, err)

	assert.Eventually(t, func() bool { return numCalls == 1 }, time.Second, 50*time.Millisecond)
}

func setupTestServer() (string, *http.ServeMux, func()) {
	router := http.NewServeMux()
	srv := httptest.NewServer(router)

	return srv.URL, router, func() { srv.Close() }
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

func resetSingleton(t *testing.T) {
	t.Helper()

	cmdparams.Once = sync.Once{}
}
