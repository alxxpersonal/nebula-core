package api

import "time"

// DefaultBaseURL is the single source of truth for the CLI API target.
const DefaultBaseURL = "http://127.0.0.1:8000"

// NewDefaultClient builds a client pointed at the default Nebula API URL.
func NewDefaultClient(apiKey string, timeout ...time.Duration) *Client {
	return NewClient(DefaultBaseURL, apiKey, timeout...)
}
