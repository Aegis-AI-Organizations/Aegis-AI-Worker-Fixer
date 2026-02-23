# 🚑 Aegis AI - Worker Pool: Fixer

**Project ID:** AEGIS-CORE-2026

## 🏗️ System Architecture & Role
The **Aegis AI Worker Fixer** handles the remediation logic phase in our distributed execution topology. Managed downstream by the Brain Orchestrator (Temporal), these Go-based workers receive vulnerability inputs and push mitigation patches natively into client infrastructure states.

* **Tech Stack:** Go (Native Goroutines, `client-go`, `controller-runtime`).
* **Role:**
  * Reconciles configuration drift generated during attacks.
  * Rapidly generates and applies PR diffs to remediated git repositories or K8s resources.
* **Architecture Justification:** Go dominates infrastructure automation. Its speed and native Kube SDK integration are vital to ensuring minimal RTO after zero-day patches.

## 🔐 Security & DevSecOps Mandates
* **No Plain-Text Secrets:** Like all Aegis resources, any git tokens or deploy keys must be passed to this stateless worker at execution time via Infisical.
* **Least Privilege:** This pool is constrained, executing strictly under isolated RBAC roles.

## 🐳 Docker Deployment
Designed to scale dynamically (Autoscaling Infinite Pool) against Temporal queues within the Kubernetes compute layer.

```bash
docker pull ghcr.io/aegis-ai/aegis-worker-fixer:latest

infisical run --env=prod -- docker run -d \
  --name aegis-worker-fixer \
  --read-only \
  --cap-drop=ALL \
  --security-opt no-new-privileges:true \
  --user 10001:10001 \
  -e INFISICAL_TOKEN=$INFISICAL_TOKEN \
  ghcr.io/aegis-ai/aegis-worker-fixer:latest
```
