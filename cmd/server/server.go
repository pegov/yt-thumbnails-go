package main

import (
	"context"
	"flag"
	"log"
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

	log.SetFlags(0)

	ctx := context.Background()

	ctxCache, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	sqliteCache, err := sqlite.New(ctxCache)
	if err != nil {
		log.Fatalf("Could not create sqlite cache: %v", err)
	}

	shutdown := make(chan error, 1)
	srv := server.NewServer(
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
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Print("Starting server")

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()
	log.Printf("Server listening at %v\n", lis.Addr())

	select {
	case <-done:
		log.Print("Stopping server")
	case err := <-shutdown:
		log.Printf("Internal server error: %v", err)
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
		log.Print("Timeout on graceful shutdown")
		grpcServer.Stop()
	case <-stop:
		log.Print("Graceful shutdown")
	}
}
