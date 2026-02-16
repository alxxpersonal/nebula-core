package api

// BulkImportRequest defines payload for bulk imports.
type BulkImportRequest struct {
	Format   string           `json:"format"`
	Data     string           `json:"data,omitempty"`
	Items    []map[string]any `json:"items,omitempty"`
	Defaults map[string]any   `json:"defaults,omitempty"`
}

// ImportEntities sends a bulk entity import request.
func (c *Client) ImportEntities(payload BulkImportRequest) (*BulkImportResult, error) {
	data, err := c.post("/api/import/entities", payload)
	if err != nil {
		return nil, err
	}
	return decodeOne[BulkImportResult](data)
}

// ImportContext sends a bulk context import request.
func (c *Client) ImportContext(payload BulkImportRequest) (*BulkImportResult, error) {
	data, err := c.post("/api/import/context", payload)
	if err != nil {
		return nil, err
	}
	return decodeOne[BulkImportResult](data)
}

// ImportRelationships sends a bulk relationship import request.
func (c *Client) ImportRelationships(payload BulkImportRequest) (*BulkImportResult, error) {
	data, err := c.post("/api/import/relationships", payload)
	if err != nil {
		return nil, err
	}
	return decodeOne[BulkImportResult](data)
}

// ImportJobs sends a bulk job import request.
func (c *Client) ImportJobs(payload BulkImportRequest) (*BulkImportResult, error) {
	data, err := c.post("/api/import/jobs", payload)
	if err != nil {
		return nil, err
	}
	return decodeOne[BulkImportResult](data)
}
