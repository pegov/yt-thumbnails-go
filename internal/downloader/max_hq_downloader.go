package downloader

import (
	"fmt"
	"io"
	"net/http"
)

const (
	urlFormatMaxRes = "https://i.ytimg.com/vi/%s/maxresdefault.jpg"
	urlFormatHq     = "https://i.ytimg.com/vi/%s/hqdefault.jpg"
)

type MaxResOrHqDownloader struct{}

func download(url string) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, ErrCouldNotMakeRequest
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return nil, ErrNotFound
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, ErrCouldNotReadBody
	}

	return body, nil
}

func (d MaxResOrHqDownloader) DownloadThumbnail(videoID string) ([]byte, error) {
	maxResURL := fmt.Sprintf(urlFormatMaxRes, videoID)
	b, err := download(maxResURL)
	if err == nil {
		return b, nil
	}

	urlHq := fmt.Sprintf(urlFormatHq, videoID)
	return download(urlHq)
}
