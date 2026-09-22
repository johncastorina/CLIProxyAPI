# Local Security Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Produce a reviewed CLIProxyAPI fork that closes the concrete local security findings while retaining account routing and translation.

**Architecture:** Harden existing boundaries rather than adding a second proxy layer. Independent runtime, filesystem, and configuration/build changes form one integration slice; ownership is disjoint, followed by spec and quality review.

**Tech Stack:** Go 1.26, Gin, YAML, GitHub Actions.

**Spec:** docs/security/local-hardening-spec.md

## Global Constraints

- Baseline: upstream 555662940411a07460e9d24d14477a5f50dffdb5.
- Preserve native provider OAuth, account selection, protocol translation, and streaming.
- No real credentials, live provider inference, native plugin installation, or remote control-plane enrollment.
- EasyCLIProxyAPI is outside this change.
- Follow TDD: one failing behavioral test, minimal fix, package validation, repeat.
- Implementation workers use disjoint file ownership; controller commits integrated work.

## Task 1: Local hardening integration slice

### Component A: Management, updater, and xAI network boundaries

**Files:** internal/api/server.go; internal/api/handlers/management/handler.go; internal/managementasset/updater.go; internal/auth/xai/xai.go; matching tests in those packages. All xAI OAuth POST paths share redirect rejection, including device-code polling.
**Interfaces:** Existing Server/Middleware, EnsureLatestManagementHTML, and XAIAuth methods. No signature changes to callers.

- [x] Add a request-level regression with a remote socket peer, `X-Forwarded-For: 127.0.0.1`, and a valid management key; require 403. Confirm RED.
```go
req := httptest.NewRequest(http.MethodGet, "/protected", nil)
req.RemoteAddr = "192.0.2.10:1234"
req.Header.Set("X-Forwarded-For", "127.0.0.1")
req.Header.Set("Authorization", "Bearer test-management-key")
```
- [x] Derive authorization locality from `c.RemoteIP()` and disable default trusted proxies with `engine.SetTrustedProxies(nil)`; retain management key checks. Confirm real local peer works and remote opt-in still requires the key.
- [x] Test missing/malformed/mismatched panel digests and primary download failure with synthetic HTTP transport; require no installed replacement/fallback. Remove fallback, require 32-byte SHA-256 decoding before downloads, and retain verified-success behavior.
```go
digest, err := hex.DecodeString(remoteHash)
if err != nil || len(digest) != sha256.Size { return nil, nil }
```
- [x] Test xAI stored non-x.ai/HTTP/userinfo endpoint rejection without any transport call, plus allowed fake-token success. Validate before POST; validate redirects or reject them for OAuth requests. Test 307 redirect does not leak the fake token.
- [x] Run `go test ./internal/api ./internal/api/handlers/management ./internal/managementasset ./internal/auth/xai`, gofmt changed files, report RED/GREEN evidence and review changes.

### Component B: Private credential/config/log writes and disabled-log behavior

**Files:** new dependency-light internal/privatefile package and tests (internal/util imports config, so it cannot serve config writers); provider token writers under internal/auth; sdk/auth/filestore.go; internal/config/config_yaml.go; internal/api/handlers/management/config_basic.go, auth_files_crud.go, auth_files_fields.go; sdk/cliproxy/service_lifecycle.go; internal/runtime/executor/helps/logging_helpers.go; internal/logging/request_logger*.go; internal/api/middleware/request_logging.go and affected tests.
**Interfaces:** Existing provider SaveTokenToFile and config/logger APIs. A reusable helper may return `(*os.File, error)` with create/truncate or append flags and guaranteed private Unix mode.

- [x] Add one public writer regression using a temp directory and an existing 0644 destination; require 0600 after writing valid fake credentials. Confirm RED.
```go
info, err := os.Stat(path)
if err != nil { t.Fatal(err) }
if info.Mode().Perm() != 0o600 { t.Fatalf("secret mode = %o", info.Mode().Perm()) }
```
- [x] Implement private file opens before secret bytes are written. Reject symlink destinations and avoid changing unrelated parent directory permissions. Apply to audited provider token, configuration, and detailed log write paths. Add create, overwrite, and symlink behavioral coverage.
- [x] Add an error-response request through logging middleware with logging disabled; require no persisted conversation log. Confirm RED, then avoid capture/persistence entirely when request logging is off. Keep explicit logging success/error behavior with private output.
- [x] Run affected packages and provider tests, gofmt changed files, and report RED/GREEN evidence.

### Component C: Local defaults, dependency fixes, and nonpublishing CI

**Files:** internal/config/config_load.go, parse.go, config.go, config_types.go comments as needed; config.example.yaml; config tests; go.mod/go.sum; internal/store/gitstore.go and gitstore_test.go for dependency compatibility; .github/workflows/*; docs/security/local-use.md; scripts/local-security-smoke.py.
**Interfaces:** Existing config parsing/defaults, module dependency graph, read-only GitHub checks.

- [x] Test both LoadConfig and ParseConfigBytes with minimal YAML: host127.0.0.1, disable-control-panel true, disable-auto-update-panel true. Confirm RED; initialize defaults before YAML decode, preserving explicit opt-ins.
```go
if cfg.Host != "127.0.0.1" || !cfg.RemoteManagement.DisableControlPanel || !cfg.RemoteManagement.DisableAutoUpdatePanel {
    t.Fatal("unsafe local defaults")
}
```
- [x] Update audited dependencies with official Go proxy/checksum verification: go-git/v6 >=6.0.0-alpha.5, klauspost/compress >=1.18.7, x/crypto >=0.56.0. Resolve minimal required graph changes; report any Go version/API compatibility conflict before widening changes.
- [x] Preserve Git-store corruption behavior after the alpha.5 upgrade: close/reopen the test fixture around on-disk corruption so cached objects cannot conceal deleted packs; classify only wrapped missing Git pack PathErrors as corruption. Keep existing dirty-file/remote-data assertions unchanged and test ordinary missing files and permission errors remain distinct. Root RCA: three fixture assertions saw cached objects; one saw a newly wrapped filesystem error instead of the prior sentinels.
- [x] Replace inherited workflows with one read-only test/build workflow; resolve checkout/setup-go immutable SHAs from their official repositories. Use committed model catalog; no curl-install, model-refresh, release publication, or Docker push.
```yaml
permissions:
  contents: read
# CI commands after pinned checkout/setup:
# go mod verify
# go test -timeout 120s ./...
# go vet ./...
# go build -trimpath -o cli-proxy-api ./cmd/server
# ./cli-proxy-api --help
```
- [x] Add a stdlib Python smoke harness that launches the actual compiled server with temporary private config/auth storage, random fake client/management keys, a minimal child environment, `--local-model`, and no accounts. Verify model-route client authentication, management-route authentication, and disabled panel. Always terminate the child. CI statically compiles the Python and runs it after the Go build.
- [x] Document local source-build invocation, private working directory/umask, unique keys, disabled panel/Home/plugins, model catalog local mode, and residual administrator authority. No secrets or account identifiers in docs.
- [x] Run config tests, module verification, and workflow static validation; report pin provenance and checks.

### Integration and review

- [x] Controller runs complete tests, vet, builds the server, and invokes the real binary help path.
- [x] Fresh reviewer compares all changes to this spec; implementers resolve genuine gaps.
- [x] Fresh quality/security review after spec passes; resolve findings and rerun affected checks.
- [ ] Commit, push to the fork, open a PR against the fork's main, and attach it to the task. Do not merge/deploy/authenticate without an explicit user request.
- [ ] Clean the worktree after verifying it is clean and pushed at the exact remote commit; clean Go build cache only when no concurrent Go builds are active.
