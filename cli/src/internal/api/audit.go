package api

import "fmt"

// QueryAuditLog retrieves audit log entries with optional filters.
func (c *Client) QueryAuditLog(params QueryParams) ([]AuditEntry, error) {
	data, err := c.get(buildQuery("/api/audit", params))
	if err != nil {
		return nil, err
	}
	return decodeList[AuditEntry](data)
}

// QueryAuditLogWithPagination builds common audit params.
func (c *Client) QueryAuditLogWithPagination(
	tableName string,
	action string,
	actorType string,
	actorID string,
	recordID string,
	scopeID string,
	limit int,
	offset int,
) ([]AuditEntry, error) {
	params := QueryParams{}
	if tableName != "" {
		params["table"] = tableName
	}
	if action != "" {
		params["action"] = action
	}
	if actorType != "" {
		params["actor_type"] = actorType
	}
	if actorID != "" {
		params["actor_id"] = actorID
	}
	if recordID != "" {
		params["record_id"] = recordID
	}
	if scopeID != "" {
		params["scope_id"] = scopeID
	}
	params["limit"] = fmt.Sprintf("%d", limit)
	params["offset"] = fmt.Sprintf("%d", offset)
	return c.QueryAuditLog(params)
}

// ListAuditScopes retrieves privacy scopes with usage stats.
func (c *Client) ListAuditScopes() ([]AuditScope, error) {
	data, err := c.get("/api/audit/scopes")
	if err != nil {
		return nil, err
	}
	return decodeList[AuditScope](data)
}

// ListAuditActors retrieves audit actor summaries.
func (c *Client) ListAuditActors(actorType string) ([]AuditActor, error) {
	params := QueryParams{}
	if actorType != "" {
		params["actor_type"] = actorType
	}
	data, err := c.get(buildQuery("/api/audit/actors", params))
	if err != nil {
		return nil, err
	}
	return decodeList[AuditActor](data)
}
