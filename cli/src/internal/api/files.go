package api

import "fmt"

// CreateFile creates a new file metadata entry.
func (c *Client) CreateFile(input CreateFileInput) (*File, error) {
	data, err := c.post("/api/files", input)
	if err != nil {
		return nil, err
	}
	return decodeOne[File](data)
}

// GetFile retrieves a file entry by id.
func (c *Client) GetFile(id string) (*File, error) {
	data, err := c.get(fmt.Sprintf("/api/files/%s", id))
	if err != nil {
		return nil, err
	}
	return decodeOne[File](data)
}

// QueryFiles queries file entries using optional parameters.
func (c *Client) QueryFiles(params QueryParams) ([]File, error) {
	data, err := c.get(buildQuery("/api/files", params))
	if err != nil {
		return nil, err
	}
	return decodeList[File](data)
}

// UpdateFile updates a file entry by id.
func (c *Client) UpdateFile(id string, input UpdateFileInput) (*File, error) {
	data, err := c.patch(fmt.Sprintf("/api/files/%s", id), input)
	if err != nil {
		return nil, err
	}
	return decodeOne[File](data)
}
