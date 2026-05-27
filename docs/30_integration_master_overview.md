# Bastion Master Overview

**Project:** Bastion - RAG Security Governance Framework
**Document Type:** Integration (Tier 4)
**Document ID:** 30-master-overview
**Version:** 2.0
**Date:** 2026-05-26
**Status:** Active
**Supersedes:** v1.0 (2026-05-17) — archived at docs/archive/v2/

> This is a **thin** integration document. It provides the map; detailed specifications live in their respective documents.

---

## 1. What is Bastion?

Bastion is a **data governance framework** for RAG (Retrieval-Augmented Generation) systems.

```
Philosophy: "We don't block data; we govern its safe flow."

5 modules, bidirectional protection,
progressive enhancement architecture.
```

---

## 2. The Five Modules

| Module | Name | Language | Role | Direction |
|---|---|---|---|---|
| A | Sentinel | Go 1.22+ | Validation gateway | Bidirectional |
| B | Vault | Go 1.22+ | Data isolation | Bidirectional |
| C | Navigator | Python 3.11+ | Search & ranking | Single |
| D | Tracker | Go 1.22+ | Observability | Observer |
| E | Anchor | Python 3.11+ | Embedding security | Bidirectional |

---

## 3. Pipeline

```
Input:  User → A(Go) → B(Go) → C(Py) → E(Py) → LLM
Output: LLM → E(Py) → B(Go) → A(Go) → User
Observer: D(Go) (Tracker) watches all

Bidirectional symmetric design.
Wire contract: JSON-over-gRPC (language-transparent).
```

---

## 4. Document Map

### Tier 1: Foundation
```
01-architecture-principles    → 3-Layer model, Progressive Enhancement
02-event-schema-standard      → Event format, trace propagation
03-module-interaction-map     → Interfaces, hooks, data flow
```

### Tier 2: Module SRS
```
10-sentinel-srs   → Validation (IN+OUT)
11-vault-srs      → Anonymization (Phase 1+2)
12-navigator-srs  → Search (single-direction)
13-anchor-srs     → Embedding security (IN+OUT)
14-tracker-srs    → Observability (cross-cutting observer)
```

### Tier 3: Cross-Cutting SRS
```
20-honey-token-srs    → Intrusion detection (Vault-owned, multi-layer)
21-multi-tenancy-srs  → Tenant isolation (CRITICAL)
22-data-lineage-srs   → Data journey tracking (Tracker-led)
```

### Tier 4: Integration
```
30-master-overview    → This document
```

---

## 5. Core Architecture: Progressive Enhancement

```
3-Layer Model:

Layer 3: Orchestration (cross-cutting, optional)
         Honey-token, Multi-tenancy, Lineage

Layer 2: Composition (module pairs, optional)
         Enhanced features

Layer 1: Standalone (always active)
         Each module's core function

Principle: "Remove any module → still works.
            Add any module → stronger."
```

---

## 6. Each Module's Layers

```
            Core (🟢)        Enhanced (🟡)      Hooks (🔴)
Sentinel    injection,       indirect inj.,     honey-token,
            metadata         deep PII           lineage

Vault       anonymize,       permission         honey-token(owner),
            transform        to Navigator       tenancy, lineage

Navigator   hybrid search,   permission         honey-token,
            rerank           filter             tenancy(prefilter)

Anchor      noise, analyze   quality tuning     bias monitor,
                                                lineage

Tracker     events, trace    multi-metrics      honey-token(aggr),
                                                lineage(coord)
```

---

## 7. Deployment Configurations

| Config | Modules | Use Case |
|---|---|---|
| Minimal | Sentinel | Dev/test |
| Basic | A+B | PII-safe RAG |
| Standard | A+B+C | Production RAG |
| Full | A+B+C+E | Sensitive data |
| Orchestrated | All + cross-cutting | Enterprise |

---

## 8. Key Design Decisions

```
1. Bidirectional modules (Sentinel, Vault, Anchor)
   → Input AND output protection

2. Single engine + multiple configs
   → Code reuse, consistency

3. Event-driven communication (NATS)
   → Loose coupling, independence

4. Hook-based cross-cutting
   → Core works with/without hooks

5. trace_id propagation
   → Enables lineage, correlation

6. Pre-filter for tenancy (Navigator)
   → CRITICAL for isolation
```

---

## 9. Cross-Cutting Summary

```
Honey-Token (20):
- Vault owns (create/inject/detect)
- Multi-layer detection (input/data/search/output)
- Tracker aggregates

Multi-Tenancy (21): CRITICAL
- Vault: key isolation
- Navigator: pre-filter (keystone)
- Sentinel: validation

Data Lineage (22):
- All modules emit
- Tracker reconstructs
- trace_id based
```

---

## 10. Technology Stack

```
Languages:
  Sentinel, Vault, Tracker: Go 1.22+
  Navigator, Anchor:         Python 3.11+
  Tracker UI:                React/TypeScript

Navigator (Python):
  sentence-transformers (BGE-M3, in-process)
  sentence-transformers CrossEncoder (BGE-reranker, in-process)
  qdrant-client (Python SDK)
  fastapi + uvicorn + grpcio

Anchor (Python):
  numpy + scipy (noise, WEAT bias, quality metrics)
  fastapi + uvicorn + grpcio

Shared:
  Event bus: NATS 2.10+
  Vector DB: Qdrant
  Storage: PostgreSQL, Redis
  Observability: Prometheus, Loki, Jaeger
  Policy: OPA
  KMS: AWS/HashiCorp/Local
  Wire format: JSON-over-gRPC, REST JSON, NATS JSON
```

---

## 11. Quick Start (PoC)

```bash
# Full Bastion via Docker Compose
$ docker-compose up

Services started:
- Sentinel   Go     binary  (REST: 8080, gRPC: 9090)
- Vault      Go     binary  (REST: 8081, gRPC: 9091)
- Navigator  Python app     (REST: 8082, gRPC: 9092)  ← 4GB RAM, model load
- Anchor     Python app     (REST: 8083, gRPC: 9093)
- Tracker    Go     binary  (UI: 3000, REST: 8084)
- NATS, PostgreSQL, Qdrant, Redis

Note: Navigator first startup downloads BAAI/bge-m3 and
BAAI/bge-reranker-v2-m3 from HuggingFace (~2.3GB, cached after).

→ Open http://localhost:3000 (Tracker UI)
→ See live request flow
```

---

## 12. Reading Guide

```
New to Bastion?
→ Start: 01-architecture-principles
→ Then: this overview

Implementing a module?
→ Read: that module's SRS (Tier 2)
→ Reference: 02-event-schema, 03-interaction-map

Implementing cross-cutting?
→ Read: cross-cutting SRS (Tier 3)
→ Reference: module hooks

Understanding data flow?
→ Read: 03-module-interaction-map
```

---

## 13. Document Status

| Tier | Document | Version | Status |
|---|---|---|---|
| 1 | 01-architecture-principles | v3.0 | ✅ |
| 1 | 02-event-schema-standard | v2.0 | ✅ |
| 1 | 03-module-interaction-map | v2.0 | ✅ |
| 2 | 10-sentinel-srs | v3.0 | ✅ |
| 2 | 11-vault-srs | v3.0 | ✅ |
| 2 | 12-navigator-srs | v3.0 | ✅ |
| 2 | 13-anchor-srs | v3.0 | ✅ |
| 2 | 14-tracker-srs | v3.0 | ✅ |
| 3 | 20-honey-token-srs | v2.0 | ✅ |
| 3 | 21-multi-tenancy-srs | v2.0 | ✅ |
| 3 | 22-data-lineage-srs | v2.0 | ✅ |
| 4 | 30-master-overview | v2.0 | ✅ |

Archive: docs/archive/v2/ (all v2 and earlier documents)

---

## 14. Change History

| Version | Date | Changes |
|---|---|---|
| 1.0 | 2026-05-17 | Initial master overview |
| 2.0 | 2026-05-26 | Polyglot architecture: Navigator and Anchor → Python; technology stack updated; module table adds language column; pipeline diagram adds language labels; Quick Start notes model download |

---

**End of Document**

---

## Appendix: The Bastion Story in One Page

```
A user sends a query to a RAG system.

1. Sentinel checks it (injection? valid?)
2. Vault anonymizes any PII
3. Navigator searches (tenant-isolated)
4. Anchor protects embeddings
5. LLM generates response
6. Anchor analyzes response (bias?)
7. Vault applies permissions (who sees what?)
8. Sentinel validates output (PII leak?)
9. User gets safe response

Throughout:
- Tracker watches everything
- Honey-tokens detect intruders
- Lineage records the journey
- Tenants stay isolated

Each module works alone.
Together, they form complete protection.
That's Bastion.
```
