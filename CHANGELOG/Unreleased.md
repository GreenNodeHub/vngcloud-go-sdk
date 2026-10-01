# VngCloud Go SDK

- Go version: 1.22 or later

# Highlights
- **BREAKING:** the module path moved from `github.com/vngcloud/vngcloud-go-sdk/v2` to `github.com/GreenNodeHub/vngcloud-go-sdk/v2`. The repository moved to the GreenNodeHub organization; the old repository remains but receives no new commits. Importers must update `go.mod` and every import path (`go get github.com/GreenNodeHub/vngcloud-go-sdk/v2`).

# Changes since v2.21.0
## :bug: Bug Fixes
_NONE_

## :sparkles: New Features
- `loadbalancer/v2`: listener requests and responses carry the ACL `blockedCidrs` and `defaultAction` fields.

## :hammer: Others
- Move the Go module path to `github.com/GreenNodeHub/vngcloud-go-sdk/v2`.
