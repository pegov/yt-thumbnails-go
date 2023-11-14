package server

import (
	"context"
	"log"
	"log/slog"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	pb "github.com/pegov/yt-thumbnails-go/api/thumbnail_v1"
	"github.com/pegov/yt-thumbnails-go/internal/cache/sqlite"
	"github.com/pegov/yt-thumbnails-go/internal/downloader"
	"github.com/pegov/yt-thumbnails-go/internal/extractor"
)

type pair struct {
	url     string
	videoID string
	code    codes.Code
	b       []byte
}

var wantBytes, _ = os.ReadFile("../../testdata/maxres.jpg")

var pairs = []pair{
	{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ", codes.OK, wantBytes},
	{"https://www.youtube.com/watch?x=dQw4w9WgXcQ", "", codes.InvalidArgument, nil},
	{"https://www.youtube.com/watch?v=dQw4wXXXXXX", "dQw4wXXXXXX", codes.NotFound, nil},
}

func TestThumbnailService_Get(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	t.Cleanup(func() {
		lis.Close()
	})

	srv := grpc.NewServer()
	t.Cleanup(func() {
		srv.Stop()
	})

	shutdown := make(chan struct{}, 1)
	c, _ := sqlite.New(context.Background(), ":memory:")
	svc := NewServer(slog.Default(), c, extractor.RegexExtractor{}, downloader.MaxResOrHqDownloader{}, 1, shutdown)

	pb.RegisterThumbnailServiceServer(srv, svc)

	go func() {
		if err := srv.Serve(lis); err != nil {
			log.Fatalf("srv.Serve %v", err)
		}
	}()

	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}
	conn, err := grpc.DialContext(
		context.Background(),
		"", grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	t.Cleanup(func() {
		conn.Close()
	})
	if err != nil {
		t.Fatalf("grpc.DialContext %v", err)
	}

	client := pb.NewThumbnailServiceClient(conn)
	for _, pair := range pairs {
		r, err := client.Get(context.Background(), &pb.GetRequest{Url: pair.url})
		assert.Equal(t, status.Code(err), pair.code)
		if pair.code == codes.OK {
			assert.Equal(t, r.GetUrl(), pair.url)
			assert.Equal(t, r.GetVideoId(), pair.videoID)
			assert.Equal(t, r.GetData(), wantBytes)
		}
	}
}
