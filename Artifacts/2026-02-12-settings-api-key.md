# 2026-02-12 Settings Tab and API Key Flow

## Implemented

- Renamed CLI navigation label from **Profile** to **Settings**.
- Added palette action `tab:settings` and kept compatibility for legacy `tab:profile`.
- Added direct API key edit flow in Settings:
  - press `k` to open API key input prompt
  - `enter` saves key to config
  - in-memory client key is updated immediately
- Added masked API key display in Settings overview table.

## Request Wiring

- Added `Client.SetAPIKey(...)` in `cli/internal/api/client.go`.
- Settings API key save flow now updates both:
  - persisted config (`~/.nebula/config`)
  - active API client used by the running TUI session

## Tests

- Added `TestSetAPIKeyUpdatesSubsequentRequests` in:
  - `cli/src/internal/api/client_test.go`
- Added persistence + request wiring coverage in:
  - `cli/src/internal/ui/profile_test.go` (`TestProfileSetAPIKeyPersistsAndUpdatesClient`)
- Updated palette action coverage:
  - `cli/src/internal/ui/app_test.go` now validates `tab:settings`.

## Validation

- `cd cli/src && go test ./internal/api ./internal/ui -count=1`
- `./scripts/lint.sh`

