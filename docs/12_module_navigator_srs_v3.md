# Bastion-Navigator Module SRS

**Project:** Bastion - RAG Security Governance Framework
**Document Type:** Module SRS (Tier 2)
**Document ID:** 12-navigator-srs
**Module:** C - Navigator (Search & Ranking)
**Version:** 3.0
**Date:** 2026-05-26
**Status:** Active
**Supersedes:** v2.0 (2026-05-17) — archived at docs/archive/v2/

**Foundation References:**
- 01-architecture-principles (v3 — polyglot)
- 02-event-schema-standard
- 03-foundation-module-interaction-map

---

## 1. Introduction

### 1.1 Purpose

This document specifies the **Navigator** module, the search and ranking layer of Bastion. v3 reflects the migration from Go to Python, enabling in-process ML model serving with no separate model microservice.

### 1.2 Module Identity

```
Module: C - Navigator
Language: Python 3.11+
Role: Search & Ranking
Position: After Vault (input path)
Direction: Single (search)

Standalone value:
"Attach Navigator alone → hybrid search with in-process reranking for RAG"
```

### 1.3 Why Python (v3 Change)

```
v2 (Go): Embedding and reranking required a separate Python microservice
         because Go has no equivalent of sentence-transformers.
         Every search = HTTP round-trip to model server.

v3 (Python): Models run in-process.
  from sentence_transformers import SentenceTransformer, CrossEncoder
  model.encode(text)          # in-process, no network hop
  reranker.predict(pairs)     # in-process, no network hop

Benefits:
- Simpler deployment (no model sidecar)
- Lower latency (no inter-service HTTP)
- qdrant-client Python SDK is the primary maintained client
```

### 1.4 The Standalone Test (Foundation Litmus)

```
Question: "If only Navigator is attached to an LLM,
          does it provide meaningful value?"

Answer: YES
- Hybrid search (vector + BM25)
- In-process reranking
- Quality retrieval for RAG

→ Navigator passes the standalone test ✅
(Security value is in Enhanced/Hooks layer)
```

### 1.5 Why Single-Direction?

```
Navigator searches on INPUT path.
On OUTPUT path, there is nothing to "search".

Output-side concerns (response validation)
are handled by Sentinel-OUT, which may
use Navigator's context (Enhanced composition).

Navigator is single-direction by nature.
```

### 1.6 Scope

**In Scope:**
- 🟢 Core: Hybrid search (vector + BM25)
- 🟢 Core: In-process reranking (CrossEncoder, no model server)
- 🟢 Core: In-process query embedding (SentenceTransformer, no model server)
- 🟡 Enhanced: Permission-based filtering (with Vault)
- 🟡 Enhanced: Context provision (to Sentinel-OUT)
- 🔴 Hooks: Honey-token search detection
- 🔴 Hooks: Multi-tenancy pre-filter
- 🔴 Hooks: Lineage emission
- Standalone deployment

**Out of Scope:**
- Document indexing (separate Indexer service)
- Anonymization (Vault)
- Input/output validation (Sentinel)

### 1.7 Definitions

| Term | Definition |
|---|---|
| **Hybrid search** | Vector + BM25 combined via RRF |
| **RRF** | Reciprocal Rank Fusion (k=60) |
| **In-process ML** | Model loaded into the service process, no separate server |
| **SentenceTransformer** | Python library for in-process embedding (BGE-M3) |
| **CrossEncoder** | Python cross-encoder for reranking (BGE-reranker-v2-m3) |
| **qdrant-client** | Official Python SDK for Qdrant vector DB |
| **Pre-filter** | Filter applied before search |

---

## 2. Overall Description

### 2.1 Architecture

```
┌─────────────────────────────────────────────┐
│           Navigator Service (Python)         │
├─────────────────────────────────────────────┤
│                                              │
│  ┌──────────────────────────────┐            │
│  │  Search Orchestrator         │            │
│  └────────────┬─────────────────┘            │
│               │                              │
│      ┌────────┼────────┐                     │
│      ▼        ▼        ▼                     │
│  ┌───────┐┌───────┐┌──────────┐              │
│  │Embedder││Searcher││Reranker │              │
│  │(in-proc)│(Qdrant)│(in-proc) │              │
│  └───────┘└───────┘└──────────┘              │
│      │                  │                    │
│  SentenceTransformer  CrossEncoder           │
│  BAAI/bge-m3         BAAI/bge-reranker-v2-m3 │
│               │                              │
│      ┌────────┴────────┐                     │
│      ▼                 ▼                     │
│  ┌──────────┐    ┌──────────┐                │
│  │Enhancement│   │  Hooks   │                │
│  │(optional)│    │(optional)│                │
│  └──────────┘    └──────────┘                │
│                                              │
└─────────────────────────────────────────────┘
```

### 2.2 Position in Pipeline

```
Input Pipeline:
Vault → [Navigator: search] → Anchor

Navigator provides context to Sentinel-OUT
(Enhanced composition, output path)
```

### 2.3 Layer Classification

```
🟢 CORE (Standalone):
   - Hybrid search (vector + BM25)
   - In-process reranking
   - In-process query embedding

🟡 ENHANCED (Composition):
   - Permission filtering (+ Vault)
   - Context provision (+ Sentinel-OUT)

🔴 HOOKS (Cross-cutting):
   - Honey-token search detection
   - Multi-tenancy pre-filter
   - Lineage emission
```

### 2.4 Constraints

```
Language: Python 3.11+
Vector DB: Qdrant (self-hosted)
Embedding model: BAAI/bge-m3 (1024-dim, in-process)
Reranker model: BAAI/bge-reranker-v2-m3 (in-process)
RAM: ≥ 4GB (bge-m3 ~1.8GB + reranker ~0.5GB)
Scale: 1M–10M docs (SMB)
Latency: <150ms p95 (hybrid+rerank)
First startup: model download from HuggingFace (cached after)
CPU sufficient; GPU optional (10–30× speedup for batch inference)
```

### 2.5 Dependencies

```
Core dependencies (Python packages):
- sentence-transformers >= 3.0  (embedding + reranking)
- qdrant-client >= 1.9           (vector search)
- fastapi >= 0.111               (REST)
- uvicorn >= 0.30                (ASGI server)
- grpcio >= 1.64                 (gRPC server)
- pydantic >= 2.7                (models + config)
- pyyaml >= 6.0                  (config loading)

Optional (Enhanced):
- httpx >= 0.27                  (Vault HTTP client)

Optional (Hooks):
- nats-py >= 2.7                 (NATS events)
- prometheus-client >= 0.20      (metrics)
```

---

## 3. Core Functions (🟢 Standalone)

### 3.1 Query Embedding (FR-CORE-EM)

**FR-CORE-EM-001: In-process Multilingual Embedding**
```
Model: BAAI/bge-m3 (1024-dim)
Library: sentence-transformers (SentenceTransformer)
Languages: Korean + English (multilingual)
Execution: in-process — no separate model server

Implementation:
  from sentence_transformers import SentenceTransformer
  model = SentenceTransformer("BAAI/bge-m3", trust_remote_code=True)
  vector = model.encode(text, normalize_embeddings=True)

No dependency on external embedding service.
```

**FR-CORE-EM-002: Embedding Cache**
```
In-memory LRU cache for repeated queries.
Cache key: (text, model_name)
Default capacity: 1000 entries
```

**FR-CORE-EM-003: Batch Embedding**
```
model.encode(texts, normalize_embeddings=True)
Vectorized batch processing.
```

### 3.2 Vector Search (FR-CORE-VS)

**FR-CORE-VS-001: HNSW Search via qdrant-client**
```
Library: qdrant-client Python SDK (primary maintained client)
Method: client.search(collection_name, query_vector, limit)
Metric: Cosine similarity
Index: HNSW (Qdrant managed)

Implementation:
  from qdrant_client import QdrantClient
  client = QdrantClient(host=..., port=...)
  results = client.search(collection, query_vector, limit=top_k)
```

**FR-CORE-VS-002: Multi-Collection**
```
Collections map to document categories:
  customer_docs, manufacturing_docs, hr_docs
Search across all matching collections.
```

### 3.3 Keyword Search (FR-CORE-BM)

**FR-CORE-BM-001: Sparse / Text Search**
```
qdrant-client scroll with MatchText filter for lexical matching.
Korean (Nori) + English term matching.
Exact term/code matching for product codes.
```

### 3.4 Hybrid Search (FR-CORE-HS)

**FR-CORE-HS-001: RRF Fusion**
```
Combine vector + BM25 result lists.
Reciprocal Rank Fusion formula (k=60):
  score(d) = Σ 1/(k + rank_i(d))

Configurable per-source weights.
```

**FR-CORE-HS-002: 15-Step Search Pipeline**
```
Implemented in Orchestrator.search():
 1.  Merge request defaults from config
 2.  Resolve permissions (Vault or passthrough)
 3.  Map user categories → Qdrant collections
 4.  Embed query (SentenceTransformer)
 5.  Vector search (qdrant-client)
 6.  Sparse/BM25 search (qdrant-client scroll)
 7.  RRF fusion (k=60)
 8.  Apply permission post-filter
 9.  Slice to over_fetch candidate pool
10.  Rerank with CrossEncoder
11.  Slice to top_k final results
12.  Build SearchResponse with metadata
13.  Record processing_time_ms
14.  Return SearchResponse
15.  Fire events + hooks (in caller)
```

### 3.5 Reranking (FR-CORE-RR)

**FR-CORE-RR-001: In-process CrossEncoder**
```
Model: BAAI/bge-reranker-v2-m3
Library: sentence-transformers (CrossEncoder)
Execution: in-process — no separate reranker service

Implementation:
  from sentence_transformers import CrossEncoder
  reranker = CrossEncoder("BAAI/bge-reranker-v2-m3",
                          max_length=512, trust_remote_code=True)
  pairs = [[query, candidate.content] for candidate in candidates]
  scores = reranker.predict(pairs, show_progress_bar=False)

Default: top 50 candidates → top 10 results
```

**FR-CORE-RR-002: HTTP Reranker (Optional)**
```
For deployments with a separate reranker service:
BGEHttpReranker(endpoint) — HTTP POST to external service.
```

### 3.6 Core Summary

```
Standalone capabilities (Qdrant + Python):
✅ In-process query embedding (bge-m3, no model server)
✅ Vector search (qdrant-client)
✅ BM25/text search (qdrant-client scroll)
✅ Hybrid (RRF fusion)
✅ In-process reranking (bge-reranker, no model server)

Quality RAG search without other Bastion modules.
No external model services required.
```

---

## 4. Enhanced Functions (🟡 Composition)

### 4.1 Permission Filtering (FR-ENH-PF)

**Requires: Vault (provides permissions)**

**FR-ENH-PF-001: Pre-filter by Permission**
```
When Vault permissions available:
- Filter search collections by user.allowed_categories
- CRITICAL: pre-filter (collection-level, not post-filter)
- Prevent unauthorized document retrieval

Interface (Foundation — permissions passed in by caller):
  POST /v1/navigator/search/with-permissions
  grpc: SearchWithPermissions(SearchRequest with user.allowed_categories)

Caller fetches permissions from Vault, passes in request:
  perms = vault_client.get_permissions(user_id)
  navigator_client.search(query, user=UserContext(
      allowed_categories=perms.categories))

Graceful degradation:
- Without Vault: search all collections (less secure)
- Core search still fully operational
```

**FR-ENH-PF-002: Over-fetching**
```
Fetch top_k × over_fetch_multiplier (default 5) candidates
before applying permission post-filter.
Ensures top_k results survive filtering.
```

### 4.2 Context Provision (FR-ENH-CP)

**Requires: Sentinel-OUT (consumes context)**

**FR-ENH-CP-001: Search Context Export**
```
Provide search results to Sentinel-OUT
for indirect injection detection.

Sentinel-OUT uses results to verify response grounding.

Graceful degradation:
- Without Sentinel-OUT: context unused
- Core search unaffected
```

### 4.3 Enhanced Summary

```
🟡 Permission filtering (+ Vault) — CRITICAL for multi-tenancy
🟡 Context provision (+ Sentinel-OUT)

Without composition: core search works (less secure)
```

---

## 5. Hooks (🔴 Cross-Cutting)

### 5.1 Honey-Token Hooks

**Hook Point:**
```
navigator.results.scanned
- After search, scan results for honey-token metadata
- result.metadata["is_honey_token"] == "true"
- Detail: see Honey-Token SRS (Tier 3)
```

**Contract:**
```
On honey-token in search results:
→ NATS event: navigator.honey_token_retrieved
→ HookEvent: EVENT_HONEY_TOKEN_RETRIEVED
→ severity: MEDIUM (suspicious search pattern)

Full logic: Honey-Token SRS
```

### 5.2 Multi-Tenancy Hooks

**Hook Point:**
```
navigator.tenant.prefilter
- Enforce tenant pre-filter before search
- CRITICAL: prevents cross-tenant data leakage
- Detail: see Multi-tenancy SRS (Tier 3)
```

**Contract:**
```
Pre-filter by tenant_id BEFORE search.
NEVER post-filter only (security risk).

Full logic: Multi-tenancy SRS
```

### 5.3 Lineage Hooks

```
navigator.search.completed
→ Lineage event with trace_id, span_id
Detail: see Data Lineage SRS (Tier 3)
```

### 5.4 Hook Summary

```
🔴 navigator.results.scanned     → Honey-Token SRS
🔴 navigator.tenant.prefilter    → Multi-tenancy SRS
🔴 navigator.search.completed    → Lineage SRS
```

---

## 6. External Interfaces

### 6.1 gRPC Interface

```
Wire format: JSON-over-gRPC (GenericRpcHandler, JSON codec)
Service: bastion.navigator.v1.NavigatorService

Methods:
  Search(SearchRequest) → SearchResponse
  SearchWithPermissions(SearchRequest) → SearchResponse
  HybridSearch(SearchRequest) → SearchResponse
  BatchSearch(BatchSearchRequest) → BatchSearchResponse
  Embed(EmbedRequest) → EmbedResponse
  BatchEmbed(BatchEmbedRequest) → BatchEmbedResponse
  Rerank(RerankRequest) → RerankResponse
  GetCollections(_) → CollectionsResponse
  Health(_) → {"status":"ok","version":"..."}

gRPC metadata headers:
  x-trace-id, x-span-id, x-parent-span-id
  x-tenant-id, x-user-id, x-request-id
```

**SearchRequest (JSON):**
```json
{
  "request_id": "uuid",
  "query": "product warranty terms",
  "tenant_id": "tenant-acme",
  "collections": ["customer_docs"],
  "top_k": 10,
  "user": {
    "user_id": "u-123",
    "allowed_categories": ["customer", "public"]
  },
  "options": {
    "use_hybrid": true,
    "use_reranking": true,
    "vector_weight": 0.7,
    "bm25_weight": 0.3
  }
}
```

**SearchResponse (JSON):**
```json
{
  "request_id": "uuid",
  "results": [
    {
      "document_id": "doc-abc",
      "content": "...",
      "score": 0.92,
      "collection": "customer_docs",
      "metadata": {"category": "customer"}
    }
  ],
  "metadata": {
    "total_candidates": 50,
    "filtered_out": 5,
    "reranked": true,
    "strategy": "hybrid"
  },
  "processing_time_ms": 87
}
```

### 6.2 REST Interface

```
POST /v1/navigator/search
POST /v1/navigator/search/with-permissions
POST /v1/navigator/search/hybrid
POST /v1/navigator/search/batch
POST /v1/navigator/embed
POST /v1/navigator/embed/batch
POST /v1/navigator/rerank
GET  /v1/navigator/collections
GET  /v1/navigator/collections/{name}
GET  /v1/health
GET  /v1/metrics     (Prometheus)
```

**Request headers:**
```
X-Trace-Id, X-Span-Id, X-Tenant-Id, X-User-Id, X-Request-Id
```

### 6.3 CLI Interface

```bash
# Standalone server
$ python -m navigator.main --config config.yaml

# (CLI wrapper)
$ navigator-cli search --query "warranty terms" --tenant tenant-acme
$ navigator-cli server
```

### 6.4 Events (Foundation Schema)

```
Operational (NATS):
  bastion.events.navigator.search_started
  bastion.events.navigator.search_completed

Enhanced:
  bastion.events.navigator.permission_filtered

Via hooks:
  bastion.events.navigator.honey_token_retrieved
```

---

## 7. Non-Functional Requirements

### 7.1 Performance

| ID | Item | Target |
|---|---|---|
| NFR-PE-001 | Vector search (p95) | < 50ms |
| NFR-PE-002 | Hybrid search (p95) | < 80ms |
| NFR-PE-003 | Hybrid+Rerank (p95) | < 150ms |
| NFR-PE-004 | Throughput | ≥ 100 req/s |
| NFR-PE-005 | Memory (10M docs index) | ≤ 16GB |
| NFR-PE-006 | Model RAM at startup | ~2.3GB (bge-m3 + reranker) |

### 7.2 Independence (Foundation)

```
NFR-IND-001: Core works standalone (Qdrant + Python packages only)
NFR-IND-002: Graceful degradation (Vault optional)
NFR-IND-003: Loose coupling (permissions passed in request, no Vault ref held)
```

### 7.3 Startup

```
NFR-START-001: First startup downloads models from HuggingFace (one-time)
NFR-START-002: Subsequent startups load from local cache (~5s)
NFR-START-003: If Qdrant unreachable at startup, fall back to MockSearcher
               (service starts; core search returns empty; logs warning)
```

---

## 8. System Architecture

```
┌────────────────────────────────────────────────┐
│           Navigator Service (Python)            │
├────────────────────────────────────────────────┤
│  API Layer                                      │
│    FastAPI (REST, uvicorn)  │  gRPC server      │
│    port 8082                │  port 9092        │
│         ↓                              ↓        │
│  ─────────────── Orchestrator ──────────────    │
│  15-step search pipeline                        │
│         ↓                                       │
│  ┌──────────────────────────────────────┐       │
│  │  Core Components                     │       │
│  │  LocalEmbedder  (SentenceTransformer)│       │
│  │  QdrantSearcher (qdrant-client SDK)  │       │
│  │  LocalReranker  (CrossEncoder)       │       │
│  └──────────────────────────────────────┘       │
│         ↓                                       │
│  ┌──────────────────────────────────────┐       │
│  │  Optional                            │       │
│  │  VaultClient   (permission filter)   │       │
│  │  HookManager   (cross-cutting)       │       │
│  │  Publisher     (NATS events)         │       │
│  └──────────────────────────────────────┘       │
└────────────────────────────────────────────────┘
         ↓                    ↓
      Qdrant           HuggingFace (first startup)
   (vector store)      (model cache)
```

### 8.1 Component Responsibilities

| Component | Class | Responsibility |
|---|---|---|
| `LocalEmbedder` | `embedder.py` | In-process bge-m3 via SentenceTransformer |
| `BGEHttpEmbedder` | `embedder.py` | HTTP fallback to external embedder service |
| `CachedEmbedder` | `embedder.py` | LRU cache wrapper around any Embedder |
| `QdrantSearcher` | `searcher.py` | Vector + sparse search via qdrant-client |
| `MockSearcher` | `searcher.py` | Stub for testing / Qdrant-unavailable startup |
| `LocalReranker` | `reranker.py` | In-process bge-reranker via CrossEncoder |
| `BGEHttpReranker` | `reranker.py` | HTTP fallback to external reranker service |
| `MockReranker` | `reranker.py` | Stub for testing |
| `VaultClient` | `vault_client.py` | HTTP GET to Vault /v1/vault/permissions |
| `NoopVaultClient` | `vault_client.py` | Passthrough (vault.enabled=false) |
| `Orchestrator` | `orchestrator.py` | 15-step pipeline |
| `HookManager` | `hooks.py` | Thread-per-handler hook dispatch |
| `Publisher` | `events.py` | Async NATS in background thread |

---

## 9. Standalone Operation

### 9.1 Startup Log

```
[navigator] starting v3.0 (REST :8082, gRPC :9092)
[navigator] loading embedder: local (BAAI/bge-m3)
[navigator] loading reranker: local (BAAI/bge-reranker-v2-m3)
[navigator] qdrant connected (collections: customer_docs, mfg_docs)
[navigator] vault: disabled (no permission filter)
[navigator] grpc listening on :9092
[navigator] ready
```

### 9.2 Graceful Degradation

```
Qdrant unreachable at startup:
  → MockSearcher active (empty results)
  → Core service starts; NATS events fire normally
  → Warning logged; Vault/hooks unaffected

Vault disabled:
  → Permission filter inactive
  → Core search all collections
  → SearchWithPermissions: validates user.allowed_categories
    but no Vault round-trip
```

### 9.3 Standalone Test (Litmus)

```
POST /v1/navigator/search
{"query": "product warranty", "top_k": 5}

→ 200 OK
{
  "results": [...],
  "metadata": {"strategy": "hybrid", "reranked": true},
  "processing_time_ms": 112
}

(No Vault, Sentinel, Anchor, or Tracker needed)
```

---

## 10. Configuration

```yaml
# /etc/bastion-navigator/config.yaml
version: "3.0"

server:
  rest_port: 8082
  grpc_port: 9092
  workers: 1

embedder:
  type: local             # local | bge_http
  model_name: BAAI/bge-m3
  max_length: 8192
  cache_size: 1000        # LRU in-memory cache entries

reranker:
  type: local             # local | bge_http
  model_name: BAAI/bge-reranker-v2-m3
  max_length: 512

vector_db:
  hosts:
    - host: qdrant
      port: 6333

search:
  default_top_k: 10
  default_use_hybrid: true
  default_use_reranking: true
  default_vector_weight: 0.7
  default_bm25_weight: 0.3
  over_fetch_multiplier: 5
  rrf_k: 60

vault:
  enabled: false          # true = fetch permissions from Vault
  endpoint: http://vault:8081

events:
  nats_url: nats://nats:4222
```

---

## 11. Summary

```
🟢 Core (Standalone, Python):
   - In-process embedding (bge-m3, SentenceTransformer)
   - Vector search (qdrant-client SDK)
   - BM25/text search (qdrant-client scroll)
   - Hybrid fusion (RRF k=60)
   - In-process reranking (bge-reranker, CrossEncoder)
   - 15-step search pipeline in Orchestrator

🟡 Enhanced (Composition):
   - Permission filter (+ Vault) [CRITICAL for security]
   - Context provision (+ Sentinel-OUT)

🔴 Hooks (Cross-cutting):
   - Honey-token detection (→ Honey-Token SRS)
   - Multi-tenancy pre-filter (→ Multi-tenancy SRS)
   - Lineage (→ Lineage SRS)

Wire contract: unchanged from v2 (JSON-over-gRPC, REST JSON)
Language change: Go → Python (internal implementation detail only)
```

---

## 12. Change History

| Version | Date | Changes |
|---|---|---|
| 1.0 | 2026-05-17 | Initial |
| 2.0 | 2026-05-17 | Foundation-aligned, layer classification |
| 3.0 | 2026-05-26 | **Python rewrite**: in-process ML via sentence-transformers; qdrant-client Python SDK; FastAPI + grpcio |

---

**End of Document**

## Appendix: Cross-cutting References

```
Honey-Token: hook results.scanned → Honey-Token SRS
Multi-tenancy: pre-filter (CRITICAL) → Multi-tenancy SRS
Lineage: search.completed → Lineage SRS
```

## Appendix: v2 → v3 Migration Notes

```
Interface changes: NONE
  - Same gRPC service name and methods
  - Same REST endpoints
  - Same JSON request/response schemas
  - Same NATS event subjects

Internal changes:
  - Language: Go → Python 3.11+
  - Embedding: BGEClient (HTTP) → LocalEmbedder (in-process SentenceTransformer)
  - Reranking: BGERerankerClient (HTTP) → LocalReranker (in-process CrossEncoder)
  - Vector DB: Qdrant REST client → qdrant-client Python SDK
  - gRPC: ServiceDesc (protobuf) → GenericRpcHandler (JSON codec)
  - Config: Go struct + viper → Pydantic model + pyyaml
  - Models: Go structs + JSON tags → Pydantic v2 models

Old Go source: navigator/internal/ (archived; superseded)
New Python source: navigator/navigator/
```
