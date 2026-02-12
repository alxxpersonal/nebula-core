package api

// SemanticSearchResult is a ranked semantic match item from /api/search/semantic.
type SemanticSearchResult struct {
	Kind     string  `json:"kind"`
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Subtitle string  `json:"subtitle"`
	Snippet  string  `json:"snippet"`
	Score    float64 `json:"score"`
}

// SemanticSearch executes semantic search across supported kinds.
func (c *Client) SemanticSearch(query string, kinds []string, limit int) ([]SemanticSearchResult, error) {
	if limit <= 0 {
		limit = 20
	}
	payload := map[string]any{
		"query": query,
		"kinds": kinds,
		"limit": limit,
	}
	data, err := c.post("/api/search/semantic", payload)
	if err != nil {
		return nil, err
	}
	return decodeList[SemanticSearchResult](data)
}
