# Bastion-Navigator System Requirements Specification (SRS) v1.0

**Project:** Bastion - RAG Security Governance Framework  
**Module:** Module C - Navigator (Search & Ranking Layer)  
**Document Version:** 1.0  
**Date:** 2026-05-17  
**Status:** Draft  
**Target Scale:** SMB / Research (1M-10M documents)

---

## 1. Introduction

### 1.1 Purpose

This document defines the functional and non-functional requirements for the **Bastion-Navigator** module. Navigator is the search and ranking layer of the RAG (Retrieval-Augmented Generation) pipeline, responsible for:

1. **Vector Search** - Embedding-based similarity search
2. **Hybrid Search** - Combined vector and keyword search
3. **Reranking** - Result quality improvement
4. **Logical Partitioning** - Tenant-aware search scoping
5. **Access-Aware Filtering** - Permission-based result filtering

### 1.2 Scope

**In Scope:**
- Vector database integration (Qdrant)
- Self-hosted embedding service (BGE-M3, multilingual)
- Cross-encoder reranking (BGE-reranker)
- Hybrid search combining vector and BM25
- Tenant-aware search with pre-filtering
- Integration with Vault (Module B) for access control
- Standalone execution and testing
- API-based input/output (gRPC, REST)
- Manual operations via CLI

**Out of Scope:**
- Document parsing and chunking (delegated to separate Indexer service)
- Input validation (Module A - Sentinel responsibility)
- Data anonymization (Module B - Vault responsibility)
- Embedding security (Module E - Anchor responsibility)
- Audit log storage (Module D - Tracker responsibility)
- LLM response generation
- Large-scale (>100M documents) - requires architectural changes

### 1.3 Target Scale

This SRS is optimized for:
- **Organization Size:** SMB (Small-Medium Business) / Research labs
- **Document Count:** 1M - 10M documents
- **Concurrent Users:** Up to 1,000
- **Tenants:** Up to 100
- **Infrastructure:** On-premise (cloud-optional)
- **GPU Usage:** Minimal (CPU-first design)
- **Languages:** Korean + English (50/50)

### 1.4 Definitions and Acronyms

| Term | Definition |
|---|---|
| **Navigator** | Module C of the Bastion framework (search layer) |
| **RAG** | Retrieval-Augmented Generation |
| **Vector DB** | Database optimized for vector similarity search |
| **Embedding** | Numerical vector representation of text |
| **Hybrid Search** | Combining vector and keyword (BM25) search |
| **Reranking** | Re-ordering search results for relevance |
| **BM25** | Best Matching 25 - lexical search algorithm |
| **RRF** | Reciprocal Rank Fusion - result combination |
| **Pre-filter** | Filtering before search execution |
| **Cross-encoder** | Model that scores query-document pairs |
| **HNSW** | Hierarchical Navigable Small World (vector index) |
| **K** | Number of results returned |
| **NDCG** | Normalized Discounted Cumulative Gain |
| **MRR** | Mean Reciprocal Rank |

### 1.5 References

- ISO/IEC/IEEE 29148:2018
- Qdrant Documentation
- BGE Model Card (BAAI)
- BM25 Algorithm (Robertson & Zaragoza)
- RAG Best Practices (OWASP LLM Top 10)

---

## 2. Overall Description

### 2.1 Product Perspective

Navigator is the third module in the Bastion framework pipeline.

```
┌──────────────────────────────────────────────────────┐
│              User Query                               │
└────────────────────────┬─────────────────────────────┘
                         ▼
        ┌────────────────────────────────────┐
        │   Module A: Sentinel                │
        │   - Input Validation                │
        └────────────────┬───────────────────┘
                         │ (validated)
                         ▼
        ┌────────────────────────────────────┐
        │   Module B: Vault                   │
        │   - PII Anonymization               │
        │   - Access Permissions              │
        └────────────────┬───────────────────┘
                         │ (anonymized query + permissions)
                         ▼
        ┌────────────────────────────────────┐
        │   Module C: NAVIGATOR  ◄── (This doc)│
        │   - Vector Search                   │
        │   - Hybrid Search                   │
        │   - Reranking                       │
        │   - Tenant Partitioning             │
        └────────────────┬───────────────────┘
                         │ (top-K relevant docs)
                         ▼
        ┌────────────────────────────────────┐
        │   Module E: Anchor (Embedding Sec.) │
        └────────────────┬───────────────────┘
                         ▼
        ┌────────────────────────────────────┐
        │              LLM                    │
        └────────────────────────────────────┘
                         │
                         ▼ (async)
        ┌────────────────────────────────────┐
        │   Module D: Tracker (Audit)        │
        └────────────────────────────────────┘
```

### 2.2 Product Functions

1. **F1: Query Embedding**
   - Convert text query to vector representation
   - Multilingual support (Korean + English)
   - Self-hosted (no external API)

2. **F2: Vector Search**
   - Approximate nearest neighbor (ANN) search
   - HNSW-based indexing
   - Multi-tenant aware

3. **F3: Keyword Search (BM25)**
   - Lexical matching for exact terms
   - Handles product codes, technical terms

4. **F4: Hybrid Search**
   - Combine vector + BM25 with RRF
   - Configurable weights

5. **F5: Reranking**
   - Cross-encoder re-scoring
   - Top-K result refinement

6. **F6: Tenant Partitioning**
   - Logical isolation via metadata filtering
   - Pre-filter for security

7. **F7: Access Control Integration**
   - Query Vault for user permissions
   - Filter results by category

8. **F8: Result Post-processing**
   - Send results to Vault for anonymization
   - Apply user-specific access level

9. **F9: Multiple Interfaces**
   - gRPC, REST, CLI, batch file

10. **F10: Independent Operation**
    - Standalone mode without other modules
    - Mock external services

### 2.3 User Characteristics

| User Type | Purpose | Interface |
|---|---|---|
| **AI System (RAG)** | Automated retrieval | gRPC |
| **Application** | Search API | REST |
| **Developer** | Testing, debugging | CLI |
| **Data Scientist** | Query experimentation | REST/CLI |
| **System Admin** | Index management | CLI |

### 2.4 Constraints

- **Language:** Go 1.21+
- **Vector DB:** Qdrant (self-hosted)
- **Embedding Model:** BGE-M3 (multilingual, self-hosted)
- **Reranker:** BGE-reranker (Cross-encoder, self-hosted)
- **Keyword Search:** BM25 via Elasticsearch or Qdrant's built-in
- **Memory:** Up to 16GB per pod (higher for large indexes)
- **CPU-first:** GPU optional for embedding/reranking acceleration
- **Storage:** SSD recommended (NVMe ideal)

### 2.5 Assumptions and Dependencies

**Assumptions:**
- Documents are pre-indexed by separate Indexer service
- Vault provides access permissions
- Sentinel validates input before reaching Navigator
- On-premise deployment (cloud optional)

**External Dependencies:**
- Qdrant 1.10+ (vector database)
- Embedding service (BGE-M3 via FastAPI/ONNX)
- Reranker service (BGE-reranker via ONNX)
- Redis 7.0+ (query cache, optional)
- Module B (Vault) - access control
- PostgreSQL 15+ (metadata, audit)

---

## 3. External Interface Requirements

### 3.1 Interface Overview

Navigator follows the same interface pattern as Sentinel and Vault.

| Category | Interface | Target Users |
|---|---|---|
| **Input** | gRPC | AI systems, internal modules |
| **Input** | REST API (JSON) | External applications |
| **Input** | CLI | Developers, operators |
| **Input** | File input (JSONL) | Batch processing |
| **Output** | Protobuf | gRPC responses |
| **Output** | JSON | REST responses |
| **Output** | Text | Human-readable |
| **Output** | File output | Batch results |

### 3.2 Input Interface 1: gRPC (System)

```protobuf
syntax = "proto3";
package bastion.navigator.v1;

service NavigatorService {
  // Main search operations
  rpc Search(SearchRequest) returns (SearchResponse);
  rpc HybridSearch(HybridSearchRequest) returns (SearchResponse);
  rpc BatchSearch(BatchSearchRequest) returns (BatchSearchResponse);
  
  // Embedding operations
  rpc Embed(EmbedRequest) returns (EmbedResponse);
  rpc BatchEmbed(BatchEmbedRequest) returns (BatchEmbedResponse);
  
  // Reranking
  rpc Rerank(RerankRequest) returns (RerankResponse);
  
  // Management
  rpc GetCollections(CollectionsRequest) returns (CollectionsResponse);
  rpc Health(HealthRequest) returns (HealthResponse);
}

message SearchRequest {
  string request_id = 1;
  string tenant_id = 2;
  string query = 3;
  UserContext user = 4;
  SearchOptions options = 5;
}

message UserContext {
  string user_id = 1;
  string department = 2;
  repeated string roles = 3;
  repeated string allowed_categories = 4;  // From Vault
}

message SearchOptions {
  int32 top_k = 1;              // Final results
  int32 over_fetch = 2;          // Initial retrieval (for permission filter)
  bool use_reranking = 3;
  bool use_hybrid = 4;
  float vector_weight = 5;       // 0.0 - 1.0
  float bm25_weight = 6;         // 0.0 - 1.0
  map<string, string> filters = 7;
  float min_score = 8;
  int32 timeout_ms = 9;
}

message SearchResponse {
  string request_id = 1;
  repeated SearchResult results = 2;
  SearchMetadata metadata = 3;
  float processing_time_ms = 4;
}

message SearchResult {
  string document_id = 1;
  string content = 2;
  float score = 3;
  float vector_score = 4;
  float bm25_score = 5;
  float rerank_score = 6;
  map<string, string> metadata = 7;
  string category = 8;
}

message SearchMetadata {
  int32 total_candidates = 1;
  int32 filtered_out = 2;
  int32 final_count = 3;
  bool used_cache = 4;
  string strategy = 5;  // "vector", "hybrid", "rerank"
}
```

### 3.3 Input Interface 2: REST API

**Endpoints:**

```
# Search operations
POST /v1/navigator/search                # Main search
POST /v1/navigator/search/hybrid         # Explicit hybrid
POST /v1/navigator/search/batch          # Batch search

# Embedding
POST /v1/navigator/embed                 # Get embedding
POST /v1/navigator/embed/batch           # Batch embedding

# Reranking
POST /v1/navigator/rerank                # Rerank existing results

# Management
GET  /v1/navigator/collections           # List collections
GET  /v1/navigator/collections/{name}    # Collection info
GET  /v1/health                          # Health check
GET  /v1/metrics                         # Prometheus metrics
```

**Request Example:**

```http
POST /v1/navigator/search HTTP/1.1
Host: navigator.bastion.local
Content-Type: application/json
Authorization: Bearer <jwt-token>

{
  "request_id": "req-nav-001",
  "tenant_id": "tenant-acme",
  "query": "What are the warranty terms for product PROD-001?",
  "user": {
    "user_id": "user-alice",
    "department": "marketing",
    "roles": ["marketing_staff"],
    "allowed_categories": ["customer_data", "manufacturing_data"]
  },
  "options": {
    "top_k": 10,
    "over_fetch": 50,
    "use_reranking": true,
    "use_hybrid": true,
    "vector_weight": 0.7,
    "bm25_weight": 0.3,
    "filters": {
      "product_id": "PROD-001"
    },
    "min_score": 0.6,
    "timeout_ms": 500
  }
}
```

**Response Example:**

```json
{
  "request_id": "req-nav-001",
  "results": [
    {
      "document_id": "doc-123",
      "content": "Warranty for PROD-001: 2-year manufacturing defect coverage...",
      "score": 0.92,
      "vector_score": 0.89,
      "bm25_score": 0.95,
      "rerank_score": 0.94,
      "metadata": {
        "product_id": "PROD-001",
        "doc_type": "warranty",
        "language": "en"
      },
      "category": "manufacturing_data"
    },
    {
      "document_id": "doc-456",
      "content": "제품 PROD-001 보증 조건...",
      "score": 0.88,
      "vector_score": 0.91,
      "bm25_score": 0.82,
      "rerank_score": 0.87,
      "metadata": {
        "product_id": "PROD-001",
        "doc_type": "warranty",
        "language": "ko"
      },
      "category": "manufacturing_data"
    }
  ],
  "metadata": {
    "total_candidates": 50,
    "filtered_out": 12,
    "final_count": 10,
    "used_cache": false,
    "strategy": "hybrid+rerank"
  },
  "processing_time_ms": 87.3
}
```

### 3.4 Input Interface 3: CLI

```bash
# Single search
$ navigator-cli search \
    --tenant tenant-acme \
    --query "warranty for PROD-001" \
    --top-k 10 \
    --user-id alice \
    --department marketing

# Hybrid search with custom weights
$ navigator-cli search \
    --query "공정 불량률" \
    --hybrid \
    --vector-weight 0.6 \
    --bm25-weight 0.4 \
    --rerank

# Batch search
$ navigator-cli search \
    --input-file queries.jsonl \
    --output-file results.jsonl

# Embedding only
$ navigator-cli embed \
    --text "Hello world" \
    --output-format json

# Rerank existing results
$ navigator-cli rerank \
    --query "warranty terms" \
    --candidates-file candidates.jsonl \
    --top-k 5

# Interactive mode
$ navigator-cli interactive
nav> search
Query: 제품 보증 조건
Tenant: tenant-acme
✅ Found 10 results in 87ms

nav> stats
Searches: 234
Avg latency: 92ms
Cache hit rate: 23%

nav> collections
- customer_docs (1.2M vectors, 768d)
- manufacturing_docs (456K vectors, 768d)
- hr_docs (12K vectors, 768d)

nav> exit

# Server mode
$ navigator-cli server --port 8080 --grpc-port 9090
```

**CLI Options:**

| Option | Description |
|---|---|
| `--query` | Search query |
| `--tenant` | Tenant ID |
| `--top-k` | Number of results |
| `--over-fetch` | Initial retrieval size |
| `--hybrid` | Enable hybrid search |
| `--rerank` | Enable reranking |
| `--vector-weight` | Vector search weight (0-1) |
| `--bm25-weight` | BM25 weight (0-1) |
| `--filter` | Metadata filter (key=value) |
| `--min-score` | Minimum relevance score |
| `--user-id` | User context |
| `--department` | User department |
| `--output-format` | text/json/compact |

### 3.5 Output Format 1: Structured (JSON/Protobuf)

As defined in 3.2 and 3.3.

### 3.6 Output Format 2: Text (Human-Readable)

```
════════════════════════════════════════════════════════
  Bastion-Navigator Search Result
════════════════════════════════════════════════════════
Request ID:   req-nav-001
Tenant:       tenant-acme
Query:        "warranty for PROD-001"
Strategy:     Hybrid Search + Reranking
Time:         87.3 ms

─── Search Metadata ───────────────────────────────────
Initial candidates:  50
Permission filtered: 12
Final results:        10
Cache hit:           No

─── Top Results ───────────────────────────────────────

#1  [score: 0.92] doc-123  (manufacturing_data)
    Vector: 0.89 | BM25: 0.95 | Rerank: 0.94
    "Warranty for PROD-001: 2-year manufacturing 
     defect coverage..."
    Metadata: product_id=PROD-001, lang=en

#2  [score: 0.88] doc-456  (manufacturing_data)
    Vector: 0.91 | BM25: 0.82 | Rerank: 0.87
    "제품 PROD-001 보증 조건..."
    Metadata: product_id=PROD-001, lang=ko

#3  [score: 0.85] doc-789  (manufacturing_data)
    ...

─── Quality Metrics ───────────────────────────────────
Diversity score:     0.78 (good)
Coverage:            High (multiple sources)
Confidence:          0.91 (high)

─── Next Action ───────────────────────────────────────
→ Forward to Module E (Anchor)
→ Then to LLM for generation
════════════════════════════════════════════════════════
```

**Compact Format:**

```
[req-001] search query="warranty" results=10 time=87ms strategy=hybrid+rerank
[req-002] search query="공정 불량" results=8 time=92ms cache=hit
```

### 3.7 File Formats

**JSONL Input (batch queries):**

```jsonl
{"request_id":"q1","tenant_id":"acme","query":"warranty terms","top_k":5}
{"request_id":"q2","tenant_id":"acme","query":"return policy","top_k":5}
{"request_id":"q3","tenant_id":"globex","query":"제품 보증","top_k":10}
```

### 3.8 Monitoring Interface

**Prometheus Metrics:**

```
# Search operations
navigator_searches_total{tenant="acme",strategy="hybrid"} 12345
navigator_search_duration_seconds_bucket{le="0.1"} 11000
navigator_search_duration_seconds_bucket{le="0.5"} 12300

# Quality metrics
navigator_results_returned_total 45678
navigator_results_filtered_total 5432
navigator_zero_result_searches_total 123

# Embedding
navigator_embeddings_generated_total 6789
navigator_embedding_duration_seconds_bucket{le="0.05"} 6700

# Reranking
navigator_rerankings_total 5678
navigator_rerank_duration_seconds_bucket{le="0.1"} 5600

# Cache
navigator_cache_hits_total 2345
navigator_cache_misses_total 7891

# Vector DB
navigator_qdrant_calls_total{operation="search"} 12345
navigator_qdrant_call_duration_seconds_bucket{le="0.05"} 12000
```

---

## 4. Functional Requirements

### 4.1 Query Embedding (FR-EM)

**FR-EM-001: Multilingual Embedding**
- Support Korean and English natively
- Model: BGE-M3 (multilingual, 1024 dimensions)
- Self-hosted via ONNX Runtime

**FR-EM-002: Embedding Performance**
- Single query: <50ms (CPU)
- Batch (10 queries): <200ms
- GPU acceleration: optional 5x speedup

**FR-EM-003: Embedding Caching**
- Cache frequent query embeddings
- Redis with TTL 1 hour
- LRU eviction policy

**FR-EM-004: Embedding Normalization**
- L2 normalize all vectors
- Consistent with index

### 4.2 Vector Search (FR-VS)

**FR-VS-001: HNSW-based Search**
- Use Qdrant's HNSW implementation
- Configurable ef_search parameter
- Default: ef_search=128 (balance speed/accuracy)

**FR-VS-002: Multi-Collection Search**
- Separate collections per data category:
  - `customer_docs` (DC-01)
  - `manufacturing_docs` (DC-02)
  - `hr_docs` (DC-03)
- User accesses only permitted collections

**FR-VS-003: Tenant Pre-filtering**
- Apply tenant_id filter BEFORE vector search
- Prevent cross-tenant leakage
- Use Qdrant's payload filter

**FR-VS-004: Score Threshold**
- Configurable minimum cosine similarity
- Default: 0.5
- Filter low-confidence results

### 4.3 Keyword Search / BM25 (FR-BM)

**FR-BM-001: BM25 Implementation**
- Use Qdrant's sparse vector support
- Or Elasticsearch as alternative
- Multilingual analyzer (Korean + English)

**FR-BM-002: Korean Tokenization**
- Korean text: Nori or Mecab-ko analyzer
- English text: Standard analyzer
- Mixed text: Multi-language analyzer

**FR-BM-003: Exact Match Boost**
- Boost exact phrase matches
- Boost product codes, technical terms
- Configurable per collection

### 4.4 Hybrid Search (FR-HS)

**FR-HS-001: RRF (Reciprocal Rank Fusion)**
- Combine vector and BM25 results
- Formula: `score = 1/(rank + k)` where k=60
- Sum scores across methods

**FR-HS-002: Weighted Combination**
- Alternative: weighted sum
- `score = α * vector_score + (1-α) * bm25_score`
- Default α=0.7 (vector weighted higher)

**FR-HS-003: Configurable Strategy**
- Per-query strategy selection
- Default: hybrid with RRF
- Fallback: vector-only on BM25 failure

### 4.5 Reranking (FR-RR)

**FR-RR-001: Cross-encoder Reranking**
- Model: BGE-reranker-v2-m3 (multilingual)
- Re-score top-K candidates
- Default: rerank top 50 → return top 10

**FR-RR-002: Reranking Performance**
- <100ms for 50 candidates (CPU)
- <30ms with GPU
- Batch processing

**FR-RR-003: Reranking Skip Conditions**
- Skip if results < 5 (no benefit)
- Skip on timeout pressure
- Configurable via options

### 4.6 Tenant Partitioning (FR-TP)

**FR-TP-001: Logical Partitioning**
- Single Qdrant collection per category
- Filter by `tenant_id` field
- Pre-filter (NEVER post-filter)

**FR-TP-002: Tenant Isolation Verification**
- Validate tenant_id matches user context
- Reject mismatches with security alert

**FR-TP-003: Cross-Tenant Search Prevention**
- Always include tenant filter
- Test coverage for isolation

### 4.7 Access Control Integration (FR-AC)

**FR-AC-001: Vault Permission Query**
- Query Vault for user's allowed categories
- Cache permissions for 5 minutes
- Refresh on permission change events

**FR-AC-002: Category-based Result Filtering**
- Filter search results by allowed categories
- Apply slice/aggregated access rules
- Coordinate with Vault for re-anonymization

**FR-AC-003: Over-fetching for Permission Filter**
- Retrieve more candidates than top_k
- Default: over_fetch = top_k * 5
- Adapt based on filter rate

### 4.8 Result Post-processing (FR-PP)

**FR-PP-001: Vault Integration for Anonymization**
- Send results to Vault for context-aware anonymization
- Based on user's access level
- Async option for performance

**FR-PP-002: Snippet Generation**
- Extract relevant text snippets
- Highlight matched terms
- Configurable snippet length (default: 200 chars)

**FR-PP-003: Metadata Enrichment**
- Add scoring breakdown
- Include retrieval strategy
- Provide source information

### 4.9 Operational Features (FR-OP)

**FR-OP-001: Health Checks**
- `/health/live` - Service alive
- `/health/ready` - Qdrant + Embedding service ready

**FR-OP-002: Graceful Shutdown**
- Complete in-flight searches
- Maximum 30 seconds

**FR-OP-003: Hot Reload**
- Configuration changes
- Weight adjustments
- Without service restart

**FR-OP-004: Standalone Mode**
- Mock Vault responses (all categories allowed)
- Mock embedding service (random vectors)
- For development and testing

### 4.10 Quality and Diagnostics (FR-QD)

**FR-QD-001: Search Quality Metrics**
- Track Recall@K, Precision@K (when ground truth available)
- NDCG@K
- MRR
- Zero-result rate

**FR-QD-002: Result Diversity**
- Measure result diversity (avoid all-similar results)
- MMR (Maximal Marginal Relevance) option

**FR-QD-003: Query Logging**
- Log queries (anonymized) for analysis
- Identify common patterns
- Surface popular queries

---

## 5. Non-Functional Requirements

### 5.1 Performance (NFR-PE)

| ID | Item | Target |
|---|---|---|
| NFR-PE-001 | Vector search latency (p95) | < 50ms |
| NFR-PE-002 | Hybrid search latency (p95) | < 80ms |
| NFR-PE-003 | Hybrid + Rerank latency (p95) | < 150ms |
| NFR-PE-004 | Embedding generation (p95) | < 50ms |
| NFR-PE-005 | Reranking 50 docs (p95) | < 100ms |
| NFR-PE-006 | Throughput | ≥ 100 queries/s |
| NFR-PE-007 | Concurrent searches | ≥ 500 |
| NFR-PE-008 | Memory usage | ≤ 16GB (with 10M docs) |
| NFR-PE-009 | Index size on disk | ≤ 50GB (with 10M docs) |

**Note:** These targets are for SMB/research scale. Enterprise scale would require different architecture.

### 5.2 Reliability (NFR-RE)

| ID | Item | Target |
|---|---|---|
| NFR-RE-001 | Availability | 99.9% (8.7 hours/year downtime) |
| NFR-RE-002 | Error rate | < 0.5% |
| NFR-RE-003 | Data consistency | Eventually consistent (acceptable) |
| NFR-RE-004 | MTTR | < 10 minutes |
| NFR-RE-005 | Zero-result rate | < 5% |

**Note:** Lower SLA than Sentinel/Vault as Navigator is more compute-intensive.

### 5.3 Scalability (NFR-SC)

| ID | Item | Target |
|---|---|---|
| NFR-SC-001 | Documents per tenant | Up to 1M |
| NFR-SC-002 | Total documents | Up to 10M |
| NFR-SC-003 | Tenants | Up to 100 |
| NFR-SC-004 | Concurrent users | Up to 1,000 |
| NFR-SC-005 | Horizontal scaling | Yes (HPA) |

### 5.4 Security (NFR-SE)

| ID | Item | Requirement |
|---|---|---|
| NFR-SE-001 | Encryption in transit | TLS 1.3 |
| NFR-SE-002 | Encryption at rest | Filesystem level |
| NFR-SE-003 | Authentication | mTLS + JWT |
| NFR-SE-004 | Authorization | Vault integration |
| NFR-SE-005 | Cross-tenant isolation | Pre-filter enforced |
| NFR-SE-006 | Side-channel protection | Constant-time responses |
| NFR-SE-007 | Query logging | Anonymized |
| NFR-SE-008 | Index encryption | Optional (performance trade-off) |

### 5.5 Cost Efficiency (NFR-CE)

| ID | Item | Target |
|---|---|---|
| NFR-CE-001 | Infrastructure cost | < $500/month for 10M docs |
| NFR-CE-002 | GPU usage | Optional, not required |
| NFR-CE-003 | External API calls | None (fully self-hosted) |
| NFR-CE-004 | Storage efficiency | < 5KB/doc on average |

---

## 6. System Architecture

### 6.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────┐
│                Navigator Service                         │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐       │
│  │  gRPC API   │ │  REST API   │ │     CLI     │       │
│  └──────┬──────┘ └──────┬──────┘ └──────┬──────┘       │
│         └───────────────┴────────────────┘              │
│                          │                              │
│                          ▼                              │
│      ┌─────────────────────────────────┐                │
│      │      Search Orchestrator         │                │
│      └─────────────────┬───────────────┘                │
│                        │                                │
│      ┌─────────────────┼──────────────────┐             │
│      ▼                 ▼                  ▼             │
│  ┌────────┐    ┌─────────────┐    ┌──────────┐         │
│  │Embedder│    │  Searcher   │    │Reranker  │         │
│  │  (BGE) │    │  (Qdrant)   │    │ (BGE-RR) │         │
│  └────────┘    └─────────────┘    └──────────┘         │
│                                                          │
│      ┌─────────────────────────────────┐                │
│      │     Permission Filter            │                │
│      │     (Vault Integration)          │                │
│      └─────────────────────────────────┘                │
│                                                          │
│      ┌─────────────────────────────────┐                │
│      │     Result Post-processor        │                │
│      │     - Vault re-anonymization     │                │
│      │     - Snippet generation         │                │
│      └─────────────────────────────────┘                │
│                                                          │
│      ┌─────────────────────────────────┐                │
│      │     Cache Layer (Redis)          │                │
│      └─────────────────────────────────┘                │
│                                                          │
└─────────────────────────────────────────────────────────┘
         │                  │                   │
         ▼                  ▼                   ▼
   ┌──────────┐      ┌──────────┐       ┌──────────┐
   │  Qdrant  │      │Embedding │       │ Reranker │
   │ Cluster  │      │  Service │       │  Service │
   │          │      │ (BGE-M3) │       │(BGE-RR)  │
   └──────────┘      └──────────┘       └──────────┘
```

### 6.2 Component Description

| Component | Responsibility |
|---|---|
| **Search Orchestrator** | Coordinates the search pipeline |
| **Embedder** | Converts queries to vectors |
| **Searcher** | Vector + BM25 search via Qdrant |
| **Reranker** | Cross-encoder result refinement |
| **Permission Filter** | Coordinates with Vault for access control |
| **Result Post-processor** | Anonymization, snippets, metadata |
| **Cache Layer** | Query embeddings, results, permissions |

### 6.3 Data Flow

```
User Query
    ↓
[1] Receive request (user + query + filters)
    ↓
[2] Check cache (query embedding, recent results)
    ↓ (cache miss)
[3] Generate query embedding (BGE-M3)
    ↓
[4] Query Vault for user permissions
    ↓
[5] Determine searchable collections
    ↓
[6] Execute vector search (Qdrant + pre-filter)
    ↓
[7] Execute BM25 search (parallel)
    ↓
[8] Combine results (RRF)
    ↓
[9] Filter by permissions (over-fetched candidates)
    ↓
[10] Rerank top candidates (BGE-reranker)
    ↓
[11] Generate snippets
    ↓
[12] Send to Vault for re-anonymization (if needed)
    ↓
[13] Return top-K results
    ↓
[14] Cache results (async)
    ↓
[15] Log to Tracker (async)
```

### 6.4 Storage Architecture

```
Qdrant Collections (3 main collections):
├── customer_docs (DC-01)
│   ├── Dense vector field (1024d)
│   ├── Sparse vector field (BM25)
│   ├── Payload fields:
│   │   ├── tenant_id (indexed)
│   │   ├── doc_id
│   │   ├── content
│   │   ├── language (ko/en)
│   │   ├── created_at
│   │   └── custom metadata
│   └── HNSW config: m=16, ef_construct=200
│
├── manufacturing_docs (DC-02)
│   └── (same structure)
│
└── hr_docs (DC-03)
    └── (same structure)
```

---

## 7. Search Strategy Details

### 7.1 Search Pipeline

```
Default Pipeline (Hybrid + Rerank):

Stage 1: Initial Retrieval
├── Vector search (top_k * 5 = 50)
└── BM25 search (top_k * 5 = 50)
   ↓
Stage 2: Result Fusion
└── RRF combination → 50 candidates
   ↓
Stage 3: Permission Filtering
└── Filter by user's allowed categories → ~30-50 candidates
   ↓
Stage 4: Reranking
└── Cross-encoder scoring → re-ordered
   ↓
Stage 5: Final Selection
└── Top-K (default 10)
```

### 7.2 Vector Search Configuration

```yaml
vector_search:
  model: BGE-M3
  dimensions: 1024
  distance: cosine
  
  hnsw_config:
    m: 16
    ef_construct: 200
    ef_search: 128
  
  default_top_k: 50  # over-fetch for filtering
  min_score: 0.5
```

### 7.3 BM25 Configuration

```yaml
bm25_search:
  k1: 1.5
  b: 0.75
  
  analyzers:
    korean:
      type: nori
      decompound_mode: mixed
    english:
      type: standard
      lowercase: true
      stop_words: standard
    mixed:
      type: custom
      tokenizer: nori
      filters: [lowercase, asciifolding]
  
  default_top_k: 50
```

### 7.4 RRF Configuration

```yaml
result_fusion:
  method: rrf  # rrf, weighted_sum
  
  rrf:
    k: 60  # constant
  
  weighted_sum:
    vector_weight: 0.7
    bm25_weight: 0.3
```

### 7.5 Reranking Configuration

```yaml
reranking:
  enabled: true
  model: BGE-reranker-v2-m3
  
  input_top_k: 50
  output_top_k: 10
  
  skip_conditions:
    - min_candidates: 5
    - timeout_threshold_ms: 100
  
  batch_size: 16
```

---

## 8. Standalone Testing Environment

### 8.1 Independence

Navigator can operate without Vault by:
- Using mock permissions (all categories allowed)
- Returning raw (non-anonymized) results
- Skipping permission checks

### 8.2 Standalone Modes

**Mode 1: Full Server**

```bash
$ navigator-cli server

🚀 Bastion-Navigator v1.0 starting...
✅ Config loaded
✅ Embedder ready (BGE-M3, CPU mode)
✅ Reranker ready (BGE-reranker-v2-m3)
✅ Qdrant connected (3 collections, 5.2M vectors)
✅ Cache ready (Redis)
⚠️  Vault: mocked (standalone mode)
✅ REST API on :8080
✅ gRPC API on :9090
✨ Ready
```

**Mode 2: One-shot Search**

```bash
$ navigator-cli search \
    --query "warranty terms" \
    --tenant tenant-acme \
    --output-format text
```

**Mode 3: Interactive**

```bash
$ navigator-cli interactive
nav> search "warranty terms"
✅ 10 results in 87ms

nav> diagnostic
Embeddings cache hit rate: 23%
Recent latency p95: 92ms
Vector DB connection: healthy

nav> exit
```

**Mode 4: Batch Search**

```bash
$ navigator-cli search \
    --input-file queries.jsonl \
    --output-file results.jsonl \
    --parallel 5
```

### 8.3 Test Data

```
tests/
├── fixtures/
│   ├── sample_documents.jsonl       # 10,000 test documents
│   ├── korean_queries.jsonl         # Korean test queries
│   ├── english_queries.jsonl        # English test queries
│   ├── mixed_queries.jsonl          # Mixed language queries
│   └── ground_truth.jsonl           # Query → relevant docs
├── benchmarks/
│   ├── latency_tests.jsonl
│   └── quality_tests.jsonl
└── docker-compose.test.yml          # Test environment
```

### 8.4 Quality Evaluation

```bash
# Run quality benchmark
$ navigator-cli evaluate \
    --queries tests/fixtures/queries.jsonl \
    --ground-truth tests/fixtures/ground_truth.jsonl \
    --metrics recall,precision,ndcg,mrr

# Output:
Quality Report:
─────────────────────────
Recall@10:    0.85
Precision@10: 0.78
NDCG@10:      0.82
MRR:          0.71
─────────────────────────
```

---

## 9. Data Requirements

### 9.1 Configuration Schema (YAML)

```yaml
# /etc/bastion-navigator/config.yaml
version: 1.0

server:
  rest_port: 8080
  grpc_port: 9090
  workers: 4

# Qdrant configuration
vector_db:
  type: qdrant
  hosts:
    - qdrant-1:6333
    - qdrant-2:6333
  collections:
    customer_docs:
      vector_size: 1024
      distance: cosine
      hnsw:
        m: 16
        ef_construct: 200
    manufacturing_docs:
      vector_size: 1024
      distance: cosine
    hr_docs:
      vector_size: 1024
      distance: cosine

# Embedding service
embedder:
  type: bge_m3
  endpoint: http://embedder:8000
  model_path: /models/bge-m3
  batch_size: 16
  max_length: 512
  cache:
    enabled: true
    ttl: 1h
    max_size: 10000

# Reranker
reranker:
  enabled: true
  type: bge_reranker
  endpoint: http://reranker:8001
  model_path: /models/bge-reranker-v2-m3
  batch_size: 16
  max_length: 512

# Search defaults
search_defaults:
  top_k: 10
  over_fetch_multiplier: 5
  use_hybrid: true
  use_reranking: true
  vector_weight: 0.7
  bm25_weight: 0.3
  min_score: 0.5
  timeout_ms: 500

# Vault integration
vault:
  enabled: true  # false for standalone mode
  endpoint: http://vault:8080
  cache_permissions: true
  permission_ttl: 5m

# Tracker integration
tracker:
  enabled: true
  endpoint: http://tracker:8082
  async: true

# Cache
cache:
  type: redis
  url: redis://redis:6379
  query_cache_ttl: 1h
  permission_cache_ttl: 5m

# Logging
logging:
  level: info
  format: json

# Metrics
metrics:
  enabled: true
  port: 9091
```

---

## 10. Deployment and Operations

### 10.1 Deployment Environments

| Environment | Replicas | Memory | Docs |
|---|---|---|---|
| dev | 1 | 4GB | 10K |
| staging | 2 | 8GB | 1M |
| prod | 3-5 | 16GB | 10M |

### 10.2 Docker Compose (Recommended for SMB)

```yaml
version: '3.8'

services:
  navigator:
    image: bastion/navigator:1.0.0
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      - CONFIG_PATH=/etc/navigator/config.yaml
    volumes:
      - ./config:/etc/navigator
    depends_on:
      - qdrant
      - embedder
      - reranker
      - redis
    deploy:
      resources:
        limits:
          memory: 16G
          cpus: '4'
  
  qdrant:
    image: qdrant/qdrant:v1.10.0
    ports:
      - "6333:6333"
    volumes:
      - qdrant_data:/qdrant/storage
    deploy:
      resources:
        limits:
          memory: 32G
  
  embedder:
    image: bastion/embedder-bge-m3:1.0.0
    ports:
      - "8000:8000"
    volumes:
      - ./models/bge-m3:/models/bge-m3
    deploy:
      resources:
        limits:
          memory: 8G
  
  reranker:
    image: bastion/reranker-bge:1.0.0
    ports:
      - "8001:8001"
    volumes:
      - ./models/bge-reranker:/models/bge-reranker
    deploy:
      resources:
        limits:
          memory: 4G
  
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

volumes:
  qdrant_data:
```

### 10.3 Resource Requirements

**Minimum (for 1M documents):**
- 2 CPU cores
- 8GB RAM
- 20GB SSD

**Recommended (for 10M documents):**
- 4-8 CPU cores
- 32GB RAM
- 100GB NVMe SSD
- Optional: 1x GPU (T4 or better) for acceleration

### 10.4 Monitoring

**Key Metrics:**
- `navigator_search_duration_seconds` (histogram)
- `navigator_searches_total` (counter, by strategy)
- `navigator_zero_results_total` (counter)
- `navigator_cache_hit_rate` (gauge)
- `navigator_qdrant_call_duration` (histogram)

**Alert Thresholds:**
- p95 latency > 200ms: Warning
- Zero-result rate > 10%: Warning
- Qdrant unavailable: Critical
- Cache hit rate < 10%: Info

---

## 11. Appendix

### 11.1 Usage Scenarios

**Scenario 1: Marketing analyst searches customer feedback**

```bash
$ navigator-cli search \
    --query "complaints about delivery delays" \
    --tenant tenant-acme \
    --user-id analyst-1 \
    --department marketing \
    --filter category=customer_data
```

**Scenario 2: Manufacturing investigates product issue**

```bash
$ navigator-cli search \
    --query "PROD-001 defect rate increase" \
    --tenant tenant-acme \
    --user-id engineer-1 \
    --department manufacturing \
    --hybrid \
    --rerank
```

**Scenario 3: Mixed language search**

```bash
$ navigator-cli search \
    --query "warranty 보증 terms 조건" \
    --tenant tenant-acme \
    --top-k 20
```

### 11.2 Troubleshooting

| Symptom | Cause | Resolution |
|---|---|---|
| Slow search (>500ms) | Large ef_search value | Reduce to 64-128 |
| Poor quality results | Wrong embedding model | Verify BGE-M3 is loaded |
| Zero results | Aggressive filtering | Reduce min_score |
| OOM errors | Index too large | Add more RAM or shard |
| Korean tokenization issues | Wrong analyzer | Configure Nori analyzer |

### 11.3 Roadmap

- v1.1: Query expansion (synonyms, related terms)
- v1.2: Personalized ranking (user history)
- v1.3: Multi-vector retrieval (ColBERT-style)
- v2.0: GPU acceleration default
- v2.1: Distributed search (Milvus migration option)
- v2.2: Real-time index updates

### 11.4 Migration Path (Scaling Up)

If you outgrow SMB scale (>100M documents):

```
SMB Scale (Current SRS):
└── Single Qdrant + CPU-based services

Enterprise Scale:
├── Qdrant Cluster or Milvus
├── GPU-based embedding/reranking
├── Distributed cache (Redis Cluster)
├── Multi-region deployment
└── Dedicated indexer service
```

### 11.5 Change History

| Version | Date | Changes |
|---|---|---|
| 0.1 | 2026-05-15 | Initial draft |
| 0.5 | 2026-05-16 | Architecture decisions |
| 1.0 | 2026-05-17 | Initial release - SMB scale |

---

**End of Document**
