# Worker Fixer Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Worker Fixer remediation activity that patches SQLi, validates with a mini-scan, and opens a PR only when validation passes.

**Architecture:** The core activity is pure Go and depends on `GitProvider` and `Scanner` interfaces. HTTP GitHub/GitLab clients are thin adapters; tests use in-memory fakes.

**Tech Stack:** Go 1.25, standard library HTTP, Go testing, gofmt, go test.

## Global Constraints

- Branch name must be `aegis-fix-[vuln-id]`.
- Mini-scan must run before PR creation.
- PR must not be created when mini-scan still detects the vulnerability.
- Unit tests must cover success and failed validation paths.

---

### Task 1: Core Remediation Contracts and SQLi Patcher

**Files:**
- Create: `internal/fixer/remediation.go`
- Create: `internal/fixer/remediation_test.go`

**Interfaces:**
- Produces: `RemediationActivity`, `RemediationRequest`, `RemediationResult`, `GitProvider`, `Scanner`, `ApplySQLiPatch`.

- [ ] Write failing tests for branch naming, SQLi patching, successful PR creation, failed mini-scan suppression, and unsupported vulnerability type.
- [ ] Implement core contracts and remediation flow.
- [ ] Run `go test ./internal/fixer -v`.

### Task 2: GitHub/GitLab HTTP Adapters

**Files:**
- Create: `internal/fixer/git_provider.go`
- Create: `internal/fixer/git_provider_test.go`

**Interfaces:**
- Consumes: `PullRequestRequest`.
- Produces: `NewGitProviderFromEnv`, GitHub/GitLab PR API adapters.

- [ ] Write failing tests for provider selection and PR payloads.
- [ ] Implement minimal HTTP adapters.
- [ ] Run `go test ./internal/fixer -v`.

### Task 3: Worker Startup and Verification

**Files:**
- Modify: `internal/fixer/fixer.go`
- Modify: `cmd/fixer/main.go`
- Modify: `README.md`

**Interfaces:**
- Consumes: `RemediationActivity` and provider factory.
- Produces: documented env vars and startup path.

- [ ] Update startup to expose remediation initialization without breaking current binary.
- [ ] Document request shape and env vars.
- [ ] Run `gofmt`, `go test ./...`, `go build ./...`, `pre-commit run --all-files`, `graphify update .`.
- [ ] Commit, push, watch CI, close issues `#1`, `#2`, `#3` if green.

## Self-Review

- Covers PR creation branch `aegis-fix-[vuln-id]`.
- Covers mini-scan before PR creation.
- Covers failed validation blocking PR creation.
- Covers testability without live external systems.
