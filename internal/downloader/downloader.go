package downloader

import "errors"

var (
	ErrServerError           = errors.New("server error")
	ErrNotFound              = errors.New("not found")
	ErrCouldNotMakeRequest   = errors.New("error making http request")
	ErrCouldNotReadBody      = errors.New("error reading the body")
	ErrCouldNotUnmarshalBody = errors.New("error unmarshaling the body")
)
