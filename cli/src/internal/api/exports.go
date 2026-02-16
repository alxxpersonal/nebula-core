package api

// ExportEntities exports entities with optional query params.
func (c *Client) ExportEntities(params QueryParams) (*ExportResult, error) {
	data, err := c.get(buildQuery("/api/export/entities", params))
	if err != nil {
		return nil, err
	}
	return decodeOne[ExportResult](data)
}

// ExportContext exports context items with optional query params.
func (c *Client) ExportContextItems(params QueryParams) (*ExportResult, error) {
	data, err := c.get(buildQuery("/api/export/context", params))
	if err != nil {
		return nil, err
	}
	return decodeOne[ExportResult](data)
}

// ExportRelationships exports relationships with optional query params.
func (c *Client) ExportRelationships(params QueryParams) (*ExportResult, error) {
	data, err := c.get(buildQuery("/api/export/relationships", params))
	if err != nil {
		return nil, err
	}
	return decodeOne[ExportResult](data)
}

// ExportJobs exports jobs with optional query params.
func (c *Client) ExportJobs(params QueryParams) (*ExportResult, error) {
	data, err := c.get(buildQuery("/api/export/jobs", params))
	if err != nil {
		return nil, err
	}
	return decodeOne[ExportResult](data)
}

// ExportContext exports context dump (entities + context + relationships + jobs).
func (c *Client) ExportContext(params QueryParams) (*ExportResult, error) {
	data, err := c.get(buildQuery("/api/export/snapshot", params))
	if err != nil {
		return nil, err
	}
	return decodeOne[ExportResult](data)
}
