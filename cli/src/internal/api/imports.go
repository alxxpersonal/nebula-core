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
	data, err := c.post("/api/imports/entities", payload)
	if err != nil {
		return nil, err
	}
	return decodeOne[BulkImportResult](data)
}

// ImportKnowledge sends a bulk knowledge import request.
func (c *Client) ImportKnowledge(payload BulkImportRequest) (*BulkImportResult, error) {
	data, err := c.post("/api/imports/knowledge", payload)
	if err != nil {
		return nil, err
	}
	return decodeOne[BulkImportResult](data)
}

// ImportRelationships sends a bulk relationship import request.
func (c *Client) ImportRelationships(payload BulkImportRequest) (*BulkImportResult, error) {
	data, err := c.post("/api/imports/relationships", payload)
	if err != nil {
		return nil, err
	}
	return decodeOne[BulkImportResult](data)
}

// ImportJobs sends a bulk job import request.
func (c *Client) ImportJobs(payload BulkImportRequest) (*BulkImportResult, error) {
	data, err := c.post("/api/imports/jobs", payload)
	if err != nil {
		return nil, err
	}
	return decodeOne[BulkImportResult](data)
}
