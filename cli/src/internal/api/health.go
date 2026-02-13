package api

import (
	"encoding/json"
	"fmt"
)

// Health calls /api/health and returns its status string.
func (c *Client) Health() (string, error) {
	data, err := c.get("/api/health")
	if err != nil {
		return "", err
	}

	var payload struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	return payload.Status, nil
}
