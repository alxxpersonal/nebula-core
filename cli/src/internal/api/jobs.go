package api

import "fmt"

// --- Job Methods ---

func (c *Client) CreateJob(input CreateJobInput) (*Job, error) {
	data, err := c.post("/api/jobs", input)
	if err != nil {
		return nil, err
	}
	return decodeOne[Job](data)
}

func (c *Client) GetJob(id string) (*Job, error) {
	data, err := c.get(fmt.Sprintf("/api/jobs/%s", id))
	if err != nil {
		return nil, err
	}
	return decodeOne[Job](data)
}

func (c *Client) QueryJobs(params QueryParams) ([]Job, error) {
	data, err := c.get(buildQuery("/api/jobs", params))
	if err != nil {
		return nil, err
	}
	return decodeList[Job](data)
}

func (c *Client) UpdateJobStatus(id, status string) (*Job, error) {
	body := map[string]string{"status": status}
	data, err := c.patch(fmt.Sprintf("/api/jobs/%s/status", id), body)
	if err != nil {
		return nil, err
	}
	return decodeOne[Job](data)
}

func (c *Client) UpdateJob(id string, input UpdateJobInput) (*Job, error) {
	data, err := c.patch(fmt.Sprintf("/api/jobs/%s", id), input)
	if err != nil {
		return nil, err
	}
	return decodeOne[Job](data)
}

func (c *Client) CreateSubtask(jobID string, input map[string]string) (*Job, error) {
	data, err := c.post(fmt.Sprintf("/api/jobs/%s/subtasks", jobID), input)
	if err != nil {
		return nil, err
	}
	return decodeOne[Job](data)
}
