package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/gravitrone/nebula-core/cli/internal/config"
)

// loadCommandClient builds an API client for non-interactive command flows.
func loadCommandClient(requireAuth bool) (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		if requireAuth {
			return nil, fmt.Errorf("not logged in: %w", err)
		}
		return newDefaultClient(""), nil
	}
	return newDefaultClient(cfg.APIKey), nil
}

// writeCleanJSON renders predictable command output without banners.
func writeCleanJSON(out io.Writer, value any) error {
	if value == nil {
		value = map[string]any{"ok": true}
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(value)
}

// parseQueryParams converts repeated --param key=value flags into query params.
func parseQueryParams(raw []string) (api.QueryParams, error) {
	params := api.QueryParams{}
	for _, item := range raw {
		key, value, ok := strings.Cut(item, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid --param value %q (expected key=value)", item)
		}
		params[key] = strings.TrimSpace(value)
	}
	return params, nil
}

// readInputJSON reads JSON payload from --input or --input-file.
func readInputJSON(input string, inputFile string, required bool) (json.RawMessage, error) {
	if strings.TrimSpace(input) != "" && strings.TrimSpace(inputFile) != "" {
		return nil, fmt.Errorf("use either --input or --input-file, not both")
	}

	switch {
	case strings.TrimSpace(input) != "":
		raw := []byte(strings.TrimSpace(input))
		if !json.Valid(raw) {
			return nil, fmt.Errorf("invalid JSON passed to --input")
		}
		return raw, nil
	case strings.TrimSpace(inputFile) != "":
		raw, err := os.ReadFile(strings.TrimSpace(inputFile))
		if err != nil {
			return nil, fmt.Errorf("read input file: %w", err)
		}
		raw = []byte(strings.TrimSpace(string(raw)))
		if !json.Valid(raw) {
			return nil, fmt.Errorf("invalid JSON in --input-file")
		}
		return raw, nil
	default:
		if required {
			return nil, fmt.Errorf("missing input: pass --input '<json>' or --input-file <path>")
		}
		return nil, nil
	}
}
