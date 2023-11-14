package server

import (
	"context"
	"errors"
	"log"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/pegov/yt-thumbnails-go/api/thumbnail_v1"
	"github.com/pegov/yt-thumbnails-go/internal/cache"
	"github.com/pegov/yt-thumbnails-go/internal/downloader"
)

func (s *server) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	videoID, err := s.extractor.ExtractVideoIDFromURL(req.Url)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "url: invalid url")
	}

	// For cache and http request
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	b, err := s.cache.Get(ctx, videoID)
	if err == nil {
		log.Printf("%v: getting from cache...", videoID)
		return &pb.GetResponse{
			Url:     req.Url,
			VideoId: videoID,
			Data:    b,
		}, nil
	} else if err != nil && err != cache.ErrNotFound {
		if errors.Is(err, context.Canceled) {
			log.Printf("%v: timeout while looking in cache", videoID)
		} else {
			log.Printf("%v: internal cache error %v", videoID, err)
		}
		s.stopOnInternalError(err)
		return nil, status.Error(codes.Internal, "INTERNAL")
	}

	s.semaphore <- struct{}{}
	log.Printf("%v: making http request...\n", videoID)
	b, err = s.downloader.DownloadThumbnail(ctx, videoID)
	<-s.semaphore
	if err != nil {
		switch err {
		case downloader.ErrNotFound:
			log.Printf("%v: not found", videoID)
			return nil, status.Error(codes.NotFound, "NOT_FOUND")
		case downloader.ErrTimeout:
			log.Printf("%v: timeout", videoID)
			return nil, status.Error(codes.DeadlineExceeded, "timeout")
		default:
			log.Printf("%v: %v\n", videoID, err)
			return nil, status.Error(codes.Internal, "INTERNAL")
		}
	}

	err = s.cache.Set(ctx, videoID, b)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Printf("%v: timeout while saving to cache", videoID)
		} else {
			log.Printf("%v: internal cache error %v", videoID, err)
		}
		s.stopOnInternalError(err)
		return nil, status.Error(codes.Internal, "INTERNAL")
	}

	return &pb.GetResponse{
		Url:     req.Url,
		VideoId: videoID,
		Data:    b,
	}, nil
}
