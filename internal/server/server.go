package server

import (
	"context"
	"sync"

	pb "github.com/pegov/yt-thumbnails-go/api/thumbnail_v1"
)

type Cache interface {
	Get(ctx context.Context, videoID string) ([]byte, error)
	Set(ctx context.Context, videoID string, data []byte) error
}

type Downloader interface {
	DownloadThumbnail(ctx context.Context, videoID string) ([]byte, error)
}

type Extractor interface {
	ExtractVideoIDFromURL(url string) (string, error)
}

type server struct {
	pb.UnimplementedThumbnailServiceServer

	cache      Cache
	extractor  Extractor
	downloader Downloader
	semaphore  chan struct{}
	shutdown   chan<- error
	mu         sync.Mutex
	isStopping bool
}

func NewServer(
	cache Cache,
	extractor Extractor,
	downloader Downloader,
	maxParallelHTTPRequests int,
	shutdown chan<- error,
) *server {
	return &server{
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
		s.shutdown <- err
		s.isStopping = true
	}
}
