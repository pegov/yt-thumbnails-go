package extractor

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type pair struct {
	s string
	e error
}

var wantID = "dQw4w9WgXcQ"

var urls = []pair{
	{fmt.Sprintf("http://www.youtube.com/watch?v=%s&feature=feedrec_grec_index", wantID), nil},
	{fmt.Sprintf("http://www.youtube.com/watch?feature=feedrec_grec_index&v=%s", wantID), nil},
	{fmt.Sprintf("http://www.youtube.com/v/%s?fs=1&amp;hl=en_US&amp;rel=0", wantID), nil},
	{fmt.Sprintf("http://www.youtube.com/watch?v=%s#t=0m10s", wantID), nil},
	{fmt.Sprintf("http://www.youtube.com/embed/%s?rel=0", wantID), nil},
	{fmt.Sprintf("http://www.youtube.com/watch?v=%s", wantID), nil},
	{fmt.Sprintf("http://youtu.be/%s", wantID), nil},
	{wantID, nil},
	{"", ErrInvalidUrl},
	{fmt.Sprintf("http://www.youtube.com/watch?X=%s", wantID), ErrInvalidUrl},
}

var extractor = RegexExtractor{}

func TestExtractVideoIDFromURL(t *testing.T) {
	for _, p := range urls {
		videoID, err := extractor.ExtractVideoIDFromURL(p.s)
		if p.e != nil && assert.ErrorIs(t, err, p.e) {
			return
		}

		assert.Equal(t, videoID, wantID)
		if videoID != wantID {
			t.Errorf("ExtractVideoIDFromURL(%v): expect %v, got %v\n", p.s, wantID, videoID)
		}

	}
}
