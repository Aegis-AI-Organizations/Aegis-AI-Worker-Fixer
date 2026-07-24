# Worker Fixer Remediation Design

## Goal

Provide a ready-to-use remediation activity that parses a vulnerability report, patches the source repository, validates the patched branch with a mini-scan, and opens a pull request only when validation succeeds.

## Architecture

The Worker Fixer exposes a Go activity entrypoint that can be registered with Temporal. The activity delegates repository and pull request operations to a `GitProvider` interface, and validation to a `Scanner` interface, so the core remediation flow is unit-testable without live GitHub, GitLab, or Pentest Worker dependencies.

## Flow

1. Parse `RemediationRequest` and normalize the vulnerability ID.
2. Clone the target repository into a temporary workspace.
3. Create branch `aegis-fix-[vuln-id]`.
4. Apply a deterministic SQLi patch for supported Go source patterns.
5. Run a mini-scan against the patched branch environment.
6. If the mini-scan still reports the vulnerability, return `failed_validation` and do not create a PR.
7. If validation passes, commit patched files and open a pull request through GitHub or GitLab.

## Supported Patch

The first implementation supports SQL Injection reports. It replaces simple Go SQL string concatenation patterns with placeholder-based queries and argument passing. Unsupported report types return a clear error instead of generating unsafe changes.

## Failure Handling

The activity fails closed. Repository errors, patch misses, mini-scan failures, and remaining vulnerabilities all prevent PR creation. The returned result includes status and log lines for Brain/UI diagnostics.

## Testing

Unit tests cover branch naming, SQLi patch output, successful PR creation, failed-validation suppression of PR creation, and unsupported vulnerability handling. The live API clients are thin HTTP adapters behind interfaces and are not required for unit tests.
