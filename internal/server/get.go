package server

import (
	"context"
	"errors"
	"log/slog"
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
		s.logger.Info("Getting image from cache", slog.String("video_id", videoID))
		return &pb.GetResponse{
			Url:     req.Url,
			VideoId: videoID,
			Data:    b,
		}, nil
	} else if err != nil && err != cache.ErrNotFound {
		if errors.Is(err, context.Canceled) {
			s.logger.Error("Cache GET: timeout", slog.String("video_id", videoID))
		} else {
			s.logger.Error(
				"Cache GET: internal error",
				slog.String("video_id", videoID),
				slog.Any("err", err),
			)
		}
		s.stopOnInternalError(err)
		return nil, status.Error(codes.Internal, "INTERNAL")
	}

	s.semaphore <- struct{}{}
	s.logger.Info("HTTP request", slog.String("video_id", videoID))
	b, err = s.downloader.DownloadThumbnail(ctx, videoID)
	<-s.semaphore
	if err != nil {
		switch err {
		case downloader.ErrNotFound:
			s.logger.Info("HTTP request: not found", slog.String("video_id", videoID))
			return nil, status.Error(codes.NotFound, "NOT_FOUND")
		case downloader.ErrTimeout:
			s.logger.Error("HTTP request: timeout", slog.String("video_id", videoID))
			return nil, status.Error(codes.DeadlineExceeded, "timeout")
		default:
			s.logger.Error(
				"HTTP request: internal error",
				slog.String("video_id", videoID),
				slog.Any("err", err),
			)
			return nil, status.Error(codes.Internal, "INTERNAL")
		}
	}

	err = s.cache.Set(ctx, videoID, b)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			s.logger.Error("Cache SET: timeout", slog.String("video_id", videoID))
		} else {
			s.logger.Error(
				"Cache SET: internal error",
				slog.String("video_id", videoID),
				slog.Any("err", err),
			)
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
