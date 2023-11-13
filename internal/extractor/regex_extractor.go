package extractor

import (
	"regexp"
)

type RegexExtractor struct{}

// Мы предполагаем, что длина video id всегда 11 символов.
var reURL = regexp.MustCompile(`(?:youtube(?:-nocookie)?\.com\/(?:[^\/\n\s]+\/\S+\/|(?:v|e(?:mbed)?)\/|\S*?[?&]vi?=)|youtu\.be\/)([a-zA-Z0-9_-]{11})`)
var reVideoID = regexp.MustCompile("^[a-zA-Z0-9_-]{11}$")

func (p RegexExtractor) ExtractVideoIDFromURL(s string) (string, error) {
	match := reURL.FindStringSubmatch(s)
	if len(match) > 1 {
		videoID := match[1]
		return videoID, nil
	}

	// Plain video id
	if reVideoID.Match([]byte(s)) {
		return s, nil
	}

	return "", ErrInvalidUrl
}
