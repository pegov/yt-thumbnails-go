package server

import (
	"context"
	"log/slog"
	"sync"

	pb "github.com/pegov/yt-thumbnails-go/api/thumbnail_v1"
)

type Cache interface {
	Get(ctx context.Context, videoID string) ([]byte, error)
	Set(ctx context.Context, videoID string, data []byte, ts int64) error
}

type Downloader interface {
	DownloadThumbnail(ctx context.Context, videoID string) ([]byte, error)
}

type Extractor interface {
	ExtractVideoIDFromURL(url string) (string, error)
}

type server struct {
	pb.UnimplementedThumbnailServiceServer

	logger     *slog.Logger
	cache      Cache
	extractor  Extractor
	downloader Downloader
	semaphore  chan struct{}
	shutdown   chan<- struct{}
	mu         sync.Mutex
	isStopping bool
}

func NewServer(
	logger *slog.Logger,
	cache Cache,
	extractor Extractor,
	downloader Downloader,
	maxParallelHTTPRequests int,
	shutdown chan<- struct{},
) *server {
	return &server{
		logger:     logger,
		cache:      cache,
		extractor:  extractor,
		downloader: downloader,
		semaphore:  make(chan struct{}, maxParallelHTTPRequests),
		shutdown:   shutdown,
		mu:         sync.Mutex{},
		isStopping: false,
	}
}

func (s *server) stopOnInternalError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.isStopping {
		s.shutdown <- struct{}{}
		s.isStopping = true
	}
}
