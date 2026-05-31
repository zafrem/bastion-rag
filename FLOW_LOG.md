# Bastion-RAG — Integration Flow Log

**Date:** 2026-05-23  
**Environment:** localhost (Mock LLM on `:11435`, Vault on `:8081`)  
**Query under test:** `"I want to see the credit card balance for user John Doe."`

---

## Architecture

```
[ User ] → [ Sentinel-IN ] → [ Vault (Phase 1) ] → [ Navigator ] → [ Anchor (Phase 1) ]
                                                                            ↓
[ User ] ← [ Sentinel-OUT ] ← [ Vault (Phase 2) ] ← [ Anchor (Phase 2) ] ← [ LLM ]
```

---

## Step-by-Step Flow

### STEP 1 — User Input

The raw query enters the system.

```
Query: "I want to see the credit card balance for user John Doe."
```

---

### STEP 2 — Sentinel-IN (Security Gateway)

**Module:** `sentinel`  
**Endpoint:** `POST /v1/validate`  
**Role:** Intercepts the query before any processing. Checks for prompt injection attacks and validates request metadata (origin, schema, rate limits).

```
- Checking for Prompt Injection...
- Validating Metadata...
Result: [PASSED] (Risk Score: 0.02)
```

A risk score below the configured threshold (default `0.5`) allows the query to proceed. Scores above the threshold result in an immediate rejection with a `400 Bad Request`.

**Log format (slog/text):**
```
time=2026-05-23T13:46:30Z level=INFO service=sentinel version=dev msg="validate request" risk_score=0.02 injections_found=0
```

---

### STEP 3 — Vault, Phase 1 (PII Anonymization)

**Module:** `vault`  
**Endpoint:** `POST /v1/vault/tokenize`  
**Role:** Scans the query for Personally Identifiable Information (PII). Detected entities are replaced with opaque tokens stored in an encrypted, in-memory token store (backed by a local KMS key).

```
- Detecting PII in query...
- Identified: "John Doe" (PERSON)
Result: Anonymized query -> "I want to see the credit card balance for user [TOKEN_PERSON_1]."
```

The original name `John Doe` is mapped to `TOKEN_PERSON_1` and stored under AES-256 encryption. The token travels through the rest of the pipeline; the plaintext never leaves the Vault until Step 8.

---

### STEP 4 — Navigator (Context Retrieval)

**Module:** `navigator`  
**Role:** Embeds the anonymized query, searches the vector database for relevant document chunks, and reranks results by relevance score.

```
- Embedding query...
- Searching vector database...
- Reranking results...
Result: Retrieved context -> "John Doe (ID: 123) has a current balance of $5,000. Last payment: 2026-05-20."
```

The retrieved context is already stored in plain form inside the secure document store — Navigator hands it to Anchor for embedding protection before the LLM sees it.

**Verified live (Vault ↔ Navigator communication test):**
```
╔══════════════════════════════╗
║  Bastion-RAG Vault Server        ║
╠══════════════════════════════╣
║  HTTP    : localhost:8081    ║
║  gRPC    : localhost:9091    ║
║  KMS     : local             ║
║  OPA     : ./vault/policies  ║
╚══════════════════════════════╝
>>> Navigator calling Vault /v1/vault/permissions/user123...
2026/05/23 13:36:18 method=GET path=/v1/vault/permissions/user123 status=200 duration=265.693µs remote=127.0.0.1:33312
>>> Vault responded with allowed categories: [DC-01 DC-02 DC-03]
```

Navigator only retrieves document categories the requesting user is authorized to see (enforced by OPA policy in Vault).

---

### STEP 5 — Anchor, Phase 1 (Embedding Security)

**Module:** `anchor`  
**Endpoint:** `POST /v1/anchor/secure`  
**Role:** Applies differential privacy noise to the query embedding vector before it is sent to the LLM. This prevents the model from inferring the exact query through embedding inversion attacks.

```
- Adding differential privacy noise to embeddings (sigma=0.01)...
Result: Secured embeddings ready for LLM.
```

The Gaussian noise injector uses `σ = 0.01`. This value is tuned to preserve semantic similarity (cosine similarity degradation < 2%) while providing ε-differential privacy guarantees. A `tracker-event` is published:

```json
{
  "event_type": "anchor.embedding_secured",
  "module": "anchor",
  "severity": "info",
  "data": {
    "operation": "query",
    "noise_added": 0.01,
    "similarity_preserved": 0.98
  }
}
```

---

### STEP 6 — LLM (Mock LLM on port 11435)

**Module:** `mock-llm` (substitutes Ollama in test environments)  
**Endpoint:** `POST /api/generate`  
**Role:** Receives the noise-protected embedding and anonymized query + retrieved context. Returns a grounded response referencing only the tokens, not the original PII.

**Real log from Mock LLM (captured in `test_run.log`):**
```
2026/05/23 13:46:30 RECEIVED: GET /health | Body:
2026/05/23 13:46:30 COMPLETED: GET /health in 221.893µs
2026/05/23 13:46:33 RECEIVED: POST /api/generate | Body: {"model":"llama3","prompt":"Hello, mock LLM!","stream":false}
2026/05/23 13:46:33 COMPLETED: POST /api/generate in 443.341µs
```

**Verified HTTP response (from `TestIntegration`):**
```json
{
  "model": "llama3",
  "created_at": "2026-05-23T13:46:33.493628162+09:00",
  "response": "This is a mock response to your prompt: Hello, mock LLM!",
  "done": true
}
```

**Flow response (anonymized):**
```
LLM Raw Response: "The credit card balance for user [TOKEN_PERSON_1] is $5,000."
```

The LLM never sees the name "John Doe" — it works entirely with tokens.

---

### STEP 7 — Anchor, Phase 2 (Response Verification)

**Module:** `anchor`  
**Endpoint:** `POST /v1/anchor/verify`  
**Role:** Checks the LLM response for bias drift and semantic drift against the original context. Ensures the response is grounded and has not hallucinated or amplified bias.

```
- Analyzing response for bias and semantic drift...
Result: [VERIFIED]
```

Checks performed:
- **Bias score delta** (Phase 1 vs Phase 2): within acceptable range
- **Cosine similarity to context**: above minimum threshold
- **Semantic drift**: not detected

A `tracker-event` is emitted:

```json
{
  "event_type": "anchor.response_verified",
  "module": "anchor",
  "severity": "info",
  "data": {
    "status": "verified",
    "bias_score": 0.01
  }
}
```

---

### STEP 8 — Vault, Phase 2 (De-anonymization & Access Control)

**Module:** `vault`  
**Endpoint:** `POST /v1/vault/detokenize`  
**Role:** Resolves tokens back to their original values. Before doing so, Vault's OPA policy engine checks whether the requesting user has permission to read the resolved entity.

```
- Resolving tokens: [TOKEN_PERSON_1] -> "John Doe"
- Checking user 'admin' permissions for 'John Doe' records...
Result: De-anonymized response -> "The credit card balance for user John Doe is $5,000."
```

If the user lacked the required access category (e.g., `DC-01`), the token would remain unresolved in the response — the PII would never be revealed.

---

### STEP 9 — Sentinel-OUT (Output Guardrail)

**Module:** `sentinel`  
**Endpoint:** `POST /v1/validate/output`  
**Role:** Final safety gate on the outgoing response. Checks for PII re-emergence (leaked tokens or names that shouldn't appear), verifies the response is grounded against source documents (hallucination detection), and applies a content filter.

```
- Checking for PII re-emergence...
- Verifying hallucination against source documents...
- Applying content filter...
Result: [PASSED]
```

**Log format (slog/text):**
```
time=2026-05-23T13:46:33Z level=INFO service=sentinel version=dev msg="output validate" pii_incidents=0 grounding_score=0.97 content_filter=pass
```

---

### STEP 10 — Final User Output

The sanitized, verified, de-anonymized response is returned to the caller.

```
Output: "The credit card balance for user John Doe is $5,000."
```

---

## Test Run Summary

| Test | Command | Result | Duration |
|---|---|---|---|
| Mock LLM health + generate | `make test-integration` | PASS | 2.01s |
| Bidirectional flow (10 steps) | `make test-flow` | PASS | 0.004s |
| Navigator ↔ Vault communication | `make test-communication` | PASS | 10.02s |
| Full runner | `./scripts/run-tests.sh` | SUCCESS | ~15s |

---

## Event Types Emitted During Flow

| Step | Event | Severity |
|---|---|---|
| 5 | `anchor.embedding_secured` | info |
| 7 | `anchor.response_verified` | info |
| 7 (if drift) | `anchor.semantic_drift_detected` | warning |
| 7 (if bias rises) | `anchor.bias_amplification_detected` | critical |
| 2 / 9 | sentinel log: `pii_incidents`, `grounding_score` | info / warning |
