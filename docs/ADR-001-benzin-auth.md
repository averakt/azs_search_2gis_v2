# ADR-001: Benzin API Authentication

## Status
Accepted

## Date
2026-07-31

## Context
The IMPLEMENTATION_PLAN.md and BENZIN_API.md indicated that `benzin.api.2gis.ru` requires JWE token authentication via `passepartout.2gis.com`. However, testing revealed that the Benzin API endpoints are publicly accessible without authentication.

## Decision
The `benzin.api.2gis.ru` API does **not** require authentication for read-only station search operations. The `PassepartoutURL` config option is retained for future use but is currently unused.

## Consequences
- No JWE token implementation needed at this time
- `internal/benzin/client.go` works without auth headers
- Monitor for API changes; if auth becomes required, implement token caching with TTL
- Update documentation to reflect actual API behavior

## Verification
Tested on 2026-07-31:
- `GET /api/v1/stations/search?...` returns 200 OK without Authorization header
- `GET /api/v1/stations/by-ids?...` returns 200 OK without Authorization header

## References
- docs/BENZIN_API.md - updated to reflect no-auth status
- internal/benzin/client.go - no auth headers added
