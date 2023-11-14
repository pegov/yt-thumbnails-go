package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/pegov/yt-thumbnails-go/internal/cache"
)

const sqlInit = `
CREATE TABLE IF NOT EXISTS thumbnail (
	id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	video_id TEXT,
	data BLOB,
	ts INTEGER
);
CREATE INDEX IF NOT EXISTS thumbnail_video_id_idx ON thumbnail(video_id);
`

const sqlInsert = `
INSERT INTO thumbnail (video_id, data, ts) VALUES (
	?, ?, ?
);
`

const sqlSelect = `
SELECT data, ts FROM thumbnail WHERE video_id = ?;
`

type SQLiteCache struct {
	db         *sql.DB
	insertStmt *sql.Stmt
	selectStmt *sql.Stmt
}

func New(ctx context.Context) (*SQLiteCache, error) {
	db, err := sql.Open("sqlite3", "./thumbnail.db")
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	_, err = db.Exec(sqlInit)
	if err != nil {
		return nil, err
	}

	insertStmt, err := db.PrepareContext(ctx, sqlInsert)
	if err != nil {
		return nil, err
	}

	selectStmt, err := db.PrepareContext(ctx, sqlSelect)
	if err != nil {
		return nil, err
	}
	return &SQLiteCache{db, insertStmt, selectStmt}, nil
}

// Get grabs thumbnail from cache
func (c *SQLiteCache) Get(ctx context.Context, videoID string) ([]byte, error) {
	row := c.selectStmt.QueryRowContext(ctx, videoID)
	var (
		b  []byte
		ts int64
	)
	err := row.Scan(&b, &ts)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return b, cache.ErrNotFound
		} else {
			return b, cache.ErrInternal
		}
	}

	now := time.Now().Unix()
	delta := now - ts
	// Cache is valid only for 24 hours
	if delta > 60*60*24 {
		return b, cache.ErrNotFound
	}

	return b, nil
}

// Set saves thumbnails to cache
func (c *SQLiteCache) Set(
	ctx context.Context,
	videoID string,
	data []byte,
) error {
	_, err := c.insertStmt.ExecContext(ctx, videoID, data, time.Now().Unix())

	if err != nil {
		return cache.ErrInternal
	}

	// TODO: Optionally clear expired items on every ~10th set to save space.

	return nil
}

func (c *SQLiteCache) Close() {
	c.insertStmt.Close()
	c.selectStmt.Close()
	c.db.Close()
}
