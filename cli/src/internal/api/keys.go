package api

import "fmt"

// --- Key Methods ---

// Login performs first-run login (unauthenticated).
func (c *Client) Login(username string) (*LoginResponse, error) {
	data, err := c.post("/api/keys/login", LoginInput{Username: username})
	if err != nil {
		return nil, err
	}
	return decodeOne[LoginResponse](data)
}

func (c *Client) CreateKey(name string) (*CreateKeyResponse, error) {
	body := map[string]string{"name": name}
	data, err := c.post("/api/keys", body)
	if err != nil {
		return nil, err
	}
	return decodeOne[CreateKeyResponse](data)
}

func (c *Client) ListKeys() ([]APIKey, error) {
	data, err := c.get("/api/keys")
	if err != nil {
		return nil, err
	}
	return decodeList[APIKey](data)
}

func (c *Client) ListAllKeys() ([]APIKey, error) {
	data, err := c.get("/api/keys/all")
	if err != nil {
		return nil, err
	}
	return decodeList[APIKey](data)
}

func (c *Client) RevokeKey(keyID string) error {
	_, err := c.del(fmt.Sprintf("/api/keys/%s", keyID))
	return err
}
