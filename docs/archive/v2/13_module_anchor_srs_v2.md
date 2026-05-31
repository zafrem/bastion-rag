# Bastion-Anchor Module SRS

**Project:** Bastion-RAG - RAG Security Governance Framework  
**Document Type:** Module SRS (Tier 2)  
**Document ID:** 13-anchor-srs  
**Module:** E - Anchor (Embedding Security)  
**Version:** 2.0 (Foundation-aligned, IN+OUT integrated)  
**Date:** 2026-05-17  
**Status:** Draft  
**Scope:** PoC (simplified)

**Foundation References:**
- 01-architecture-principles
- 02-event-schema-standard
- 03-module-interaction-map

---

## 1. Introduction

### 1.1 Purpose

This document specifies the **Anchor** module, the embedding security layer of Bastion-RAG. Anchor operates bidirectionally:
- **Phase 1 (Input):** Inject noise into embeddings
- **Phase 2 (Output):** Analyze response for bias/anomaly

This SRS is **intentionally simplified for PoC**, demonstrating the concept of embedding security.

### 1.2 Module Identity

```
Module: E - Anchor
Role: Embedding Security (bidirectional)
Position: Before LLM (input) + after LLM (output)

Standalone value:
"Attach Anchor → embeddings protected from inversion"
```

### 1.3 The Standalone Test (Foundation Litmus)

```
Question: "If only Anchor is attached,
          does it provide meaningful security?"

Answer: YES
- Input: noise injection (inversion defense)
- Output: bias/anomaly detection

→ Anchor passes the standalone test ✅
```

### 1.4 Why Embedding Security Matters

```
Embeddings can be inverted to ~90% original text
(Morris 2023). In RAG:
- Vector DB stores all embeddings
- Breach = potential data exposure
- Anchor adds protection layer
```

### 1.5 Scope

**In Scope (PoC):**
- 🟢 Core: Gaussian noise injection (Phase 1)
- 🟢 Core: Bias detection (Phase 1 + 2)
- 🟢 Core: Response analysis (Phase 2)
- 🟢 Core: Anomaly detection (Phase 2)
- 🟡 Enhanced: Search quality optimization (with Navigator)
- 🔴 Hooks: Pipeline-wide bias monitoring
- 🔴 Hooks: Lineage emission
- Bidirectional (Phase 1 + 2)

**Out of Scope (Future):**
- Formal differential privacy
- Embedding integrity HMAC
- Adversarial robustness
- Detailed cross-cutting (respective SRS)

### 1.6 Definitions

| Term | Definition |
|---|---|
| **Anchor Phase 1** | Embedding noise (input) |
| **Anchor Phase 2** | Response analysis (output) |
| **Noise injection** | Adding perturbation |
| **WEAT** | Bias measurement method |
| **Inversion** | Reconstructing text from embedding |

---

## 2. Overall Description

### 2.1 Bidirectional Architecture

```
┌─────────────────────────────────────────────┐
│             Anchor Service                   │
├─────────────────────────────────────────────┤
│  ┌────────────┐         ┌────────────┐      │
│  │ Phase 1 API│         │ Phase 2 API│      │
│  │  (/secure) │         │ (/analyze) │      │
│  └─────┬──────┘         └─────┬──────┘      │
│        └───────────┬──────────┘             │
│                    ▼                        │
│      ┌──────────────────────────┐           │
│      │  Shared Embedding Service│           │
│      │  (BGE-M3 client)         │           │
│      └────────┬─────────────────┘           │
│               │                             │
│      ┌────────┼─────────┐                   │
│      ▼        ▼         ▼                   │
│  ┌───────┐┌───────┐┌──────────┐             │
│  │Noise  ││Analyzer││ Hooks   │             │
│  │Injector││       ││(optional)│             │
│  └───────┘└───────┘└──────────┘             │
│                                              │
└─────────────────────────────────────────────┘
```

### 2.2 Position in Pipeline

```
Input (Phase 1):
Navigator → [Anchor: noise] → LLM

Output (Phase 2):
LLM → [Anchor: analyze] → Vault-OUT
```

### 2.3 Layer Classification

```
🟢 CORE (Standalone):
   Phase 1: noise injection, bias measurement
   Phase 2: response analysis, anomaly, coherence

🟡 ENHANCED (Composition):
   - Search quality (+ Navigator)

🔴 HOOKS (Cross-cutting):
   - Pipeline bias monitoring
   - Lineage emission
```

### 2.4 Constraints

```
Language: Go 1.21+
Embedding: BGE-M3 (1024-dim)
Memory: ≤ 1GB
Latency: <5ms (noise), <50ms (analyze)
CPU-first (GPU optional)
```

### 2.5 Dependencies

```
Core dependencies:
- Embedding service (BGE-M3)

Optional (Enhanced):
- Navigator (search quality feedback)

Optional (Hooks):
- NATS, coordinators
```

---

## 3. Core Functions (🟢 Standalone)

### 3.1 Phase 1: Noise Injection (FR-CORE-NI)

**FR-CORE-NI-001: Gaussian Noise**
```
Add noise to embeddings:
- N(0, σ²), default σ=0.01
- L2 normalize after
- Inversion defense

Dependency: NONE (math only)
```

**FR-CORE-NI-002: Norm Preservation**
```
Maintain unit vector (cosine similarity)
```

**FR-CORE-NI-003: Configurable Strength**
```
Per-tenant noise levels
Trade-off: security vs search quality
```

### 3.2 Phase 1: Bias Measurement (FR-CORE-BM)

**FR-CORE-BM-001: WEAT-based**
```
Measure embedding bias:
- gender, ethnicity, age
- Compare attribute pairs
- Score 0-1
```

### 3.3 Phase 2: Response Analysis (FR-CORE-RA)

**FR-CORE-RA-001: Response Bias Detection**
```
Embed LLM response
Check for bias (statistical/embedding level)
Complements Sentinel-OUT (text level)
```

**FR-CORE-RA-002: Quality Measurement**
```
- Conciseness
- Diversity
- Relevance to query
```

**FR-CORE-RA-003: Anomaly Detection**
```
Statistical outlier detection
Length anomaly
Embedding distance from baseline
```

**FR-CORE-RA-004: Coherence Check**
```
Sentence-to-sentence similarity
Internal consistency
```

### 3.4 Core Summary

```
Standalone (embedding service only):

Phase 1:
✅ Noise injection
✅ Bias measurement

Phase 2:
✅ Response bias detection
✅ Quality measurement
✅ Anomaly detection
✅ Coherence check

Works without other Bastion-RAG modules.
```

---

## 4. Enhanced Functions (🟡 Composition)

### 4.1 Search Quality Optimization (FR-ENH-SQ)

**Requires: Navigator (search feedback)**

**FR-ENH-SQ-001: Noise-Quality Tuning**
```
When Navigator available:
- Measure search quality with noise
- Optimize noise level
- Balance security vs recall

Interface (data passed in):
OptimizeNoise(searchMetrics)

Graceful degradation:
- Without Navigator: use default noise
- Core noise injection works
```

### 4.2 Enhanced Summary

```
🟡 Search quality optimization (+ Navigator)

Without Navigator: default noise works
```

---

## 5. Hooks (🔴 Cross-Cutting)

### 5.1 Pipeline Bias Monitoring Hooks

**Hook Points:**
```
anchor.bias.measured
- Emit bias measurements
- Pipeline-wide bias tracking
- Detail: see relevant cross-cutting
```

**Brief Contract:**
```
On bias detection:
→ event: anchor.bias_detected
→ Tracker aggregates pipeline bias

Full logic: cross-cutting SRS.
```

### 5.2 Lineage Hooks

```
anchor.embedding.secured
anchor.response.analyzed
→ Lineage events with trace_id
Detail: see Data Lineage SRS
```

### 5.3 Hook Summary

```
🔴 anchor.bias.measured       → bias monitoring
🔴 anchor.*.secured/analyzed  → Lineage SRS
```

---

## 6. External Interfaces

### 6.1 gRPC Interface

```protobuf
service AnchorService {
  // Core - Phase 1
  rpc SecureEmbedding(SecureRequest) returns (SecureResponse);
  rpc MeasureBias(BiasRequest) returns (BiasResponse);
  
  // Core - Phase 2
  rpc AnalyzeResponse(AnalyzeRequest) returns (AnalyzeResponse);
  
  rpc Health(HealthRequest) returns (HealthResponse);
}

message SecureRequest {
  string request_id = 1;
  string trace_id = 2;
  repeated float embedding = 3;
  float noise_sigma = 4;
}

message AnalyzeRequest {
  string request_id = 1;
  string trace_id = 2;
  string llm_response = 3;
  string original_query = 4;
}
```

### 6.2 REST Interface

```
# Core Phase 1
POST /v1/anchor/secure
POST /v1/anchor/measure-bias

# Core Phase 2
POST /v1/anchor/analyze

# Standard
GET  /v1/health
```

### 6.3 CLI Interface

```bash
# Core secure
$ anchor-cli secure --embedding-file emb.json --sigma 0.01

# Core analyze
$ anchor-cli analyze --response "..." --query "..."

# Standalone
$ anchor-cli server
```

### 6.4 Events (Foundation Schema)

```
Operational:
- anchor.embedding_secured
- anchor.response_analyzed

Via hooks:
- anchor.bias_detected
- anchor.anomaly_detected
```

---

## 7. Non-Functional Requirements

### 7.1 Performance

| ID | Item | Target |
|---|---|---|
| NFR-PE-001 | Noise injection (p95) | < 5ms |
| NFR-PE-002 | Response analysis (p95) | < 50ms |
| NFR-PE-003 | Bias check | < 10ms |
| NFR-PE-004 | Memory | ≤ 1GB |

### 7.2 Independence (Foundation)

```
NFR-IND-001: Core works standalone (embedder only)
NFR-IND-002: Graceful degradation (Navigator optional)
NFR-IND-003: Loose coupling
```

---

## 8. System Architecture

```
┌────────────────────────────────────────────┐
│           Anchor Service                    │
├────────────────────────────────────────────┤
│  API (gRPC/REST/CLI)                        │
│         ↓                                   │
│  Phase Dispatcher (secure/analyze)          │
│         ↓                                   │
│  Shared Embedding Service (BGE-M3)          │
│         ↓                                   │
│  ┌─────────────────────────────────┐        │
│  │  Core                           │        │
│  │  Phase 1: Noise Injector        │        │
│  │  Phase 2: Analyzer              │        │
│  │   - Bias, Quality, Anomaly      │        │
│  └─────────────────────────────────┘        │
│         ↓                                   │
│  Hook Manager (optional)                    │
│         ↓                                   │
│  Event Publisher (NATS)                     │
└────────────────────────────────────────────┘
```

---

## 9. Standalone Operation

### 9.1 Standalone Mode

```bash
$ anchor-cli server

🚀 Bastion-Anchor v2.0 starting...
✅ Embedding service ready (BGE-M3)
✅ Noise injector ready (σ=0.01)
✅ Analyzer ready
⚠️  Navigator: not connected (no quality tuning)
✅ Core: FULLY OPERATIONAL
✨ Ready (standalone)
```

### 9.2 Standalone Test (Litmus)

```bash
$ anchor-cli secure \
    --embedding "[0.42,-0.31,...]" \
    --standalone

✅ Secured
Noise added: 0.029
Similarity: 0.985
(Inversion defense, no other module needed)
```

### 9.3 Degradation

```
Without other modules:
✅ Noise injection: works
✅ Response analysis: works
✅ Bias detection: works
⚠️ Quality tuning: inactive (needs Navigator)

Core fully functional ✅
```

---

## 10. Configuration

```yaml
# /etc/bastion-anchor/config.yaml
version: 2.0

# Core
core:
  embedding:
    service: http://embedder:8000
  noise:
    default_sigma: 0.01
    preserve_norm: true
  bias:
    categories: [gender, ethnicity, age]
  analysis:
    quality: true
    anomaly: true
    coherence: true

# Enhanced
enhanced:
  quality_tuning: true  # If Navigator present

# Hooks
hooks:
  bias_monitoring: true
  lineage: true

# Events
events:
  nats_url: nats://nats:4222
```

---

## 11. Summary

```
🟢 Core (Standalone):
   Phase 1: noise injection, bias measurement
   Phase 2: response analysis, anomaly, coherence

🟡 Enhanced (Composition):
   - Search quality tuning (+ Navigator)

🔴 Hooks (Cross-cutting):
   - Pipeline bias monitoring
   - Lineage (→ Lineage SRS)

PoC scope: simplified but demonstrates concept.
```

---

## 12. Change History

| Version | Date | Changes |
|---|---|---|
| 1.0 | 2026-05-17 | Initial (separate IN/OUT) |
| 2.0 | 2026-05-17 | Foundation-aligned, IN+OUT integrated |

---

**End of Document**

## Appendix: Cross-cutting References

```
Lineage: hooks secured/analyzed
Bias monitoring: pipeline-wide aggregation
```

## Appendix: PoC Simplification Note

```
Simplified for PoC:
- Gaussian noise (not formal DP)
- Basic WEAT bias
- Statistical anomaly

Future (v2.0+):
- Differential privacy
- Embedding integrity
- Adversarial robustness
```
