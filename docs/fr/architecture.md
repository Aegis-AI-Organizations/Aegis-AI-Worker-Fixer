# 🚑 Architecture du Fixer : Remédiation Sécurité Automatisée

Le **Worker Fixer Aegis AI** est le "Moteur de Remédiation" de la plateforme. Écrit en **Go** pour son intégration native avec l'outillage cloud-native et les contrôleurs Git, il répare automatiquement les faiblesses de sécurité identifiées dans l'infrastructure cible.

---

## 🏗️ Principes de Conception de Base

Le Worker Fixer est conçu pour l'**automatisation**, la **sécurité** et la **réconciliation** :

1. **Intégration GitOps** : Génère automatiquement des Pull Requests (PR) avec des correctifs de sécurité, s'intégrant parfaitement dans les flux CI/CD existants du client.
2. **Réconciliation d'État** : Détecte la dérive de configuration et applique des correctifs à chaud (hotfixes) non bloquants aux manifestes Kubernetes et aux états Terraform.
3. **Orchestration Interne** : Gérée par le `Aegis-AI-Brain` via **Temporal**, garantissant que les actions de remédiation sont idempotentes et peuvent être réessayées en toute sécurité en cas d'échec.

---

## 🔐 Sécurité et Contrôle d'Accès (RBAC)

Étant donné que ces workers modifient l'état de l'infrastructure, ils fonctionnent sous des contraintes de sécurité strictes.

- **RBAC au Moindre Privilège** : Chaque pod Fixer se voit attribuer un **ServiceAccount** limité avec des permissions RBAC localisées, autorisé uniquement à modifier les ressources explicitement marquées pour la remédiation.
- **Pistes d'Audit** : Chaque action (création de PR, mise à jour de manifeste) est enregistrée cryptographiquement pour fournir un historique complet des efforts de remédiation.
- **Gestion Sécurisée des Secrets** : Les jetons Git et les identités de déploiement sont injectés via **Infisical**/Secrets Kubernetes et ne sont jamais persistés sur le disque.

---

## 🌊 Mise à l'Échelle Dynamique (KEDA)

Le pool Fixer est géré par **KEDA** (Kubernetes Event-Driven Autoscaling) pour gérer efficacement les retards de remédiation :

- **Mise à l'Échelle Pilotée par la Demande** : Ajuste le nombre de réplicas en fonction du nombre de tâches de "Remédiation" actives dans le cluster Temporal.
- **Scale-to-Zero (Mise à l'échelle vers zéro)** : Lorsqu'aucune action de remédiation n'est en attente, le pool se réduit à **0 réplica**, minimisant les coûts opérationnels.

---

## 🛰️ Capacités de Remédiation

Le worker cible plusieurs couches de la pile :
- **Logique Applicative** : Génération de correctifs pour les vulnérabilités web courantes (SQLi, XSS).
- **Infrastructure-as-Code** : Correction des mauvaises configurations Terraform ou CloudFormation.
- **Ressources Kubernetes** : Mise à jour des `NetworkPolicies`, `SecurityContexts` et `PodDisruptionBudgets`.

```mermaid
graph LR
    Brain([Orchestrateur Brain]) -- "Tâche Fix" --> Fixer[Fixer Worker (Go)]
    Fixer -- "Générer correctif" --> Logic[Logique de Remédiation]
    Logic -- "PR" --> Git[Client GitHub/GitLab]
    Logic -- "Patch" --> K8s[Client Cluster K8s]
```

---

*Remédiation et Automatisation Aegis AI — 2026*
