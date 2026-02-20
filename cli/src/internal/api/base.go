package api

import "time"

// DefaultAPIPort is the default local Nebula API port used by the CLI.
const DefaultAPIPort = 8765

// DefaultBaseURL is the single source of truth for the CLI API target.
const DefaultBaseURL = "http://127.0.0.1:8765"

// NewDefaultClient builds a client pointed at the default Nebula API URL.
func NewDefaultClient(apiKey string, timeout ...time.Duration) *Client {
	return NewClient(DefaultBaseURL, apiKey, timeout...)
}
