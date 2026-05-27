# Bastion-Vault Cloud LLM Connector Extension SRS

**Project:** Bastion - RAG Security Governance Framework
**Document Type:** Module Extension SRS (Tier 2.5)
**Document ID:** 21-vault-cloud-llm-ext
**Modules:** B - Vault (primary), A - Sentinel (required upstream)
**Version:** 1.0
**Date:** 2026-05-27
**Status:** Draft

**Base Module References:**
- 11-vault-srs (v3.0) — base module
- 10-sentinel-srs (v3.0) — required upstream gate

**Foundation References:**
- 01-architecture-principles (v3)
- 02-event-schema-standard

---

## 1. Introduction

### 1.1 Purpose

This document specifies the **Cloud LLM Connector** extension for Vault. It enables calls to external cloud LLMs using anonymized data, ensuring PII never reaches cloud providers while preserving the LLM's ability to reason about the data through deterministic tokens.

### 1.2 Design Principle

**Vault owns anonymization. Sentinel gates all LLM access. Neither is optional.**

```
User request
    → Sentinel (validate: injection, content policy)   [required]
    → Vault: anonymize payload by data_category        [this extension]
    → Cloud LLM provider                               [this extension]
    → Vault: de-anonymize tokens in response           [this extension]
    → Sentinel-OUT (validate response)                 [required]
    → User (de-anonymized if authorized; tokenized if not)
```

Vault's existing Phase 1 deterministic tokenization (FR-CORE-AN-002) is what makes this safe: `"Hong Gildong" → always "KR_NAME_8f3d2a"`. The cloud LLM reasons about the token consistently across calls, and the response can be de-anonymized by Vault's Phase 2 (FR-CORE-TR-002) before returning to authorized callers.

### 1.3 Who Decides What Gets Anonymized

Three layers, in priority order. **The caller only provides `data_category`. Vault decides all field rules.**

| Layer | Owner | Mechanism |
|---|---|---|
| Field mapping | Vault config (administrator) | Explicit `field → strategy` rules in `config.yaml` |
| Auto-detection | Vault (FR-CORE-AN-003) | Regex + optional ML on free-text fields |
| Category rules | Caller (request field) | `data_category` selects which field mapping set and K-anonymity threshold applies |

Callers cannot specify individual field-level rules. This prevents callers from accidentally bypassing protection by omitting a sensitive field.

---

## 2. Architecture

```
┌──────────────────────────────────────────────────────┐
│               Vault Service (Go)                      │
├──────────────────────────────────────────────────────┤
│  Existing Core + Enhanced (unchanged)                 │
│  /v1/vault/anonymize                                  │
│  /v1/vault/transform                                  │
│                                                       │
│  Cloud LLM Extension (new Enhanced component)         │
│  /v1/vault/llm/complete                               │
│  gRPC: CallAnonymized                                 │
│         ↓                                             │
│  ┌──────────────────────────────────────────────┐     │
│  │  CloudLLMConnector                           │     │
│  │  1. Anonymize messages (reuses Core engine)  │     │
│  │  2. Call provider via CloudLLMProvider       │     │
│  │  3. De-anonymize tokens in response          │     │
│  │  4. Emit mandatory audit event               │     │
│  └──────────────────────────────────────────────┘     │
│         ↓                                             │
│  Provider Abstraction                                 │
│  OpenAI | Claude | Gemini | Azure OpenAI | Custom     │
│         ↓                                             │
│  KMS Abstraction (API key retrieval)                  │
└──────────────────────────────────────────────────────┘
```

`CloudLLMConnector` reuses the existing `Anonymizer` and `Transformer` from Vault Core. No new anonymization logic is introduced.

---

## 3. Provider Abstraction

### 3.1 Interface

```go
type CloudLLMProvider interface {
    ID() string
    Complete(ctx context.Context, req AnonymizedLLMRequest) (LLMResponse, error)
}
```

### 3.2 Built-in Implementations

| Provider ID | Target | Auth method |
|---|---|---|
| `openai` | OpenAI Chat Completions API | API key via KMS |
| `claude` | Anthropic Messages API | API key via KMS |
| `gemini` | Google Generative Language API | API key via KMS |
| `azure_openai` | Azure OpenAI endpoint | API key via KMS |
| `custom` | User-defined HTTP endpoint | Configurable |

**API keys are never stored in config files.** They are retrieved from Vault's KMS abstraction (FR-CORE-KMS-001) at request time using a KMS reference string (e.g., `local://cloud-llm-openai-key`).

---

## 4. API Interface

### 4.1 REST

```
POST /v1/vault/llm/complete
```

**Request:**
```json
{
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user",   "content": "Summarize Hong Gildong's (hong@company.com) warranty claim from last week."}
  ],
  "data_category": "customer_data",
  "provider": "openai",
  "model": "gpt-4o",
  "user_id": "u-123",
  "tenant_id": "tenant-acme",
  "options": {
    "temperature": 0.3,
    "max_tokens": 1024
  }
}
```

**Response:**
```json
{
  "message": {
    "role": "assistant",
    "content": "Hong Gildong filed a warranty claim on 2026-05-20 regarding..."
  },
  "anonymization": {
    "fields_anonymized": 2,
    "tokens_in_response": 1,
    "tokens_deanonymized": 1
  },
  "provider": "openai",
  "model": "gpt-4o",
  "trace_id": "abc-123"
}
```

The `content` in the response shows `Hong Gildong` because this caller holds `full_access` permission. A `anonymized`-level caller receives `KR_NAME_8f3d2a` in the same field.

### 4.2 gRPC

New method added to `bastion.vault.v1.VaultService`:

```
rpc CallAnonymized(LLMCallRequest) returns (LLMCallResponse);
```

---

## 5. Anonymization Flow

Step-by-step for the example request above:

```
Input messages:
  "Summarize Hong Gildong's (hong@company.com) warranty claim"

Step 1 — Resolve data_category rules:
  data_category = "customer_data"
  → Field mapping rules: korean_name=tokenization, email=fpe

Step 2 — Apply field mapping to named JSON fields:
  "Hong Gildong"     → "KR_NAME_8f3d2a"   (tokenization, reversible)
  "hong@company.com" → "EMAIL_a1b2c3"      (FPE, reversible)

Step 3 — Auto-detect remaining PII in free-text (FR-CORE-AN-003):
  Regex scan: no additional PII found in this message.

Step 4 — Build anonymized request:
  "Summarize KR_NAME_8f3d2a's (EMAIL_a1b2c3) warranty claim"

Step 5 — Call cloud LLM with anonymized messages.

Step 6 — LLM response contains token references:
  "KR_NAME_8f3d2a filed a warranty claim on 2026-05-20..."

Step 7 — De-anonymize tokens in response (permission-based, FR-CORE-TR-002):
  authorized caller  → "Hong Gildong filed a warranty claim on 2026-05-20..."
  anonymized caller  → "KR_NAME_8f3d2a filed a warranty claim on 2026-05-20..."
```

Tokens that appear in the LLM response are recognized by Vault's token store (PostgreSQL) and reversed in place. Tokens that do not appear in the response are ignored — there is no partial-de-anonymization failure.

---

## 6. Audit Trail

Every cloud LLM call emits a mandatory audit event. **This event cannot be disabled.**

```
bastion.events.vault.cloud_llm_called
```

```json
{
  "subject": "bastion.events.vault.cloud_llm_called",
  "data": {
    "provider": "openai",
    "model": "gpt-4o",
    "data_category": "customer_data",
    "pii_fields_anonymized": 2,
    "tokens_in_response": 1,
    "tokens_deanonymized": 1,
    "user_id": "u-123",
    "tenant_id": "tenant-acme",
    "trace_id": "abc-123",
    "latency_ms": 1240,
    "anonymized_request_hash": "sha256:e3b0c44298fc..."
  }
}
```

`anonymized_request_hash` is a SHA-256 of the **anonymized** payload sent to the cloud provider — never the raw PII. This allows audit reconstruction and forensic matching without re-exposing sensitive data.

If NATS is unreachable, the connector **refuses to process the request** and returns HTTP 503. An LLM call without an audit record is not permitted.

---

## 7. Configuration

```yaml
vault:
  cloud_llm:
    enabled: true

    providers:
      - id: openai
        type: openai
        model: gpt-4o
        api_key_kms: local://cloud-llm-openai-key   # KMS reference, never plaintext
        timeout_seconds: 30
        max_retries: 2

      - id: claude
        type: claude
        model: claude-sonnet-4-6
        api_key_kms: local://cloud-llm-claude-key
        timeout_seconds: 30
        max_retries: 2

      - id: internal
        type: custom
        endpoint: http://internal-llm:8080/v1/chat/completions
        auth: none
        timeout_seconds: 10

    default_provider: openai

    audit:
      # enabled is always true; this block exists only for future extension
      log_anonymized_hash: true       # SHA-256 of anonymized payload
      log_raw_response: false         # never log raw LLM output
```

---

## 8. Non-Functional Requirements

| ID | Requirement |
|---|---|
| NFR-CLLM-001 | Sentinel MUST validate the request before `POST /v1/vault/llm/complete` is reached |
| NFR-CLLM-002 | API keys MUST be stored in KMS; plaintext keys in config are rejected at startup |
| NFR-CLLM-003 | Audit event is mandatory; connector returns HTTP 503 if NATS is unavailable |
| NFR-CLLM-004 | Anonymization reuses Vault Core engine; no new anonymization logic in this extension |
| NFR-CLLM-005 | Callers cannot specify field-level rules; only `data_category` is accepted |
| NFR-CLLM-006 | Response de-anonymization respects caller's permission level (FR-CORE-TR-002) |
| NFR-CLLM-007 | Cloud provider timeout propagates as HTTP 504 to the caller |
| NFR-CLLM-008 | Connector is disabled by default (`vault.cloud_llm.enabled: false`) |

---

## 9. Change History

| Version | Date | Changes |
|---|---|---|
| 1.0 | 2026-05-27 | Initial draft |

---

**End of Document**
