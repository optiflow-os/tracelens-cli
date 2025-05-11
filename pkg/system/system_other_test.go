//go:build !linux

package system_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/optiflow-os/tracelens-cli/pkg/system"
	"github.com/stretchr/testify/assert"
)

func TestOSName(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("skipping test on non-darwin and non-windows system")
	}

	name := system.OSName(context.Background())

	assert.Equal(t, runtime.GOOS, name)
}
