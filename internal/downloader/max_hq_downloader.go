package downloader

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
)

const (
	urlFormatMaxRes = "https://i.ytimg.com/vi/%s/maxresdefault.jpg"
	urlFormatHq     = "https://i.ytimg.com/vi/%s/hqdefault.jpg"
)

type MaxResOrHqDownloader struct{}

func download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, ErrCouldNotCreateRequest
	}

	client := http.DefaultClient
	res, err := client.Do(req)
	if res != nil {
		defer res.Body.Close()
	}

	if e, ok := err.(net.Error); ok && e.Timeout() {
		return nil, ErrTimeout
	} else if err != nil {
		return nil, ErrCouldNotMakeRequest
	}

	if res.StatusCode == 404 {
		return nil, ErrNotFound
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, ErrCouldNotReadBody
	}

	return body, nil
}

func (d MaxResOrHqDownloader) DownloadThumbnail(
	ctx context.Context,
	videoID string,
) ([]byte, error) {
	maxResURL := fmt.Sprintf(urlFormatMaxRes, videoID)
	b, err := download(ctx, maxResURL)
	if err == nil {
		return b, nil
	}

	urlHq := fmt.Sprintf(urlFormatHq, videoID)
	return download(ctx, urlHq)
}
