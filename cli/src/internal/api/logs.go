package api

import "fmt"

// CreateLog creates a new log entry.
func (c *Client) CreateLog(input CreateLogInput) (*Log, error) {
	data, err := c.post("/api/logs", input)
	if err != nil {
		return nil, err
	}
	return decodeOne[Log](data)
}

// GetLog retrieves a log entry by id.
func (c *Client) GetLog(id string) (*Log, error) {
	data, err := c.get(fmt.Sprintf("/api/logs/%s", id))
	if err != nil {
		return nil, err
	}
	return decodeOne[Log](data)
}

// QueryLogs queries log entries using optional parameters.
func (c *Client) QueryLogs(params QueryParams) ([]Log, error) {
	data, err := c.get(buildQuery("/api/logs", params))
	if err != nil {
		return nil, err
	}
	return decodeList[Log](data)
}

// UpdateLog updates a log entry by id.
func (c *Client) UpdateLog(id string, input UpdateLogInput) (*Log, error) {
	data, err := c.patch(fmt.Sprintf("/api/logs/%s", id), input)
	if err != nil {
		return nil, err
	}
	return decodeOne[Log](data)
}
