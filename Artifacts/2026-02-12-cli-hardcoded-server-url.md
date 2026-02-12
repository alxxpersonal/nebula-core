# 2026-02-12 - CLI hardcoded server URL

## Scope
Task 3: remove user-facing server URL configuration and hardcode one base URL for CLI network calls.

## Changes
- Added single source of truth constant in `cli/src/internal/api/base.go`:
  - `DefaultBaseURL = "http://localhost:8000"`
  - `NewDefaultClient(apiKey)` helper.
- Removed `server_url` from CLI config schema in `cli/src/internal/config/config.go`.
- Kept backward compatibility with existing config files:
  - legacy `server_url` key is ignored on load.
  - only `api_key` is required.
- Updated runtime wiring to use default client:
  - `cli/src/cmd/nebula/main.go`
  - `cli/src/internal/cmd/login.go`
  - `cli/src/internal/cmd/agent.go`
  - `cli/src/internal/cmd/keys.go`
- Removed server row from Settings table in `cli/src/internal/ui/profile.go`.

## Tests
- Updated config tests for legacy field handling in `cli/src/internal/config/config_test.go`.
- Updated settings/profile tests in `cli/src/internal/ui/profile_test.go`.
- Ran:
  - `cd cli/src && go test ./... -count=1`
  - `./scripts/lint.sh`
