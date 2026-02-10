package api

// ExportEntities exports entities with optional query params.
func (c *Client) ExportEntities(params QueryParams) (*ExportResult, error) {
	data, err := c.get(buildQuery("/api/exports/entities", params))
	if err != nil {
		return nil, err
	}
	return decodeOne[ExportResult](data)
}

// ExportKnowledge exports knowledge items with optional query params.
func (c *Client) ExportKnowledge(params QueryParams) (*ExportResult, error) {
	data, err := c.get(buildQuery("/api/exports/knowledge", params))
	if err != nil {
		return nil, err
	}
	return decodeOne[ExportResult](data)
}

// ExportRelationships exports relationships with optional query params.
func (c *Client) ExportRelationships(params QueryParams) (*ExportResult, error) {
	data, err := c.get(buildQuery("/api/exports/relationships", params))
	if err != nil {
		return nil, err
	}
	return decodeOne[ExportResult](data)
}

// ExportJobs exports jobs with optional query params.
func (c *Client) ExportJobs(params QueryParams) (*ExportResult, error) {
	data, err := c.get(buildQuery("/api/exports/jobs", params))
	if err != nil {
		return nil, err
	}
	return decodeOne[ExportResult](data)
}

// ExportContext exports context dump (entities + knowledge + relationships + jobs).
func (c *Client) ExportContext(params QueryParams) (*ExportResult, error) {
	data, err := c.get(buildQuery("/api/exports/context", params))
	if err != nil {
		return nil, err
	}
	return decodeOne[ExportResult](data)
}
