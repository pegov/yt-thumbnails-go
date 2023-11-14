package sqlite

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/pegov/yt-thumbnails-go/internal/cache"
)

var ctx = context.Background()

var c *SQLiteCache

var b = []byte("test")

func TestMain(m *testing.M) {
	c, _ = New(ctx, ":memory:")
	code := m.Run()
	c.Close()
	os.Exit(code)
}

func TestSetGet(t *testing.T) {
	id := "videoID1"
	c.Set(ctx, id, b, time.Now().Unix())
	r, err := c.Get(ctx, id)
	assert.Nil(t, err)
	assert.True(t, bytes.Equal(r, b))
}

func TestGetExpired(t *testing.T) {
	id := "videoID2"
	wantTS := time.Now().Unix() - exp - 10
	c.Set(ctx, id, b, wantTS)
	_, err := c.Get(ctx, id)
	assert.ErrorIs(t, err, cache.ErrNotFound)
}

func TestGetNotFound(t *testing.T) {
	id := "videoID3"
	_, err := c.Get(ctx, id)
	assert.ErrorIs(t, err, cache.ErrNotFound)
}
