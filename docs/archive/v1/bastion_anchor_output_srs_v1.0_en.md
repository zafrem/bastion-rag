# Bastion-Anchor Output Phase SRS v1.0

**Project:** Bastion-RAG - RAG Security Governance Framework  
**Module:** Module E - Anchor (Output Phase)  
**Document Version:** 1.0  
**Date:** 2026-05-17  
**Status:** Draft  
**Parent Document:** Bastion-Anchor SRS v1.0 (Input Phase)  
**Scope:** Output Pipeline Bias & Quality Verification (PoC)

---

## 1. Introduction

### 1.1 Purpose

This document defines the **Output Phase** of the Bastion-Anchor module. Anchor operates as a bidirectional embedding security module:

- **Phase 1 (Input/Indexing):** Apply noise to embeddings during storage
- **Phase 2 (Output/Response):** Verify bias and quality in LLM responses ⭐ (This SRS)

This SRS focuses specifically on:
1. **Response Bias Detection** - Analyze LLM responses for bias
2. **Embedding Drift Verification** - Confirm Phase 1 noise didn't compromise retrieval
3. **Response Embedding Analysis** - Check response characteristics
4. **Quality Feedback Loop** - Measure end-to-end quality

**Note:** Like Phase 1, this is **intentionally simplified for PoC**. The goal is to demonstrate that bias and embedding-related concerns extend to the output side, not to implement every advanced technique.

### 1.2 Background: Completing the Bidirectional Pattern

#### Bastion-RAG's Bidirectional Architecture

```
Input Path:                Output Path:
A (Sentinel-IN)            A (Sentinel-OUT) ✓
B (Vault-IN)               B (Vault-OUT) ✓
C (Navigator)              [Response from LLM]
E (Anchor-IN)              E (Anchor-OUT) ⭐ This doc
   ↓                          ↑
  LLM ────────────────────────┘
```

This SRS completes the bidirectional pattern across all Bastion-RAG modules.

#### Why Output-Side Bias Matters

Even with Phase 1 protections (noise injection, bias measurement), the LLM can:

```
Risk 1: Amplify Existing Bias
─────────────────────────────
Phase 1 measured bias = 0.08 (low)
But LLM amplifies patterns:
Response: "엔지니어는 보통 남성이..."
        ↑ Bias amplified to 0.25 in response

Risk 2: Introduce New Bias
─────────────────────────────
Embeddings: neutral
LLM training data: contains stereotypes
Response: Reflects LLM's training bias
        ↑ New bias from LLM, not embeddings

Risk 3: Context-Specific Bias
─────────────────────────────
Embeddings: appropriately diverse
Search results: balanced
LLM response: assumes specific demographic
        ↑ Bias in generation, not retrieval
```

### 1.3 Scope

**In Scope (PoC Implementation):**
- Response text bias detection (using simple methods)
- Comparison with Phase 1 bias measurements
- Response embedding analysis
- Quality metrics for end-to-end RAG
- Integration with Tracker for visualization
- Standalone testing
- gRPC, REST, CLI interfaces

**Out of Scope (Future Versions):**
- Advanced fairness metrics (demographic parity, equalized odds)
- Counterfactual analysis
- Causal bias detection
- LLM-based bias judging (LLM judges)
- Adversarial debiasing
- Multi-modal bias detection

### 1.4 Design Philosophy

```
Principle 1: Conceptual Demonstration
- Show that bias matters in BOTH directions
- Even simple checks add value
- Build awareness, not perfect defense

Principle 2: Same Module, Two Modes
- Single Anchor service
- Phase 1: Index-time embedding noise
- Phase 2: Output-time bias verification

Principle 3: Lightweight by Default
- Should not significantly slow responses
- Async where possible
- Sampling for expensive checks

Principle 4: Actionable Output
- Don't just detect, suggest fixes
- Provide bias scores for ops team
- Enable feedback loops

Principle 5: Honest About Limitations
- This is PoC level
- Real bias detection is research-grade
- Set realistic expectations
```

### 1.5 Definitions and Acronyms

| Term | Definition |
|---|---|
| **Anchor-IN** | Anchor in Phase 1 (input/indexing) |
| **Anchor-OUT** | Anchor in Phase 2 (output/response) |
| **Response Bias** | Bias detected in LLM-generated text |
| **WEAT** | Word Embedding Association Test |
| **Bias Score** | Quantitative measure of bias (0-1) |
| **Drift** | Difference between expected and observed bias |
| **Quality Score** | End-to-end RAG quality measure |
| **PoC** | Proof of Concept |

### 1.6 References

- Parent: Bastion-Anchor SRS v1.0 (Phase 1)
- Related: Bastion-Sentinel Output SRS v1.0
- Related: Bastion-Vault Output SRS v1.0
- Related: Bastion-Tracker SRS v1.0
- Caliskan et al. (2017) - "Semantics derived automatically from language corpora..."

---

## 2. Overall Description

### 2.1 Position in Pipeline

```
┌──────────────────────────────────────────────────────┐
│              User Query                               │
└────────────────────────┬─────────────────────────────┘
                         ▼
                  [Sentinel-IN]
                         ▼
                  [Vault-IN]
                         ▼
                  [Navigator]
                         ▼
                  [Anchor-IN]
                         ▼
                       [LLM]
                         ▼
        ┌────────────────────────────────────┐
        │   Anchor-OUT  ◄── (This doc)        │
        │   - Response Bias Detection         │
        │   - Embedding Analysis              │
        │   - Quality Verification            │
        │   - Drift Detection                 │
        └────────────────┬───────────────────┘
                         ▼
                  [Vault-OUT]
                         ▼
                  [Sentinel-OUT]
                         ▼
                  [User Response]
                         │
                         ▼ (async)
                     Tracker
```

### 2.2 Unified Anchor Architecture

```
┌─────────────────────────────────────────────────────┐
│              Unified Anchor Service                  │
├─────────────────────────────────────────────────────┤
│                                                      │
│  ┌────────────────┐    ┌────────────────┐           │
│  │   Phase 1 API  │    │   Phase 2 API  │           │
│  │  (/secure)     │    │  (/verify)     │           │
│  └────────┬───────┘    └────────┬───────┘           │
│           │                     │                    │
│           └─────────┬───────────┘                   │
│                     ▼                                │
│      ┌──────────────────────────────────┐           │
│      │     Mode Dispatcher              │           │
│      └──────────────┬───────────────────┘           │
│                     ▼                                │
│      ┌──────────────────────────────────┐           │
│      │   Phase-Specific Processors      │           │
│      │  ┌──────────────┐ ┌────────────┐ │           │
│      │  │ Phase 1:     │ │ Phase 2:   │ │           │
│      │  │ Embedding    │ │ Response   │ │           │
│      │  │ Secure       │ │ Verify     │ │ ⭐        │
│      │  │              │ │            │ │           │
│      │  │ - Noise      │ │ - Bias     │ │           │
│      │  │ - Bias       │ │   detect   │ │           │
│      │  │   measure    │ │ - Embed.   │ │           │
│      │  │ - Quality    │ │   analyze  │ │           │
│      │  │   check      │ │ - Drift    │ │           │
│      │  └──────────────┘ │ - Quality  │ │           │
│      │                   └────────────┘ │           │
│      └──────────────────────────────────┘           │
│                                                      │
│      ┌──────────────────────────────────┐           │
│      │    Shared Resources              │           │
│      │  - Embedding Model (BGE-M3)      │           │
│      │  - Bias Test Sets                │           │
│      │  - Configuration                 │           │
│      └──────────────────────────────────┘           │
│                                                      │
│      ┌──────────────────────────────────┐           │
│      │    Event Publisher (NATS)        │           │
│      └──────────────────────────────────┘           │
│                                                      │
└─────────────────────────────────────────────────────┘
```

### 2.3 Functions (Phase 2 Specific)

1. **F1: Response Bias Detection** ⭐ (Primary Function)
   - Analyze LLM response text for bias
   - Compare against Phase 1 baseline
   - Generate bias scores

2. **F2: Response Embedding Analysis**
   - Generate embedding of LLM response
   - Compare with retrieved context
   - Detect semantic drift

3. **F3: Bias Drift Detection**
   - Compare Phase 1 (embedding) bias vs Phase 2 (response) bias
   - Identify amplification or introduction
   - Alert on significant drift

4. **F4: Quality Metrics**
   - End-to-end retrieval-to-response quality
   - Grounding score (response based on context?)
   - Diversity metrics

5. **F5: Configurable Strategies**
   - Per-tenant bias sensitivity
   - Per-category test sets
   - Hot-reload of configurations

6. **F6: Event Reporting**
   - Send bias detection events to Tracker
   - Quality degradation alerts
   - Actionable insights

7. **F7: Standalone Operation**
   - Mock mode for testing
   - Works without other modules

### 2.4 User Characteristics

| User | Phase 2 Use Case |
|---|---|
| **Sentinel-OUT** | Receives bias info to inform decisions |
| **Tracker** | Visualizes bias trends |
| **Data Scientist** | Analyzes response patterns |
| **Operations Team** | Monitors bias alerts |
| **Demo Audience** | Sees bias detection in action |

### 2.5 Constraints

- **Language:** Go (same as Anchor-IN)
- **Memory:** Shared with Phase 1 (≤ 2GB total)
- **Latency:** <100ms p95 (response analysis is heavier than embedding noise)
- **Throughput:** ≥ 1,000 responses/s
- **CPU-only:** No GPU required (consistent with PoC)
- **Dependencies:** BGE-M3 (shared with Phase 1)

### 2.6 Assumptions

- LLM responses are received as text
- Same embedding model used as Phase 1 (BGE-M3)
- Retrieval context is available for comparison
- Tracker is available (graceful degradation)

---

## 3. External Interface Requirements

### 3.1 Interface Overview

Anchor-OUT extends Anchor's existing interfaces.

| Category | Interface | Purpose |
|---|---|---|
| **Input** | gRPC | Internal calls (from LLM) |
| **Input** | REST API | External integrations |
| **Input** | CLI | Manual testing, demos |
| **Output** | Verified response + metrics | To Vault-OUT |
| **Output** | Events | To Tracker via NATS |

### 3.2 gRPC Interface

```protobuf
// anchor.proto (extended)
syntax = "proto3";
package bastion-rag.anchor.v1;

service AnchorService {
  // Existing Phase 1 methods
  rpc SecureEmbedding(SecureRequest) returns (SecureResponse);
  rpc SecureBatch(BatchSecureRequest) returns (BatchSecureResponse);
  rpc MeasureBias(BiasRequest) returns (BiasResponse);
  
  // NEW: Phase 2 methods
  rpc VerifyResponse(VerifyRequest) returns (VerifyResponse);
  rpc AnalyzeResponseEmbedding(EmbeddingAnalysisRequest) returns (EmbeddingAnalysisResponse);
  rpc CheckDrift(DriftRequest) returns (DriftResponse);
  
  // Health
  rpc Health(HealthRequest) returns (HealthResponse);
}

message VerifyRequest {
  string request_id = 1;
  string trace_id = 2;
  string tenant_id = 3;
  
  // The LLM response to verify
  string llm_response = 4;
  
  // Original query for context
  string original_query = 5;
  
  // Retrieved documents that LLM saw
  repeated string source_documents = 6;
  
  // User context (for category-specific checks)
  UserContext user = 7;
  
  // Phase 1 baseline (for drift detection)
  BiasBaseline phase1_baseline = 8;
  
  // Options
  VerifyOptions options = 9;
}

message UserContext {
  string user_id = 1;
  string tenant_id = 2;
  string department = 3;
  string category = 4;  // Data category being queried
}

message BiasBaseline {
  float overall_score = 1;
  repeated CategoryBaseline categories = 2;
}

message CategoryBaseline {
  string category = 1;  // "gender", "ethnicity", "age"
  float score = 2;
  int64 measured_at = 3;
}

message VerifyOptions {
  bool check_bias = 1;
  bool analyze_embedding = 2;
  bool check_drift = 3;
  bool measure_quality = 4;
  float bias_threshold = 5;
  int32 timeout_ms = 6;
}

message VerifyResponse {
  string request_id = 1;
  Status status = 2;
  
  BiasAnalysis bias_analysis = 3;
  EmbeddingAnalysis embedding_analysis = 4;
  DriftAnalysis drift_analysis = 5;
  QualityMetrics quality = 6;
  
  repeated string warnings = 7;
  repeated string recommendations = 8;
  
  float processing_time_ms = 9;
  
  enum Status {
    UNKNOWN = 0;
    PASSED = 1;          // No significant issues
    WARNING = 2;         // Issues detected but acceptable
    CRITICAL = 3;        // Significant bias/issues
  }
}

message BiasAnalysis {
  float overall_score = 1;
  repeated CategoryBias categories = 2;
  string severity = 3;        // "low", "medium", "high"
  bool exceeds_threshold = 4;
}

message CategoryBias {
  string category = 1;
  float score = 2;
  repeated string evidence = 3;  // Words/phrases triggering detection
}

message EmbeddingAnalysis {
  float similarity_to_query = 1;
  float similarity_to_context = 2;
  float diversity_score = 3;
  bool semantic_drift_detected = 4;
}

message DriftAnalysis {
  bool drift_detected = 1;
  float phase1_bias_score = 2;
  float phase2_bias_score = 3;
  float drift_amount = 4;
  string drift_direction = 5;  // "amplification", "introduction", "reduction"
}

message QualityMetrics {
  float grounding_score = 1;     // How well response is grounded
  float coherence_score = 2;     // Internal consistency
  float relevance_score = 3;     // Relevance to query
  float overall_quality = 4;
}

message EmbeddingAnalysisRequest {
  string text = 1;
  repeated string comparison_texts = 2;
}

message EmbeddingAnalysisResponse {
  repeated float embedding = 1;
  repeated float similarities = 2;
  float diversity = 3;
}

message DriftRequest {
  BiasBaseline phase1 = 1;
  BiasAnalysis phase2 = 2;
}

message DriftResponse {
  bool drift_detected = 1;
  repeated CategoryDrift drifts = 2;
  string severity = 3;
}

message CategoryDrift {
  string category = 1;
  float phase1_score = 2;
  float phase2_score = 3;
  float change = 4;
  string direction = 5;
}
```

### 3.3 REST API

**New Endpoints:**

```
# Response verification (Phase 2)
POST /v1/anchor/verify                    # Verify LLM response
POST /v1/anchor/analyze-embedding         # Analyze response embedding
POST /v1/anchor/check-drift               # Compare Phase 1 vs Phase 2

# Bias trends
GET  /v1/anchor/bias/trends               # Bias trend analysis
GET  /v1/anchor/bias/baseline             # Current Phase 1 baseline
```

**Request Example:**

```http
POST /v1/anchor/verify HTTP/1.1
Host: anchor.bastion-rag.local
Content-Type: application/json

{
  "request_id": "req-anchor-out-001",
  "trace_id": "trace-12345",
  "tenant_id": "tenant-acme",
  "llm_response": "엔지니어들은 보통 분석적이고 논리적인 사고를 합니다. 이러한 자질을 가진 남성들이 이 분야에서 두각을 나타냅니다.",
  "original_query": "엔지니어의 특성은 무엇인가요?",
  "source_documents": [
    "엔지니어는 분석적 사고와 문제 해결 능력이 중요합니다.",
    "엔지니어링은 다양한 배경의 사람들이 활약하는 분야입니다."
  ],
  "user": {
    "user_id": "user-alice",
    "category": "manufacturing_data"
  },
  "phase1_baseline": {
    "overall_score": 0.08,
    "categories": [
      {"category": "gender", "score": 0.05},
      {"category": "ethnicity", "score": 0.10}
    ]
  },
  "options": {
    "check_bias": true,
    "analyze_embedding": true,
    "check_drift": true,
    "measure_quality": true,
    "bias_threshold": 0.15
  }
}
```

**Response Example (Bias Detected):**

```json
{
  "request_id": "req-anchor-out-001",
  "status": "CRITICAL",
  "processing_time_ms": 67.3,
  
  "bias_analysis": {
    "overall_score": 0.28,
    "categories": [
      {
        "category": "gender",
        "score": 0.28,
        "evidence": ["남성들이", "이 분야에서 두각"]
      }
    ],
    "severity": "high",
    "exceeds_threshold": true
  },
  
  "embedding_analysis": {
    "similarity_to_query": 0.85,
    "similarity_to_context": 0.62,
    "diversity_score": 0.45,
    "semantic_drift_detected": true
  },
  
  "drift_analysis": {
    "drift_detected": true,
    "phase1_bias_score": 0.05,
    "phase2_bias_score": 0.28,
    "drift_amount": 0.23,
    "drift_direction": "amplification"
  },
  
  "quality": {
    "grounding_score": 0.62,
    "coherence_score": 0.85,
    "relevance_score": 0.78,
    "overall_quality": 0.72
  },
  
  "warnings": [
    "Gender bias detected in response (0.28, threshold 0.15)",
    "Bias amplified from Phase 1 (0.05 → 0.28)",
    "Response contains generalizations not in source documents"
  ],
  
  "recommendations": [
    "Consider rephrasing response to avoid gender assumptions",
    "Source documents are gender-neutral; align response",
    "Review LLM model for gender bias"
  ]
}
```

**Response Example (Acceptable):**

```json
{
  "request_id": "req-anchor-out-002",
  "status": "PASSED",
  "processing_time_ms": 45.1,
  
  "bias_analysis": {
    "overall_score": 0.06,
    "severity": "low",
    "exceeds_threshold": false
  },
  
  "embedding_analysis": {
    "similarity_to_query": 0.88,
    "similarity_to_context": 0.81,
    "diversity_score": 0.72,
    "semantic_drift_detected": false
  },
  
  "drift_analysis": {
    "drift_detected": false,
    "phase1_bias_score": 0.05,
    "phase2_bias_score": 0.06,
    "drift_amount": 0.01,
    "drift_direction": "none"
  },
  
  "quality": {
    "grounding_score": 0.87,
    "coherence_score": 0.89,
    "relevance_score": 0.85,
    "overall_quality": 0.87
  }
}
```

### 3.4 CLI Interface

```bash
# Verify a response
$ anchor-cli verify \
    --response "엔지니어는 보통 남성이..." \
    --query "엔지니어 특성?" \
    --context-file source-docs.json \
    --output-format text

# Output:
══════════════════════════════════════════════════════
  Anchor-OUT Verification Report
══════════════════════════════════════════════════════
Status:           🚨 CRITICAL
Processing:       67.3ms

─── Bias Analysis ────────────────────────────────────
Overall Score:    0.28 (high)
Threshold:        0.15
Status:           ⚠️ EXCEEDS THRESHOLD

Category Breakdown:
  gender:  0.28  🚨 HIGH
    Evidence: "남성들이", "이 분야에서 두각"

─── Drift Analysis ───────────────────────────────────
Phase 1 (embedding) bias:  0.05  ✅
Phase 2 (response) bias:   0.28  🚨
Drift amount:              0.23
Direction:                 amplification

🔴 LLM AMPLIFIED bias not present in embeddings!

─── Embedding Analysis ───────────────────────────────
Query similarity:    0.85  ✅
Context similarity:  0.62  ⚠️ (below 0.75)
Diversity:           0.45  ⚠️ (low)
Semantic drift:      🚨 DETECTED

─── Quality Metrics ──────────────────────────────────
Grounding:    0.62  ⚠️ (some claims not in source)
Coherence:    0.85  ✅
Relevance:    0.78  ✅
Overall:      0.72  ⚠️

─── Recommendations ──────────────────────────────────
1. Rephrase to avoid gender assumptions
2. Source documents are gender-neutral - align response
3. Consider reviewing LLM model for gender bias

══════════════════════════════════════════════════════

# Check bias drift trends
$ anchor-cli drift-trend --days 7

Bias Drift Over Last 7 Days:
─────────────────────────────────────
Day      Avg Drift   Max Drift   Alerts
2026-05-11   0.03      0.08         0
2026-05-12   0.04      0.12         1
2026-05-13   0.05      0.18         2
2026-05-14   0.06      0.21         3
2026-05-15   0.08      0.25         5
2026-05-16   0.07      0.20         3
2026-05-17   0.09      0.28         6  ⚠️

Trend: 📈 Increasing
Recommendation: Review LLM behavior

# Compare specific response
$ anchor-cli compare \
    --phase1-baseline baseline.json \
    --response-file response.txt

Phase 1 baseline: 0.05
Phase 2 detected: 0.28
Drift: +0.23 (amplification)

# Analyze embedding
$ anchor-cli analyze-embedding \
    --text "response text" \
    --compare-against source-docs.json

Embedding similarity matrix:
                    doc-1   doc-2   doc-3
response            0.62    0.71    0.45
doc-1               -       0.55    0.40

Diversity: 0.58

# Interactive mode
$ anchor-cli interactive
anchor> mode output
Mode: output (Phase 2)

anchor> verify "엔지니어는 보통 남성이 강합니다"
🚨 Bias detected: gender (0.32)
   Evidence: ["남성이 강합니다"]
   Recommendation: Rephrase neutrally

anchor> stats --phase output
Today's Statistics:
  Verifications:  234
  Bias detected:  12 (5.1%)
  Drift alerts:   3
  Avg processing: 67ms

anchor> exit
```

### 3.5 Text Output (Demo Format)

```
══════════════════════════════════════════════════════
  Anchor-OUT: Response Bias Verification
══════════════════════════════════════════════════════
Trace ID:     trace-12345
Tenant:       tenant-acme
User:         alice@manufacturing
Time:         67.3ms

─── Original Query ──────────────────────────────────
"엔지니어의 특성은 무엇인가요?"

─── Source Documents (from Navigator) ───────────────
[doc-1] "엔지니어는 분석적 사고와 문제 해결 능력이 
         중요합니다."
[doc-2] "엔지니어링은 다양한 배경의 사람들이 활약하는 
         분야입니다."
[doc-3] "현대 엔지니어는 협업과 의사소통도 중요합니다."

─── LLM Response ────────────────────────────────────
"엔지니어들은 보통 분석적이고 논리적인 사고를 합니다. 
 이러한 자질을 가진 남성들이 이 분야에서 두각을 
 나타냅니다."

─── Bias Analysis ───────────────────────────────────
Phase 1 baseline (embeddings):
  gender: 0.05 ✅ (low bias)

Phase 2 detected (response):
  gender: 0.28 🚨 (high bias)
  
  Evidence found:
  - "남성들이"
  - "이 분야에서 두각을 나타냅니다"

─── Drift Analysis ──────────────────────────────────
Direction: AMPLIFICATION 🚨
  
  Phase 1: 0.05  Phase 2: 0.28
  ▓░░░░░░░       ▓▓▓▓▓▓▓░░░
  
  +0.23 bias added by LLM (not in embeddings)

─── Embedding Analysis ──────────────────────────────
Query → Response:    0.85 ✅ (relevant)
Context → Response:  0.62 ⚠️ (drift)
Diversity:           0.45 ⚠️ (homogeneous)

Source documents are gender-neutral.
Response introduces gender-specific framing.

─── Quality Verdict ─────────────────────────────────
Overall Quality:     0.72 ⚠️
Grounding:           0.62 ⚠️ (claims beyond source)
Coherence:           0.85 ✅
Relevance:           0.78 ✅

─── Recommendations ─────────────────────────────────
1. ⭐ Critical: Rephrase to remove gender assumptions
2. 📝 Align response with neutral source documents
3. 🔍 Review LLM for gender bias patterns
4. 📊 Monitor trend (3rd similar response this week)

─── Tracker Event ───────────────────────────────────
event: anchor.bias_amplification_detected
severity: high
category: gender
drift: 0.23

═══════════════════════════════════════════════════════
```

---

## 4. Functional Requirements

### 4.1 Response Bias Detection (FR-RB)

**FR-RB-001: Text-based Bias Detection**
- Use simple methods for PoC:
  - Keyword-based detection (gendered terms, ethnic markers)
  - Pattern matching (stereotypical phrases)
  - Sentiment analysis per category
- Multilingual support (Korean + English)

**FR-RB-002: Word List Comparison**
- Pre-defined word lists per category
- Detect imbalanced usage (e.g., only male pronouns)
- Compute representation ratios

**FR-RB-003: Embedding-based Bias Detection**
- Generate response embedding
- Compare with bias direction vectors
- Apply WEAT-inspired algorithms

**FR-RB-004: Category Coverage**
- Gender bias
- Ethnicity/national origin bias
- Age bias
- Disability bias (optional)
- Custom categories (extensible)

**FR-RB-005: Evidence Collection**
- Identify specific words/phrases triggering bias detection
- Provide context (surrounding text)
- Enable human review

### 4.2 Response Embedding Analysis (FR-EA)

**FR-EA-001: Embedding Generation**
- Use same model as Phase 1 (BGE-M3)
- 1024-dimensional vectors
- L2 normalized

**FR-EA-002: Similarity Metrics**
- Query → Response similarity
- Context → Response similarity
- Inter-document diversity

**FR-EA-003: Semantic Drift Detection**
- Threshold-based detection
- Response too different from context = potential issue
- Configurable thresholds

**FR-EA-004: Diversity Measurement**
- Compute diversity of source documents
- Compare with response diversity
- Detect homogenization

### 4.3 Drift Detection (FR-DD)

**FR-DD-001: Phase 1 vs Phase 2 Comparison**
- Retrieve Phase 1 baseline bias scores
- Compare with current Phase 2 measurements
- Compute drift amount

**FR-DD-002: Drift Categorization**
- **Amplification**: Phase 2 > Phase 1 + threshold
- **Introduction**: Phase 1 ≈ 0, Phase 2 > threshold
- **Reduction**: Phase 1 > Phase 2 (positive case)
- **None**: Within tolerance

**FR-DD-003: Drift Trending**
- Track drift over time
- Per-tenant, per-category trends
- Alert on persistent drift

**FR-DD-004: Drift Alerts**
- Real-time alerts for significant drift
- Send to Tracker
- Aggregate alerts to avoid noise

### 4.4 Quality Verification (FR-QV)

**FR-QV-001: Grounding Score**
- Verify response is based on context
- Use lexical overlap + semantic similarity
- Score 0-1

**FR-QV-002: Coherence Score**
- Check internal consistency of response
- Detect contradictions
- Simple heuristic for PoC

**FR-QV-003: Relevance Score**
- Verify response addresses query
- Embedding similarity query → response
- Threshold for acceptable

**FR-QV-004: Overall Quality**
- Weighted combination of metrics
- Configurable weights
- Used for trending

### 4.5 Configuration Management (FR-CM)

**FR-CM-001: Per-Tenant Settings**
- Bias sensitivity thresholds
- Active bias categories
- Custom word lists

**FR-CM-002: Category Configuration**
- Per-category test sets
- Industry-specific (HR, marketing, manufacturing)
- Multilingual support

**FR-CM-003: Hot-Reload**
- Update configuration without restart
- Atomic updates
- Validation before applying

### 4.6 Event Reporting (FR-ER)

**FR-ER-001: Bias Detection Events**
- Published to NATS
- Sent to Tracker
- Async, non-blocking

**FR-ER-002: Quality Events**
- Low quality alerts
- Drift trends
- Anomalies

**FR-ER-003: Aggregate Reports**
- Daily summaries
- Weekly trends
- Monthly analysis

---

## 5. Non-Functional Requirements

### 5.1 Performance (NFR-PE)

| ID | Item | Target |
|---|---|---|
| NFR-PE-001 | Bias detection latency (p95) | < 50ms |
| NFR-PE-002 | Embedding analysis (p95) | < 30ms |
| NFR-PE-003 | Full verification (p95) | < 100ms |
| NFR-PE-004 | Throughput | ≥ 1,000 responses/s |
| NFR-PE-005 | Memory overhead | ≤ 500MB (Phase 2) |

### 5.2 Reliability (NFR-RE)

| ID | Item | Target |
|---|---|---|
| NFR-RE-001 | Availability | 99.9% (PoC level) |
| NFR-RE-002 | False positive rate (bias) | < 10% (PoC tolerance) |
| NFR-RE-003 | False negative rate (bias) | < 20% (PoC tolerance) |
| NFR-RE-004 | Embedding consistency | 100% (same model) |

**Note:** Higher error tolerances reflect PoC nature. Production would require lower rates.

### 5.3 Security (NFR-SE)

| ID | Item | Requirement |
|---|---|---|
| NFR-SE-001 | Encryption in transit | TLS 1.3 |
| NFR-SE-002 | Authentication | mTLS |
| NFR-SE-003 | Response content | Not stored long-term |
| NFR-SE-004 | Bias scores | Logged with audit |

### 5.4 Usability (NFR-US)

| ID | Item | Target |
|---|---|---|
| NFR-US-001 | Clear bias evidence | All detections include reasoning |
| NFR-US-002 | Actionable recommendations | Suggested fixes provided |
| NFR-US-003 | Demo-friendly output | Text format readable |

---

## 6. System Architecture

### 6.1 Phase 2 Components

```
┌────────────────────────────────────────────────────────┐
│           Anchor Service (Phase 2 Focus)               │
├────────────────────────────────────────────────────────┤
│                                                         │
│  ┌─────────────────────────────────────────────────┐  │
│  │  Phase 2 Request Handler                        │  │
│  └─────────────────┬───────────────────────────────┘  │
│                    │                                   │
│                    ▼                                   │
│  ┌─────────────────────────────────────────────────┐  │
│  │  Verification Orchestrator                      │  │
│  │  - Parallel processing                          │  │
│  │  - Result aggregation                           │  │
│  └─────────────────┬───────────────────────────────┘  │
│                    │                                   │
│         ┌──────────┼──────────┐                       │
│         ▼          ▼          ▼                       │
│  ┌──────────┐ ┌──────────┐ ┌────────────┐            │
│  │  Bias    │ │Embedding │ │ Quality    │            │
│  │ Analyzer │ │ Analyzer │ │ Evaluator  │            │
│  │          │ │          │ │            │            │
│  │-Keyword  │ │-Generate │ │-Grounding  │            │
│  │-Pattern  │ │-Compare  │ │-Coherence  │            │
│  │-WEAT     │ │-Diversity│ │-Relevance  │            │
│  └────┬─────┘ └────┬─────┘ └────┬───────┘            │
│       │            │             │                    │
│       └────────────┼─────────────┘                    │
│                    ▼                                  │
│  ┌─────────────────────────────────────────────────┐ │
│  │  Drift Detector                                 │ │
│  │  - Phase 1 baseline comparison                  │ │
│  │  - Categorization                               │ │
│  └─────────────────┬───────────────────────────────┘ │
│                    ▼                                  │
│  ┌─────────────────────────────────────────────────┐ │
│  │  Recommendation Engine                          │ │
│  │  - Actionable insights                          │ │
│  │  - Severity-based suggestions                   │ │
│  └─────────────────┬───────────────────────────────┘ │
│                    ▼                                  │
│  ┌─────────────────────────────────────────────────┐ │
│  │  Response Builder + Event Publisher             │ │
│  └─────────────────────────────────────────────────┘ │
│                                                       │
│  Shared with Phase 1:                                │
│  - BGE-M3 embedding model                            │
│  - Bias test sets                                    │
│  - Configuration                                     │
│                                                       │
└────────────────────────────────────────────────────────┘
```

### 6.2 Data Flow

```
Phase 2 Verification Flow:

[LLM Response]
    ↓
[Anchor-OUT Handler]
    ↓
[Get Phase 1 baseline] ← From cache or storage
    ↓
[Parallel Analysis]:
├─ Bias Analyzer
│   ├─ Keyword matching
│   ├─ Pattern detection
│   └─ Embedding-based WEAT
├─ Embedding Analyzer
│   ├─ Generate response embedding
│   ├─ Compare with query
│   └─ Compare with context
├─ Quality Evaluator
│   ├─ Grounding check
│   ├─ Coherence check
│   └─ Relevance check
    ↓
[Drift Detector]
    ↓ (compare Phase 1 vs Phase 2)
[Categorize: amplification/introduction/reduction]
    ↓
[Recommendation Engine]
    ↓
[Build Response]
    ↓
[Publish Event to Tracker]
    ↓
[Return to Vault-OUT]
```

---

## 7. Implementation Details

### 7.1 Bias Detection Algorithm (Simple PoC)

```go
type BiasAnalyzer struct {
    wordLists      map[string]WordList
    embeddingModel EmbeddingModel
    config         BiasConfig
}

type WordList struct {
    Category string
    GroupA   []string  // e.g., male terms
    GroupB   []string  // e.g., female terms
    Neutral  []string
}

func (b *BiasAnalyzer) Analyze(response string) *BiasAnalysisResult {
    result := &BiasAnalysisResult{}
    
    // 1. Word-based detection
    for _, wordList := range b.wordLists {
        score := b.wordBasedScore(response, wordList)
        if score > b.config.WordThreshold {
            result.AddCategoryBias(CategoryBias{
                Category: wordList.Category,
                Score:    score,
                Evidence: b.findEvidence(response, wordList),
                Method:   "word-based",
            })
        }
    }
    
    // 2. Pattern-based detection (PoC simple)
    patterns := map[string][]string{
        "gender": []string{
            "보통 남성이", "주로 여성이",
            "남자들은", "여자들은",
            // Add more patterns
        },
        "ethnicity": []string{
            "보통 외국인은", "한국인은 모두",
            // Add more patterns
        },
    }
    
    for category, patternList := range patterns {
        for _, pattern := range patternList {
            if strings.Contains(response, pattern) {
                result.AddCategoryBias(CategoryBias{
                    Category: category,
                    Score:    0.4,  // Pattern match = significant
                    Evidence: []string{pattern},
                    Method:   "pattern",
                })
            }
        }
    }
    
    // 3. Embedding-based (WEAT-inspired)
    responseEmb, _ := b.embeddingModel.Encode(response)
    
    for _, testSet := range b.config.WEATTests {
        score := b.computeWEATScore(responseEmb, testSet)
        if score > b.config.EmbeddingThreshold {
            result.AddCategoryBias(CategoryBias{
                Category: testSet.Category,
                Score:    score,
                Method:   "embedding",
            })
        }
    }
    
    // 4. Aggregate and classify
    result.OverallScore = b.aggregateScores(result.Categories)
    result.Severity = b.classifySeverity(result.OverallScore)
    
    return result
}

func (b *BiasAnalyzer) wordBasedScore(text string, wl WordList) float64 {
    countA := 0
    countB := 0
    
    for _, word := range wl.GroupA {
        countA += strings.Count(text, word)
    }
    for _, word := range wl.GroupB {
        countB += strings.Count(text, word)
    }
    
    if countA + countB == 0 {
        return 0
    }
    
    // Imbalance score (0 = balanced, 1 = completely imbalanced)
    return math.Abs(float64(countA - countB)) / float64(countA + countB)
}
```

### 7.2 Drift Detection

```go
type DriftDetector struct {
    config DriftConfig
}

func (d *DriftDetector) Detect(
    phase1 *BiasBaseline,
    phase2 *BiasAnalysisResult,
) *DriftResult {
    result := &DriftResult{}
    
    for _, p2Category := range phase2.Categories {
        p1Score := d.getPhase1Score(phase1, p2Category.Category)
        p2Score := p2Category.Score
        
        diff := p2Score - p1Score
        
        categoryDrift := CategoryDrift{
            Category:    p2Category.Category,
            Phase1Score: p1Score,
            Phase2Score: p2Score,
            Change:      diff,
        }
        
        // Categorize drift
        switch {
        case diff > d.config.AmplificationThreshold:
            categoryDrift.Direction = "amplification"
            result.DriftDetected = true
        case p1Score < d.config.LowBiasThreshold && p2Score > d.config.ConcernThreshold:
            categoryDrift.Direction = "introduction"
            result.DriftDetected = true
        case diff < -d.config.ReductionThreshold:
            categoryDrift.Direction = "reduction"
        default:
            categoryDrift.Direction = "none"
        }
        
        result.CategoryDrifts = append(result.CategoryDrifts, categoryDrift)
    }
    
    // Overall severity
    result.Severity = d.determineSeverity(result)
    
    return result
}
```

### 7.3 Recommendation Engine

```go
type RecommendationEngine struct {
    templates map[string]RecommendationTemplate
}

func (r *RecommendationEngine) Generate(
    bias *BiasAnalysisResult,
    drift *DriftResult,
    quality *QualityResult,
) []string {
    var recommendations []string
    
    // Bias-based recommendations
    for _, categoryBias := range bias.Categories {
        if categoryBias.Score > 0.15 {
            template := r.templates[categoryBias.Category]
            recommendation := template.GenerateMessage(categoryBias)
            recommendations = append(recommendations, recommendation)
        }
    }
    
    // Drift-based recommendations
    if drift.DriftDetected {
        for _, categoryDrift := range drift.CategoryDrifts {
            switch categoryDrift.Direction {
            case "amplification":
                recommendations = append(recommendations,
                    fmt.Sprintf("LLM amplified %s bias not present in embeddings. Review LLM model.",
                        categoryDrift.Category))
            case "introduction":
                recommendations = append(recommendations,
                    fmt.Sprintf("LLM introduced %s bias. Source documents are neutral.",
                        categoryDrift.Category))
            }
        }
    }
    
    // Quality-based recommendations
    if quality.GroundingScore < 0.5 {
        recommendations = append(recommendations,
            "Response not well-grounded in source. Verify claims.")
    }
    
    return recommendations
}
```

---

## 8. Standalone Testing

### 8.1 Standalone Modes

```bash
# Server mode
$ anchor-cli server --mode output

# Demo mode
$ anchor-cli demo --phase 2 --scenario bias-amplification

# Test mode (no other modules needed)
$ anchor-cli verify --standalone \
    --response "test text" \
    --query "test query"
```

### 8.2 Demo Scenarios (for Tracker)

```yaml
scenario: "Bias Amplification Detection"
description: "LLM amplifies bias not in embeddings"
steps:
  1. Show neutral source documents
  2. Show Phase 1 baseline: bias = 0.05
  3. Show LLM response with gender stereotypes
  4. Anchor-OUT detects: bias = 0.28
  5. Visualize drift (Phase 1 → Phase 2)
  6. Show evidence and recommendations
  7. Send alert to Tracker

scenario: "Bias Introduction Detection"
description: "Neutral input, biased output"
steps:
  1. Source: neutral technical content
  2. Phase 1: bias near zero
  3. LLM response: introduces ethnic stereotypes
  4. Anchor-OUT detects: introduction pattern
  5. Show evidence and source comparison
  6. Recommend LLM model review

scenario: "Quality Verification"
description: "Check response groundedness"
steps:
  1. Source: specific facts about product
  2. LLM response: includes fabricated details
  3. Anchor-OUT detects: low grounding score
  4. Identify ungrounded claims
  5. Recommend fact-checking
```

### 8.3 Test Data

```
tests/output/
├── fixtures/
│   ├── biased_responses_ko.jsonl     # Korean biased examples
│   ├── biased_responses_en.jsonl     # English biased examples
│   ├── neutral_responses.jsonl       # Neutral examples
│   ├── source_documents.jsonl
│   └── phase1_baselines.jsonl
├── bias_lists/
│   ├── gender_words_ko.txt
│   ├── gender_words_en.txt
│   ├── ethnicity_words_ko.txt
│   └── ethnicity_words_en.txt
├── patterns/
│   ├── gender_patterns.txt
│   └── ethnicity_patterns.txt
└── expected_outputs/
```

---

## 9. Configuration Schema

```yaml
# /etc/bastion-anchor/output-config.yaml
version: 1.0

mode: output

# Bias detection
bias_detection:
  enabled: true
  
  # Word-based detection
  word_lists:
    gender:
      enabled: true
      lists_path: /etc/anchor/bias_lists/gender_*.txt
      threshold: 0.15
    
    ethnicity:
      enabled: true
      lists_path: /etc/anchor/bias_lists/ethnicity_*.txt
      threshold: 0.15
    
    age:
      enabled: true
      lists_path: /etc/anchor/bias_lists/age_*.txt
      threshold: 0.20
  
  # Pattern-based detection
  pattern_detection:
    enabled: true
    patterns_path: /etc/anchor/patterns/
  
  # Embedding-based (WEAT-style)
  embedding_based:
    enabled: true
    test_sets_path: /etc/anchor/weat_tests/
    threshold: 0.15
  
  # Overall settings
  severity_thresholds:
    low: 0.10
    medium: 0.20
    high: 0.30
  
  # Per-tenant overrides
  tenant_overrides:
    tenant-acme:
      gender_threshold: 0.10  # Stricter
    tenant-research:
      enabled: false  # Disabled for research

# Embedding analysis
embedding_analysis:
  enabled: true
  
  model: bge-m3
  endpoint: http://embedder:8000
  
  thresholds:
    query_similarity_min: 0.5
    context_similarity_min: 0.6
    diversity_min: 0.4

# Drift detection
drift_detection:
  enabled: true
  
  thresholds:
    amplification: 0.10
    introduction:  0.15  # When Phase 1 is near zero
    reduction:     0.10
    low_bias:      0.05  # Phase 1 threshold for "introduction"
  
  trending:
    enabled: true
    window_days: 7
    alert_on_increase: true

# Quality verification
quality:
  enabled: true
  
  grounding:
    method: lexical_overlap  # PoC simple
    threshold: 0.5
  
  coherence:
    enabled: true
    threshold: 0.7
  
  relevance:
    enabled: true
    threshold: 0.6
  
  overall_weights:
    grounding: 0.4
    coherence: 0.3
    relevance: 0.3

# Recommendations
recommendations:
  enabled: true
  templates_path: /etc/anchor/recommendation_templates/

# Performance
performance:
  parallel_analysis: true
  timeout_ms: 100
  cache_results: true
  cache_ttl: 1m

# Tracker integration
tracker:
  enabled: true
  nats_url: nats://nats:4222
  event_subject: bastion-rag.events.anchor

  alerts:
    bias_high: true
    drift_amplification: true
    drift_introduction: true
    quality_low: true

# Logging
logging:
  level: info
  format: json
```

---

## 10. Deployment

### 10.1 Same Service

Anchor remains single service handling both phases:

```yaml
anchor:
  image: bastion-rag/anchor:1.1.0
  ports:
    - "8080:8080"
    - "9090:9090"
  environment:
    - INPUT_CONFIG=/etc/anchor/input-config.yaml
    - OUTPUT_CONFIG=/etc/anchor/output-config.yaml
  resources:
    limits:
      memory: 2G
      cpu: '2'
```

---

## 11. Tracker Integration

### 11.1 New Events

```yaml
events:
  - anchor.response_verified              # Successful verification
  - anchor.bias_detected_response         # Bias in response
  - anchor.bias_amplification_detected    # Drift: amplification
  - anchor.bias_introduction_detected     # Drift: introduction
  - anchor.bias_reduction_detected        # Drift: reduction (good)
  - anchor.quality_low                    # Low quality response
  - anchor.semantic_drift_detected        # Embedding drift
  - anchor.trending_bias_increase         # Long-term trend
```

### 11.2 Visualization

```
Live Flow (with both Anchor phases):

[Anchor-IN] → [Storage]
   ↓
   Bias baseline established

Later, during retrieval:

[Anchor-IN] → applies noise

After LLM response:
[LLM] → [Anchor-OUT] → [Vault-OUT]
            ↓
       [Detail Panel]
       
       Phase 1 Bias:  0.05 ✅
       Phase 2 Bias:  0.28 🚨
       Drift:         +0.23 (amplification)
       
       Action: Alert + recommendation
```

### 11.3 Demo Dashboard

```
┌──────────────────────────────────────────┐
│ Anchor Bias Tracking                     │
├──────────────────────────────────────────┤
│                                          │
│ Phase 1 (Embedding) Bias:    [chart]    │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━ stable        │
│                                          │
│ Phase 2 (Response) Bias:     [chart]    │
│ ━━━━━━━━╱━━━━━━━━━━━━━━━━━━ increasing  │
│                                          │
│ Drift Detection:                         │
│ ●●○○○○○○○○ amplifications today        │
│                                          │
│ Top Categories:                          │
│ 1. Gender:     0.28 🚨                  │
│ 2. Ethnicity:  0.15 ⚠️                  │
│ 3. Age:        0.08 ✅                  │
│                                          │
│ Recommendation: Review LLM behavior      │
└──────────────────────────────────────────┘
```

---

## 12. Future Enhancements (Out of PoC)

### v1.1 Enhancements
- LLM-based bias judging
- More sophisticated WEAT variants
- Counterfactual analysis

### v2.0 Enhancements
- Causal bias detection
- Adversarial testing
- Multi-modal bias (images, audio)
- Real-time bias correction

### Why Not Now:
These advanced features require:
- Significant research depth
- Larger compute resources
- Specialized expertise
- Beyond PoC scope

The current SRS provides **conceptual completeness** for bidirectional bias awareness.

---

## 13. Appendix

### 13.1 Demo Walkthrough (2 minutes)

```
0:00-0:30  Show pipeline with both Anchor phases highlighted
0:30-1:00  Run query with gender-neutral source documents
1:00-1:30  Show LLM response containing gender stereotypes
1:30-2:00  Demonstrate Anchor-OUT detection:
           - Phase 1 baseline (low bias)
           - Phase 2 detection (high bias)
           - Drift visualization (amplification)
           - Recommendations
```

### 13.2 Common Patterns

**Pattern: Amplification (LLM bias)**
```
Embeddings:   0.05 (low bias)
Response:     0.28 (high bias)
Drift:        +0.23 amplification
Cause:        LLM's training data bias
Action:       Review LLM model
```

**Pattern: Introduction (New bias)**
```
Embeddings:   0.02 (near zero)
Response:     0.20 (medium bias)
Drift:        Introduction
Cause:        LLM generated content beyond source
Action:       Improve grounding
```

**Pattern: Reduction (Good)**
```
Embeddings:   0.15 (some bias)
Response:     0.08 (lower bias)
Drift:        -0.07 reduction
Cause:        LLM neutralizes source
Action:       None (positive case)
```

### 13.3 Roadmap

- v1.0: PoC (current)
- v1.1: Improved bias word lists
- v1.2: LLM-based judging
- v2.0: Causal analysis
- v2.1: Counterfactuals

### 13.4 Change History

| Version | Date | Changes |
|---|---|---|
| 0.1 | 2026-05-17 | Initial draft - bidirectional Anchor |
| 1.0 | 2026-05-17 | First release - Phase 2 specifications |

---

## 14. Summary

### What This SRS Adds

```
Before:
Anchor = Only handled embedding security at indexing time (Phase 1)
Response-side bias was invisible

After:
Anchor = Bidirectional bias awareness
- Phase 1: Embedding noise + bias baseline (existing)
- Phase 2: Response verification + drift detection (this SRS) ✅
```

### Key Design Decisions

1. **Bidirectional Service**: Single Anchor handles both phases
2. **Shared Resources**: Same embedding model, test sets, config
3. **PoC Simplification**: Simple methods, not research-grade
4. **Drift Focus**: Compare Phase 1 vs Phase 2 to find LLM-introduced issues
5. **Actionable Output**: Recommendations, not just detection
6. **Demo-friendly**: Clear visualization for PoC presentations

### Value Delivered

- ✅ Complete bidirectional pattern across all Bastion-RAG modules
- ✅ Bias awareness at both ends
- ✅ LLM behavior monitoring
- ✅ Quality verification
- ✅ Maintains PoC simplicity
- ✅ Foundation for future enhancements

### Note on Honest Limitations

This PoC implementation:
- Uses simple bias detection methods
- May have higher false positive/negative rates than production
- Is meant to **demonstrate the concept** of bidirectional bias awareness
- Should be expanded for real production use

The goal is showing that:
> "Bias and quality concerns matter in BOTH input embeddings AND output responses"

---

**End of Document**

---

## Note on Simplification

This SRS intentionally keeps Phase 2 simple, like Phase 1:
- Word/pattern-based detection (not advanced NLP)
- Simple drift detection (not statistical analysis)
- Basic quality metrics (not academic frameworks)

The complexity of real-world bias detection is a research topic. This PoC provides:
- Conceptual completeness (bidirectional)
- Working demonstration
- Foundation for enhancement

For production, consider:
- Advanced fairness libraries (e.g., Fairlearn, AIF360)
- LLM-based evaluation (LLM-as-judge)
- Continuous bias monitoring
- Domain-specific bias dimensions
