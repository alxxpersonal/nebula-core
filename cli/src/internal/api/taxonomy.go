package api

import "fmt"

// ListTaxonomy returns taxonomy rows for the given kind.
func (c *Client) ListTaxonomy(kind string, includeInactive bool, search string, limit, offset int) ([]TaxonomyEntry, error) {
	params := QueryParams{}
	if includeInactive {
		params["include_inactive"] = "true"
	}
	if search != "" {
		params["search"] = search
	}
	if limit > 0 {
		params["limit"] = fmt.Sprintf("%d", limit)
	}
	if offset > 0 {
		params["offset"] = fmt.Sprintf("%d", offset)
	}
	data, err := c.get(buildQuery(fmt.Sprintf("/api/taxonomy/%s", kind), params))
	if err != nil {
		return nil, err
	}
	return decodeList[TaxonomyEntry](data)
}

// CreateTaxonomy inserts a new taxonomy row.
func (c *Client) CreateTaxonomy(kind string, input CreateTaxonomyInput) (*TaxonomyEntry, error) {
	data, err := c.post(fmt.Sprintf("/api/taxonomy/%s", kind), input)
	if err != nil {
		return nil, err
	}
	return decodeOne[TaxonomyEntry](data)
}

// UpdateTaxonomy updates an existing taxonomy row.
func (c *Client) UpdateTaxonomy(kind, id string, input UpdateTaxonomyInput) (*TaxonomyEntry, error) {
	data, err := c.patch(fmt.Sprintf("/api/taxonomy/%s/%s", kind, id), input)
	if err != nil {
		return nil, err
	}
	return decodeOne[TaxonomyEntry](data)
}

// ArchiveTaxonomy marks a taxonomy row as inactive.
func (c *Client) ArchiveTaxonomy(kind, id string) (*TaxonomyEntry, error) {
	data, err := c.post(fmt.Sprintf("/api/taxonomy/%s/%s/archive", kind, id), nil)
	if err != nil {
		return nil, err
	}
	return decodeOne[TaxonomyEntry](data)
}

// ActivateTaxonomy marks a taxonomy row as active.
func (c *Client) ActivateTaxonomy(kind, id string) (*TaxonomyEntry, error) {
	data, err := c.post(fmt.Sprintf("/api/taxonomy/%s/%s/activate", kind, id), nil)
	if err != nil {
		return nil, err
	}
	return decodeOne[TaxonomyEntry](data)
}
