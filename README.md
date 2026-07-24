# 🚑 Aegis AI — Fixer Worker

**Project ID:** AEGIS-CORE-2026

> The **Aegis AI Fixer Worker** is the automated remediation engine of the platform. Orchestrated by the Brain via **Temporal**, these Go-based workers receive vulnerability insights and automatically generate/apply mitigation patches (Git PRs, K8s manifests, or Terraform diffs) to secure the target infrastructure.

---

## 🏗️ Role in the Ecosystem

The Fixer pool closes the security loop by moving from "Detection" to "Resolution".

- **Vulnerability Reconciliation**: Analyzes identified weaknesses and maps them to known secure configurations.
- **GitOps Automation**: Automatically creates Pull Requests with optimized security patches in the client's repositories.
- **Infrastructure Patching**: Directly applies non-breaking security hotfixes to Kubernetes resources when authorized.

```mermaid
graph LR
    Brain[Brain Orchestrator] -- "Fix Instructions" --> Fixer[Fixer Worker (Go)]
    Fixer -- "Generate PR" --> Git[Client Git Repo]
    Fixer -- "Apply Manifest" --> K8s[Client K8s Cluster]
    Git -- "Webhook" --> ArgoCD[ArgoCD Sync]
```

---

## 🛠️ Tech Stack

| Component | Technology | Version |
|---|---|---|
| Language | **Go** | 1.22+ |
| Orchestration | **Temporal SDK** | 1.x |
| K8s Integration | **client-go**, controller-runtime | — |
| Git Integration | **go-git** | — |

---

## 🔐 Security & DevSecOps

- **Least Privilege RBAC**: Operates with strictly scoped Kubernetes roles, only allowed to modify specific resources.
- **Secret Management**: Deployment keys and Git tokens are injected via **Infisical** at runtime.
- **Audit Logging**: Every remediation action is logged with a complete cryptographic audit trail.

---

## Remediation Activity

The worker exposes `RemediationActivity.RemediateVulnerability` as the Temporal activity entrypoint. The activity:

- parses the vulnerability report,
- clones the target repository,
- creates branch `aegis-fix-[vuln-id]`,
- applies a supported SQL Injection remediation,
- runs a mini-scan on the patched branch through the `Scanner` interface,
- opens a GitHub Pull Request or GitLab Merge Request only if the mini-scan no longer detects the vulnerability.

Minimal request shape:

```json
{
  "repository": {
    "provider": "github",
    "owner": "acme",
    "name": "vulnerable-app",
    "clone_url": "https://github.com/acme/vulnerable-app.git",
    "base_branch": "main"
  },
  "vulnerability": {
    "id": "SQLI-123",
    "type": "SQL Injection",
    "title": "SQL injection in login handler",
    "file_path": "handler.go"
  }
}
```

Configure one provider token at runtime:

```bash
export GITHUB_TOKEN=ghp_xxx
# or
export GITLAB_TOKEN=glpat_xxx
```

Optional endpoints for self-hosted providers:

```bash
export GITHUB_API_URL=https://github.example/api/v3
export GITLAB_API_URL=https://gitlab.example
```

If the mini-scan still reports the original vulnerability, the activity returns `failed_validation` and does not create a pull request.

---

## 🐳 Deployment (Kubernetes)

Autoscaled by **KEDA** based on the remediation task queue depth.

```yaml
# Helm values example
image:
  repository: ghcr.io/aegis-ai/aegis-worker-fixer
  tag: latest
keda:
  enabled: true
  minReplicas: 0
  maxReplicas: 20
```

---

## 🛠️ Development

```bash
# Run locally
go run ./cmd/fixer

# Run unit tests
go test ./...
```

---

*Aegis AI — Remediation & Automation — 2026*
