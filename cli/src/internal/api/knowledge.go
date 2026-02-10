package api

import "fmt"

// --- Knowledge Methods ---

func (c *Client) CreateKnowledge(input CreateKnowledgeInput) (*Knowledge, error) {
	data, err := c.post("/api/knowledge", input)
	if err != nil {
		return nil, err
	}
	return decodeOne[Knowledge](data)
}

func (c *Client) GetKnowledge(id string) (*Knowledge, error) {
	data, err := c.get(fmt.Sprintf("/api/knowledge/%s", id))
	if err != nil {
		return nil, err
	}
	return decodeOne[Knowledge](data)
}

func (c *Client) QueryKnowledge(params QueryParams) ([]Knowledge, error) {
	data, err := c.get(buildQuery("/api/knowledge", params))
	if err != nil {
		return nil, err
	}
	return decodeList[Knowledge](data)
}

func (c *Client) UpdateKnowledge(id string, input UpdateKnowledgeInput) (*Knowledge, error) {
	data, err := c.patch(fmt.Sprintf("/api/knowledge/%s", id), input)
	if err != nil {
		return nil, err
	}
	return decodeOne[Knowledge](data)
}

func (c *Client) LinkKnowledge(id, entityID string) error {
	body := map[string]string{"entity_id": entityID}
	_, err := c.post(fmt.Sprintf("/api/knowledge/%s/link", id), body)
	return err
}
