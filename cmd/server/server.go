package main

import (
	"context"
	"flag"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/pegov/yt-thumbnails-go/api/thumbnail_v1"
	"github.com/pegov/yt-thumbnails-go/internal/cache"
	"github.com/pegov/yt-thumbnails-go/internal/cache/sqlite"
	"github.com/pegov/yt-thumbnails-go/internal/downloader"
	"github.com/pegov/yt-thumbnails-go/internal/extractor"
)

type Cache interface {
	Get(ctx context.Context, videoID string) ([]byte, error)
	Set(ctx context.Context, videoID string, data []byte) error
}

type Extractor interface {
	ExtractVideoIDFromURL(url string) (string, error)
}

type Downloader interface {
	DownloadThumbnail(videoID string) ([]byte, error)
}

type server struct {
	cache      Cache
	extractor  Extractor
	downloader Downloader
	semaphore  chan struct{}
	pb.UnimplementedThumbnailServiceServer
}

var (
	addr                    = flag.String("addr", "localhost:8080", "address")
	maxParallelHTTPRequests = flag.Int(
		"max-parallel-http-requests",
		16,
		"max parallel http requests to youtube",
	)
)

func (s *server) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	videoID, err := s.extractor.ExtractVideoIDFromURL(req.Url)

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "url: invalid url")
	}

	b, err := s.cache.Get(ctx, videoID)
	if err == nil {
		log.Printf("%v: getting from cache...", videoID)
		return &pb.GetResponse{
			Url:     req.Url,
			VideoId: videoID,
			Data:    b,
		}, nil
	} else if err != nil && err != cache.ErrNotFound {
		log.Fatalf("%v: internal cache error %v", videoID, err)
	}

	s.semaphore <- struct{}{}
	log.Printf("%v: making http request...\n", videoID)
	b, err = s.downloader.DownloadThumbnail(videoID)
	<-s.semaphore
	if err != nil {
		switch err {
		case downloader.ErrNotFound:
			log.Printf("%v: not found\n", videoID)
			return nil, status.Error(codes.NotFound, "NOT_FOUND")
		default:
			log.Printf("%v: %v\n", videoID, err)
			return nil, status.Error(codes.Internal, "INTERNAL")
		}
	}

	s.cache.Set(ctx, videoID, b)

	return &pb.GetResponse{
		Url:     req.Url,
		VideoId: videoID,
		Data:    b,
	}, nil
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	s := &server{
		cache:      sqlite.New(),
		extractor:  extractor.RegexExtractor{},
		downloader: downloader.MaxResOrHqDownloader{},
		semaphore:  make(chan struct{}, *maxParallelHTTPRequests),
	}
	pb.RegisterThumbnailServiceServer(grpcServer, s)
	log.Printf("Server listening at %v\n", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
