# Bastion — Modular RAG Requirements

**Project:** Bastion - RAG Security Governance Framework
**Document Type:** Cross-Cutting SRS (Tier 3)
**Document ID:** 31-modular-rag-requirements
**Feature:** Modular RAG Architecture
**Version:** 1.2
**Date:** 2026-05-29
**Status:** Active (Priority 1 + 2 + 3 implemented)

**Foundation References:**
- 01-architecture-principles (Progressive Enhancement model)
- 02-event-schema-standard (trace_id propagation)
- 03-module-interaction-map (module interfaces and hooks)

**Affected Modules:**
- Navigator (C) — primary: routing, query transformation, iterative loop
- Vault (B) — purpose-based access control, per-iteration anonymization
- Sentinel (A) — per-iteration validation, circuit breaker
- Tracker (D) — chunk-level lineage, query-result binding
- Anchor (E) — quality signal for re-search decisions

---

## 1. Introduction

### 1.1 Purpose

This document specifies the requirements for evolving Bastion's pipeline from a
**linear single-pass architecture** into a **Modular RAG architecture** that supports
adaptive routing, query transformation, iterative re-search, purpose-based data
governance, and enterprise-grade data integration.

The requirements are additive: every new capability is expressed as a progressive
enhancement over the existing pipeline. The existing linear path remains correct
and fully functional when no modular capabilities are configured.

### 1.2 Why Modular RAG

```
Current state (Bastion v0.1.0):

  User → [Sentinel-IN] → [Vault P1] → [Navigator] → [Anchor] → LLM
                                           ↑
                         Single pass. One query. One retrieval. Fixed path.

Problem:
  Complex enterprise queries are not single-pass problems.
  A question that spans multiple data domains, requires disambiguation,
  or involves PII across retrieval attempts cannot be handled correctly
  by a fixed linear pipeline.

Target state (Modular RAG):

  User → [Sentinel-IN] → [Vault P1] → [Router] → [Transformer] → [Retriever] ←
                                          │                            │ relevance
                                          │                         [Evaluator]
                                          │                            │ retry?
                                          └────────────────────────────┘
                                          (bounded loop, security on every iteration)
```

### 1.3 Bastion Invariant

All new capabilities in this document are **constrained** by the following invariant
that must not be violated:

```
BASTION INVARIANT:
  Every iteration of every search loop must pass through
  Sentinel-IN and Vault Phase 1 independently.

  No module may bypass, cache, or short-circuit security
  checks on behalf of a retry or re-search.

  Rationale: A crafted query may be designed to look safe on
  iteration 1 and inject on iteration 2 after the path is
  established. Each iteration is an independent security surface.
```

### 1.4 Scope

**In Scope:**
- 🟢 MR-01: Adaptive Query Routing
- 🟢 MR-02: Query Transformation (rewriting, HyDE, sub-query decomposition)
- 🟢 MR-03: Iterative Re-search Loop with circuit breaker
- 🟡 MR-04: Purpose-Based Access Control (separation of authority)
- 🟡 MR-05: Chunk-Level Data Lineage and query-result binding
- 🔴 MR-06: Enterprise Data Integration (connectors, CDC, data freshness)

**Out of Scope:**
- LLM provider management (handled by Vault Cloud LLM Connector, doc 21)
- Prompt template management
- Model fine-tuning

### 1.5 Definitions

| Term | Definition |
|---|---|
| **Modular RAG** | Pipeline architecture where retrieval components (routing, transformation, search, evaluation) are independently composable and replaceable |
| **Query routing** | Selecting the retrieval strategy and target collection(s) based on query intent, not only user permissions |
| **Query transformation** | Converting the raw (anonymized) query into a form better suited for embedding or retrieval |
| **HyDE** | Hypothetical Document Embeddings — generate a plausible answer, embed that instead of the question |
| **Sub-query decomposition** | Splitting a multi-hop question into parallel independent sub-searches |
| **Iterative re-search** | Retrieval loop that evaluates result quality and re-searches with a modified query if below threshold |
| **Circuit breaker** | Hard iteration cap that terminates a loop unconditionally, regardless of result quality |
| **Purpose limitation** | Access to data granted only when the stated access purpose matches the document's permitted purposes |
| **Data steward** | Role that owns and controls access authorisation for a specific data domain |
| **CDC** | Change Data Capture — detecting and propagating source document updates into the index |
| **Chunk provenance** | The lineage trail from a retrieved chunk back to its parent document and original source |

---

## 2. Overall Architecture

### 2.1 Modular RAG Pipeline

```
┌────────────────────────────────────────────────────────────────────────┐
│  INPUT PATH                                                            │
│                                                                        │
│  User Query                                                            │
│      │                                                                 │
│      ▼                                                                 │
│  [Sentinel-IN]  ← validates every iteration independently             │
│      │                                                                 │
│      ▼                                                                 │
│  [Vault Phase 1]  ← anonymizes every iteration independently          │
│      │                                                                 │
│      ▼                                                                 │
│  ┌──────────────────────────────────────────────────────────┐         │
│  │  Navigator — Modular Search Orchestrator                 │         │
│  │                                                          │         │
│  │  ┌────────────┐   ┌──────────────┐   ┌───────────────┐  │         │
│  │  │   Router   │──▶│ Transformer  │──▶│   Retriever   │  │         │
│  │  │ (MR-01)   │   │  (MR-02)    │   │  (existing)   │  │         │
│  │  └────────────┘   └──────────────┘   └───────┬───────┘  │         │
│  │                                              │           │         │
│  │                                      ┌───────▼───────┐  │         │
│  │                                      │   Evaluator   │  │         │
│  │                                      │   (MR-03)    │  │         │
│  │                                      └───────┬───────┘  │         │
│  │                                              │           │         │
│  │              sufficient? ◀───────────────────┘           │         │
│  │                  │ no: rewrite + loop (bounded)           │         │
│  │                  │ yes: pass to Anchor                    │         │
│  └──────────────────┼───────────────────────────────────────┘         │
│                     │                                                  │
│                     ▼                                                  │
│  [Anchor-IN]  → LLM                                                   │
└────────────────────────────────────────────────────────────────────────┘

Observer: Tracker receives chunk-level lineage events on every iteration (MR-05)
Policy:   Vault enforces purpose-based access control per request (MR-04)
```

### 2.2 Relationship to Existing Pipeline

```
Existing (linear):             Modular RAG extension:

Sentinel → Vault → Navigator   Sentinel → Vault → [Router → Transformer →
→ Anchor → LLM                  Retriever → Evaluator] → Anchor → LLM

The bracket [ ] is the Modular Search Orchestrator.
It replaces the single Navigator.search() call.
The surrounding pipeline (Sentinel, Vault, Anchor, Tracker) is unchanged.
```

### 2.3 Progressive Enhancement Classification

```
🟢 CORE (MR-01, MR-02, MR-03):
   Navigator internal — no other module's interface changes.
   Deployable without any cross-module coordination.

🟡 ENHANCED (MR-04, MR-05):
   Requires Vault (purpose tagging) and Tracker (chunk lineage).
   Cross-module coordination at the interface level.

🔴 HOOKS (MR-06):
   Enterprise connectors, CDC.
   Infrastructure dependency (source systems).
   Optional; pipeline functions without it.
```

---

## 3. MR-01 — Adaptive Query Routing

### 3.1 Problem

The current orchestrator selects collections by mapping user permission categories
to collection names via a hardcoded dict. The query content is not considered.

```python
# Current: static permission-to-collection mapping
_CATEGORY_TO_COLLECTION = {
    "customer_data":       "customer_docs",
    "manufacturing_data":  "manufacturing_docs",
    "hr_data":             "hr_docs",
}
```

Consequences:
- A query about "annual leave policy" from a user with `customer_data` permission
  searches `customer_docs` even though `hr_docs` is the correct collection.
- No mechanism to select retrieval strategy (vector-only vs. hybrid vs. hybrid+rerank)
  based on query characteristics.
- A low-confidence ambiguous query searches as broadly as a high-confidence factual one.

### 3.2 Requirements

**FR-MR-01-001: Intent Classification**
```
The router MUST classify each query into one of the following intent types
before collection selection:

  factual      — specific lookup with expected direct answer
                 ("What is the defect rate on Line 7?")
  analytical   — aggregation or comparison across multiple records
                 ("Compare Q3 vs Q4 defect trends")
  procedural   — how-to or policy lookup
                 ("What is the leave application procedure?")
  multi_hop    — answer requires chaining across multiple documents
                 ("What policy applies to employees who joined before 2020 with grade B?")
  ambiguous    — insufficient signal to classify with confidence

Classification method: lightweight keyword + embedding cosine distance to
intent exemplars. Must complete in < 5ms p95 (no LLM call).
```

**FR-MR-01-002: Strategy Selection**
```
The router MUST select a retrieval strategy based on intent:

  intent        → strategy
  ─────────────────────────────────────────────────
  factual       → vector_only        (fast, precise)
  analytical    → hybrid + rerank    (broad recall)
  procedural    → hybrid             (keyword-rich)
  multi_hop     → decompose → parallel sub-searches (MR-02-003)
  ambiguous     → hybrid, low min_score, log warning

Strategy override via SearchOptions.strategy is permitted (caller explicit).
```

**FR-MR-01-003: Domain-Aware Collection Selection**
```
In addition to permission filtering, the router MUST score each available
collection for relevance to the query using collection-level topic embeddings
(pre-computed at index time, stored in collection metadata).

Collections with affinity score < routing_threshold (default: 0.25) are
excluded from the search, even if the user has permission to access them.

This prevents vocabulary leakage across domains under a single tenant.

routing_threshold is configurable per tenant.
```

**FR-MR-01-004: Routing Audit Event**
```
Every routing decision MUST emit a Tracker event:

  event_type:  navigator.query.routed
  fields:
    intent:         <classified intent>
    strategy:       <selected strategy>
    collections:    [<list of selected collections>]
    excluded:       [<list of excluded collections with reason>]
    confidence:     <float 0–1>
    routing_ms:     <latency>
```

### 3.3 Failure Mode

```
If intent classification fails or routing_threshold excludes all collections:
  → Fall back to current behaviour (permission-based collection selection)
  → Emit routing event with intent="fallback", reason=<error>
  → Do NOT return an error to the caller (fail-open for search, fail-closed for security)
```

---

## 4. MR-02 — Query Transformation

### 4.1 Problem

After Vault Phase 1 anonymization, the query contains structured tokens
(`KR_NAME_8f3d2a`, `EMAIL_c3a91f`) in place of real PII values. These tokens
carry near-zero semantic signal for the embedding model.

```
User types:    "Find Hong Gildong's purchase history, email hong@naver.com"
Vault output:  "Find KR_NAME_8f3d2a's purchase history, email EMAIL_c3a91f"
After MR-02:   "Find Logan Anderson_8f3d2a's purchase history,
                email minjun.kim@example.com_c3a91f"
BGE-M3 embeds: a query about a named person contacting via email —
               full semantic signal, zero real PII
```

Additionally, short or ambiguous queries retrieve poorly against long document
chunks without query expansion or hypothetical document generation.

### 4.2 Requirements

**FR-MR-02-001: Faker Token Rewriting**
```
The transformer MUST rewrite Vault PII tokens into realistic pseudonyms
before the text is passed to the embedder AND to the LLM prompt.
The hex suffix from the original Vault token MUST be preserved on every
rewritten value — it is the only key Vault Phase 2 can use to reverse
the substitution in the LLM response.

Why the hex suffix is required:
  The rewritten value appears in the LLM response. Vault Phase 2 scans
  the response for <value>_<hex> patterns, extracts the hex, and looks
  up the original in the token store. Without the suffix there is no
  reversible link.

  박민준 → [Vault P1] → KR_NAME_4d9e1b
                ↓ MR-02
           Logan_4d9e1b        ← in LLM prompt AND embedding
  LLM: "Logan_4d9e1b의 잔액은 ₩5,000,000입니다"
                ↓ Vault Phase 2
  "_4d9e1b" → look up KR_NAME_4d9e1b → 박민준
  User: "박민준의 잔액은 ₩5,000,000입니다"

────────────────────────────────────────────────────────────
TOKEN REWRITING TABLE
────────────────────────────────────────────────────────────

Vault token      Rewritten form              Approach
─────────────────────────────────────────────────────────────────────────
KR_NAME_<hex>  → <EN first> <EN last>_<hex>  Cross-language name
                 e.g. Logan Anderson_4d9e1b
                 Source: en_given_names.csv (230) × en_surnames.csv (100)
                 = 23,000 combinations
                 Why cross-language: prevents collision with real Korean
                 names in the system. A Korean user named 유준희 entering
                 a query would be tokenized by Vault to KR_NAME_<hex2>,
                 then rewritten to an English name — never generating
                 유준희 as a fake value for someone else.

EMAIL_<hex>    → <kr_given>.<kr_sur>@example.com_<hex>
                 e.g. minjun.kim@example.com_c3a91f
                 Source: kr_given_names.csv + kr_surnames.csv (romanized)
                 Why @example.com: RFC 2606 / RFC 2606bis reserves
                 example.com, example.net, example.org for documentation
                 and testing. These domains cannot exist in real email.
                 Sentinel PII detection skips *@example.com by policy.

MOBILE_<hex>   → 010-0000-<4digits>_<hex>
                 e.g. 010-0000-4821_b7e4d1
                 Why 010-0000-: the 0000 block is not assigned by Korean
                 carriers (MSISDN ranges 010-1000 through 010-9999 are
                 issued; 0000 is reserved). Cannot match a real number.

RRN_TOKEN_<hex>→ [ID_NUMBER]                 Generic label — no faker
                 Reason: a structurally valid fake RRN (YYMMDD-N123456)
                 could match a real person's national ID. The embedding
                 gain is also negligible — RRN is a lookup key, not a
                 semantic search term.

EMP_<hex>      → [EMPLOYEE_ID]              Generic label — no faker
WRK_<hex>      → [WORKER_ID]               Generic label — no faker
                 Reason: internal IDs carry no semantic value for the
                 embedding model. A fake employee ID does not improve
                 retrieval quality.

EMAIL_honey_<hex> → [EMAIL]                 Preserve class, hide honey signal
                 Vault honey-token detection happens before MR-02.
                 After detection the honey token is rewritten to the
                 same generic label as a normal email.

────────────────────────────────────────────────────────────
GENERATION ALGORITHM (Navigator, Python)
────────────────────────────────────────────────────────────

  # Load once at startup from pii-pattern-engine/datas/
  EN_FIRST = load_csv("en_given_names.csv")   # 230 names
  EN_LAST  = load_csv("en_surnames.csv")      # 100 names
  KR_GIVEN = load_csv("kr_given_names.csv")   # 1,051 names
  KR_SUR   = load_csv("kr_surnames.csv")      # 93 names

  def rewrite(token: str) -> str:
      prefix, hex_part = token.rsplit("_", 1)

      if prefix == "KR_NAME":
          seed  = int(hex_part, 16)
          first = EN_FIRST[seed % len(EN_FIRST)]
          last  = EN_LAST[(seed // len(EN_FIRST)) % len(EN_LAST)]
          return f"{first} {last}_{hex_part}"

      if prefix == "EMAIL":
          seed  = int(hex_part, 16)
          given = KR_GIVEN[seed % len(KR_GIVEN)]          # romanized below
          sur   = KR_SUR[(seed // len(KR_GIVEN)) % len(KR_SUR)]
          local = f"{_romanize(given)}.{_romanize(sur)}"
          return f"{local}@example.com_{hex_part}"

      if prefix == "MOBILE":
          seed   = int(hex_part, 16)
          digits = f"{seed % 10000:04d}"
          return f"010-0000-{digits}_{hex_part}"

      if prefix == "RRN_TOKEN":   return "[ID_NUMBER]"
      if prefix in ("EMP", "WRK"): return f"[{prefix}_ID]"
      if prefix == "EMAIL_honey":  return "[EMAIL]"

      return f"[{prefix}]"   # fallback for unknown types

────────────────────────────────────────────────────────────
DETERMINISM AND UNIQUENESS
────────────────────────────────────────────────────────────

  The hex suffix is the HMAC of the original value (Vault).
  Same original → same hex → same faker value every time.

  Two different people may produce the same faker prefix but
  always have different hex suffixes:
    박민준 → Logan Anderson_4d9e1b
    이민준 → Logan Anderson_8f3d2a   ← same name, different hex

  Vault Phase 2 uses only the hex suffix for lookup.
  The faker prefix is purely for LLM / embedding readability.

────────────────────────────────────────────────────────────
SCOPE
────────────────────────────────────────────────────────────

  Vault token store: never modified.
    KR_NAME_4d9e1b → 박민준 mapping is unchanged throughout.

  Rewriting happens: Navigator orchestrator, MR-02 transformer step.
    NOT in Vault. The faker form is an optimisation artifact,
    not a canonical anonymisation record.

  The rewritten text is used for:
    - Embedding (BGE-M3 query vector)
    - LLM prompt input
  It is NOT used for:
    - Vault token store entries
    - Audit log canonical form
    - Any value returned directly to the caller
```

**FR-MR-02-002: HyDE (Hypothetical Document Embeddings)**
```
When intent = factual or procedural AND the query is short (< 12 words):
  1. Generate a hypothetical short answer using a local template or LLM call
  2. Embed the hypothetical answer instead of the question
  3. Use the hypothetical embedding for vector search
  4. Use the original query (rewritten) for BM25 sparse search

The hypothetical generation step MUST be bounded:
  - If using a local LLM: timeout 2s, fallback to original query on failure
  - The generated hypothetical is NEVER shown to the caller
  - The generated hypothetical MUST be passed through Vault Phase 1 to catch
    any PII the LLM may have hallucinated into it

HyDE is disabled by default. Enabled via SearchOptions.use_hyde = true.
```

**FR-MR-02-003: Sub-Query Decomposition**
```
When intent = multi_hop:
  1. Decompose the query into N independent sub-queries (N ≤ 4)
  2. Execute sub-queries in parallel, each as a full Navigator search
  3. Each sub-query independently passes through Sentinel-IN and Vault Phase 1
  4. Merge results using RRF across all sub-query result sets
  5. Apply reranking on the merged set

Decomposition method: rule-based splitting at conjunctions and temporal/
conditional clauses. No LLM call required for decomposition.

Security constraint: each sub-query is an independent security surface
(Bastion Invariant, §1.3).

Sub-query count limit (N ≤ 4) is a circuit breaker against decomposition
explosion from adversarial inputs.
```

**FR-MR-02-004: Transformation Audit Event**
```
Every transformation MUST emit:

  event_type:  navigator.query.transformed
  fields:
    transformation_type:  [token_rewrite | hyde | decompose | none]
    original_length:      <char count before>
    transformed_length:   <char count after>
    sub_query_count:      <int, only for decompose>
    transformation_ms:    <latency>

The original query content and the hypothetical text are NEVER included
in the event payload.
```

---

## 5. MR-03 — Iterative Re-search Loop

### 5.1 Problem

If the top search results score below the retrieval quality threshold, the current
pipeline returns whatever it found. The LLM then generates a response grounded in
low-quality or irrelevant context — a controlled form of hallucination.

There is no feedback path from retrieval quality to query refinement.

### 5.2 Requirements

**FR-MR-03-001: Retrieval Quality Evaluator**
```
After each retrieval pass, the evaluator MUST score result quality:

  Quality signals (all computed locally, no LLM):
    - top_score:      score of the best result
    - score_gap:      difference between rank-1 and rank-2 scores
    - coverage:       fraction of query keywords present in top-K results
    - diversity:      heading_path diversity across top-K chunks

  Quality verdict:
    sufficient   → top_score ≥ quality_threshold (default: 0.60)
                   AND coverage ≥ 0.40
    insufficient → below either threshold
    uncertain    → top_score in [0.45, 0.60) — retry once only

Thresholds are configurable per tenant and per collection.
```

**FR-MR-03-002: Query Refinement on Insufficient Results**
```
On insufficient verdict:
  1. Apply query refinement strategy based on failure mode:

     low top_score     → broaden: remove specific tokens, generalise
     low coverage      → keyword expansion: add synonyms for uncovered terms
     low diversity     → diversify: add "different aspect" instruction to query

  2. Refined query passes through Sentinel-IN and Vault Phase 1 independently
     (Bastion Invariant, §1.3)

  3. Execute retrieval with refined query

Refinement is rule-based. No LLM call. Refinement must complete in < 10ms.
```

**FR-MR-03-003: Circuit Breaker**
```
The loop MUST terminate unconditionally when:
  - Iteration count reaches max_iterations (default: 3, configurable 1–5)
  - Total loop elapsed time exceeds loop_timeout_ms (default: 500ms)
  - Two consecutive iterations produce identical result sets (detected by
    comparing sorted document_id lists)
  - A refined query is classified as a potential injection by Sentinel-IN

On circuit breaker activation:
  - Return best results collected across all iterations (union, re-ranked)
  - Emit loop.terminated event with termination reason
  - Do NOT surface the circuit breaker reason to the caller
```

**FR-MR-03-004: Loop Audit Events**
```
Each iteration MUST emit:

  event_type:  navigator.search.iteration
  fields:
    iteration:         <int, 1-based>
    verdict:           <sufficient | insufficient | uncertain>
    top_score:         <float>
    coverage:          <float>
    refinement:        <strategy applied, or null>
    iteration_ms:      <latency>

On loop completion:

  event_type:  navigator.search.loop_completed
  fields:
    total_iterations:  <int>
    termination:       <quality_met | max_iterations | timeout | duplicate | injection_detected>
    final_result_count: <int>
    total_ms:          <latency>
```

### 5.3 Interaction with MR-02

```
Iteration 1:   original query → transformer → retriever → evaluator
                                                               │ insufficient
Iteration 2:   refined query → [Sentinel] → [Vault] → transformer → retriever → evaluator
                                                               │ insufficient
Iteration 3:   refined query → [Sentinel] → [Vault] → transformer → retriever → evaluator
                                                               │ circuit breaker (max_iterations)
                                                         return best union
```

---

## 6. MR-04 — Purpose-Based Access Control

### 6.1 Problem

The current RBAC model in Vault answers: *"Can this role in this department access
data at this sensitivity level?"*

It does not answer: *"Is this the purpose for which this data was collected, and has
the data subject's consent covered this use?"*

This gap matters for GDPR Article 5(1)(b) (purpose limitation), PIPA §3 (collection
minimisation), and enterprise data governance frameworks where the same dataset may
be legitimately accessed for customer support but not for marketing analytics.

### 6.2 Requirements

**FR-MR-04-001: Document Purpose Tagging**
```
Each document indexed into Navigator MUST carry a permitted_purposes field
in its Qdrant payload alongside existing metadata:

  permitted_purposes: ["customer_support", "audit", "hr_analytics"]

Purpose tags are set at index time by the data steward for that document's
domain. They are immutable after indexing (change requires re-indexing).

Built-in purposes (extensible):
  customer_support    — respond to customer queries
  audit               — regulatory and internal audit
  hr_analytics        — aggregate workforce analysis (no individual lookup)
  product_development — inform product decisions
  legal               — legal proceedings and compliance
  training_data       — model training (requires explicit opt-in)
```

**FR-MR-04-002: Request Purpose Declaration**
```
SearchRequest MUST be extended with an optional purpose field:

  purpose: str  (default: "customer_support" if unset)

Navigator MUST filter retrieved results to only those whose
permitted_purposes includes the declared purpose, BEFORE
permission filtering (MR-04-003).

Purpose filtering happens in the Retriever, not post-hoc.
It is a pre-filter equivalent to tenant pre-filter (doc 21 §4).
```

**FR-MR-04-003: Purpose × RBAC Conjunction**
```
Access is granted only when BOTH conditions hold:

  (1) RBAC: user role + department allows access_level of document  [existing]
  (2) Purpose: declared purpose ∈ document.permitted_purposes       [new]

Neither alone is sufficient. A data steward may grant a user HR admin
access level yet restrict the purpose to "audit" only — preventing the
same user from running "hr_analytics" queries.

This conjunction is evaluated in Vault's access controller (OPA policy
or built-in RBAC extension). Navigator enforces it as a pre-filter.
```

**FR-MR-04-004: Data Steward Role**
```
A new role "data_steward" MUST be defined in the RBAC model:

  data_steward:
    - may set and update permitted_purposes on documents in their domain
    - may grant purpose-specific access exceptions to specific users
    - may NOT access the document content directly (separation of authority)
    - audit events for all steward actions are emitted to Tracker

A data steward for "hr_docs" domain cannot act as steward for "financial_docs".
Domain assignment is set in Vault configuration per tenant.
```

**FR-MR-04-005: Purpose Audit Event**
```
When a result is excluded due to purpose mismatch:

  event_type:  vault.purpose_filtered
  fields:
    document_id:       <string>
    declared_purpose:  <string>
    permitted_purposes: [<list>]
    tenant_id:         <string>
    trace_id:          <string>

When purpose = "training_data" is declared, a CRITICAL alert is raised
regardless of whether opt-in consent exists (human review required).
```

---

## 7. MR-05 — Chunk-Level Data Lineage

### 7.1 Problem

The existing lineage SRS (doc 22) tracks events at the document level. With semantic
chunking now implemented, retrieval operates at the chunk level — but Tracker receives
only `search.completed` with document IDs.

Two gaps result:

1. **Chunk provenance gap:** Which chunk of a document was retrieved? A document
   may have 12 chunks; only chunk 3 may be relevant. Lineage should record this.

2. **Query-result binding gap:** Which retrieved chunk contributed to which part of
   the LLM's response? In regulated industries (GDPR Right of Explanation, financial
   audit), this binding is required for explainability.

### 7.2 Requirements

**FR-MR-05-001: Chunk Retrieved Event**
```
Navigator MUST emit a chunk-level retrieval event for every chunk that
passes permission and purpose filtering and enters the result set:

  event_type:  navigator.chunk.retrieved
  fields:
    chunk_id:           <string>  — e.g. "doc-302_0003"
    parent_document_id: <string>  — e.g. "doc-302"
    chunk_index:        <int>
    heading_path:       <string>  — "# Report > ## Line 7 Analysis"
    collection:         <string>
    score:              <float>
    contains_table:     <bool>
    contains_link:      <bool>
    tenant_id:          <string>
    trace_id:           <string>
    iteration:          <int>     — which re-search iteration produced this
```

**FR-MR-05-002: Source Attribution in SearchResult**
```
SearchResult MUST be extended with provenance fields:

  chunk_id:      str     — chunk identifier
  heading_path:  str     — ancestor heading breadcrumb
  char_start:    int     — byte offset in parent document
  char_end:      int     — byte offset in parent document
  last_indexed:  str     — ISO-8601 timestamp of last index operation

These fields flow through the full pipeline to Sentinel-OUT, which can use
them to verify that LLM-stated facts appear at the attributed source location.
```

**FR-MR-05-003: Query-Result Binding**
```
Sentinel-OUT MUST receive the retrieved chunk list alongside the LLM response
for hallucination verification (this is the existing "context provision"
Enhanced composition — doc 12 §2.3).

Tracker MUST correlate:
  trace_id → [retrieved chunks] → LLM response

and store this binding in the lineage graph so that:

  GET /v1/lineage/{trace_id}/sources
  → returns: which chunk_ids were retrieved and used for this response

This endpoint does not exist yet and is required.
```

**FR-MR-05-004: Document Staleness Tracking**
```
Each indexed chunk carries a last_indexed timestamp.

Sentinel-OUT MUST check:
  if (now - chunk.last_indexed) > staleness_threshold (default: 7 days):
    emit event_type: navigator.chunk.stale
    include a staleness warning in SearchResult.metadata

Staleness threshold is configurable per tenant and per collection category.

A stale chunk is still returned (fail-open) but flagged. The LLM prompt
template may use this flag to qualify the response ("based on data indexed
as of <date>").
```

---

## 8. MR-06 — Enterprise Data Integration

### 8.1 Problem

The current ingestion path is manual: `navigator-cli index --file docs.jsonl`. There
is no mechanism to keep the index current as source documents change, nor to integrate
with enterprise content systems.

### 8.2 Requirements

**FR-MR-06-001: Source Connector Interface**
```
Navigator MUST define a SourceConnector interface:

  class SourceConnector:
    def list_documents(since: datetime) -> Iterator[Document]
    def get_document(id: str) -> Document
    def document_updated_at(id: str) -> datetime

Built-in connectors (implementation order):
  1. JSONL file (exists — navigator-cli index)
  2. Directory watcher (fs events → re-index changed files)
  3. REST pull (periodic GET to a configured endpoint)
  4. [Future] SharePoint, Confluence, S3 (plugin interface)

All connectors feed into the same indexing pipeline:
  Document → Chunker → Embedder → Qdrant upsert
```

**FR-MR-06-002: Change Detection and Delta Indexing**
```
The indexer MUST detect document changes and re-index only changed documents:

  Change detection methods (in priority order):
    1. Explicit CDC event (source system pushes change notification)
    2. Content hash comparison (SHA-256 of document content vs. stored hash)
    3. Last-modified timestamp comparison

On change detected:
  1. Delete all existing chunks for the document from Qdrant
     (by parent_document_id filter)
  2. Re-chunk the new document content
  3. Embed and upsert new chunks
  4. Emit navigator.document.reindexed event with:
       old_chunk_count, new_chunk_count, changed_sections (heading paths)

Deletion of old chunks MUST happen before insertion of new chunks
to prevent mixed-version retrieval during the transition.
```

**FR-MR-06-003: Document Content Hash Storage**
```
The content hash of each indexed document MUST be stored in Qdrant
collection metadata alongside the chunk payload:

  Payload fields added:
    content_hash:   <SHA-256 of document.content at index time>
    last_indexed:   <ISO-8601 timestamp>
    source_version: <version string from source system, if available>
```

**FR-MR-06-004: Schema-Aware Chunking Profiles**
```
ChunkerConfig MUST support named profiles for different document types:

  Profile         max_chars  overlap  Strategy
  ──────────────────────────────────────────────────────────────────
  markdown        1200       120      Heading-boundary, table-atomic
  plain_text      800        80       Sentence-boundary (NLTK)
  structured_csv  0          0        One row per chunk (no splitting)
  json_record     0          0        One top-level object per chunk
  html            1200       120      Tag-boundary (strip tags first)

Profile selection is automatic based on document MIME type or file extension,
overridable per connector configuration.
```

---

## 9. Security Constraints Summary

Every capability in this document is subject to the following constraints,
derived from the Bastion Invariant (§1.3) and the existing security architecture:

| # | Constraint | Applies to |
|---|---|---|
| SC-01 | Every loop iteration re-enters Sentinel-IN independently | MR-03 |
| SC-02 | Every loop iteration re-enters Vault Phase 1 independently | MR-02, MR-03 |
| SC-03 | Hypothetical text (HyDE) passes through Vault Phase 1 before embedding | MR-02 |
| SC-04 | Each sub-query in decomposition is an independent security surface | MR-02 |
| SC-05 | Circuit breaker terminates on injection detection, not only on timeout | MR-03 |
| SC-06 | Purpose filter is a pre-filter (before access), not post-filter | MR-04 |
| SC-07 | Chunk content is never included in lineage event payloads | MR-05 |
| SC-08 | Stale chunk flag does not suppress the result — fail-open for availability | MR-05 |
| SC-09 | Source connector credentials are stored in Vault KMS, not in config | MR-06 |
| SC-10 | Delta index deletion is atomic with insertion (no mixed-version window > 0) | MR-06 |

---

## 10. Non-Functional Requirements

| ID | Requirement | Target |
|---|---|---|
| NFR-MR-01 | Query routing (intent classification) | < 5 ms p95 |
| NFR-MR-02 | Token-aware rewrite | < 1 ms p95 |
| NFR-MR-03 | Full loop (3 iterations, hybrid+rerank) | < 1500 ms p95 |
| NFR-MR-04 | Single iteration latency budget | < 400 ms p95 |
| NFR-MR-05 | Purpose filter evaluation | < 2 ms p95 |
| NFR-MR-06 | Chunk retrieved event emission | < 1 ms p95 (async, non-blocking) |
| NFR-MR-07 | Delta index (per document) | < 5 s p95 |
| NFR-MR-08 | Sub-query decomposition (N=4 parallel) | < 800 ms p95 |

---

## 11. Module Impact Matrix

| Module | MR-01 | MR-02 | MR-03 | MR-04 | MR-05 | MR-06 |
|---|---|---|---|---|---|---|
| Navigator (C) | **Primary** — Router, intent classifier | **Primary** — Transformer, HyDE, decomposition | **Primary** — Evaluator, circuit breaker | Filter integration | Chunk event emission, staleness flag | Connector interface, delta indexer |
| Vault (B) | — | Re-enter on each sub-query and iteration | Re-enter on each iteration | **Primary** — Purpose tagging, steward role, OPA extension | purpose_filtered event | Connector credential storage |
| Sentinel (A) | — | Validate HyDE hypothetical | Re-enter + injection → terminate | — | Receive chunk provenance for hallucination check | — |
| Tracker (D) | Receive routing events | Receive transformation events | Receive iteration events | Receive purpose filter events | **Primary** — chunk lineage graph, binding endpoint | Receive reindex events |
| Anchor (E) | — | — | Quality signal consumer (score threshold input) | — | — | — |

---

## 12. Implementation Priority

```
Priority 1 — ✅ COMPLETE (2026-05-29)

  FR-MR-02-001  Token-aware query rewriting          ✅ navigator/token_rewriter.py
  FR-MR-05-001  Chunk retrieved event                ✅ navigator/events.py + orchestrator
  FR-MR-05-002  Source attribution in SearchResult   ✅ models.py + searcher.py + orchestrator
  FR-MR-03-001  Retrieval quality evaluator          ✅ navigator/evaluator.py

Priority 2 — ✅ COMPLETE (2026-05-29)

  FR-MR-01-001  Intent classification                ✅ navigator/router.py
  FR-MR-01-002  Strategy selection                   ✅ navigator/router.py
  FR-MR-01-003  Domain-aware collection selection    ✅ navigator/router.py (keyword proxy)
  FR-MR-01-004  Routing audit event                  ✅ navigator/events.py (event_query_routed)
  FR-MR-03-002  Query refinement on insufficient     ✅ navigator/evaluator.py (refine_query)
  FR-MR-03-003  Circuit breaker                      ✅ navigator/orchestrator.py (loop)
  FR-MR-03-004  Loop audit events                    ✅ navigator/events.py (event_search_iteration,
                                                        event_loop_completed)
  FR-MR-05-003  Query-result binding (Tracker)       🔲 Pending — Tracker endpoint not yet built

Priority 3 — ✅ COMPLETE (2026-05-29)

  FR-MR-04-001  Document purpose tagging        ✅ IndexRequest.permitted_purposes + Qdrant payload
  FR-MR-04-002  Request purpose declaration     ✅ SearchRequest.purpose + purpose pre-filter
  FR-MR-04-003  Purpose × RBAC conjunction      ✅ Vault access/controller.go:DecideWithPurpose()
  FR-MR-02-002  HyDE                            ✅ navigator/hyde.py + orchestrator wiring
  FR-MR-05-003  Query-result binding (Tracker)  ✅ GET /v1/lineage/{trace_id}/sources
  FR-MR-05-004  Staleness tracking              ✅ _check_staleness() + event_chunk_stale

Priority 4 — 🔲 PENDING (infrastructure dependency)

  FR-MR-06-001  Source connector interface
  FR-MR-06-002  Delta indexing
  FR-MR-02-003  Sub-query decomposition
  FR-MR-04-004  Data steward role
```

---

## 13. Implementation Status

| FR | Title | Status | Location |
|---|---|---|---|
| FR-MR-01-001 | Intent classification | ✅ | `navigator/navigator/router.py` · `Router._classify()` |
| FR-MR-01-002 | Strategy selection | ✅ | `navigator/navigator/router.py` · `_INTENT_STRATEGY` |
| FR-MR-01-003 | Domain-aware collection selection | ✅ | `router.py` · `Router._select_collections()` (keyword proxy) |
| FR-MR-01-004 | Routing audit event | ✅ | `events.py` · `event_query_routed()` |
| FR-MR-02-001 | Faker token rewriting | ✅ | `navigator/navigator/token_rewriter.py` · `TokenRewriter` |
| FR-MR-02-002 | HyDE | ✅ | `navigator/navigator/hyde.py` · `HyDETransformer`; wired in `orchestrator.py` |
| FR-MR-02-003 | Sub-query decomposition | 🔲 | — |
| FR-MR-02-004 | Transformation audit event | ✅ | Covered by `event_query_routed` (routing step) |
| FR-MR-03-001 | Retrieval quality evaluator | ✅ | `navigator/navigator/evaluator.py` · `Evaluator.evaluate()` |
| FR-MR-03-002 | Query refinement | ✅ | `evaluator.py` · `Evaluator.refine_query()` |
| FR-MR-03-003 | Circuit breaker | ✅ | `orchestrator.py` · `search()` loop (max_iter / timeout / duplicate) |
| FR-MR-03-004 | Loop audit events | ✅ | `events.py` · `event_search_iteration()`, `event_loop_completed()` |
| FR-MR-04-001 | Document purpose tagging | ✅ | `models.py` · `IndexRequest.permitted_purposes`; stored in Qdrant payload |
| FR-MR-04-002 | Request purpose declaration | ✅ | `models.py` · `SearchRequest.purpose`; `orchestrator.py` · `_filter_by_purpose()` |
| FR-MR-04-003 | Purpose × RBAC conjunction | ✅ | `vault/internal/access/controller.go` · `DecideWithPurpose()`, `purposeMatrix` |
| FR-MR-04-004 | Data steward role | 🔲 | — |
| FR-MR-04-005 | Purpose audit event | ✅ | `events.py` · `event_purpose_filtered()` |
| FR-MR-05-001 | Chunk retrieved event | ✅ | `events.py` · `event_chunk_retrieved()`; emitted per result in `orchestrator.py` |
| FR-MR-05-002 | Source attribution in SearchResult | ✅ | `models.py` · `SearchResult` (chunk_id, heading_path, char_start, char_end, last_indexed); `searcher.py` · `_to_search_result()` |
| FR-MR-05-003 | Query-result binding (Tracker endpoint) | ✅ | `tracker/internal/store/memory.go` · `AddChunkLineage/GetLineageSources`; `GET /v1/lineage/{trace_id}/sources` |
| FR-MR-05-004 | Document staleness tracking | ✅ | `orchestrator.py` · `_check_staleness()`; `events.py` · `event_chunk_stale()` |
| FR-MR-06-001 | Source connector interface | 🔲 | — |
| FR-MR-06-002 | Delta indexing / CDC | 🔲 | — |
| FR-MR-06-003 | Document content hash storage | 🔲 | — |
| FR-MR-06-004 | Schema-aware chunking profiles | 🔲 | — |

**Note on FR-MR-01-003:** The SRS specifies collection-level topic embeddings pre-computed at index time. The current implementation uses keyword-based domain affinity scoring as a practical proxy. The embedding-based version (requiring Qdrant collection metadata for topic vectors) is deferred to the next iteration.

---

## 14. Change History

| Version | Date | Changes |
|---|---|---|
| 1.0 | 2026-05-28 | Initial draft — 6 capability areas, 24 functional requirements |
| 1.1 | 2026-05-29 | Priority 1 + Priority 2 implemented; §12 updated with completion status; §13 Implementation Status table added; status changed to Active |
| 1.2 | 2026-05-29 | Priority 3 implemented: FR-MR-04-001/002/003 (purpose tagging + filter + Vault conjunction), FR-MR-02-002 (HyDE), FR-MR-05-003 (Tracker lineage sources endpoint), FR-MR-05-004 (staleness tracking) |

---

**End of Document**
