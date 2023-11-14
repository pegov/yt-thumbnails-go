package downloader

import "errors"

var (
	ErrServerError           = errors.New("server error")
	ErrNotFound              = errors.New("not found")
	ErrCouldNotCreateRequest = errors.New("error creating http request")
	ErrCouldNotMakeRequest   = errors.New("error making http request")
	ErrTimeout               = errors.New("timeout")
	ErrCouldNotReadBody      = errors.New("error reading the body")
	ErrCouldNotUnmarshalBody = errors.New("error unmarshaling the body")
)
