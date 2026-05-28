# Bastion — RAG Security Governance Framework

> *"We don't block data. We govern its safe flow."*

**Version 3.0** &nbsp;·&nbsp; Go 1.22+ / Python 3.11+ &nbsp;·&nbsp; Apache 2.0

---

## What problem does this solve?

When an organisation deploys an AI assistant powered by its own documents — a technology called RAG (Retrieval-Augmented Generation) — the AI needs to read private data to answer questions. That creates real risks:

| Risk | Example |
|---|---|
| Prompt injection | A user hides instructions inside a question to trick the AI into revealing secrets |
| PII leakage | The AI accidentally includes someone's name, email, or ID number in a response |
| Cross-tenant data exposure | Customer A's data surfaces in Customer B's response on a shared platform |
| Hallucination | The AI states facts that don't exist in any source document |
| Intrusion | An attacker queries specific records, revealing they already know data they shouldn't |

**Bastion is the security layer that wraps your RAG pipeline and prevents all of these — in under 3 milliseconds of added latency.**

It works in both directions, like an airport that screens passengers on the way in *and* inspects luggage on the way out. Most security tools only check input. Bastion checks both.

---

## Architecture

```
                              ┌─ INPUT PATH ──────────────────────────────────────┐
  User Query                  │                                                   │
      │                       ▼                                                   ▼
      │         ┌─────────────────┐   ┌───────────────┐   ┌──────────┐   ┌──────────────┐
      └────────▶│  Sentinel-IN    │──▶│  Vault Phase1 │──▶│Navigator │──▶│  Anchor-IN   │──▶ LLM
                │ (Go)            │   │  (Go)         │   │ (Python) │   │  (Python)    │
                │ injection check │   │ PII tokenize  │   │ search   │   │ noise inject │
                └─────────────────┘   └───────────────┘   └──────────┘   └──────────────┘
                ┌─────────────────┐   ┌───────────────┐                  ┌──────────────┐
      ┌─────────│  Sentinel-OUT   │◀──│  Vault Phase2 │◀─────────────────│  Anchor-OUT  │◀── LLM
      │         │ (Go)            │   │  (Go)         │                  │  (Python)    │
      │         │ PII / hallucin. │   │ detokenize    │                  │ bias detect  │
      ▼         └─────────────────┘   └───────────────┘                  └──────────────┘
  Safe Response          │                   │                   │               │
                         └───────────────────┴───────────────────┴───────────────┘
                                                      │
                                          ┌───────────────────────┐
                                          │  Tracker  (Go)        │
                                          │  observes every step  │
                                          │  audit · metrics · UI │
                                          └───────────────────────┘
```

Security is applied **symmetrically** — every module that processes input also processes output. This is not common in AI security tooling and is a deliberate architectural constraint: an attacker who bypasses the input layer is still caught at the output layer.

---

## The Five Modules

### 🛡️ Sentinel — Bidirectional Validation Gateway
*Language: Go &nbsp;·&nbsp; Latency: ~0.3 ms (IN), ~0.9 ms (OUT)*

Sentinel is the first and last line of defence. It runs the same engine in both directions, configured differently for each.

**Input (Sentinel-IN)**
- Detects **prompt injection** attacks: instructions hidden in user queries designed to override the AI's behaviour
- Enforces **industry-specific rulesets** — healthcare (PHI), finance (PCI), defence (export controls) — via a pluggable registry
- Validates request metadata (tenant ID, user ID, timestamp format) before any data is touched

**Output (Sentinel-OUT)**
- Scans LLM responses for **PII re-emergence** — personal data that leaked through despite anonymization upstream
- Detects **hallucinations** by checking whether stated facts appear in the retrieved source documents
- Applies **output permission filters** — a user without access to certain data categories cannot receive it in the response, even if the LLM generated it

**Key design decision:** Single engine, two configurations. This avoids the divergence problem where input and output rules drift apart over time.

---

### 🔐 Vault — Privacy Shield (Two-Phase)
*Language: Go &nbsp;·&nbsp; Pattern: tokenize → process → detokenize*

Vault ensures the AI model never sees real personal data. It replaces sensitive values with structured, recognisable tokens before the query reaches the LLM, and restores them (for authorised users) after the response returns.

```
User types:   "Show me Hong Gildong's account, email hong@company.com"
LLM sees:     "Show me [PERSON_a3f2c1]'s account, email [EMAIL_9b4d2a]"
User gets:    "Hong Gildong's balance is ₩5,000,000"   ← Vault decoded it
```

**Phase 1 — Anonymization (input path)**
- PII detection across 30+ entity types (names, emails, phone numbers, national IDs, financial data)
- Structured token format (`[TYPE_hex6]`) that the LLM treats as a real person/value, not as a placeholder
- Tenant-scoped token storage: token `[PERSON_a3f2c1]` in tenant A maps to a different person than the same token in tenant B

**Phase 2 — Access control (output path)**
- OPA (Open Policy Agent) policy evaluation: each user's role determines what they can see in full vs redacted
- K-anonymity enforcement: a response must generalise enough that no individual can be uniquely re-identified from it
- Selective detokenization: tokens are only decoded for fields the user is authorised to view

**Also includes:** Cloud LLM Connector (doc 21) — anonymizes payloads before sending to external AI APIs (OpenAI, Claude, Gemini), so real data is never transmitted to third-party providers.

---

### 🧭 Navigator — Tenant-Isolated Hybrid Search
*Language: Python &nbsp;·&nbsp; Models: BGE-M3 (embed), BGE-reranker-v2-m3 (rerank)*

Navigator handles vector search with two properties that distinguish it from a standard RAG retrieval layer:

**Pre-filter isolation (not post-filter)**
Most multi-tenant search implementations retrieve broadly and then filter out what users shouldn't see. This is wrong — the data was still accessed. Navigator applies tenant filters *before* the search, so a tenant's documents are never touched by another tenant's query, even transiently.

**Hybrid search with RRF fusion**
```
Vector search (BGE-M3 dense embeddings)  ─┐
                                           ├─▶ RRF merge ──▶ Cross-encoder rerank ──▶ Results
BM25 keyword search                       ─┘
```
Reciprocal Rank Fusion (RRF) combines ranked lists from vector and keyword search without requiring score normalisation. The BGE cross-encoder reranker then re-scores the merged set for semantic relevance.

**Federated search**
Multiple Bastion deployments in different data centres can collaborate on a single query. A hop-depth counter prevents circular queries. Each node shares only what its own policy permits.

---

### 📊 Tracker — Observability and Human-in-the-Loop
*Language: Go + React &nbsp;·&nbsp; Transport: NATS event bus, WebSocket*

Tracker is an observer — it touches no data in the pipeline, but records everything that happens to it.

**Event tracing**
Every module emits Foundation-standard events (doc 02) tagged with a `trace_id`. Tracker aggregates these into a complete data lineage: for any request, you can reconstruct every security decision, every PII match, every access control check, from query to response.

**Audit log**
Events are HMAC-signed as they are stored. Any tampering with the log — even a single byte — is detectable. This is a compliance requirement in regulated industries.

**Pipeline monitoring mode** — *switchable at runtime, zero restart*

| Mode | Behaviour |
|---|---|
| `off` | Standard operation, zero overhead (default) |
| `observe` | Every request's pipeline journey captured step-by-step; operators annotate in real time |
| `gate` | Pipeline pauses at each stage and waits for human approval; configurable timeout (default: reject on timeout — fail-safe) |

```bash
# Switch to gate mode — all pipeline stages now require approval
curl -X POST http://localhost:8080/v1/monitor/mode \
     -d '{"mode": "gate", "reason": "incident investigation"}'

# An operator approves a pending checkpoint
curl -X POST http://localhost:8080/v1/monitor/checkpoints/{id}/decide \
     -d '{"decision": "approve", "notes": "verified — legitimate request"}'
```

Real-time updates are pushed over WebSocket so operator dashboards update without polling.

---

### ⚓ Anchor — Embedding Security
*Language: Python &nbsp;·&nbsp; Techniques: differential noise, WEAT bias analysis*

Anchor protects the vector embeddings that power search — the mathematical fingerprints that represent document meaning.

**Noise injection**
Adds calibrated Gaussian noise to embeddings during both indexing (write) and querying (read). The noise is small enough that search quality is not significantly degraded, but large enough that an attacker who extracted raw vectors from the database could not reconstruct the original documents.

**Bias monitoring**
Uses Word Embedding Association Test (WEAT) statistics to detect whether the system is responding differently for different user groups — a sign that the underlying model has demographic bias that could produce discriminatory outputs.

---

## Cross-Cutting Features

### 🍯 Honey-Token Intrusion Detection

Bastion plants invisible decoy records — fake customers, fake API keys, fake emails — throughout the data. No legitimate query ever touches them. When one is accessed, it proves the querier has prior knowledge of data they should not know.

Detection is multi-layer: Vault detects honey tokens in the query, Navigator detects them in search results, Sentinel detects them in the LLM response. Tracker correlates across all three. A single hit might be coincidence; the same user triggering all three layers simultaneously is a confirmed breach.

### 🏢 Multi-Tenant Isolation

Tenant isolation is enforced **independently** in every module:
- Vault: per-tenant encryption key derivation
- Navigator: pre-filter on `tenant_id` before search (not after)
- Sentinel: per-tenant rule configurations

If any single layer's isolation fails, the others still hold. This is defence-in-depth applied to the multi-tenancy problem.

### 🌐 Federated Search

Multiple Bastion deployments can collaborate on a single query across data centres or organisational boundaries. Each deployment shares only what its own policy permits. Hop-depth counters prevent circular query loops. Result sets are merged using RRF at the coordinating node.

---

## Performance

Measured in-process across all five modules (latency test: 20 iterations, p50/p95):

| Stage | Engine cost (p50) | With HTTP overhead |
|---|---|---|
| Sentinel-IN | 0.14 ms | ~0.3 ms |
| Sentinel-OUT | 0.59 ms | ~0.9 ms |
| **Full pipeline (Bastion only)** | **~1.2 ms** | **~1–3 ms** |
| *(LLM API call, not included)* | — | *(200–800 ms)* |

Bastion's entire security wrapper adds less than 1% to the total round-trip time.

---

## Test Coverage

| Module | Test files | Test cases |
|---|---|---|
| Sentinel (Go) | 12 | ~140 |
| Vault (Go) | 18 | ~120 |
| Tracker (Go) | 13 | ~277 |
| Navigator (Python) | 3 | 61 |
| Anchor (Python) | 4 | 129 |
| End-to-end (Go) | 5 | ~60 |
| **Total** | **55** | **~787** |

Test suites include: unit tests, HTTP integration tests, end-to-end pipeline scenario tests (8 scenarios), latency benchmarks, and system tests that exercise the full stack without mocking.

---

## Getting Started

### Quick start

```bash
git clone --recurse-submodules https://github.com/zafrem/bastion-rag.git
cd bastion-rag
docker-compose up
```

> **Note:** Navigator downloads `BAAI/bge-m3` (~1.4 GB) and `BAAI/bge-reranker-v2-m3` (~900 MB) on first run. Cached after that. Requires ~4 GB RAM.

| Service | Endpoint |
|---|---|
| Tracker dashboard | http://localhost:3000 |
| Tracker REST API | http://localhost:8080 |
| Sentinel | REST :8080 · gRPC :9090 |
| Vault | REST :8081 · gRPC :9091 |
| Navigator | REST :8082 · gRPC :9092 |
| Anchor | REST :8083 · gRPC :9093 |

### Run demos and tests

```bash
make demo          # replay all 8 built-in security scenarios
make run-tests     # full test suite (Go + Python)
```

### Deployment configurations

| Configuration | Modules | Suitable for |
|---|---|---|
| Minimal | Sentinel | Development, proof-of-concept |
| Basic | Sentinel + Vault | GDPR-compliant AI, PII protection |
| Standard | Sentinel + Vault + Navigator | Production multi-tenant RAG |
| Full | All five modules | Sensitive data, regulated environments |
| Enterprise | All + cross-cutting features | Regulated industries, large-scale SaaS |

---

## Repository Structure

```
bastion-rag/
├── sentinel/            Go — injection detection, output validation
├── vault/               Go — PII anonymization, access control, Cloud LLM connector
├── navigator/           Python — hybrid vector search, federation
├── anchor/              Python — embedding noise, bias detection
├── tracker/             Go — observability, audit, monitoring mode, React UI
├── tests/               End-to-end, scenario, and latency tests
├── docs/                SRS specification library (16 documents, v3.0)
├── BASTION_OVERVIEW.md  Plain-English architecture guide
└── docker-compose.yml
```

---

## Documentation

| Audience | Document |
|---|---|
| Non-technical stakeholders | [BASTION_OVERVIEW.md](./BASTION_OVERVIEW.md) |
| Architecture overview | [docs/30_integration_master_overview.md](./docs/30_integration_master_overview.md) |
| Architecture principles | [docs/01_foundation_architecture_principles.md](./docs/01_foundation_architecture_principles.md) |
| Event schema standard | [docs/02_foundation_event_schema_standard.md](./docs/02_foundation_event_schema_standard.md) |
| Module interaction map | [docs/03_foundation_module_interaction_map.md](./docs/03_foundation_module_interaction_map.md) |
| Sentinel SRS v3.0 | [docs/10_module_sentinel_srs_v3.md](./docs/10_module_sentinel_srs_v3.md) |
| Vault SRS v3.0 | [docs/11_module_vault_srs_v3.md](./docs/11_module_vault_srs_v3.md) |
| Navigator SRS v3.0 | [docs/12_module_navigator_srs_v3.md](./docs/12_module_navigator_srs_v3.md) |
| Anchor SRS v3.0 | [docs/13_module_anchor_srs_v3.md](./docs/13_module_anchor_srs_v3.md) |
| Tracker SRS v3.0 | [docs/14_module_tracker_srs_v3.md](./docs/14_module_tracker_srs_v3.md) |
| Honey-token SRS | [docs/20_extension_sentinel_industry_v1.md](./docs/20_extension_sentinel_industry_v1.md) |
| Multi-tenancy SRS | [docs/21_extension_vault_cloud_llm_v1.md](./docs/21_extension_vault_cloud_llm_v1.md) |
| Data lineage SRS | [docs/22_extension_navigator_federation_v1.md](./docs/22_extension_navigator_federation_v1.md) |

---

## License

Apache License 2.0
