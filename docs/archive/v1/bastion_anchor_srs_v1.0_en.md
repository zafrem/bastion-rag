# Bastion-Anchor System Requirements Specification (SRS) v1.0

**Project:** Bastion-RAG - RAG Security Governance Framework  
**Module:** Module E - Anchor (Embedding Security)  
**Document Version:** 1.0  
**Date:** 2026-05-17  
**Status:** Draft  
**Scope:** PoC Implementation (Simplified)  
**Target Scale:** SMB / Research

---

## 1. Introduction

### 1.1 Purpose

This document defines the requirements for the **Bastion-Anchor** module, the embedding security layer of the RAG (Retrieval-Augmented Generation) pipeline. Anchor provides protection against embedding-based attacks and bias issues in vector representations.

This SRS is **intentionally simplified** for PoC purposes, focusing on demonstrating the core concept of embedding security rather than implementing every advanced technique.

### 1.2 Scope

**In Scope (PoC Implementation):**
- **Phase 1: Input Embedding Security**
  - Gaussian and Laplacian noise injection into embeddings
  - Configurable noise intensity
  - Basic bias measurement (WEAT)
  - Static bias checks at indexing time
  - Search quality measurement (recall preservation)
- **Phase 2: Output Response Security** ⭐
  - Response text bias detection
  - Semantic drift analysis
  - Quality verification (Grounding, Coherence, Relevance)
  - (See [Output SRS](bastion_anchor_output_srs_v1.0_en.md) for details)
- **General**
  - Standalone execution and testing
  - API-based integration (gRPC, REST)
  - CLI for manual testing and demonstration
  - Visualization of security operations for Tracker

**Out of Scope (Future Versions):**
- Differential Privacy with formal mathematical guarantees (ε-DP)
- Advanced embedding inversion defense
- Dynamic bias monitoring during retrieval
- Embedding integrity (HMAC signatures)
- Adversarial training
- Multi-modal embedding security

**Why Simplified:**
This PoC focuses on demonstrating that embedding security is **conceptually necessary** in RAG pipelines, rather than achieving production-grade defenses. The full implementation can be added incrementally based on real-world threats encountered.

### 1.3 Why Embedding Security Matters

Embeddings, while appearing as harmless numerical vectors, contain enough information to reconstruct original text with high accuracy (90%+ for short text). In RAG systems:

- Vector databases store all embeddings
- A breach exposes embeddings → potential original data exposure
- Embedding models carry biases that propagate to results
- Without protection, embeddings = unprotected data

Anchor demonstrates how to mitigate these risks at the embedding layer.

### 1.4 Definitions and Acronyms

| Term | Definition |
|---|---|
| **Anchor** | Module E - Embedding security layer |
| **Embedding** | Numerical vector representation of text/data |
| **Noise Injection** | Adding random perturbation to embeddings |
| **Bias** | Systematic skew in embedding representations |
| **Recall** | Retrieval quality metric (relevant results found) |
| **Cosine Similarity** | Distance metric for vector comparison |
| **L2 Norm** | Vector magnitude (Euclidean length) |
| **Gaussian Noise** | Random values from normal distribution |
| **Inversion Attack** | Reconstructing original text from embedding |
| **PoC** | Proof of Concept |

### 1.5 References

- Morris et al. (2023) - "Text Embeddings Reveal Almost as Much as Text"
- NIST AI Risk Management Framework
- OpenTelemetry Specification (for tracing)
- BGE-M3 Model Documentation

---

## 2. Overall Description

### 2.1 Product Perspective

Anchor sits between Navigator (search) and the LLM in the Bastion-RAG pipeline.

```
┌──────────────────────────────────────────────────────┐
│              User Query                               │
└────────────────────────┬─────────────────────────────┘
                         ▼
        ┌────────────────────────────────────┐
        │   Module A: Sentinel                │
        └────────────────┬───────────────────┘
                         ▼
        ┌────────────────────────────────────┐
        │   Module B: Vault                   │
        └────────────────┬───────────────────┘
                         ▼
        ┌────────────────────────────────────┐
        │   Module C: Navigator               │
        └────────────────┬───────────────────┘
                         │ (embeddings)
                         ▼
        ┌────────────────────────────────────┐
        │   Module E: ANCHOR  ◄── (This doc)  │
        │   - Noise Injection                 │
        │   - Bias Measurement                │
        │   - Quality Preservation            │
        └────────────────┬───────────────────┘
                         ▼
        ┌────────────────────────────────────┐
        │              LLM                    │
        └────────────────────────────────────┘
                         │
                         ▼ (async events)
        ┌────────────────────────────────────┐
        │   Module D: Tracker (Audit)        │
        └────────────────────────────────────┘
```

### 2.2 Operation Modes

Anchor operates in two modes:

**Mode 1: Indexing Time (Write Path)**
```
Document → Embedding Model → Embedding
                                ↓
                            Anchor (apply noise)
                                ↓
                            Vector DB Storage
```

**Mode 2: Search Time (Read Path)**
```
Query → Embedding Model → Query Embedding
                            ↓
                        Anchor (apply noise)
                            ↓
                        Vector DB Search
                            ↓
                        Results → LLM
```

### 2.3 Product Functions

1. **F1: Noise Injection** (Primary Feature)
   - Add Gaussian noise to embeddings
   - Configurable noise intensity
   - Maintain vector normalization

2. **F2: Bias Measurement**
   - Measure embedding bias for sensitive attributes
   - Static analysis at configuration time
   - Generate bias reports

3. **F3: Quality Preservation**
   - Measure recall impact of noise injection
   - Optimize noise level for security/quality balance
   - Alert if quality degrades

4. **F4: Configurable Strategies**
   - Per-tenant noise settings
   - Per-category noise settings
   - Strategy hot-reload

5. **F5: Standalone Operation**
   - Work without other Bastion-RAG modules
   - Mock mode for testing
   - Local development support

6. **F6: Multiple Interfaces**
   - gRPC for internal communication
   - REST API for external access
   - CLI for manual testing

7. **F7: Event Reporting**
   - Send events to Tracker (Module D)
   - Bias detection alerts
   - Quality degradation alerts

### 2.4 User Characteristics

| User Type | Purpose | Interface |
|---|---|---|
| **Navigator (Module C)** | Embedding security | gRPC |
| **Developer** | Testing, tuning | CLI |
| **Data Scientist** | Bias analysis | REST/CLI |
| **PoC Demo Audience** | Understanding | Visualization via Tracker |

### 2.5 Constraints

- **Language:** Go (consistent with other modules)
- **Embedding Compatibility:** BGE-M3 (1024 dimensions)
- **Memory Usage:** ≤ 1GB per pod
- **Processing Time:** ≤ 5ms per embedding
- **CPU-only:** No GPU required for noise injection
- **Library:** Standard math libraries (no external dependencies for noise)

### 2.6 Assumptions and Dependencies

**Assumptions:**
- Navigator provides clean embeddings
- BGE-M3 is the embedding model
- 1024-dimensional vectors
- Vault has already handled PII concerns

**External Dependencies:**
- Navigator (Module C) - integration partner
- Tracker (Module D) - event reporting
- NATS - event bus

---

## 3. External Interface Requirements

### 3.1 Interface Overview

Anchor follows the same interface pattern as other Bastion-RAG modules.

| Category | Interface | Purpose |
|---|---|---|
| **Input** | gRPC | From Navigator |
| **Input** | REST API | External testing |
| **Input** | CLI | Manual operations |
| **Output** | Processed embeddings | To Navigator |
| **Output** | Events | To Tracker via NATS |
| **Output** | Text reports | For demos |

### 3.2 Input Interface 1: gRPC

```protobuf
syntax = "proto3";
package bastion-rag.anchor.v1;

service AnchorService {
  // Phase 1: Embedding Security (Anchor-IN)
  rpc SecureEmbedding(SecureRequest)          returns (SecureResponse);
  rpc SecureBatch(BatchSecureRequest)         returns (BatchSecureResponse);
  rpc MeasureBias(BiasRequest)                returns (BiasResponse);
  rpc MeasureQuality(QualityRequest)          returns (QualityResponse);

  // Phase 2: Output Verification (Anchor-OUT)
  rpc VerifyResponse(VerifyRequest)           returns (VerifyResponse);
  rpc CheckDrift(DriftRequest)                returns (DriftResponse);

  // Configuration & Health
  rpc GetConfig(ConfigRequest)                returns (AnchorConfig);
  rpc UpdateConfig(AnchorConfig)              returns (AnchorConfig);
  rpc Health(HealthRequest)                   returns (HealthResponse);
}

// ... message definitions follow (see proto/anchor/v1/anchor.proto)
```


### 3.3 Input Interface 2: REST API

**Endpoints:**

```
# Embedding security
POST /v1/anchor/secure                  # Single embedding
POST /v1/anchor/secure/batch            # Batch embeddings

# Analysis
POST /v1/anchor/measure-bias            # Bias measurement
POST /v1/anchor/measure-quality         # Quality impact

# Configuration
GET  /v1/anchor/config                  # Current config
POST /v1/anchor/config                  # Update config

# Operations
GET  /v1/health                         # Health check
GET  /v1/metrics                        # Prometheus metrics
```

**Request Example:**

```http
POST /v1/anchor/secure HTTP/1.1
Host: anchor.bastion-rag.local
Content-Type: application/json

{
  "request_id": "req-anchor-001",
  "tenant_id": "tenant-acme",
  "embedding": [0.42, -0.31, 0.78, ..., 0.56],
  "operation": "index",
  "category": "customer_data",
  "options": {
    "noise_sigma": 0.01,
    "measure_bias": false,
    "preserve_norm": true
  }
}
```

**Response Example:**

```json
{
  "request_id": "req-anchor-001",
  "secured_embedding": [0.45, -0.28, 0.80, ..., 0.54],
  "metrics": {
    "noise_added": 0.0285,
    "similarity_to_original": 0.985,
    "bias_score": null,
    "strategy_used": "gaussian_noise"
  },
  "processing_time_ms": 2.3
}
```

### 3.4 Input Interface 3: CLI

```bash
# Apply noise to embedding
$ anchor-cli secure \
    --embedding-file query.json \
    --noise-sigma 0.01 \
    --output secured.json

# Measure bias
$ anchor-cli measure-bias \
    --test-words doctor,nurse,engineer \
    --pairs gender:male,female \
    --output bias-report.json

# Output:
Bias Analysis Report:
─────────────────────────────────
Word         Category   Bias    Severity
doctor       gender     0.15    🟡 medium
nurse        gender     0.18    🟡 medium  
engineer     gender     0.12    🟢 low
─────────────────────────────────
Overall: 0.15 (medium)
Recommendation: Review embedding model

# Test quality impact
$ anchor-cli test-quality \
    --queries test-queries.jsonl \
    --noise-sigma 0.01

# Output:
Quality Impact Test:
─────────────────────────────────
Recall@10 without noise: 0.85
Recall@10 with noise:    0.83 (-0.02)
Noise impact: 2.4% degradation
Status: ✅ Acceptable (< 5%)
─────────────────────────────────

# Compare different noise levels
$ anchor-cli compare-noise \
    --queries test.jsonl \
    --sigmas 0.005,0.01,0.02,0.05

# Output:
Noise Level Comparison:
─────────────────────────────────
Sigma   Recall@10   Security   Verdict
0.005   0.847       Low        ❌ Too weak
0.010   0.831       Medium     ✅ Recommended
0.020   0.798       High       ⚠️ Quality impact
0.050   0.654       Very High  ❌ Too aggressive
─────────────────────────────────

# Interactive mode
$ anchor-cli interactive
anchor> config
{
  "default_noise_sigma": 0.01,
  "preserve_norm": true,
  "measure_bias_enabled": false
}

anchor> secure-test "What is the price?"
Original: [0.42, -0.31, ...]
Secured:  [0.45, -0.28, ...]
Similarity: 0.985

anchor> exit

# Server mode
$ anchor-cli server --port 8080
```

### 3.5 Output: Text Report (for Demos)

```
══════════════════════════════════════════════════════
  Anchor Security Report
══════════════════════════════════════════════════════
Request:        req-anchor-001
Operation:      Index
Category:       customer_data
Processing:     2.3ms

─── Security Applied ────────────────────────────────
Strategy:        Gaussian Noise Injection
Noise Sigma:     0.01
Noise Magnitude: 0.0285 (avg)
L2 Norm:         Preserved

─── Quality Metrics ─────────────────────────────────
Original Norm:   1.000
Secured Norm:    1.000
Cosine Sim:      0.985 (✅ high preservation)
Expected Recall: ~98% of unsecured

─── Visualization ───────────────────────────────────
Original: [0.42, -0.31, 0.78, ..., 0.56]
           │     │     │           │
           +0.03 +0.03 +0.02      -0.02
           ▼     ▼     ▼           ▼
Secured:  [0.45, -0.28, 0.80, ..., 0.54]

─── Interpretation ──────────────────────────────────
✅ Embedding is now resistant to inversion attacks
✅ Search quality minimally affected
✅ Suitable for storage in Vector DB

═════════════════════════════════════════════════════
```

### 3.6 Output Events (to Tracker)

```yaml
# Event: embedding_secured
{
  "event_type": "anchor.embedding_secured",
  "module": "anchor",
  "severity": "info",
  "data": {
    "tenant_id": "tenant-acme",
    "operation": "index",
    "noise_added": 0.0285,
    "similarity_preserved": 0.985
  }
}

# Event: bias_detected
{
  "event_type": "anchor.bias_detected",
  "module": "anchor",
  "severity": "warning",
  "data": {
    "category": "gender",
    "bias_score": 0.18,
    "affected_terms": ["nurse", "secretary"]
  }
}

# Event: quality_degraded
{
  "event_type": "anchor.quality_degraded",
  "module": "anchor",
  "severity": "warning",
  "data": {
    "expected_recall": 0.85,
    "current_recall": 0.78,
    "noise_sigma": 0.02
  }
}
```

---

## 4. Functional Requirements

### 4.1 Noise Injection (FR-NI)

**FR-NI-001: Gaussian Noise**
- Generate random values from N(0, σ²)
- Default σ = 0.01
- Configurable per request

**FR-NI-002: Norm Preservation**
- L2-normalize embedding after noise injection
- Maintain unit vector (norm = 1.0)
- Critical for cosine similarity

**FR-NI-003: Deterministic Mode (Optional)**
- Use seeded random for testing
- Reproducible noise patterns
- Helps in debugging

**FR-NI-004: Per-tenant Configuration**
- Different noise levels per tenant
- Stricter for sensitive data
- Relaxed for public data

**FR-NI-005: Noise Strategy Selection**
- **Gaussian** (default): Normal distribution noise.
- **Laplacian**: Laplacian distribution noise (implemented for differential privacy readiness).
- Configurable via API or configuration file.

### 4.6 Phase 2 Requirements (FR-P2)

See [Output SRS (bastion_anchor_output_srs_v1.0_en.md)](bastion_anchor_output_srs_v1.0_en.md) for detailed functional requirements regarding response bias detection, drift analysis, and quality verification.

### 4.2 Bias Measurement (FR-BM)

**FR-BM-001: Word Embedding Association Test (WEAT)**
- Implement WEAT methodology
- Test predefined attribute pairs
- Compute bias scores

**FR-BM-002: Configurable Test Sets**
- Built-in: gender, ethnicity, age
- Custom: user-defined word lists
- Multilingual support (Korean + English)

**FR-BM-003: Bias Reporting**
- Generate human-readable reports
- Categorize severity (low/medium/high)
- Suggest mitigations

**FR-BM-004: Static Analysis Mode**
- Run bias tests at configuration time
- Not on every request (performance)
- Cache results

### 4.3 Quality Measurement (FR-QM)

**FR-QM-001: Recall@K Calculation**
- Measure retrieval quality with ground truth
- Compare with/without noise
- Track over time

**FR-QM-002: Cosine Similarity Tracking**
- Measure original vs secured embedding similarity
- Per-request metric
- Aggregate statistics

**FR-QM-003: Quality Threshold Alerts**
- Alert if recall drops > 5%
- Alert if similarity < 0.95
- Notify operations team

### 4.4 Configuration Management (FR-CM)

**FR-CM-001: Configuration File**
- YAML format
- Hot-reload capability
- Validation before applying

**FR-CM-002: Runtime Configuration Override**
- Per-request options
- Override defaults
- For testing/demo

**FR-CM-003: Tenant Policy Storage**
- PostgreSQL for tenant configs
- Cached in memory
- Refresh on change

### 4.5 Operational Features (FR-OP)

**FR-OP-001: Health Checks**
- /health/live
- /health/ready (verifies dependencies)

**FR-OP-002: Metrics Export**
- Prometheus format
- Latency, throughput, quality
- Bias scores

**FR-OP-003: Event Publishing**
- NATS integration
- Async to Tracker
- Non-blocking

**FR-OP-004: Standalone Mode**
- Run without Tracker (events to stdout)
- Run without Navigator (test mode)

---

## 5. Non-Functional Requirements

### 5.1 Performance (NFR-PE)

| ID | Item | Target |
|---|---|---|
| NFR-PE-001 | Single embedding processing (p95) | < 5ms |
| NFR-PE-002 | Batch processing (100 embeddings) | < 50ms |
| NFR-PE-003 | Bias measurement (full test set) | < 500ms |
| NFR-PE-004 | Throughput | ≥ 5,000 embeddings/s |
| NFR-PE-005 | Memory usage | ≤ 1GB |
| NFR-PE-006 | CPU usage | ≤ 1 vCPU (normal) |

### 5.2 Reliability (NFR-RE)

| ID | Item | Target |
|---|---|---|
| NFR-RE-001 | Availability | 99.9% |
| NFR-RE-002 | Error rate | < 0.1% |
| NFR-RE-003 | Quality preservation | Recall@K within 5% |

### 5.3 Security (NFR-SE)

| ID | Item | Requirement |
|---|---|---|
| NFR-SE-001 | Encryption in transit | TLS 1.3 |
| NFR-SE-002 | Authentication | mTLS |
| NFR-SE-003 | Configuration access | RBAC |
| NFR-SE-004 | Noise randomness | Cryptographically secure |

### 5.4 Maintainability (NFR-MA)

| ID | Item | Target |
|---|---|---|
| NFR-MA-001 | Code coverage | ≥ 80% |
| NFR-MA-002 | Documentation | godoc + examples |
| NFR-MA-003 | API versioning | URL path |

---

## 6. System Architecture

### 6.1 High-Level Architecture

```
┌──────────────────────────────────────────────────────┐
│                Anchor Service                         │
├──────────────────────────────────────────────────────┤
│                                                       │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐   │
│  │  gRPC API   │ │  REST API   │ │     CLI     │   │
│  └──────┬──────┘ └──────┬──────┘ └──────┬──────┘   │
│         └────────────────┴────────────────┘         │
│                          │                           │
│                          ▼                           │
│      ┌──────────────────────────────────┐           │
│      │       Request Handler            │           │
│      └──────────────────┬───────────────┘           │
│                         │                           │
│      ┌──────────────────┼──────────────────┐        │
│      ▼                  ▼                  ▼        │
│  ┌─────────┐    ┌──────────────┐  ┌──────────────┐ │
│  │  Noise  │    │     Bias     │  │   Quality    │ │
│  │Injector │    │   Analyzer   │  │   Monitor    │ │
│  └─────────┘    └──────────────┘  └──────────────┘ │
│                         │                           │
│                         ▼                           │
│      ┌──────────────────────────────────┐           │
│      │     Configuration Manager        │           │
│      └──────────────────────────────────┘           │
│                                                      │
│      ┌──────────────────────────────────┐           │
│      │     Event Publisher (NATS)       │           │
│      └──────────────────────────────────┘           │
│                                                      │
└──────────────────────────────────────────────────────┘
                          │
                          ▼
                    ┌──────────┐
                    │  NATS    │ → To Tracker
                    └──────────┘
```

### 6.2 Components

| Component | Responsibility |
|---|---|
| **Request Handler** | Route requests, validate input |
| **Noise Injector** | Apply Gaussian noise to embeddings |
| **Bias Analyzer** | Measure bias in embedding model |
| **Quality Monitor** | Track recall impact |
| **Config Manager** | Hot-reload configurations |
| **Event Publisher** | Send events to Tracker |

### 6.3 Data Flow

```
Indexing Time:
[Document Embedding] → Request Handler
                          ↓
                      Get Tenant Config
                          ↓
                      Noise Injector
                          ↓
                      Quality Monitor (sample)
                          ↓
                      Event Publisher → Tracker
                          ↓
                      [Return Secured Embedding]

Search Time:
[Query Embedding] → Request Handler
                          ↓
                      Get Tenant Config
                          ↓
                      Noise Injector
                          ↓
                      [Return Secured Query Embedding]
```

---

## 7. Noise Injection Algorithm

### 7.1 Core Algorithm

```go
package anchor

import (
    "math/rand"
    "math"
)

type NoiseInjector struct {
    sigma           float64
    preserveNorm    bool
    randomSource    *rand.Rand
}

// Apply Gaussian noise to embedding
func (n *NoiseInjector) Inject(embedding []float64) []float64 {
    // 1. Generate noise
    noise := make([]float64, len(embedding))
    for i := range noise {
        noise[i] = n.gaussianRandom() * n.sigma
    }
    
    // 2. Add noise to embedding
    secured := make([]float64, len(embedding))
    for i := range embedding {
        secured[i] = embedding[i] + noise[i]
    }
    
    // 3. Normalize if required
    if n.preserveNorm {
        secured = normalize(secured)
    }
    
    return secured
}

// Generate Gaussian random number (Box-Muller transform)
func (n *NoiseInjector) gaussianRandom() float64 {
    u1 := n.randomSource.Float64()
    u2 := n.randomSource.Float64()
    
    z := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
    return z
}

// L2 normalize vector
func normalize(vec []float64) []float64 {
    var sumSquares float64
    for _, v := range vec {
        sumSquares += v * v
    }
    norm := math.Sqrt(sumSquares)
    
    if norm == 0 {
        return vec
    }
    
    normalized := make([]float64, len(vec))
    for i, v := range vec {
        normalized[i] = v / norm
    }
    return normalized
}
```

### 7.2 Parameter Selection Guide

| Sigma | Use Case | Quality Impact | Security Level |
|---|---|---|---|
| 0.001 | Minimal | Negligible | Very Low |
| 0.005 | Light | ~1% recall loss | Low |
| **0.01** | **Recommended** | **~2-3% recall loss** | **Medium** |
| 0.02 | Stronger | ~5-7% recall loss | High |
| 0.05 | Aggressive | ~15-20% recall loss | Very High |
| 0.1 | Extreme | ~30%+ recall loss | Extreme (unusable) |

---

## 8. Bias Measurement (Simplified WEAT)

### 8.1 WEAT-inspired Algorithm

```go
type BiasAnalyzer struct {
    embedModel EmbeddingModel
}

type BiasResult struct {
    Word     string
    Category string
    Score    float64
    Severity string
}

func (b *BiasAnalyzer) Analyze(
    testWords []string,
    pairs []AttributePair,
) []BiasResult {
    var results []BiasResult
    
    for _, word := range testWords {
        for _, pair := range pairs {
            wordEmb, _ := b.embedModel.Encode(word)
            attrAEmb, _ := b.embedModel.Encode(pair.AttributeA)
            attrBEmb, _ := b.embedModel.Encode(pair.AttributeB)
            
            // Calculate cosine similarities
            simA := cosineSimilarity(wordEmb, attrAEmb)
            simB := cosineSimilarity(wordEmb, attrBEmb)
            
            // Bias = absolute difference
            bias := math.Abs(simA - simB)
            
            severity := classifyBias(bias)
            
            results = append(results, BiasResult{
                Word:     word,
                Category: pair.Category,
                Score:    bias,
                Severity: severity,
            })
        }
    }
    
    return results
}

func classifyBias(score float64) string {
    switch {
    case score < 0.1:
        return "low"
    case score < 0.2:
        return "medium"
    default:
        return "high"
    }
}
```

### 8.2 Built-in Test Sets

```yaml
# Default bias test configurations
bias_tests:
  gender:
    test_words:
      - doctor / 의사
      - nurse / 간호사
      - engineer / 엔지니어
      - teacher / 교사
      - manager / 매니저
    attribute_pairs:
      - { a: "male", b: "female" }
      - { a: "남성", b: "여성" }
      - { a: "he", b: "she" }
  
  ethnicity:
    test_words:
      - intelligent / 똑똑한
      - hardworking / 부지런한
      - athletic / 운동을 잘하는
    attribute_pairs:
      - { a: "Korean", b: "Foreign" }
      - { a: "한국인", b: "외국인" }
  
  age:
    test_words:
      - capable / 능력있는
      - modern / 현대적인
      - traditional / 전통적인
    attribute_pairs:
      - { a: "young", b: "old" }
      - { a: "젊은", b: "나이든" }
```

---

## 9. Standalone Testing

### 9.1 Operation Modes

**Mode 1: Server Mode**
```bash
$ anchor-cli server

🚀 Bastion-Anchor v1.0 starting...
✅ Config loaded
✅ Noise injector ready (sigma=0.01)
✅ Bias analyzer ready
✅ Quality monitor ready
✅ NATS connected (events to Tracker)
✅ REST API on :8080
✅ gRPC API on :9090
✨ Ready
```

**Mode 2: Demo Mode**
```bash
$ anchor-cli demo

🎬 Anchor Demo Mode

Demo 1: Embedding Noise Injection
─────────────────────────────────
Original embedding: [0.42, -0.31, 0.78, ..., 0.56]
Applying noise (sigma=0.01)...
Secured embedding:  [0.45, -0.28, 0.80, ..., 0.54]
Similarity:         0.985 ✅
Status:             Embedding secured!

Demo 2: Bias Measurement
─────────────────────────────────
Testing word: "engineer"
vs. "male" / "female"
Similarity to "male":   0.823
Similarity to "female": 0.687
Bias score:             0.136 🟡 medium
Recommendation:         Consider model review

Demo 3: Quality Impact
─────────────────────────────────
Running 100 test queries...
Recall@10 (no noise):   0.85
Recall@10 (with noise): 0.83
Quality loss:           2.4% ✅ acceptable
```

**Mode 3: Test Mode**
```bash
$ anchor-cli test \
    --queries test-queries.jsonl \
    --noise-sigmas 0.005,0.01,0.02,0.05 \
    --output results.json
```

### 9.2 Test Data

```
tests/
├── fixtures/
│   ├── sample_embeddings.json     # 1000 test embeddings
│   ├── bias_test_words_ko.json    # Korean bias tests
│   ├── bias_test_words_en.json    # English bias tests
│   └── ground_truth_queries.json  # For recall measurement
├── benchmarks/
│   └── performance_tests.json
└── docker-compose.test.yml
```

---

## 10. Data Requirements

### 10.1 Configuration Schema

```yaml
# /etc/bastion-anchor/config.yaml
version: 1.0

server:
  rest_port: 8080
  grpc_port: 9090

# Noise injection
noise:
  default_sigma: 0.01
  preserve_norm: true
  strategy: gaussian   # gaussian, laplacian
  
  # Per-tenant override
  tenant_overrides:
    tenant-acme:
      sigma: 0.015      # Stricter for sensitive
    tenant-public:
      sigma: 0.005      # Lighter for public

# Bias measurement
bias:
  enabled: true
  cache_ttl: 1h
  
  alert_thresholds:
    medium: 0.15
    high: 0.25
  
  test_sets_path: /etc/anchor/bias-tests/

# Quality monitoring
quality:
  enabled: true
  sample_rate: 0.1     # 10% of requests
  
  alert_thresholds:
    recall_drop_percent: 5.0
    similarity_minimum: 0.95

# Embedding compatibility
embedding:
  expected_dimensions: 1024
  expected_model: BGE-M3

# Tracker integration
tracker:
  enabled: true
  nats_url: nats://nats:4222
  event_subject: bastion-rag.events.anchor

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

## 11. Deployment

### 11.1 Docker Compose

```yaml
services:
  anchor:
    image: bastion-rag/anchor:1.0.0
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      - CONFIG_PATH=/etc/anchor/config.yaml
    volumes:
      - ./config:/etc/anchor
    deploy:
      resources:
        limits:
          memory: 1G
          cpus: '1'
```

### 11.2 Resources

**Minimum:**
- 1 CPU core
- 512MB RAM
- No GPU

**Recommended (PoC):**
- 2 CPU cores
- 1GB RAM

---

## 12. Tracker Integration (Visualization)

### 12.1 Events for Visualization

When Tracker displays Anchor in the live flow:

```
Live Flow Visualization:

[Navigator] → [Anchor] → [LLM]
                ↓
              [Detail Panel]
              
              Anchor Stats:
              - Processing: 2.3ms
              - Noise added: 0.029
              - Quality: 98.5% similarity
              - Bias check: ✅ passed
```

### 12.2 Demo Scenarios for Tracker

```yaml
demo_scenarios:
  - name: "Anchor Protection Demo"
    description: "Show embedding being secured"
    events:
      - anchor.embedding_received
      - anchor.noise_injected
      - anchor.quality_measured
      - anchor.bias_checked
      - anchor.embedding_returned
    speed: 0.5x
    annotations:
      - "Original embedding from Navigator"
      - "Adding Gaussian noise..."
      - "Quality preserved: 98%"
      - "No bias detected"
      - "Secured embedding sent to LLM"
```

---

## 13. Future Enhancements (Out of PoC Scope)

### v1.1 Enhancements
- Laplacian noise for stronger DP
- Per-category noise tuning
- Bias remediation suggestions

### v2.0 Enhancements
- Formal differential privacy (ε-DP)
- Embedding integrity verification (HMAC)
- Adversarial robustness
- Multi-modal embedding security

### Why Not Now:
These advanced features require:
- Significant complexity
- More compute resources
- Specialized knowledge
- May not fit PoC timeline

The current SRS provides **conceptual proof** of embedding security, which is the primary goal of PoC.

---

## 14. Appendix

### 14.1 Demo Walkthrough (2 minutes)

```
0:00-0:30  Show embedding visualization
           - Original embedding vector
           - Explain what numbers represent

0:30-1:00  Apply Anchor security
           - Live noise injection
           - Show before/after vectors
           - Explain similarity preservation

1:00-1:30  Bias analysis
           - Run bias test
           - Show results visualization
           - Discuss implications

1:30-2:00  Quality verification
           - Show recall@10 with/without noise
           - Confirm acceptable quality
           - Q&A
```

### 14.2 Common Questions

**Q: Why not encrypt the embeddings instead?**
A: Encryption prevents similarity search. Noise injection preserves search capability while adding protection.

**Q: Is noise injection a perfect defense?**
A: No, it's a layer of defense. Combined with Vault's data protection, it significantly raises the bar.

**Q: How is this different from differential privacy?**
A: Gaussian noise is the basis of DP, but formal DP requires careful budget management. Our PoC uses noise injection without formal DP guarantees, which is a simpler starting point.

**Q: Can we increase noise for more security?**
A: Yes, but at the cost of search quality. The trade-off must be tuned per use case.

### 14.3 Roadmap

- v1.0: PoC (current)
- v1.1: Improved bias detection
- v1.2: Per-tenant fine-tuning
- v2.0: Formal differential privacy
- v2.1: Embedding integrity (HMAC)
- v2.2: Multi-modal support

### 14.4 Change History

| Version | Date | Changes |
|---|---|---|
| 0.1 | 2026-05-16 | Initial draft |
| 0.5 | 2026-05-17 | Simplified for PoC |
| 1.0 | 2026-05-17 | PoC-ready release |

---

**End of Document**

---

## Note on Simplification

This SRS intentionally simplifies advanced topics like:
- Differential Privacy (formal ε-DP)
- Embedding inversion defense techniques
- Adversarial robustness

The goal of this module in the Bastion-RAG PoC is to **demonstrate that embedding security matters** and provide a working implementation that can be expanded based on production needs. The full complexity of embedding security is a research-level topic that warrants dedicated study and is beyond the scope of an initial PoC.
