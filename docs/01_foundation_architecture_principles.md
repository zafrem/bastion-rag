# Bastion Architecture & Design Principles

**Project:** Bastion - RAG Security Governance Framework
**Document Type:** Foundation (Tier 1)
**Document ID:** 01-architecture-principles
**Version:** 3.0
**Date:** 2026-05-26
**Status:** Active
**Supersedes:** v2.0 (2026-05-17) — archived at docs/archive/v2/

---

## 1. Introduction

### 1.1 Purpose

This document establishes the foundational architecture and design principles for the Bastion RAG Security Governance Framework. It is the **single source of truth** for architectural decisions that all module and cross-cutting specifications must follow.

### 1.2 What Changed in v3

v3 introduces a **polyglot architecture**: Navigator (C) and Anchor (E) are implemented in Python, while Sentinel (A), Vault (B), and Tracker (D) remain in Go. All inter-module contracts (gRPC + REST interfaces, NATS event schema) are unchanged — the language change is an internal implementation detail.

| Module | Language | Reason |
|---|---|---|
| A – Sentinel | Go | High-throughput text pattern matching; Go excels |
| B – Vault | Go | Cryptographic operations, KMS integration; Go excels |
| C – Navigator | **Python** | In-process ML model serving (`sentence-transformers`, `qdrant-client`) |
| D – Tracker | Go | Event aggregation, time-series; Go excels |
| E – Anchor | **Python** | Numerical computing — noise, WEAT, quality (`numpy`/`scipy`) |

### 1.3 Document Hierarchy

```
Tier 1: Foundation (THIS DOCUMENT + 2 others)
        ↓ defines principles for
Tier 2: Module SRS (5 modules)
        ↓ uses
Tier 3: Cross-cutting SRS (features)
        ↓ summarized in
Tier 4: Integration (master overview)
```

---

## 2. System Overview

### 2.1 What is Bastion?

Bastion is a **data governance framework** for RAG (Retrieval-Augmented Generation) systems.

```
Philosophy:
"We don't block data; we govern its safe flow."
```

### 2.2 The Five Modules

| Module | Name | Language | Primary Responsibility |
|---|---|---|---|
| **A** | Sentinel | Go | Input/Output validation gateway |
| **B** | Vault | Go | Data isolation & anonymization |
| **C** | Navigator | Python | Search & ranking |
| **D** | Tracker | Go | Observability & audit |
| **E** | Anchor | Python | Embedding security |

### 2.3 Pipeline Position

```
                    User Query
                       ↓
        ┌──────────────────────────────┐
        │  Input Pipeline               │
        │  A → B → C → E → LLM          │
        │  Go  Go  Py  Py               │
        └──────────────────────────────┘
                       ↓
        ┌──────────────────────────────┐
        │  Output Pipeline              │
        │  LLM → E → B → A → User       │
        │        Py  Go  Go             │
        └──────────────────────────────┘
                       ↓
        ┌──────────────────────────────┐
        │  D (Tracker): Go             │
        │  cross-cutting observer       │
        └──────────────────────────────┘
```

---

## 3. Core Design Principle: Progressive Enhancement

### 3.1 The Central Idea

```
Each module provides value INDEPENDENTLY.
Combining modules ENHANCES security.
Full orchestration enables ADVANCED features.

Key property:
"Remove any module and the system still works,
 just with less security.
 Add any module and security increases."
```

### 3.2 The Litmus Test

Every module must pass this test:

```
"If I attach ONLY this module directly to an LLM,
 does it provide meaningful security?"

Sentinel → LLM: ✅  Vault → LLM: ✅
Navigator → LLM: ✅  Anchor → LLM: ✅
```

---

## 4. The Three-Layer Model

### 4.1 Layer Definitions

```
┌─────────────────────────────────────────────┐
│ Layer 3: ORCHESTRATION                      │
│ - Requires multiple modules + coordination  │
│ - Cross-cutting features                    │
│ - Optional (system works without)           │
├─────────────────────────────────────────────┤
│ Layer 2: COMPOSITION                        │
│ - Enhanced by combining 2+ modules          │
│ - Optional enhancement                      │
├─────────────────────────────────────────────┤
│ Layer 1: STANDALONE                         │
│ - Each module's core function               │
│ - Works independently, no dependencies      │
│ - ALWAYS active                             │
└─────────────────────────────────────────────┘
```

### 4.2 Module Feature Classification

| Module | 🟢 Core | 🟡 Enhanced | 🔴 Orchestrated |
|---|---|---|---|
| Sentinel | Injection defense, metadata validation | Indirect injection (+ Navigator) | Honey-token, Lineage |
| Vault | PII anonymization, tokenization | Permission filtering (+ Navigator) | Honey-token, Multi-tenancy |
| Navigator | Hybrid search, reranking | Permission filter (+ Vault) | Honey-token, Tenancy |
| Anchor | Noise injection, bias analysis, response verify | Quality tuning (+ Navigator) | Pipeline bias monitoring |
| Tracker | Event collection, visualization | Multi-module metrics | Lineage, Correlation |

---

## 5. Independence Principles

### 5.1 Three Types of Independence

```
1. Deployment Independence  — each module deploys separately
2. Functional Independence  — core features need no other module
3. Runtime Independence     — one failure doesn't cascade
```

### 5.2 Loose Coupling Requirement

```
REQUIRED: No direct module-to-module dependencies.

Synchronous (in-request data path):
  - gRPC/REST calls with data PASSED IN the request by the caller
  - No module holds a reference to another

Asynchronous (cross-cutting, observability):
  - Events via NATS (fire-and-forget)

❌ FORBIDDEN:
   class Navigator:
       vault: Vault  # Direct dependency — NO

✅ REQUIRED:
   # Caller fetches permissions, passes to Navigator
   perms = vault_client.get_permissions(user_id)
   navigator_client.search_with_permissions(query, perms)
```

---

## 6. Cross-Cutting Concern Handling

### 6.1 The Hook Pattern

```python
# Python (Navigator, Anchor)
class SearchService:
    def __init__(self, hook_manager: HookManager): ...

    def search(self, req):
        result = self._core_search(req)          # always runs
        self._hook_manager.fire(HookEvent(...))  # if hooks registered
        return result
```

```go
// Go (Sentinel, Vault, Tracker)
func (s *Service) Process(req Request) Response {
    result := s.coreProcess(req)         // always runs
    s.hm.Fire(hooks.Event{...})          // if hooks registered
    return result
}
```

### 6.2 Cross-Cutting Features

| Feature | Modules Involved | Coordinator |
|---|---|---|
| **Honey-token** | Vault, Sentinel, Navigator, Tracker | Vault-led |
| **Multi-tenancy** | Vault, Navigator, Sentinel | Shared |
| **Data Lineage** | All modules | Tracker-led |

---

## 7. Polyglot Architecture Details

### 7.1 Interoperability Contract

Language is an internal implementation detail. All modules expose identical **wire contracts**:

```
gRPC: JSON codec (both Go and Python servers use JSON-over-gRPC,
      not protobuf binary, for uniform tooling compatibility)

REST: JSON over HTTP/1.1

Events: JSON payload on NATS subjects
        bastion.events.{module}.{event_type}
```

### 7.2 Why Python for Navigator and Anchor

**Navigator:**
- The `sentence-transformers` library runs BGE-M3 and BGE-reranker in-process
- In Go, model serving requires a separate Python microservice and HTTP round-trips
- In Python: `model.encode(text)` — one function call, no network hop
- `qdrant-client` Python SDK is the primary maintained client for Qdrant

**Anchor:**
- Noise injection, WEAT bias analysis, quality metrics are numerical algorithms
- Go requires manual implementation of Box-Muller, L2 norm, cosine similarity
- Python: `np.random.normal()`, `np.linalg.norm()` — single calls on optimized BLAS
- `scipy` provides differential privacy and statistical primitives out of the box

### 7.3 Shared Patterns Across Languages

Despite different languages, all modules share the same patterns:

| Pattern | Go (A, B, D) | Python (C, E) |
|---|---|---|
| Config | YAML → struct | YAML → Pydantic model |
| Models | Go structs with JSON tags | Pydantic models |
| gRPC codec | `encoding.RegisterCodec(JSONCodec{})` | `GenericRpcHandler` + JSON serializer |
| Hook manager | `hooks.Manager` (goroutine-per-handler) | `HookManager` (thread-per-handler) |
| NATS publisher | `nats.go` in background goroutine | `nats-py` in background thread + asyncio |
| Metrics | `prometheus/client_golang` | `prometheus-client` |

### 7.4 Deployment Topology

```
┌──────────────────────────────────────────────┐
│  Kubernetes / Docker Compose                  │
│                                               │
│  sentinel:   Go binary  (port 8080/9090)      │
│  vault:      Go binary  (port 8081/9091)      │
│  navigator:  Python app (port 8082/9092)      │
│  anchor:     Python app (port 8083/9093)      │
│  tracker:    Go binary  (port 8084/9094)      │
│  nats:       nats:2.10  (port 4222)           │
│  qdrant:     qdrant/qdrant (port 6333)        │
└──────────────────────────────────────────────┘
```

### 7.5 Python Module Requirements

```
Python modules require:
- Python 3.11+
- 4GB RAM minimum (model loading: bge-m3 ~1.8GB, reranker ~0.5GB)
- First startup: model download from HuggingFace (cached after)
- CPU sufficient; GPU optional (10-30× speedup for batch inference)
```

---

## 8. Deployment Configurations

### 8.1 Progressive Deployment

```
Minimal:    Sentinel (Go)
Basic:      Sentinel → Vault (both Go)
Standard:   Sentinel → Vault → Navigator (Go + Python)
Full:       Sentinel → Vault → Navigator → Anchor (Go + Python)
Complete:   All modules + Tracker
```

### 8.2 Configuration Matrix

| Config | Modules | Security Level | Use Case |
|---|---|---|---|
| Minimal | A | Basic | Dev/test |
| Basic | A+B | PII-safe | Simple RAG |
| Standard | A+B+C | Search-safe | Production RAG |
| Full | A+B+C+E | Complete | Sensitive data |
| Orchestrated | All + CC | Maximum | Enterprise |

---

## 9. Design Constraints

### 9.1 Technical Constraints

```
Language:
  Sentinel (A), Vault (B), Tracker (D): Go 1.22+
  Navigator (C), Anchor (E):            Python 3.11+

Communication:  Event-driven (NATS 2.10+)
Deployment:     Docker / Kubernetes
Interfaces:     gRPC (JSON codec), REST, CLI — ALL modules
Standards:      OpenTelemetry, W3C Trace Context
Event bus:      NATS (at-most-once operational, at-least-once security)
```

### 9.2 Architectural Constraints

```
1. Every module MUST work standalone
2. Inter-module communication MUST NOT use direct dependencies
3. Cross-cutting concerns MUST use hooks
4. Core functions MUST NOT depend on other modules
5. Failures MUST degrade gracefully
6. gRPC wire format MUST be JSON (language-neutral tooling)
```

### 9.3 Quality Attributes (Priority Order)

```
1. Independence   — modules don't break each other
2. Security       — defense in depth
3. Observability  — everything traceable
4. Performance    — minimal overhead
5. Flexibility    — configurable deployment
```

---

## 10. Glossary

| Term | Definition |
|---|---|
| **Polyglot** | Multiple implementation languages sharing identical contracts |
| **Standalone** | Module operating independently |
| **Composition** | Two+ modules working together |
| **Orchestration** | Cross-cutting coordination |
| **Hook** | Extension point for cross-cutting |
| **JSON-over-gRPC** | gRPC using JSON serialization instead of protobuf binary |
| **In-process ML** | ML model loaded into the service process (no separate model server) |

---

## 11. Change History

| Version | Date | Changes |
|---|---|---|
| 1.0 | 2026-05-17 | Initial |
| 2.0 | 2026-05-17 | Foundation-aligned |
| 3.0 | 2026-05-26 | Polyglot architecture: Navigator (C) and Anchor (E) → Python |

---

**End of Document**
