package iplookup

import "fmt"

// HTTPError is returned when the API responds with a non-2xx status.
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("ip lookup endpoint returned HTTP %d: %s", e.StatusCode, e.Body)
}
