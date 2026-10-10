# VngCloud Go SDK

- Go version: 1.22 or later

# Highlights
- **BREAKING:** a consumer that recognises rate limiting by matching `EcPermissionDenied` no longer sees it. HTTP 429 now returns `EcTooManyRequests`; code that must treat throttling specially should match that, and code that reads `GetParameters()["statusCode"]` is unaffected.
- **BREAKING:** the module path moved from `github.com/vngcloud/vngcloud-go-sdk/v2` to `github.com/GreenNodeHub/vngcloud-go-sdk/v2`. The repository moved to the GreenNodeHub organization; the old repository remains but receives no new commits. Importers must update `go.mod` and every import path (`go get github.com/GreenNodeHub/vngcloud-go-sdk/v2`).

# Changes since v2.21.0
## :bug: Bug Fixes
- HTTP 429 now carries its own error code `EcTooManyRequests` (`VngCloudApiTooManyRequests`) instead of sharing `EcPermissionDenied` with HTTP 403, so a throttle can be told apart from an authorisation failure without reading the raw status code. 403 is unchanged, and so are the error parameters (`statusCode`, `responseHeaders`, `method`, `url`).

## :sparkles: New Features
- `loadbalancer/v2`: listener requests and responses carry the ACL `blockedCidrs` and `defaultAction` fields.
- `compute/v2`: `GetServerConsoleLog` returns a server's serial console output (`GET /v2/{projectId}/servers/{serverId}/console-log`).

## :hammer: Others
- Move the Go module path to `github.com/GreenNodeHub/vngcloud-go-sdk/v2`.
