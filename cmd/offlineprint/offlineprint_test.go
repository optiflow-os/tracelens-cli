package offlineprint_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/optiflow-os/tracelens-cli/cmd/offlineprint"
	"github.com/optiflow-os/tracelens-cli/pkg/exitcode"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func TestPrintOfflineHeartbeats(t *testing.T) {
	// setup offline queue
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

	v := viper.New()
	v.Set("print-offline-heartbeats", 10)
	v.Set("key", "00000000-0000-4000-8000-000000000000")
	v.Set("offline-queue-file", f.Name())

	stdout := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	code, err := offlineprint.Run(context.Background(), v)
	require.NoError(t, err)

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		_, err = io.Copy(&buf, r)
		require.NoError(t, err)
		outC <- buf.String()
	}()

	w.Close()

	os.Stdout = stdout
	output := <-outC

	offlineHeartbeat, err := os.ReadFile("testdata/offline_heartbeat.json")
	require.NoError(t, err)

	assert.Equal(t, exitcode.Success, code)
	assert.Equal(t, string(offlineHeartbeat)+"\n", output)
}

func TestPrintOfflineHeartbeats_Empty(t *testing.T) {
	// setup offline queue
	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	defer f.Close()

	v := viper.New()
	v.Set("print-offline-heartbeats", 10)
	v.Set("key", "00000000-0000-4000-8000-000000000000")
	v.Set("offline-queue-file", f.Name())

	stdout := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	code, err := offlineprint.Run(context.Background(), v)
	require.NoError(t, err)

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		_, err = io.Copy(&buf, r)
		require.NoError(t, err)
		outC <- buf.String()
	}()

	w.Close()

	os.Stdout = stdout
	output := <-outC

	assert.Equal(t, exitcode.Success, code)
	assert.Equal(t, "[]\n", output)
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
