# VngCloud Go SDK

- Go version: 1.22 or later

# Highlights
- **BREAKING:** a consumer that recognises rate limiting by matching `EcPermissionDenied` no longer sees it. HTTP 429 now returns `EcTooManyRequests`; code that must treat throttling specially should match that, and code that reads `GetParameters()["statusCode"]` is unaffected.
- **BREAKING:** the module path moved from `github.com/vngcloud/vngcloud-go-sdk/v2` to `github.com/GreenNodeHub/vngcloud-go-sdk/v2`. The repository moved to the GreenNodeHub organization; the old repository remains but receives no new commits. Importers must update `go.mod` and every import path (`go get github.com/GreenNodeHub/vngcloud-go-sdk/v2`).

# Changes since v2.21.0
## :bug: Bug Fixes
- `DoRequest` reauthenticates at most once per call. An endpoint that answered 401 to every token, or IAM issuing tokens that `NeedReauth` already treats as stale, used to make it fetch a new token and call itself again with no bound: one IAM token request per round, a goroutine stack that grew until the process was OOM-killed, and a caller that never returned. A 401 that survives a fresh token now returns `EcUnauthorized` (`VngCloudApiUnauthorized`) with the usual `statusCode`/`url`/`method` parameters.
- A 401 that arrives after a redirect to another host no longer triggers reauthentication: Go drops the `Authorization` header on such a hop, so no token could help. The error is `EcUnauthorized` with a `redirectedTo` parameter and a message naming both URLs.
- HTTP 429 now carries its own error code `EcTooManyRequests` (`VngCloudApiTooManyRequests`) instead of sharing `EcPermissionDenied` with HTTP 403, so a throttle can be told apart from an authorisation failure without reading the raw status code. 403 is unchanged, and so are the error parameters (`statusCode`, `responseHeaders`, `method`, `url`).

## :sparkles: New Features
- `loadbalancer/v2`: listener requests and responses carry the ACL `blockedCidrs` and `defaultAction` fields.
- `compute/v2`: `GetServerConsoleLog` returns a server's serial console output (`GET /v2/{projectId}/servers/{serverId}/console-log`).

## :hammer: Others
- Move the Go module path to `github.com/GreenNodeHub/vngcloud-go-sdk/v2`.
