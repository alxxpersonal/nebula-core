package ui

import (
	"fmt"
	"strings"

	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/gravitrone/nebula-core/cli/internal/ui/components"
)

func relationshipSummaryRows(nodeType, nodeID string, rels []api.Relationship, maxRows int) []components.TableRow {
	if maxRows <= 0 {
		maxRows = 5
	}
	rows := make([]components.TableRow, 0, maxRows+1)
	for i, rel := range rels {
		if i >= maxRows {
			break
		}
		relType := strings.TrimSpace(components.SanitizeOneLine(rel.Type))
		if relType == "" {
			relType = "-"
		}
		direction, endpoint := relationshipDirectionAndEndpoint(nodeType, nodeID, rel)
		rows = append(rows, components.TableRow{
			Label: fmt.Sprintf("%s %d", relType, i+1),
			Value: fmt.Sprintf("%s %s", direction, endpoint),
		})
	}
	if extra := len(rels) - maxRows; extra > 0 {
		rows = append(rows, components.TableRow{
			Label: "More",
			Value: fmt.Sprintf("+%d relationships", extra),
		})
	}
	return rows
}

func relationshipDirectionAndEndpoint(nodeType, nodeID string, rel api.Relationship) (string, string) {
	sourceID := strings.TrimSpace(rel.SourceID)
	targetID := strings.TrimSpace(rel.TargetID)
	sourceType := strings.TrimSpace(strings.ToLower(rel.SourceType))
	targetType := strings.TrimSpace(strings.ToLower(rel.TargetType))

	sourceLabel := relationshipNodeLabel(rel.SourceName, sourceID, sourceType)
	targetLabel := relationshipNodeLabel(rel.TargetName, targetID, targetType)

	switch {
	case sourceType == nodeType && sourceID == nodeID:
		return "->", targetLabel
	case targetType == nodeType && targetID == nodeID:
		return "<-", sourceLabel
	default:
		return "<>", fmt.Sprintf("%s <-> %s", sourceLabel, targetLabel)
	}
}

func relationshipNodeLabel(name, nodeID, nodeType string) string {
	clean := strings.TrimSpace(components.SanitizeOneLine(name))
	if clean != "" {
		return clean
	}
	switch strings.TrimSpace(nodeType) {
	case "entity":
		return "entity:" + shortID(nodeID)
	case "context":
		return "context:" + shortID(nodeID)
	case "job":
		return "job:" + shortID(nodeID)
	case "log":
		return "log:" + shortID(nodeID)
	case "file":
		return "file:" + shortID(nodeID)
	case "protocol":
		return "protocol:" + shortID(nodeID)
	case "agent":
		return "agent:" + shortID(nodeID)
	default:
		if strings.TrimSpace(nodeID) != "" {
			return shortID(nodeID)
		}
		return "unknown"
	}
}
