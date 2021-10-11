package lib

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUtils(t *testing.T) {
	host, err := os.Hostname()
	require.NoError(t, err)

	start := float64(time.Now().Unix())
	tested := GetTimestamp()

	year, month, day := time.Now().Date()

	t.Run("Test util functions", func(t *testing.T) {
		assert.True(t, start <= tested)
		assert.True(t, float64(time.Now().Unix()) >= GetTimestamp())
		assert.Equal(t, fmt.Sprintf("%s-wubba", host), FormatPublisher("wubba"))
		assert.Equal(t, fmt.Sprintf("lubba-%s.%d.%02d.%02d", host, year, month, day), FormatIndex("lubba"))
	})
}
