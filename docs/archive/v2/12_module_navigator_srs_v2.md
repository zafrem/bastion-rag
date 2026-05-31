# Bastion-Navigator Module SRS

**Project:** Bastion-RAG - RAG Security Governance Framework  
**Document Type:** Module SRS (Tier 2)  
**Document ID:** 12-navigator-srs  
**Module:** C - Navigator (Search & Ranking)  
**Version:** 2.0 (Foundation-aligned)  
**Date:** 2026-05-17  
**Status:** Draft

**Foundation References:**
- 01-architecture-principles
- 02-event-schema-standard
- 03-module-interaction-map

---

## 1. Introduction

### 1.1 Purpose

This document specifies the **Navigator** module, the search and ranking layer of Bastion-RAG. Unlike Sentinel/Vault/Anchor, Navigator is **single-direction** (search happens on input path; output validation is delegated to Sentinel-OUT).

### 1.2 Module Identity

```
Module: C - Navigator
Role: Search & Ranking
Position: After Vault (input path)
Direction: Single (search)

Standalone value:
"Attach Navigator alone → safe hybrid search for RAG"
```

### 1.3 The Standalone Test (Foundation Litmus)

```
Question: "If only Navigator is attached to an LLM,
          does it provide meaningful value?"

Answer: YES
- Hybrid search (vector + BM25)
- Reranking
- Quality retrieval for RAG

→ Navigator passes the standalone test ✅
(Note: security value is in Enhanced/Hooks)
```

### 1.4 Why Single-Direction?

```
Navigator searches on INPUT path.
On OUTPUT path, there's nothing to "search".

Output-side concerns (response validation)
are handled by Sentinel-OUT, which may
USE Navigator's context (Enhanced composition).

So Navigator is single-direction by nature.
```

### 1.5 Scope

**In Scope:**
- 🟢 Core: Hybrid search (vector + BM25)
- 🟢 Core: Reranking
- 🟢 Core: Query embedding (BGE-M3)
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

### 1.6 Definitions

| Term | Definition |
|---|---|
| **Hybrid search** | Vector + BM25 combined |
| **RRF** | Reciprocal Rank Fusion |
| **Reranking** | Cross-encoder re-scoring |
| **Pre-filter** | Filter before search |
| **Quasi-identifier** | Combinable identifying field |

---

## 2. Overall Description

### 2.1 Architecture

```
┌─────────────────────────────────────────────┐
│           Navigator Service                  │
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
│  │(BGE-M3)││(Qdrant)││(BGE-RR) │              │
│  └───────┘└───────┘└──────────┘              │
│               │                              │
│      ┌────────┴────────┐                     │
│      ▼                 ▼                      │
│  ┌──────────┐    ┌──────────┐               │
│  │Enhancement│    │  Hooks   │               │
│  │(optional)│    │(optional)│               │
│  └──────────┘    └──────────┘               │
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
   - Reranking
   - Query embedding

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
Language: Go 1.21+
Vector DB: Qdrant (self-hosted)
Embedding: BGE-M3 (multilingual)
Reranker: BGE-reranker-v2-m3
Scale: 1M-10M docs (SMB)
Memory: ≤ 16GB
Latency: <150ms p95 (hybrid+rerank)
```

### 2.5 Dependencies

```
Core dependencies:
- Qdrant (vector DB)
- Embedding service (BGE-M3)
- Reranker service (BGE-RR)

Optional (Enhanced):
- Vault (permissions)

Optional (Hooks):
- NATS, coordinators
```

---

## 3. Core Functions (🟢 Standalone)

### 3.1 Query Embedding (FR-CORE-EM)

**FR-CORE-EM-001: Multilingual Embedding**
```
Model: BGE-M3 (1024-dim)
Languages: Korean + English
Self-hosted (no external API)
Dependency: Embedding service only
```

### 3.2 Vector Search (FR-CORE-VS)

**FR-CORE-VS-001: HNSW Search**
```
Qdrant HNSW-based ANN
Cosine similarity
Configurable ef_search
```

**FR-CORE-VS-002: Multi-Collection**
```
Collections: customer_docs, manufacturing_docs, hr_docs
(Per category)
```

### 3.3 Keyword Search (FR-CORE-BM)

**FR-CORE-BM-001: BM25**
```
Lexical matching
Korean (Nori) + English analyzers
Exact term/code matching
```

### 3.4 Hybrid Search (FR-CORE-HS)

**FR-CORE-HS-001: RRF Fusion**
```
Combine vector + BM25
Reciprocal Rank Fusion (k=60)
Configurable weights
```

### 3.5 Reranking (FR-CORE-RR)

**FR-CORE-RR-001: Cross-encoder**
```
Model: BGE-reranker-v2-m3
Rerank top candidates
Default: top 50 → top 10
```

### 3.6 Core Summary

```
Standalone capabilities (Qdrant+models):
✅ Query embedding
✅ Vector search
✅ BM25 search
✅ Hybrid (RRF)
✅ Reranking

Quality RAG search without other Bastion-RAG modules.
```

---

## 4. Enhanced Functions (🟡 Composition)

### 4.1 Permission Filtering (FR-ENH-PF)

**Requires: Vault (provides permissions)**

**FR-ENH-PF-001: Pre-filter by Permission**
```
When Vault permissions available:
- Pre-filter search by allowed categories
- CRITICAL: pre-filter (not post-filter)
- Prevent unauthorized results

Interface (Foundation - permissions passed in):
SearchWithPermissions(query, permissions)

Graceful degradation:
- Without Vault: search all (less secure)
- Core search still works
```

**FR-ENH-PF-002: Over-fetching**
```
Fetch more candidates for permission filter
Default: top_k * 5
Adapt to filter rate
```

### 4.2 Context Provision (FR-ENH-CP)

**Requires: Sentinel-OUT (consumes context)**

**FR-ENH-CP-001: Search Context Export**
```
Provide search results to Sentinel-OUT
for indirect injection detection.

Sentinel-OUT uses to verify response grounding.

Graceful degradation:
- Without Sentinel-OUT: context unused
- Core search unaffected
```

### 4.3 Enhanced Summary

```
🟡 Permission filtering (+ Vault)
   CRITICAL for multi-tenancy
🟡 Context provision (+ Sentinel-OUT)

Without composition: core search works
(but less secure)
```

---

## 5. Hooks (🔴 Cross-Cutting)

### 5.1 Honey-Token Hooks

**Hook Points:**
```
navigator.results.scanned
- After search, scan results for honey-tokens
- Detail: see Honey-Token SRS
```

**Brief Contract:**
```
On honey-token in search results:
→ event: navigator.honey_token_retrieved
→ severity: MEDIUM (suspicious search pattern)

Full logic: Honey-Token SRS (Tier 3).
```

### 5.2 Multi-Tenancy Hooks

**Hook Points:**
```
navigator.tenant.prefilter
- Enforce tenant pre-filter
- CRITICAL: prevents cross-tenant leak
- Detail: see Multi-tenancy SRS
```

**Brief Contract:**
```
Pre-filter by tenant_id BEFORE search.
NEVER post-filter (security risk).

Full logic: Multi-tenancy SRS (Tier 3).
```

### 5.3 Lineage Hooks

```
navigator.search.completed
→ Lineage event with trace_id
Detail: see Data Lineage SRS
```

### 5.4 Hook Summary

```
🔴 navigator.results.scanned   → Honey-Token SRS
🔴 navigator.tenant.prefilter  → Multi-tenancy SRS
🔴 navigator.search.completed  → Lineage SRS
```

---

## 6. External Interfaces

### 6.1 gRPC Interface

```protobuf
service NavigatorService {
  // Core
  rpc Search(SearchRequest) returns (SearchResponse);
  
  // Enhanced (permissions passed in)
  rpc SearchWithPermissions(PermSearchRequest) returns (SearchResponse);
  
  rpc Health(HealthRequest) returns (HealthResponse);
}

message SearchRequest {
  string request_id = 1;
  string trace_id = 2;
  string tenant_id = 3;
  string query = 4;
  SearchOptions options = 5;
}

message PermSearchRequest {
  SearchRequest base = 1;
  // Enhanced: Vault permissions passed in
  repeated string allowed_categories = 2;
}

message SearchResponse {
  string request_id = 1;
  repeated SearchResult results = 2;
  SearchMetadata metadata = 3;
}
```

### 6.2 REST Interface

```
# Core
POST /v1/navigator/search

# Enhanced
POST /v1/navigator/search/with-permissions

# Standard
GET  /v1/health
GET  /v1/metrics
```

### 6.3 CLI Interface

```bash
# Core search
$ navigator-cli search \
    --query "warranty terms" \
    --tenant tenant-acme

# Standalone
$ navigator-cli server
```

### 6.4 Events (Foundation Schema)

```
Operational:
- navigator.search_started
- navigator.search_completed

Enhanced:
- navigator.permission_filtered

Via hooks:
- navigator.honey_token_retrieved
```

---

## 7. Non-Functional Requirements

### 7.1 Performance

| ID | Item | Target |
|---|---|---|
| NFR-PE-001 | Vector search (p95) | < 50ms |
| NFR-PE-002 | Hybrid search (p95) | < 80ms |
| NFR-PE-003 | Hybrid+Rerank (p95) | < 150ms |
| NFR-PE-004 | Throughput | ≥ 100/s |
| NFR-PE-005 | Memory (10M docs) | ≤ 16GB |

### 7.2 Independence (Foundation)

```
NFR-IND-001: Core works standalone (Qdrant+models)
NFR-IND-002: Graceful degradation (Vault optional)
NFR-IND-003: Loose coupling (permissions passed in)
```

---

## 8. System Architecture

```
┌────────────────────────────────────────────┐
│           Navigator Service                 │
├────────────────────────────────────────────┤
│  API (gRPC/REST/CLI)                        │
│         ↓                                   │
│  Search Orchestrator                        │
│         ↓                                   │
│  ┌─────────────────────────────────┐        │
│  │  Core: Embed → Search → Rerank  │        │
│  └─────────────────────────────────┘        │
│         ↓                                   │
│  Enhancement: Permission filter (optional)  │
│         ↓                                   │
│  Hook Manager (optional)                    │
│         ↓                                   │
│  Event Publisher (NATS)                     │
└────────────────────────────────────────────┘
         ↓
   Qdrant + Embedder + Reranker
```

---

## 9. Standalone Operation

### 9.1 Standalone Mode

```bash
$ navigator-cli server

🚀 Bastion-Navigator v2.0 starting...
✅ Qdrant connected (5M vectors)
✅ Embedder ready (BGE-M3)
✅ Reranker ready
⚠️  Vault: not connected (no permission filter)
✅ Core search: FULLY OPERATIONAL
✨ Ready (standalone)
```

### 9.2 Standalone Test (Litmus)

```bash
$ navigator-cli search \
    --query "product warranty" \
    --standalone

✅ Found 10 results (87ms)
Strategy: hybrid + rerank
(No other modules needed for search)
```

### 9.3 Degradation

```
Without other modules:
✅ Hybrid search: works
✅ Reranking: works
⚠️ Permission filter: inactive (needs Vault)
⚠️ Honey-token: inactive

Core search fully functional ✅
```

---

## 10. Configuration

```yaml
# /etc/bastion-navigator/config.yaml
version: 2.0

# Core
core:
  vector_db:
    type: qdrant
    collections: [customer_docs, manufacturing_docs, hr_docs]
  embedder:
    model: bge_m3
    dimensions: 1024
  reranker:
    model: bge_reranker_v2_m3
  search:
    use_hybrid: true
    use_reranking: true
    vector_weight: 0.7
    bm25_weight: 0.3

# Enhanced
enhanced:
  permission_filter: true  # If Vault present
  over_fetch_multiplier: 5

# Hooks
hooks:
  honey_token: false
  multi_tenancy: true  # Pre-filter
  lineage: true

# Events
events:
  nats_url: nats://nats:4222
```

---

## 11. Summary

```
🟢 Core (Standalone):
   - Hybrid search (vector + BM25)
   - Reranking
   - Query embedding

🟡 Enhanced (Composition):
   - Permission filter (+ Vault) [CRITICAL for security]
   - Context provision (+ Sentinel-OUT)

🔴 Hooks (Cross-cutting):
   - Honey-token detection (→ Honey-Token SRS)
   - Multi-tenancy pre-filter (→ Multi-tenancy SRS)
   - Lineage (→ Lineage SRS)

Note: Navigator is single-direction (search).
Security value mostly in Enhanced + Hooks.
```

---

## 12. Change History

| Version | Date | Changes |
|---|---|---|
| 1.0 | 2026-05-17 | Initial |
| 2.0 | 2026-05-17 | Foundation-aligned, layer classification |

---

**End of Document**

## Appendix: Cross-cutting References

```
Honey-Token: hook results.scanned
Multi-tenancy: pre-filter (CRITICAL)
Lineage: search.completed
```
