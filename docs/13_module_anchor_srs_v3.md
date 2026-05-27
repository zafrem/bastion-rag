# Bastion-Anchor Module SRS

**Project:** Bastion - RAG Security Governance Framework
**Document Type:** Module SRS (Tier 2)
**Document ID:** 13-anchor-srs
**Module:** E - Anchor (Embedding Security)
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

This document specifies the **Anchor** module, the embedding security layer of Bastion. v3 reflects the migration from Go to Python, enabling numerical computing algorithms via `numpy`/`scipy` with no manual implementation of mathematical primitives.

Anchor operates bidirectionally:
- **Phase 1 (Input):** Inject noise into embeddings before they reach the LLM
- **Phase 2 (Output):** Analyze LLM response for bias and anomaly

### 1.2 Module Identity

```
Module: E - Anchor
Language: Python 3.11+
Role: Embedding Security (bidirectional)
Position: Before LLM (input) + after LLM (output)

Standalone value:
"Attach Anchor → embeddings protected from inversion attacks"
```

### 1.3 Why Python (v3 Change)

```
v2 (Go): All numerical algorithms implemented manually.
  Box-Muller transform: ~30 lines of manual Go code
  WEAT bias:           ~100 lines (cosine similarity, set differences)
  L2 norm:             ~10 lines (sqrt of dot product)
  Laplacian noise:     ~20 lines (manual inverse-CDF)

v3 (Python): Single library calls on BLAS-optimized backends.
  import numpy as np
  noise = np.random.normal(0.0, sigma, size=len(embedding))
  norm  = np.linalg.norm(embedding)
  bias  = np.dot(a, b) / (norm_a * norm_b)   # cosine similarity

Benefits:
- Shorter, more readable code
- BLAS/LAPACK-optimized numerical operations
- scipy provides statistical primitives (differential privacy, distributions)
- Easier to extend with new AI security techniques
```

### 1.4 The Standalone Test (Foundation Litmus)

```
Question: "If only Anchor is attached,
          does it provide meaningful security?"

Answer: YES
- Phase 1: noise injection (inversion defense)
- Phase 2: bias/anomaly detection

→ Anchor passes the standalone test ✅
```

### 1.5 Why Embedding Security Matters

```
Embeddings can be inverted to ~90% original text
(Morris 2023). In RAG systems:
- Vector DB stores embeddings for all documents
- A breach exposes not just metadata but content
- Anchor adds a noise layer that degrades inversion
  while preserving search quality
```

### 1.6 Scope

**In Scope:**
- 🟢 Core: Gaussian/Laplacian noise injection (Phase 1)
- 🟢 Core: Norm preservation after noise
- 🟢 Core: Per-tenant noise configuration
- 🟢 Core: WEAT bias analysis (Phase 1 + 2)
- 🟢 Core: Response verification: bias, quality, anomaly, drift (Phase 2)
- 🟡 Enhanced: Search quality optimization (with Navigator)
- 🔴 Hooks: Pipeline-wide bias monitoring
- 🔴 Hooks: Lineage emission
- Bidirectional (Phase 1 + Phase 2)

**Out of Scope:**
- Formal differential privacy guarantees (PoC scope)
- Embedding integrity HMAC
- Adversarial robustness
- Detailed cross-cutting (respective SRS)

### 1.7 Definitions

| Term | Definition |
|---|---|
| **Phase 1** | Input path: noise injection into embeddings |
| **Phase 2** | Output path: LLM response analysis |
| **Noise injection** | Adding random perturbation to an embedding |
| **Norm preservation** | Scaling noised vector to original L2 norm |
| **WEAT** | Word Embedding Association Test — bias measurement |
| **Inversion** | Reconstructing text from embedding (attack) |
| **Sigma (σ)** | Gaussian noise standard deviation |
| **Strategy** | gaussian or laplacian noise distribution |

---

## 2. Overall Description

### 2.1 Bidirectional Architecture

```
┌─────────────────────────────────────────────┐
│             Anchor Service (Python)          │
├─────────────────────────────────────────────┤
│  ┌────────────┐         ┌────────────┐      │
│  │ Phase 1 API│         │ Phase 2 API│      │
│  │  (/secure) │         │  (/verify) │      │
│  └─────┬──────┘         └─────┬──────┘      │
│        └───────────┬──────────┘             │
│                    ▼                        │
│      ┌─────────────────────────────┐        │
│      │  Core Components            │        │
│      │  NoiseInjector  (numpy)     │        │
│      │  BiasAnalyzer   (numpy)     │        │
│      │  QualityMonitor (numpy)     │        │
│      │  Verifier       (numpy)     │        │
│      └─────────────────────────────┘        │
│                    ↓                        │
│      ┌─────────────────────────────┐        │
│      │  Optional                   │        │
│      │  HookManager                │        │
│      │  Publisher (NATS)           │        │
│      └─────────────────────────────┘        │
└─────────────────────────────────────────────┘
```

### 2.2 Position in Pipeline

```
Input Pipeline (Phase 1):
Navigator → [Anchor: noise injection] → LLM

Output Pipeline (Phase 2):
LLM → [Anchor: response verify] → Vault-OUT
```

### 2.3 Layer Classification

```
🟢 CORE (Standalone):
   Phase 1: Noise injection, WEAT bias measurement
   Phase 2: Response verify (bias, quality, anomaly, drift)

🟡 ENHANCED (Composition):
   - Search quality tuning (+ Navigator)

🔴 HOOKS (Cross-cutting):
   - Pipeline bias monitoring
   - Lineage emission
```

### 2.4 Constraints

```
Language: Python 3.11+
RAM: ≤ 1GB (no ML models loaded; numpy only)
Latency: <5ms (noise injection), <50ms (response analysis)
CPU-first (no GPU required; BLAS auto-used for matrix ops)
```

### 2.5 Dependencies

```
Core dependencies (Python packages):
- numpy >= 1.26          (noise, cosine similarity, WEAT, norms)
- scipy >= 1.13          (statistical distributions, future DP)
- fastapi >= 0.111       (REST)
- uvicorn >= 0.30        (ASGI server)
- grpcio >= 1.64         (gRPC server)
- pydantic >= 2.7        (models + config)
- pyyaml >= 6.0          (config loading)

Optional (Hooks):
- nats-py >= 2.7         (NATS events)
- prometheus-client >= 0.20 (metrics)
```

---

## 3. Core Functions (🟢 Standalone)

### 3.1 Phase 1: Noise Injection (FR-CORE-NI)

**FR-CORE-NI-001: Gaussian Noise**
```
Add isotropic Gaussian noise to an embedding vector.

Implementation (numpy):
  noise = rng.normal(0.0, sigma, size=len(embedding))
  noised = embedding + noise

Default sigma: 0.01
Inversion defense: higher sigma = harder to invert

No external dependencies; pure numpy.
```

**FR-CORE-NI-002: Laplacian Noise**
```
Alternative strategy for stronger privacy guarantees.

Implementation (numpy):
  b = sigma / sqrt(2)          # scale param for same variance
  noise = rng.laplace(0.0, b, size=len(embedding))
  noised = embedding + noise

Select via config: noise.strategy: laplacian
```

**FR-CORE-NI-003: Norm Preservation**
```
Preserve original L2 norm to maintain cosine similarity.

Implementation (numpy):
  original_norm = np.linalg.norm(embedding)
  if preserve_norm and original_norm > 0:
      noised_norm = np.linalg.norm(noised)
      if noised_norm > 0:
          noised = noised * (original_norm / noised_norm)

Prevents noise from corrupting search quality.
```

**FR-CORE-NI-004: Per-Tenant Sigma**
```
noise.tenant_overrides:
  tenant-acme: 0.02
  tenant-beta: 0.005

Higher sigma = more protection, lower recall.
sigma_for_tenant() resolves override or falls back to default.
```

**FR-CORE-NI-005: Deterministic Seed**
```
Optional: inject with seed for reproducibility (testing/audit).
SecureOptions.seed → NoiseInjector(sigma, preserve, seed=seed)
```

### 3.2 Phase 1: Bias Measurement (FR-CORE-BM)

**FR-CORE-BM-001: WEAT Bias Analysis**
```
Word Embedding Association Test (WEAT):
- Compare associations of target word sets
  with attribute word pairs (e.g. male/female)
- Compute effect size d:
    d = mean_cosine(target_A, attr) - mean_cosine(target_B, attr)
        / pooled_std

Implementation (numpy):
  def cosine_similarity(a, b):
      return np.dot(a, b) / (np.linalg.norm(a) * np.linalg.norm(b))
```

**FR-CORE-BM-002: Built-in Test Sets**
```
Default bias categories:
  gender:    [man, male, he, ...] vs [woman, female, she, ...]
  ethnicity: [white, european, ...] vs [black, african, ...]
  age:       [young, youth, ...] vs [old, elderly, ...]

Each attribute pair has positive/negative word lists.
```

**FR-CORE-BM-003: Severity Thresholds**
```
bias_score ∈ [0.0, 1.0]

severity:
  low:    score < bias.medium_threshold  (default 0.3)
  medium: score < bias.high_threshold    (default 0.6)
  high:   score ≥ bias.high_threshold

High severity → NATS event + hook fire.
```

### 3.3 Phase 2: Response Verification (FR-CORE-RV)

**FR-CORE-RV-001: Response Bias Detection**
```
Apply WEAT analysis to LLM response embedding.
Complements Sentinel-OUT (text-level pattern matching)
with embedding-level statistical bias detection.
```

**FR-CORE-RV-002: Quality Evaluation**
```
Estimate recall impact of noise via QualityMonitor:
  - Inject deterministic noise (seed=42) to query embeddings
  - Measure cosine similarity delta vs original
  - Estimate recall drop

QualityMonitor uses NoiseInjector internally.
```

**FR-CORE-RV-003: Anomaly Detection**
```
Statistical checks on LLM response:
  - Length anomaly: response length vs baseline distribution
  - Jaccard similarity of content words vs context_chunks
  - Embedding distance from query embedding
```

**FR-CORE-RV-004: Drift Detection**
```
Compare current embedding pairs vs baseline pairs:
  avg_cosine_similarity(current) vs avg_cosine_similarity(baseline)
  delta = |current_avg - baseline_avg|

If delta > threshold → drift detected.
```

**FR-CORE-RV-005: Overall Status**
```
Verifier._determine_status() aggregates all signals:
  bias_score, anomaly flags, quality, coherence

Status values: "approved" | "review" | "flagged"
```

**FR-CORE-RV-006: Insight Generation**
```
Natural-language insights for each detected issue:
  - "High gender bias detected (score: 0.72)"
  - "Response length anomaly (ratio: 0.12)"
```

### 3.4 Noise Result Metrics

```
After injection, NoiseInjector returns NoiseResult:
  secured:              noised embedding (list[float])
  noise_magnitude:      ||noise||₂ / ||original||₂
  similarity_to_original: cosine_similarity(original, noised)
  strategy:             "gaussian" | "laplacian"
  original_norm:        ||original||₂
  secured_norm:         ||noised||₂
```

### 3.5 Core Summary

```
Standalone capabilities (numpy only, no models):

Phase 1:
✅ Gaussian noise injection
✅ Laplacian noise injection
✅ Norm preservation
✅ Per-tenant sigma
✅ WEAT bias analysis

Phase 2:
✅ Response bias detection
✅ Quality estimation
✅ Anomaly detection
✅ Drift detection
✅ Status determination + insights

Works without other Bastion modules.
No ML models required.
```

---

## 4. Enhanced Functions (🟡 Composition)

### 4.1 Search Quality Optimization (FR-ENH-SQ)

**Requires: Navigator (search quality feedback)**

**FR-ENH-SQ-001: Noise-Quality Tuning**
```
When Navigator available:
- Caller passes search quality metrics into Anchor
- Anchor measures recall impact at current sigma
- Returns recommended sigma adjustment

Interface (data passed in by caller):
  POST /v1/anchor/quality
  { "noise_sigma": 0.01, "embedding_pairs": [...] }

QualityMonitor.estimate(sigma, embedding_pairs)
  → QualityResponse(recall_drop_estimate, similarity_preservation)

Graceful degradation:
- Without Navigator: use default sigma; no tuning
- Core noise injection fully operational
```

### 4.2 Enhanced Summary

```
🟡 Search quality tuning (+ Navigator)

Without Navigator: default sigma works; core noise injection active
```

---

## 5. Hooks (🔴 Cross-Cutting)

### 5.1 Pipeline Bias Monitoring Hooks

**Hook Point:**
```
anchor.bias.measured
- Fired after MeasureBias returns a "high" severity measurement
- Detail: see relevant cross-cutting SRS
```

**Contract:**
```
On high-severity bias:
→ NATS event: bastion.events.anchor.bias_detected
→ HookEvent: EVENT_BIAS_DETECTED
  data: {"category": "gender", "bias_score": 0.74}
→ Tracker aggregates pipeline bias metrics

Full logic: cross-cutting SRS
```

### 5.2 Lineage Hooks

```
After SecureEmbedding:
→ NATS event: bastion.events.anchor.embedding_secured
→ HookEvent: EVENT_EMBEDDING_SECURED

After VerifyResponse:
→ NATS event: bastion.events.anchor.response_verified
→ HookEvent: EVENT_RESPONSE_VERIFIED

Detail: see Data Lineage SRS (Tier 3)
```

### 5.3 Hook Summary

```
🔴 anchor.bias.measured          → bias monitoring
🔴 anchor.embedding.secured      → Lineage SRS
🔴 anchor.response.verified      → Lineage SRS
```

---

## 6. External Interfaces

### 6.1 gRPC Interface

```
Wire format: JSON-over-gRPC (GenericRpcHandler, JSON codec)
Service: bastion.anchor.v1.AnchorService

Methods:
  SecureEmbedding(SecureRequest) → SecureResponse
  SecureBatch(BatchSecureRequest) → BatchSecureResponse
  MeasureBias(BiasRequest) → BiasResponse
  MeasureQuality(QualityRequest) → QualityResponse
  GetConfig(_) → AnchorConfig
  UpdateConfig(AnchorConfig) → AnchorConfig
  Health(_) → {"status":"ok","version":"..."}
  VerifyResponse(VerifyRequest) → VerifyResponse
  AnalyzeResponse(VerifyRequest) → VerifyResponse
  AnalyzeResponseEmbedding(EmbeddingAnalysisRequest) → EmbeddingAnalysisResponse
  CheckDrift(DriftRequest) → DriftResponse

gRPC metadata headers:
  x-trace-id, x-span-id, x-parent-span-id
  x-tenant-id, x-user-id, x-request-id
```

**SecureRequest (JSON):**
```json
{
  "request_id": "uuid",
  "embedding": [0.42, -0.31, 0.18, "..."],
  "tenant_id": "tenant-acme",
  "operation": "query",
  "options": {
    "noise_sigma": 0.0,
    "seed": ""
  }
}
```

**SecureResponse (JSON):**
```json
{
  "request_id": "uuid",
  "secured_embedding": [0.43, -0.30, 0.19, "..."],
  "metrics": {
    "noise_added": 0.029,
    "similarity_to_original": 0.985,
    "strategy_used": "gaussian",
    "original_norm": 1.0,
    "secured_norm": 1.0
  }
}
```

**VerifyRequest (JSON):**
```json
{
  "request_id": "uuid",
  "tenant_id": "tenant-acme",
  "llm_response": "The warranty period is 12 months...",
  "original_query": "product warranty terms",
  "context_chunks": ["doc1 content", "doc2 content"],
  "response_embedding": [0.42, "..."],
  "query_embedding": [0.41, "..."]
}
```

### 6.2 REST Interface

```
# Phase 1
POST /v1/anchor/secure
POST /v1/anchor/secure/batch
POST /v1/anchor/bias

# Phase 2
POST /v1/anchor/verify
POST /v1/anchor/analyze
POST /v1/anchor/analyze/embedding
POST /v1/anchor/drift

# Config
GET  /v1/anchor/config
PUT  /v1/anchor/config

# Standard
GET  /v1/health
GET  /v1/metrics     (Prometheus)
POST /v1/anchor/quality
```

### 6.3 CLI Interface

```bash
# Standalone server
$ python -m anchor.main --config config.yaml

# Phase 1
$ anchor-cli secure --embedding-file emb.json --tenant-id tenant-acme

# Phase 2
$ anchor-cli verify --response "..." --query "..."

# Server
$ anchor-cli server
```

### 6.4 Events (Foundation Schema)

```
Operational (NATS):
  bastion.events.anchor.embedding_secured
  bastion.events.anchor.response_verified

Via hooks:
  bastion.events.anchor.bias_detected
  bastion.events.anchor.anomaly_detected
```

---

## 7. Non-Functional Requirements

### 7.1 Performance

| ID | Item | Target |
|---|---|---|
| NFR-PE-001 | Noise injection (p95) | < 5ms |
| NFR-PE-002 | Response verify (p95) | < 50ms |
| NFR-PE-003 | Bias analysis (p95) | < 10ms |
| NFR-PE-004 | Memory | ≤ 1GB |
| NFR-PE-005 | Startup time | < 3s (no model load) |

### 7.2 Independence (Foundation)

```
NFR-IND-001: Core works standalone (numpy only, no services)
NFR-IND-002: Graceful degradation (Navigator optional)
NFR-IND-003: Loose coupling (search metrics passed in, no Navigator ref held)
```

---

## 8. System Architecture

```
┌────────────────────────────────────────────────┐
│           Anchor Service (Python)               │
├────────────────────────────────────────────────┤
│  API Layer                                      │
│    FastAPI (REST, uvicorn)  │  gRPC server      │
│    port 8083                │  port 9093        │
│         ↓                              ↓        │
│  ─────────── AnchorServiceHandler ──────────    │
│  Routes to Phase 1 or Phase 2 methods           │
│         ↓                                       │
│  ┌──────────────────────────────────────┐       │
│  │  Core Components (numpy)             │       │
│  │  NoiseInjector   — Gaussian/Laplace  │       │
│  │  BiasAnalyzer    — WEAT (cosine sim) │       │
│  │  QualityMonitor  — recall estimation │       │
│  │  Verifier        — Phase 2 analysis  │       │
│  └──────────────────────────────────────┘       │
│         ↓                                       │
│  ┌──────────────────────────────────────┐       │
│  │  Optional                            │       │
│  │  HookManager   (cross-cutting)       │       │
│  │  Publisher     (NATS events)         │       │
│  └──────────────────────────────────────┘       │
└────────────────────────────────────────────────┘
```

### 8.1 Component Responsibilities

| Component | File | Responsibility |
|---|---|---|
| `NoiseInjector` | `noise.py` | Gaussian/Laplacian noise, norm preservation |
| `BiasAnalyzer` | `bias.py` | WEAT analysis via numpy cosine similarity |
| `QualityMonitor` | `quality.py` | Recall-impact estimation via NoiseInjector |
| `Verifier` | `verifier.py` | Phase 2: bias, quality, anomaly, drift, status |
| `HookManager` | `hooks.py` | Thread-per-handler async hook dispatch |
| `Publisher` | `events.py` | NATS events via background asyncio thread |
| `Config` | `config.py` | Pydantic config; `sigma_for_tenant()` |
| `_AnchorServiceHandler` | `grpc_server.py` | gRPC GenericRpcHandler |
| REST routes | `rest.py` | FastAPI app |

---

## 9. Standalone Operation

### 9.1 Startup Log

```
[anchor] starting v3.0 (REST :8083, gRPC :9093)
[anchor] noise: strategy=gaussian, sigma=0.01, preserve_norm=true
[anchor] bias: enabled (medium=0.3, high=0.6)
[anchor] quality: enabled
[anchor] grpc listening on :9093
[anchor] ready
```

### 9.2 Standalone Test (Litmus)

```
POST /v1/anchor/secure
{
  "embedding": [0.42, -0.31, 0.18, ...],
  "tenant_id": "tenant-acme"
}

→ 200 OK
{
  "secured_embedding": [0.43, -0.30, 0.19, ...],
  "metrics": {
    "noise_added": 0.029,
    "similarity_to_original": 0.985,
    "strategy_used": "gaussian"
  }
}

(No Navigator, Vault, Sentinel, or Tracker needed)
```

### 9.3 Degradation

```
Without other modules:
✅ Noise injection: fully operational
✅ WEAT bias analysis: fully operational
✅ Response verification: fully operational
⚠️ Quality tuning: inactive (needs Navigator search metrics)

Core fully functional ✅
```

---

## 10. Configuration

```yaml
# /etc/bastion-anchor/config.yaml
version: "3.0"

server:
  rest_port: 8083
  grpc_port: 9093
  workers: 1

noise:
  default_sigma: 0.01
  preserve_norm: true
  strategy: gaussian         # gaussian | laplacian
  tenant_overrides:
    # tenant-acme: 0.02      # uncomment for per-tenant override

bias:
  enabled: true
  medium_threshold: 0.3
  high_threshold: 0.6
  dims: 1024                 # embedding dimension

quality:
  enabled: true
  recall_drop_alert: 0.1     # alert if recall drops > 10%
  similarity_minimum: 0.8    # minimum cosine similarity

events:
  nats_url: nats://nats:4222
```

---

## 11. Summary

```
🟢 Core (Standalone, Python + numpy):
   Phase 1:
     - Gaussian/Laplacian noise injection (rng.normal / rng.laplace)
     - Norm preservation (np.linalg.norm)
     - Per-tenant sigma overrides
     - WEAT bias analysis (cosine similarity, np.dot)

   Phase 2:
     - Response bias detection (embedding-level)
     - Quality estimation (recall impact)
     - Anomaly detection (length, Jaccard, embedding distance)
     - Drift detection (cosine similarity delta)
     - Status determination (approved / review / flagged)
     - Natural-language insights

🟡 Enhanced (Composition):
   - Search quality tuning (+ Navigator)

🔴 Hooks (Cross-cutting):
   - Pipeline bias monitoring (→ cross-cutting SRS)
   - Lineage (→ Lineage SRS)

Wire contract: unchanged from v2 (JSON-over-gRPC, REST JSON)
Language change: Go → Python (internal implementation detail only)
```

---

## 12. Change History

| Version | Date | Changes |
|---|---|---|
| 1.0 | 2026-05-17 | Initial (separate IN/OUT) |
| 2.0 | 2026-05-17 | Foundation-aligned, IN+OUT integrated |
| 3.0 | 2026-05-26 | **Python rewrite**: numpy/scipy for all numerical ops; FastAPI + grpcio; Phase 2 verifier expanded |

---

**End of Document**

## Appendix: Cross-cutting References

```
Lineage: hooks embedding.secured / response.verified → Lineage SRS
Bias monitoring: pipeline-wide aggregation → Tracker
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
  - Noise: Box-Muller (Go manual) → np.random.normal() / np.random.laplace()
  - Cosine sim: Go manual loop → np.dot / np.linalg.norm
  - Norm: manual sqrt → np.linalg.norm
  - Config: Go struct + viper → Pydantic model + pyyaml
  - gRPC: ServiceDesc (protobuf) → GenericRpcHandler (JSON codec)

Old Go source: anchor/internal/ (archived; superseded)
New Python source: anchor/anchor/
```

## Appendix: PoC Note

```
Simplified for PoC:
  - Gaussian/Laplacian noise (not formal differential privacy)
  - Basic WEAT bias (not full SEAT/debiasing)
  - Statistical anomaly (not adversarial robustness)

Future (post-PoC):
  - scipy-based differential privacy (Laplace/Gaussian mechanisms)
  - Embedding integrity HMAC
  - Adversarial example detection
```
