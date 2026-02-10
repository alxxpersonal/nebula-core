package ui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gravitrone/nebula-core/cli/internal/ui/components"
)

func parseMetadataInput(input string) (map[string]any, error) {
	if strings.TrimSpace(input) == "" {
		return nil, nil
	}
	root := map[string]any{}
	stack := []map[string]any{root}

	lines := strings.Split(input, "\n")
	for idx, raw := range lines {
		lineNum := idx + 1
		line := strings.TrimRight(raw, " \t")
		if strings.TrimSpace(line) == "" {
			continue
		}
		spaces := leadingSpaces(line)
		if spaces%2 != 0 {
			return nil, fmt.Errorf("line %d: indent must use 2 spaces", lineNum)
		}
		level := spaces / 2
		if level > len(stack)-1 {
			return nil, fmt.Errorf("line %d: indent has no parent key", lineNum)
		}
		if level < len(stack)-1 {
			stack = stack[:level+1]
		}
		content := strings.TrimSpace(line)
		if strings.HasPrefix(content, "- ") {
			return nil, fmt.Errorf("line %d: list items not supported, use key: [a, b]", lineNum)
		}
		parts := strings.SplitN(content, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("line %d: missing ':' separator", lineNum)
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			return nil, fmt.Errorf("line %d: key is empty", lineNum)
		}
		valueRaw := strings.TrimSpace(parts[1])
		current := stack[len(stack)-1]
		if valueRaw == "" {
			child := map[string]any{}
			current[key] = child
			stack = append(stack, child)
			continue
		}
		value, err := parseMetadataValue(valueRaw, lineNum)
		if err != nil {
			return nil, err
		}
		current[key] = value
	}
	return root, nil
}

func parseMetadataValue(raw string, lineNum int) (any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]") {
		inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(raw, "["), "]"))
		if inner == "" {
			return []any{}, nil
		}
		parts := strings.Split(inner, ",")
		items := make([]any, 0, len(parts))
		for _, part := range parts {
			items = append(items, parseMetadataScalar(strings.TrimSpace(part)))
		}
		return items, nil
	}
	if strings.HasPrefix(raw, "{") && strings.HasSuffix(raw, "}") {
		return nil, fmt.Errorf("line %d: inline objects not supported yet", lineNum)
	}
	return parseMetadataScalar(raw), nil
}

func parseMetadataScalar(raw string) any {
	if raw == "" {
		return ""
	}
	lower := strings.ToLower(raw)
	switch lower {
	case "true":
		return true
	case "false":
		return false
	case "null", "nil":
		return nil
	}
	if i, err := strconv.Atoi(raw); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(raw, 64); err == nil {
		return f
	}
	if (strings.HasPrefix(raw, "\"") && strings.HasSuffix(raw, "\"")) ||
		(strings.HasPrefix(raw, "'") && strings.HasSuffix(raw, "'")) {
		return strings.Trim(raw, "\"'")
	}
	return raw
}

func metadataToInput(data map[string]any) string {
	if len(data) == 0 {
		return ""
	}
	lines := metadataInputLines(data, 0)
	return strings.Join(lines, "\n")
}

func metadataInputLines(data map[string]any, indent int) []string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var lines []string
	pad := strings.Repeat(" ", indent)
	for _, k := range keys {
		switch typed := data[k].(type) {
		case map[string]any:
			lines = append(lines, pad+k+":")
			lines = append(lines, metadataInputLines(typed, indent+2)...)
		default:
			value := formatMetadataValue(typed)
			lines = append(lines, fmt.Sprintf("%s%s: %s", pad, k, value))
		}
	}
	return lines
}

func formatMetadataValue(value any) string {
	switch typed := value.(type) {
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, formatMetadataInline(item))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]any:
		b, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprintf("%v", typed)
		}
		return string(b)
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func formatMetadataInline(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func renderMetadataInput(input string) string {
	if strings.TrimSpace(input) == "" {
		return "-"
	}
	lines := strings.Split(input, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		spaces := leadingSpaces(line)
		content := strings.TrimSpace(line)
		pad := strings.Repeat(" ", spaces)

		if strings.HasPrefix(content, "- ") {
			value := strings.TrimSpace(strings.TrimPrefix(content, "- "))
			lines[i] = pad + MetaPunctStyle.Render("- ") + MetaValueStyle.Render(value)
			continue
		}

		if idx := strings.Index(content, ":"); idx != -1 {
			key := strings.TrimSpace(content[:idx])
			rest := strings.TrimSpace(content[idx+1:])
			rendered := MetaKeyStyle.Render(key) + MetaPunctStyle.Render(":")
			if rest != "" {
				rendered += " " + MetaValueStyle.Render(rest)
			}
			lines[i] = pad + rendered
			continue
		}

		lines[i] = pad + MetaValueStyle.Render(content)
	}
	return strings.Join(lines, "\n")
}

func leadingSpaces(s string) int {
	count := 0
	for _, r := range s {
		if r != ' ' {
			break
		}
		count++
	}
	return count
}

func metadataPreview(data map[string]any, maxLen int) string {
	if len(data) == 0 || maxLen <= 0 {
		return ""
	}
	keys := []string{"summary", "notes", "content", "url", "author"}
	for _, key := range keys {
		if val, ok := data[key]; ok {
			if preview := metadataValuePreview(val, maxLen); preview != "" {
				return preview
			}
		}
	}
	sorted := make([]string, 0, len(data))
	for k := range data {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	if len(sorted) == 0 {
		return ""
	}
	return metadataValuePreview(data[sorted[0]], maxLen)
}

func metadataValuePreview(value any, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return truncateString(strings.TrimSpace(typed), maxLen)
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, fmt.Sprintf("%v", item))
		}
		return truncateString(strings.Join(parts, ", "), maxLen)
	default:
		return truncateString(fmt.Sprintf("%v", value), maxLen)
	}
}

func renderMetadataBlock(data map[string]any, width int, expanded bool) string {
	if len(data) == 0 {
		return ""
	}
	if expanded {
		return components.MetadataTable(data, width)
	}
	lines := metadataLines(data, 0)
	maxLines := 6
	if len(lines) > maxLines {
		lines = append(lines[:maxLines], MutedStyle.Render("..."))
	}
	return components.TitledBox("Metadata", strings.Join(lines, "\n"), width)
}

func metadataLines(data map[string]any, indent int) []string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var lines []string
	pad := strings.Repeat(" ", indent)
	for _, k := range keys {
		switch typed := data[k].(type) {
		case map[string]any:
			lines = append(lines, pad+k+":")
			lines = append(lines, metadataLines(typed, indent+2)...)
		default:
			lines = append(lines, fmt.Sprintf("%s%s: %s", pad, k, formatMetadataValue(typed)))
		}
	}
	return lines
}
