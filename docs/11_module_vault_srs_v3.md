# Bastion-Vault Module SRS

**Project:** Bastion - RAG Security Governance Framework
**Document Type:** Module SRS (Tier 2)
**Document ID:** 11-vault-srs
**Module:** B - Vault (Data Isolation & Anonymization)
**Version:** 3.0 (Foundation-aligned, Phase1+2 integrated)
**Date:** 2026-05-26
**Status:** Active
**Supersedes:** v2.0 (2026-05-17) — archived at docs/archive/v2/

**Foundation References:**
- 01-architecture-principles (v3 — polyglot)
- 02-event-schema-standard
- 03-module-interaction-map

---

## 1. Introduction

### 1.1 Purpose

This document specifies the **Vault** module, the data protection core of Bastion. Vault operates bidirectionally:
- **Phase 1 (Input/Storage):** Anonymize data before storage
- **Phase 2 (Output/Use):** Apply permissions on retrieval

### 1.2 What Changed in v3

```
Vault itself is UNCHANGED.
Language: Go 1.22+ (was 1.21+)

Polyglot context:
- Vault (B): Go — unchanged
- Navigator (C): now Python (was Go)
- Anchor (E): now Python (was Go)

Vault's permission API is consumed by Navigator (now a Python service).
Wire contract (JSON-over-gRPC, REST) is unchanged — language is internal.
```

### 1.3 Module Identity

```
Module: B - Vault
Language: Go 1.22+
Role: Data Isolation & Anonymization (bidirectional)
Position: After Sentinel (input) + before Sentinel-OUT (output)

Standalone value:
"Attach Vault alone to an LLM → PII never reaches LLM"
```

### 1.4 The Standalone Test (Foundation Litmus)

```
Question: "If only Vault is attached to an LLM,
          does it provide meaningful security?"

Answer: YES
- Input: anonymizes PII before LLM sees it
- Input: masks sensitive data
- Output: re-applies based on permissions

→ Vault passes the standalone test ✅
```

### 1.5 Scope

**In Scope:**
- 🟢 Core: PII anonymization (Phase 1)
- 🟢 Core: Deterministic tokenization
- 🟢 Core: Permission-based transformation (Phase 2)
- 🟢 Core: KMS abstraction (AWS/HashiCorp/Local)
- 🟡 Enhanced: Permission provision to Navigator
- 🟡 Enhanced: Search result transformation
- 🔴 Hooks: Honey-token creation/injection/detection
- 🔴 Hooks: Multi-tenancy coordination
- 🔴 Hooks: Lineage event emission
- Bidirectional operation (Phase 1 + Phase 2)
- Standalone deployment

**Out of Scope:**
- Input validation (Sentinel)
- Search (Navigator)
- Detailed honey-token logic (Honey-Token Cross-cutting SRS)
- Detailed multi-tenancy logic (Multi-tenancy Cross-cutting SRS)

### 1.6 Definitions

| Term | Definition |
|---|---|
| **Vault Phase 1** | Anonymization (storage/input) |
| **Vault Phase 2** | Permission transform (use/output) |
| **Tokenization** | Reversible PII → token |
| **K-anonymity** | Group-based privacy |
| **KMS** | Key Management Service |
| **DC** | Data Category (DC-01/02/03) |

---

## 2. Overall Description

### 2.1 Bidirectional Architecture

```
┌─────────────────────────────────────────────┐
│             Vault Service (Go)               │
├─────────────────────────────────────────────┤
│  ┌────────────┐         ┌────────────┐      │
│  │ Phase 1 API│         │ Phase 2 API│      │
│  │(/anonymize)│         │(/transform)│      │
│  └─────┬──────┘         └─────┬──────┘      │
│        └───────────┬──────────┘             │
│                    ▼                        │
│      ┌──────────────────────────┐           │
│      │  Shared Core Services    │           │
│      │  - Token mappings        │           │
│      │  - KMS abstraction       │           │
│      └────────┬─────────────────┘           │
│               │                             │
│      ┌────────┼─────────┐                   │
│      ▼        ▼         ▼                   │
│  ┌───────┐┌───────┐┌──────────┐             │
│  │Anonym ││Transform││ Hooks   │             │
│  │izer   ││er      ││(optional)│             │
│  └───────┘└───────┘└──────────┘             │
└─────────────────────────────────────────────┘
```

### 2.2 Position in Pipeline

```
Input Pipeline (Phase 1):
Sentinel → [Vault-IN: anonymize] → Navigator(Py)

Output Pipeline (Phase 2):
Anchor(Py) → [Vault-OUT: transform] → Sentinel-OUT

Navigator and Anchor are Python services in v3 polyglot architecture.
Vault communicates with them via JSON-over-gRPC (wire contract unchanged).
```

### 2.3 Layer Classification

```
🟢 CORE (Standalone):
   Phase 1: anonymize PII, tokenize
   Phase 2: permission transform, deanonymize
   KMS abstraction

🟡 ENHANCED (Composition):
   - Permission provision (+ Navigator — Python service)
   - Search result transform (+ Navigator — Python service)

🔴 HOOKS (Cross-cutting):
   - Honey-token (create/inject/detect)
   - Multi-tenancy coordination
   - Lineage emission
```

### 2.4 Constraints

```
Language: Go 1.22+
Memory: ≤ 2GB (shared Phase 1+2)
Latency: Anonymize <5ms, Transform <10ms (p95)
KMS: AWS + HashiCorp + Local
```

### 2.5 Dependencies

```
Core dependencies:
- PostgreSQL (token storage)
- KMS (one of: AWS/HashiCorp/Local)

Optional (Enhanced):
- Navigator (permission consumer; Python service, JSON-over-gRPC)

Optional (Hooks):
- NATS, cross-cutting coordinators
```

---

## 3. Core Functions (🟢 Standalone)

### 3.1 Phase 1: PII Anonymization (FR-CORE-AN)

**FR-CORE-AN-001: Multi-Strategy Anonymization**
```
Strategies:
- Tokenization (reversible): names, emails
- Hashing (irreversible): RRN, SSN
- FPE (format-preserving): credit cards
- Masking: phones
- Generalization: age, location

Dependency: KMS only (no other module)
```

**FR-CORE-AN-002: Deterministic Tokenization**
```
Same input → same token (consistency)
Enables: same person tracked across records
"Hong Gildong" → always "KR_NAME_8f3d2a"
```

**FR-CORE-AN-003: PII Detection (Hybrid)**
```
- Explicit field mapping (priority)
- Regex patterns
- ML detection (optional)
```

**FR-CORE-AN-004: Data Categories**
```
DC-01: Customer (marketing access)
DC-02: Manufacturing (mfg + marketing)
DC-03: HR/Finance (HR only)
```

### 3.2 Phase 2: Permission Transform (FR-CORE-TR)

**FR-CORE-TR-001: Permission-Based Transformation**
```
Same data, different output by user:
- Full access: deanonymize
- Anonymized: keep tokens
- K-anonymized: aggregate
- Aggregated: statistics only

Dependency: OPA (policy) - bundled
```

**FR-CORE-TR-002: Selective Deanonymization**
```
Reverse tokenization for authorized users
Field-level permissions
Audit every deanonymization
```

**FR-CORE-TR-003: K-anonymity Enforcement**
```
K values: DC-01=5, DC-02=3, DC-03=10
Auto-generalization on violation
```

### 3.3 KMS Abstraction (FR-CORE-KMS)

**FR-CORE-KMS-001: Multi-Provider**
```
interface KMSProvider {
    Encrypt/Decrypt
    GenerateDataKey
    HMAC
}

Implementations: AWS, HashiCorp, Local
```

**FR-CORE-KMS-002: Envelope Encryption**
```
Master Key (KMS) → Data Key → encrypts PII
```

### 3.4 Core Summary

```
Standalone capabilities (KMS + DB only):

Phase 1:
✅ PII anonymization
✅ Deterministic tokenization
✅ Category classification

Phase 2:
✅ Permission transform
✅ Selective deanonymization
✅ K-anonymity

These work even if Vault is the only module.
```

---

## 4. Enhanced Functions (🟡 Composition)

### 4.1 Permission Provision to Navigator (FR-ENH-PP)

**Requires: Navigator (permission consumer; Python service)**

**FR-ENH-PP-001: Permission API**
```
Provide permissions for Navigator's search filtering.

Interface (per Foundation — caller fetches, then passes in):
  GET /v1/vault/permissions/{user_id}

The pipeline orchestrator calls Vault.GetPermissions()
and passes the result to Navigator.SearchWithPermissions().
Navigator (Python) does NOT call Vault directly (Foundation prohibition).

Graceful degradation:
- Without Navigator: permissions unused
- Core anonymization still works
```

### 4.2 Search Result Transformation (FR-ENH-ST)

**Requires: Navigator (provides results; Python service)**

**FR-ENH-ST-001: Result Anonymization**
```
Transform Navigator's search results by permission:

Input: search results (passed in request)
Process: apply Phase 2 transform
Output: permission-appropriate results

Graceful degradation:
- Without Navigator: no results to transform
- Core transform still works on direct data
```

### 4.3 Enhanced Summary

```
🟡 Permission provision (+ Navigator — Python service)
🟡 Search result transform (+ Navigator — Python service)

Wire contract: JSON-over-gRPC (unchanged from v2)
Without Navigator: core anonymization/transform works
```

---

## 5. Hooks (🔴 Cross-Cutting)

### 5.1 Honey-Token Hooks

**Hook Points:**
```
vault.honey.inject
- During indexing, inject honey-tokens
- Vault owns this (data layer)
- Detail: see Honey-Token SRS

vault.data.accessed
- On data access, check if honey-token
- Detail: see Honey-Token SRS
```

**Brief Contract:**
```
Vault is the OWNER of honey-tokens:
- Creates fake data
- Injects into datasets
- Marks/identifies
- Detects data-layer access

On honey-token data access:
→ event: vault.honey_token_accessed
→ severity: HIGH

Full logic: Honey-Token SRS (Tier 3).
```

### 5.2 Multi-Tenancy Hooks

**Hook Point:**
```
vault.tenant.isolate
- Enforce tenant key separation
- Detail: see Multi-tenancy SRS

On cross-tenant attempt:
→ event: vault.cross_tenant_attempt
→ severity: CRITICAL
```

### 5.3 Lineage Hooks

```
vault.anonymize.completed
vault.transform.completed
→ Lineage events with trace_id
Detail: see Data Lineage SRS
```

### 5.4 Hook Summary

```
🔴 vault.honey.inject       → Honey-Token SRS
🔴 vault.data.accessed      → Honey-Token SRS
🔴 vault.tenant.isolate     → Multi-tenancy SRS
🔴 vault.*.completed        → Lineage SRS
```

---

## 6. External Interfaces

### 6.1 gRPC Interface

```
Wire format: JSON-over-gRPC (Go encoding.RegisterCodec JSONCodec)
Service: bastion.vault.v1.VaultService

Methods:
  Anonymize(AnonymizeRequest) → AnonymizeResponse
  Transform(TransformRequest) → TransformResponse
  Deanonymize(DeanonymizeRequest) → DeanonymizeResponse
  CheckAccess(AccessRequest) → AccessResponse
  GetPermissions(PermRequest) → PermResponse
  Health(HealthRequest) → HealthResponse
```

### 6.2 REST Interface

```
# Core Phase 1
POST /v1/vault/anonymize

# Core Phase 2
POST /v1/vault/transform
POST /v1/vault/deanonymize
POST /v1/vault/access/check

# Enhanced (Navigator consumes)
GET  /v1/vault/permissions/{user_id}

# Standard
GET  /v1/health
GET  /v1/metrics
```

### 6.3 CLI Interface

```bash
$ vault-cli anonymize --category customer_data \
    --data '{"name":"Hong","email":"hong@naver.com"}'
$ vault-cli transform --user-id alice --role marketing_analyst --input-file data.json
$ vault-cli server --kms local
```

### 6.4 Events (Foundation Schema)

```
Operational:
  bastion.events.vault.anonymized
  bastion.events.vault.transform_executed

Security:
  bastion.events.vault.access_denied
  bastion.events.vault.deanonymization_performed
  bastion.events.vault.k_anonymity_enforced

Via hooks:
  bastion.events.vault.honey_token_accessed
  bastion.events.vault.cross_tenant_attempt
```

---

## 7. Non-Functional Requirements

### 7.1 Performance

| ID | Item | Target |
|---|---|---|
| NFR-PE-001 | Anonymize (p95) | < 5ms |
| NFR-PE-002 | Transform (p95) | < 10ms |
| NFR-PE-003 | Deanonymize (p95) | < 20ms |
| NFR-PE-004 | K-anon check | < 50ms |
| NFR-PE-005 | Memory | ≤ 2GB |

### 7.2 Independence (Foundation)

```
NFR-IND-001: Core works standalone (KMS+DB only)
NFR-IND-002: Graceful degradation (Navigator optional)
NFR-IND-003: Loose coupling (events, no direct calls to other modules)
```

### 7.3 Compliance

```
PIPA: RRN irreversible, retention limits
GDPR: right to erasure (token deletion)
PCI DSS: card tokenization
```

---

## 8. System Architecture

```
┌────────────────────────────────────────────┐
│             Vault Service (Go 1.22+)        │
├────────────────────────────────────────────┤
│  API (gRPC/REST/CLI)                        │
│  REST :8081 | gRPC :9091                    │
│         ↓                                   │
│  Phase Dispatcher (anonymize/transform)     │
│         ↓                                   │
│  Core Engine                                │
│  - Anonymizer (Phase 1)                     │
│  - Transformer (Phase 2)                    │
│  - K-anon validator                         │
│         ↓                                   │
│  KMS Abstraction (AWS/HashiCorp/Local)      │
│         ↓                                   │
│  Hook Manager                               │
│  hm.Fire(hooks.Event{...})                  │
│         ↓                                   │
│  Storage: PostgreSQL (tokens) + Redis       │
│         ↓                                   │
│  Event Publisher (NATS)                     │
└────────────────────────────────────────────┘
```

---

## 9. Standalone Operation

### 9.1 Startup Log

```
[vault] starting v3.0 (REST :8081, gRPC :9091)
[vault] KMS: local
[vault] PostgreSQL connected
[vault] Navigator: not connected (Enhanced limited)
[vault] core: FULLY OPERATIONAL
[vault] ready
```

### 9.2 Standalone Test (Litmus)

```
POST /v1/vault/anonymize
{"data": {"name": "Hong Gildong", "rrn": "850315-1234567"}, "category": "customer_data"}

→ 200 OK
{
  "anonymized": {
    "name": "KR_NAME_8f3d2a",
    "rrn": "RRN_a1b2c3..."
  }
}

(No other modules needed)
```

### 9.3 Degradation

```
Without other modules:
✅ Anonymization: works (KMS+DB)
✅ Transform: works
✅ K-anonymity: works
⚠️ Permission to Navigator: inactive (no consumer)
⚠️ Honey-token: inactive (needs coordinator)

Core fully functional standalone ✅
```

---

## 10. Configuration

```yaml
# /etc/bastion-vault/config.yaml
version: "3.0"

server:
  rest_port: 8081
  grpc_port: 9091

core:
  kms:
    provider: local     # local | aws | hashicorp
  anonymization:
    strategies:
      korean_name: {strategy: tokenization, reversible: true}
      korean_rrn: {strategy: hmac, reversible: false}
      email: {strategy: fpe, reversible: true}
  data_categories:
    customer_data: {k_anonymity: 5}
    manufacturing_data: {k_anonymity: 3}
    hr_finance_data: {k_anonymity: 10}

enhanced:
  permission_provision: true

hooks:
  honey_token: false
  multi_tenancy: true
  lineage: true

storage:
  token_db: postgresql://postgres:5432/vault
  cache: redis://redis:6379

events:
  nats_url: nats://nats:4222
```

---

## 11. Summary

```
🟢 Core (Standalone, Go 1.22+):
   Phase 1: anonymize, tokenize, categorize
   Phase 2: transform, deanonymize, K-anon
   KMS abstraction (AWS/HashiCorp/Local)

🟡 Enhanced (Composition):
   - Permission provision (+ Navigator — Python service)
   - Search transform (+ Navigator — Python service)

🔴 Hooks (Cross-cutting):
   - Honey-token OWNER (→ Honey-Token SRS)
   - Multi-tenancy (→ Multi-tenancy SRS)
   - Lineage (→ Lineage SRS)

Wire contract: unchanged from v2
Language: Go (unchanged)
Polyglot note: Navigator (consumer of GetPermissions) is now Python;
               wire contract (JSON-over-gRPC) is identical.
```

---

## 12. Change History

| Version | Date | Changes |
|---|---|---|
| 1.0 | 2026-05-17 | Initial (separate Phase docs) |
| 2.0 | 2026-05-17 | Foundation-aligned, Phase1+2 integrated |
| 3.0 | 2026-05-26 | Go 1.22+; polyglot context (Navigator/Anchor → Python); foundation ref v3 |

---

**End of Document**

## Appendix: Cross-cutting References

```
Honey-Token (Tier 3): Vault is OWNER
- Hooks: honey.inject, data.accessed

Multi-tenancy (Tier 3):
- Core: tenant_id, key separation

Data Lineage (Tier 3):
- Hooks: *.completed
```
