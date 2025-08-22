# Copilot Instructions for `holydb`

Concise guidance for AI agents to be productive quickly in this repo. Focus on present, observable patterns (not aspirations).

## 1. Object Storage Focus (Current Core)
The implemented, test-covered core is a filesystem-backed, S3‑like minimal object store (`pkg/storage`). Higher-level DB (`pkg/db`) is a stub. Treat storage layer + HTTP server as the authoritative domain.

Key interface: `pkg/storage/storage.go` (`Storage`). Primary implementation: `LocalStorage` (`storage_local.go`). Objects are directories under `<Root>/<bucket>/<key>` containing `part.N` files plus `data.meta` (JSON). Directory keys end with `/` and create a directory containing a `.dir` marker (no parts / metadata file).

Multipart uploads: staged under `<bucket>/.multipart/<uploadID>/part.N` then atomically moved on `CompleteMultipart`.

Metadata: arbitrary `map[string]string` serialized deterministically into `data.meta`. Bucket metadata stored as `<bucket>/.bucket.meta` (`BucketMetadata`).

Stats: walks bucket; counts `part.*` sizes & number of object directories. Reconstruct: writes an output file with an 8‑byte big‑endian length prefix + selected metadata JSON + concatenated parts.

## 2. HTTP API Shape
HTTP server composition in `internal/server/server.go` (preferred) and a legacy duplicate in `cmd/holydb/server.go` (kept for backward compatibility tests; prefer NOT to extend this duplicate—update `internal/server/*`).

Routes are mounted under `/v1/storage` (see `contrib/openapi.yaml`). Handlers are split by concern:
- `handlers_objects.go`: PUT/GET/DELETE object (optionally multipart part upload via `?uploadId=&partNumber=`; metadata via header `X-Meta-JSON` JSON string)
- `handlers_multipart.go`: finalize / abort multipart (`POST ?complete=uploadId`, `DELETE ?abort=`)
- `handlers_bucket.go`: list objects (`GET /{bucket}?prefix=`) & start multipart session (`POST ?uploads=`)
- `handlers_bucket_meta.go`: bucket metadata read/write (`/{bucket}/.bucket.meta`)
- `handlers_stats.go`: stats (`/{bucket}/_stats`)
- `handlers_reconstruct.go`: reconstruct (`POST /{bucket}/_reconstruct/{key}?out=...&include=key1,key2`)

Always register new storage endpoints in `internal/server` and update `openapi.yaml` accordingly.

## 3. CLI Layer
Entry point: `main.go` -> `cmd/holydb/root.go`. Simple manual flag parsing (no Cobra). Subcommands:
- `serve --addr :8080 --root . [--background]`
- `version` / `--version`
- `help` / `help <command>`
Background mode forks a detached process (env `HOLYDB_BACKGROUND=1`). Prefer adding new commands by extending the `switch` in `Execute()`.

## 4. Project Conventions
- Go version declared: `go 1.24.5` (assume toolchain >= this).
- Only external dependency: `github.com/gorilla/mux`.
- Tests live alongside code (`*_test.go`) and exercise storage behaviors extensively (`pkg/storage/storage_test.go`). Preserve existing test semantics when modifying storage layout.
- Metadata & multipart directories start with a dot (`.bucket.meta`, `.multipart`) and are excluded in listings & stats via prefix checks. Maintain this convention for internal/system files (dot‑prefixed).
- Directory objects (keys ending with `/`) intentionally skip creating `data.meta`; code paths check `strings.HasSuffix(key, "/")`— replicate that logic if adding features involving directory keys.

## 5. Build & Dev Workflow
Primary commands (see `Makefile`):
- Build: `make build` (outputs binary `holydb` in repo root)
- Test: `make test` (runs `go test -v ./...`)
- Format/Vet: `make fmt`, `make vet`; optional lint: `make lint` (no hard dependency on golangci-lint).
Run server for manual testing: `./holydb serve --addr :8080 --root ./data` then interact per OpenAPI spec.

## 6. Extension Guidelines for Agents
When adding features:
- Prefer extending `Storage` interface only if HTTP/API needs new primitive; keep `LocalStorage` consistent. Update tests or add new focused tests under `pkg/storage`.
- For new HTTP endpoints: create a new `handlers_*.go` under `internal/server/`, register within `server.go`, document path in `openapi.yaml`. Follow existing pattern: one top-level function `RegisterXHandlers(r *mux.Router, ls storage.Storage)`.
- Keep legacy duplicate server (`cmd/holydb/server.go`) untouched unless fixing a bug mirrored from `internal/server`—avoid divergence; ideally plan deprecation.
- Use temp file + atomic rename pattern for writes requiring durability (see `writeMeta`, bucket meta) to avoid partial writes.
- Maintain deterministic ordering when concatenating parts or listing (current code sorts part filenames and collected keys).

## 7. Testing Patterns
- Use `t.TempDir()` for filesystem isolation.
- Multipart tests verify header reconstruction logic (8‑byte length prefix + JSON). Reuse helper patterns if extracting common code.
- Add tests mirroring new behaviors before refactoring core write/list logic to catch regressions.

## 8. Common Pitfalls / Gotchas
- Forgetting to exclude dot‑prefixed paths will leak internal data in listings/stats.
- Completing multipart for directory key is explicitly invalid (`CompleteMultipart` guard) — preserve this restriction.
- Reconstruct always forces inclusion of `filename` metadata if present; keep this invariant.
- Ensure readers from `Get` are fully closed (multiReadCloser manages cascading closes).

## 9. Future Stub Awareness
`pkg/db` is currently a placeholder; avoid building features that rely on unimplemented DB semantics. Treat it as out of scope unless explicitly prioritized.

## 10. Quick Reference File Map
- Interface: `pkg/storage/storage.go`
- Impl: `pkg/storage/storage_local.go`
- HTTP assembly: `internal/server/server.go`
- Handlers: `internal/server/handlers_*.go`
- CLI: `cmd/holydb/root.go`
- OpenAPI spec: `contrib/openapi.yaml`
- Tests: `pkg/storage/storage_test.go`, `internal/config/config_test.go`

Feedback welcome: clarify any unclear area (e.g. multipart flow, reconstruct format) for iterative refinement.
