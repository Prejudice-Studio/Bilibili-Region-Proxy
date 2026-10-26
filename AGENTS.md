# Repository working rules

## Scope

- This repository is the Go HTTP proxy for Bilibili main-site API requests.
- Keep changes focused on this project. Do not add unrelated client, deployment, or reverse-engineering artifacts.
- The SEA bangumi adapter uses the Bilibili `/zh-Hans/bangumi` web search page and returns the normal bangumi search response shape.

## Go conventions

- Target Go 1.23 as declared in `go.mod`.
- Format all Go files with `gofmt`.
- Keep proxy behavior in `internal/proxy`; keep process flags and server startup in `main.go`.
- Prefer the standard library. Explain any new direct dependency in the pull request.
- Keep credentials, account data, captured traffic, local configs, and build outputs out of source control.
- Never log query strings, cookies, authorization values, access keys, or device identifiers.

## Before submitting

- Run `gofmt` on changed Go files.
- Run `go test ./...`, `go vet ./...`, and `go build ./...`.
- Update the Chinese README when command-line behavior, routes, configuration, or response behavior changes.
- Add or update focused tests when changing request routing, parsing, configuration, or result injection.
- Use a descriptive Conventional Commit message, such as `fix(proxy): preserve intl search parameters`.
