# 🚑 Fixer Architecture: Automated Security Remediation

The **Aegis AI Fixer Worker** is the "Remediation Engine" of the platform. Written in **Go** for its native integration with cloud-native tooling and Git controllers, it automatically repairs identified security weaknesses in the target infrastructure.

---

## 🏗️ Core Design Principles

The Fixer Worker is built for **automation**, **safety**, and **reconciliation**:

1. **GitOps Integration**: Automatically generates Pull Requests (PRs) with security patches, integrating seamlessly into the client's existing CI/CD workflows.
2. **State Reconciliation**: Detects configuration drift and applies non-breaking hotfixes to Kubernetes manifests and Terraform states.
3. **Internal Orchestration**: Managed by the `Aegis-AI-Brain` via **Temporal**, ensuring remediation actions are idempotent and can be safely retried upon failure.

---

## 🔐 Security & Access Control (RBAC)

Since these workers modify infrastructure state, they operate under strict security constraints.

- **Least Privilege RBAC**: Every Fixer pod is assigned a limited **ServiceAccount** with localized RBAC permissions, only allowed to modify resources explicitly marked for remediation.
- **Audit Trails**: Every action (PR creation, manifest update) is cryptographically logged to provide a complete history of remediation efforts.
- **Secure Secret Handling**: Git tokens and deployment identities are injected via **Infisical**/Kubernetes Secrets and are never persisted on disk.

---

## 🌊 Dynamic Scaling (KEDA)

The Fixer pool is managed by **KEDA** (Kubernetes Event-Driven Autoscaling) to efficiently manage remediation backlogs:

- **Demand-Driven Scaling**: Scales replicas based on the count of active "Remediation" tasks in the Temporal cluster.
- **Scale-to-Zero**: When no remediation actions are pending, the pool scales down to **0 replicas**, minimizing operational costs.

---

## 🛰️ Remediation Capabilities

The worker targets various layers of the stack:
- **Application Logic**: Generating patches for common web vulnerabilities (SQLi, XSS).
- **Infastructure-as-Code**: Patching Terraform or CloudFormation misconfigurations.
- **Kubernetes Resources**: Updating `NetworkPolicies`, `SecurityContexts`, and `PodDisruptionBudgets`.

```mermaid
graph LR
    Brain([Brain Orchestrator]) -- "Fix Task" --> Fixer[Fixer Worker (Go)]
    Fixer -- "Generate Patch" --> Logic[Remediation Logic]
    Logic -- "PR" --> Git[Client GitHub/GitLab]
    Logic -- "Patch" --> K8s[Target K8s Cluster]
```

---

*Aegis AI Remediation & Automation — 2026*
