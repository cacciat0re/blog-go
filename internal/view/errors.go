package view

import (
	"net/http"
)

// HTTPError is a controller error that already knows which status page it should
// produce. Controllers return one when the outcome is part of the site's
// contract (an article that does not exist is a 404); anything else is an
// internal error.
type HTTPError struct {
	Status int
	Err    error
}

func (e *HTTPError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return http.StatusText(e.Status)
}

func (e *HTTPError) Unwrap() error { return e.Err }

// NotFound returns the 404 sentinel, wrapping the cause when there is one.
func NotFound(err error) *HTTPError {
	return &HTTPError{Status: http.StatusNotFound, Err: err}
}

// Internal returns the 500 sentinel.
func Internal(err error) *HTTPError {
	return &HTTPError{Status: http.StatusInternalServerError, Err: err}
}
