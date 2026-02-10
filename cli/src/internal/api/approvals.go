package api

import "fmt"

// --- Approval Methods ---

func (c *Client) GetPendingApprovals() ([]Approval, error) {
	data, err := c.get("/api/approvals/pending")
	if err != nil {
		return nil, err
	}
	return decodeList[Approval](data)
}

func (c *Client) GetApproval(id string) (*Approval, error) {
	data, err := c.get(fmt.Sprintf("/api/approvals/%s", id))
	if err != nil {
		return nil, err
	}
	return decodeOne[Approval](data)
}

func (c *Client) ApproveRequest(id string) (*Approval, error) {
	data, err := c.post(fmt.Sprintf("/api/approvals/%s/approve", id), nil)
	if err != nil {
		return nil, err
	}
	return decodeOne[Approval](data)
}

func (c *Client) RejectRequest(id string, notes string) (*Approval, error) {
	body := map[string]string{"review_notes": notes}
	data, err := c.post(fmt.Sprintf("/api/approvals/%s/reject", id), body)
	if err != nil {
		return nil, err
	}
	return decodeOne[Approval](data)
}

func (c *Client) GetApprovalDiff(id string) (*ApprovalDiff, error) {
	data, err := c.get(fmt.Sprintf("/api/approvals/%s/diff", id))
	if err != nil {
		return nil, err
	}
	return decodeOne[ApprovalDiff](data)
}
