package downloader

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var d MaxResOrHqDownloader

var (
	videoIDMaxRes = "dQw4w9WgXcQ"
	videoIDHq     = "jNQXAC9IVRw"
)

func TestMaxOrHqDownloaderMaxRes(t *testing.T) {
	wantMaxRes, _ := os.ReadFile("../../testdata/maxres.jpg")
	actualMaxRes, err := d.DownloadThumbnail(context.Background(), videoIDMaxRes)
	if err != nil {
		t.Fatalf("TestMaxOrHqDownloaderMaxRes: http error %v", err)
	}

	assert.Equal(t, actualMaxRes, wantMaxRes)
}

func TestMaxOrHqDownloaderHq(t *testing.T) {
	wantHq, _ := os.ReadFile("../../testdata/hq.jpg")
	actualHq, err := d.DownloadThumbnail(context.Background(), videoIDHq)
	if err != nil {
		t.Fatalf("TestMaxOrHqDownloaderMaxRes: http error %v", err)
	}

	assert.Equal(t, actualHq, wantHq)
}
