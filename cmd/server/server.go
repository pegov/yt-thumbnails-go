package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	pb "github.com/pegov/yt-thumbnails-go/api/thumbnail_v1"
	"github.com/pegov/yt-thumbnails-go/internal/cache/sqlite"
	"github.com/pegov/yt-thumbnails-go/internal/downloader"
	"github.com/pegov/yt-thumbnails-go/internal/extractor"
	"github.com/pegov/yt-thumbnails-go/internal/server"
)

var (
	addr                    = flag.String("addr", "localhost:8080", "address")
	maxParallelHTTPRequests = flag.Int(
		"max-parallel-http-requests",
		16,
		"max parallel http requests to youtube",
	)
)

func main() {
	flag.Parse()

	logLevel, ok := os.LookupEnv("LOG_LEVEL")
	if !ok {
		logLevel = "INFO"
	}

	logger := setupLogger(logLevel)

	ctx := context.Background()

	ctxCache, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	sqliteCache, err := sqlite.New(ctxCache, "./thumbnail.db")
	if err != nil {
		logger.Error("Could not create sqlite cache", slog.Any("err", err))
		os.Exit(1)
	}

	shutdown := make(chan struct{}, 1)
	srv := server.NewServer(
		logger,
		sqliteCache,
		extractor.RegexExtractor{},
		downloader.MaxResOrHqDownloader{},
		*maxParallelHTTPRequests,
		shutdown,
	)

	grpcServer := grpc.NewServer()
	pb.RegisterThumbnailServiceServer(grpcServer, srv)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		logger.Error("Failed to listen: %v", err)
		os.Exit(1)
	}

	logger.Info("Starting server")

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("Failed to serve", slog.Any("err", err))
			os.Exit(1)
		}
	}()
	logger.Info("Server listening", slog.Any("addr", lis.Addr()))

	select {
	case <-done:
		logger.Warn("Stopping server")
	case <-shutdown:
	}

	ctxShutdown, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	stop := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		sqliteCache.Close()
		stop <- struct{}{}
	}()

	select {
	case <-ctxShutdown.Done():
		logger.Warn("Timeout on graceful shutdown")
		grpcServer.Stop()
	case <-stop:
		logger.Warn("Graceful shutdown")
	}
}

func setupLogger(levelString string) *slog.Logger {
	var level slog.Level
	switch levelString {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		panic("unknown level name")
	}

	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}
