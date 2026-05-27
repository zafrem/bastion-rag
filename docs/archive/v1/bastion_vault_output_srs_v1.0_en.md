# Bastion-Vault Output Validation SRS v1.0

**Project:** Bastion - RAG Security Governance Framework  
**Module:** Module B - Vault (Output Validation / Phase 2)  
**Document Version:** 1.0  
**Date:** 2026-05-17  
**Status:** Draft  
**Parent Document:** Bastion-Vault SRS v1.0 (Input / Phase 1)  
**Companion Document:** Bastion-Sentinel Output SRS v1.0

---

## 1. Introduction

### 1.1 Purpose

This document defines the **Output Validation (Phase 2)** functionality of the Bastion-Vault module. While Phase 1 handles data anonymization at storage time, Phase 2 ensures that data presented in LLM responses respects the user's permissions and the security boundaries established at storage.

Vault-OUT focuses on:
1. **Permission-based Re-application** - Adjusting data presentation based on user's access level
2. **Search Result Filtering** - Ensuring retrieved documents match user permissions
3. **Re-identification Prevention** - Detecting inference attacks
4. **Slice/Aggregation Enforcement** - Enforcing access level constraints (full/anonymized/k-anonymized/aggregated)
5. **Token Resolution Control** - Selective deanonymization based on permissions
6. **Cross-Reference Validation** - Detecting combined data attacks

### 1.2 Why Vault-OUT?

Phase 1 (Input/Storage) anonymizes data when it enters the system:
```
Customer Data → Vault Phase 1 → Anonymized in DB
```

However, during retrieval and LLM response generation, several risks emerge:
```
Anonymized DB → Navigator retrieves → LLM responds → User
                                                       ↑
                                              Permission check needed!
```

Vault-OUT addresses:

#### Risk 1: Over-Disclosure to Authorized Users
```
HR Manager queries: "What is John's salary?"
Phase 1 stored: salary encrypted
Without Phase 2: Always returns encrypted (useless)
With Phase 2: Returns decrypted (HR Manager has permission)
```

#### Risk 2: Under-Disclosure to Limited Users
```
Marketing Analyst queries: "Customer demographics?"
Phase 1 stored: anonymized with tokens
Without Phase 2: Returns tokens (confusing)
With Phase 2: Returns K-anonymized aggregated data (useful + safe)
```

#### Risk 3: Slice Access Violations
```
Manufacturing queries: "Product complaints"
User has slice access (product-related only)
Phase 1 stored: full customer data
Phase 2: Returns ONLY product-related fields, K-anonymized
```

#### Risk 4: Inference Attacks
```
User combines multiple K-anonymized results:
Query 1: "30s male customers" (K=5 each query)
Query 2: "Seoul Gangnam customers" (K=5 each query)
Query 3: "Premium members" (K=5 each query)
Intersection: Possibly 1 person identified!
```

#### Risk 5: Token Leakage to Unauthorized Users
```
LLM response contains: "USER_8f3d2a purchased..."
User without permission to see specific identifiers
Phase 2: Replace tokens with generic descriptors
```

### 1.3 Scope

**In Scope:**
- Permission-based deanonymization
- Access level enforcement during retrieval
- K-anonymity validation on output sets
- Slice access enforcement
- Aggregation enforcement
- Inference attack detection
- Token resolution control
- Cross-tenant verification at output
- Audit logging of all decisions

**Out of Scope:**
- Anonymization (Phase 1 - Parent SRS)
- LLM response content validation (Sentinel-OUT)
- Embedding security (Anchor)
- Vector search itself (Navigator)

### 1.4 Design Philosophy

```
Principle 1: Two-Phase Asymmetry
- Phase 1: Conservative (anonymize everything potentially sensitive)
- Phase 2: Permissive (release only what user is authorized to see)
- Net result: Defense in depth with operational flexibility

Principle 2: Same Data, Different Views
- One anonymized record can produce multiple views
- View determined by requester's permissions
- No data duplication; computed views on read

Principle 3: Fail-Safe Defaults
- Unknown permission → most restrictive view
- Failure to determine → block
- Always log decisions

Principle 4: Verifiability
- Every disclosure decision auditable
- Reproducible from logs
- Compliance-ready trail
```

### 1.5 Definitions and Acronyms

| Term | Definition |
|---|---|
| **Vault-IN** | Vault in input mode (Phase 1, anonymization) |
| **Vault-OUT** | Vault in output mode (Phase 2, this doc) |
| **Re-application** | Applying anonymization based on consumer's permissions |
| **View** | Permission-filtered representation of data |
| **Slice** | Subset of data limited to specific context |
| **Aggregation** | Statistical summary without individual records |
| **Inference Attack** | Combining permitted data to deduce restricted data |
| **K-anonymity** | Each record indistinguishable from K-1 others |
| **Token Resolution** | Converting anonymized token back to original |

### 1.6 References

- Parent: Bastion-Vault SRS v1.0 (Input/Phase 1)
- Companion: Bastion-Sentinel Output SRS v1.0
- Related: Bastion-Navigator SRS v1.0
- Related: Bastion-Tracker SRS v1.0
- NIST SP 800-188 (De-Identification Guidelines)

---

## 2. Overall Description

### 2.1 Position in Pipeline

```
┌──────────────────────────────────────────────────────┐
│              User Query                              │
└────────────────────────┬─────────────────────────────┘
                         ▼
        ┌────────────────────────────────────┐
        │   Sentinel-IN (Input Validation)   │
        └────────────────┬───────────────────┘
                         ▼
        ┌────────────────────────────────────┐
        │   Vault-IN (Anonymization)         │
        │   - Phase 1                        │
        └────────────────┬───────────────────┘
                         ▼
        ┌────────────────────────────────────┐
        │   Navigator (Search)               │
        └────────────────┬───────────────────┘
                         ▼
        ┌────────────────────────────────────┐
        │   Anchor-IN (Embedding Security)   │
        └────────────────┬───────────────────┘
                         ▼
                        LLM
                         ▼
        ┌────────────────────────────────────┐
        │   Anchor-OUT (Bias Check)          │
        └────────────────┬───────────────────┘
                         ▼
        ┌────────────────────────────────────┐
        │   Vault-OUT  ◄── (This doc)        │
        │   - Phase 2                        │
        │   - Permission Re-application      │
        │   - K-anonymity Verification       │
        │   - Inference Attack Detection     │
        └────────────────┬───────────────────┘
                         ▼
        ┌────────────────────────────────────┐
        │   Sentinel-OUT (Output Validation) │
        └────────────────┬───────────────────┘
                         ▼
        ┌────────────────────────────────────┐
        │           User Response            │
        └────────────────────────────────────┘
```

### 2.2 Unified Vault Architecture

Vault operates as a single service with two phases:

```
┌─────────────────────────────────────────────────────┐
│              Unified Vault Service                  │
├─────────────────────────────────────────────────────┤
│                                                     │
│  ┌──────────────────────────────────────────────┐   │
│  │           Vault Engine (Shared)              │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────────┐  │   │
│  │  │ Encrypt/ │ │ Tokenize │ │  Hash/HMAC   │  │   │
│  │  │ Decrypt  │ │ Manager  │ │   Engine     │  │   │
│  │  └──────────┘ └──────────┘ └──────────────┘  │   │
│  └──────────┬───────────────────────────────────┘   │
│             │                                       │
│  ┌──────────┴────────────────────────────────┐      │
│  │     Configuration & Policies              │      │
│  │  ┌────────────────────────────────────┐   │      │
│  │  │ Phase 1 Config (Input)             │   │      │
│  │  │ - Anonymization Strategies         │   │      │
│  │  │ - PII Type Mappings                │   │      │
│  │  │ - Storage Encryption               │   │      │
│  │  └────────────────────────────────────┘   │      │
│  │  ┌────────────────────────────────────┐   │      │
│  │  │ Phase 2 Config (Output) ⭐         │   │      │
│  │  │ - Access Level Rules               │   │      │
│  │  │ - Permission Matrices              │   │      │
│  │  │ - K-anonymity Parameters           │   │      │
│  │  │ - Slice Definitions                │   │      │
│  │  │ - Inference Attack Detection       │   │      │
│  │  └────────────────────────────────────┘   │      │
│  └───────────────────────────────────────────┘      │
│                                                     │
│  ┌──────────────────────────────────────────────┐   │
│  │       Storage & Integration Layer            │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────────┐  │   │
│  │  │   KMS    │ │ Token DB │ │   OPA Policy │  │   │
│  │  │(AWS/HC)  │ │(Postgres)│ │   Engine     │  │   │
│  │  └──────────┘ └──────────┘ └──────────────┘  │   │
│  └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

### 2.3 Operating Modes

| Mode | Trigger | Configuration | Examples |
|---|---|---|---|
| **Phase 1** | `/v1/vault/anonymize` | Input Config | Storage-time anonymization |
| **Phase 2** | `/v1/vault/apply-permissions` | Output Config ⭐ | Read-time view generation |
| **Auto** | `/v1/vault/process` | Mode in request | API user choice |

### 2.4 Output Functions

1. **F1: Permission-based View Generation** ⭐ (Core Function)
   - Generate user-specific data views
   - Apply access level rules
   - Six levels: full, read, anonymized, k_anonymized, slice, aggregated

2. **F2: Selective Token Resolution**
   - Resolve tokens to originals if authorized
   - Keep tokens for limited users
   - Mask for unauthorized requesters

3. **F3: K-anonymity Verification on Output Sets**
   - Validate result sets meet K-anonymity
   - Auto-generalize if violated
   - Block release if cannot meet

4. **F4: Slice Access Enforcement**
   - Filter to specific context only
   - Verify slice boundaries
   - Block cross-slice queries

5. **F5: Aggregation Enforcement**
   - Convert detailed data to aggregates
   - Statistics only, no individuals
   - For executives, analysts

6. **F6: Inference Attack Detection** ⭐
   - Detect query sequences that combine to identify
   - Track per-user query history
   - Block suspicious patterns

7. **F7: Cross-tenant Verification**
   - Ensure output data matches user's tenant
   - Double-check after retrieval
   - Final safety net

8. **F8: Audit Trail Generation**
   - Log every disclosure decision
   - Reproducible from logs
   - Compliance reporting

9. **F9: Break-Glass Override Handling**
   - Special handling for emergency access
   - Enhanced auditing
   - Time-bounded permissions

### 2.5 Constraints

- **Language:** Go (consistent with Vault Phase 1)
- **Memory:** Shared with Phase 1 (≤ 2GB total)
- **Latency:** <50ms p95 for permission-based view
- **Throughput:** ≥ 2,000 operations/s
- **Dependencies:** OPA, Token DB, KMS (shared with Phase 1)

### 2.6 Assumptions and Dependencies

**Assumptions:**
- Phase 1 has properly anonymized data at storage
- User authentication completed by upstream (mTLS/JWT)
- Permission information available in request context
- OPA policies are loaded and valid

**External Dependencies:**
- KMS (AWS KMS / HashiCorp Vault / Local) - for token resolution
- Token DB (PostgreSQL) - for token-to-original mappings
- OPA - policy evaluation
- Navigator - for cross-reference with retrieval context
- Tracker - audit event publishing

---

## 3. External Interface Requirements

### 3.1 Interface Overview

Vault-OUT shares interface infrastructure with Vault-IN, distinguished by endpoint/mode.

| Category | Interface | Purpose |
|---|---|---|
| **Input** | gRPC | Internal module calls (Navigator → Vault) |
| **Input** | REST API | External applications |
| **Input** | CLI | Operations, testing |
| **Output** | Permission-filtered data | To Sentinel-OUT → User |
| **Output** | Events | To Tracker via NATS |

### 3.2 gRPC Interface

```protobuf
syntax = "proto3";
package bastion.vault.v1;

service VaultService {
  // Phase 1 methods (existing)
  rpc Anonymize(AnonymizeRequest) returns (AnonymizeResponse);
  
  // Phase 2 methods (NEW) ⭐
  rpc ApplyPermissions(PermissionRequest) returns (PermissionResponse);
  rpc ResolveTokens(TokenResolutionRequest) returns (TokenResolutionResponse);
  rpc VerifyAccess(AccessVerifyRequest) returns (AccessVerifyResponse);
  rpc ValidateOutputSet(OutputSetRequest) returns (OutputSetResponse);
  rpc DetectInference(InferenceRequest) returns (InferenceResponse);
  
  // Health (shared)
  rpc Health(HealthRequest) returns (HealthResponse);
}

// Apply permissions to retrieved data
message PermissionRequest {
  string request_id = 1;
  string trace_id = 2;
  
  // The retrieved data (potentially anonymized)
  repeated DataRecord records = 3;
  
  // User context
  UserContext user = 4;
  
  // Options
  PermissionOptions options = 5;
}

message DataRecord {
  string record_id = 1;
  string category = 2;        // DC-01, DC-02, DC-03
  string tenant_id = 3;
  map<string, FieldValue> fields = 4;
}

message FieldValue {
  string value = 1;
  string type = 2;            // "original", "token", "masked", "encrypted"
  string pii_type = 3;        // "korean_name", "email", etc.
}

message UserContext {
  string user_id = 1;
  string tenant_id = 2;
  string department = 3;
  repeated string roles = 4;
  repeated string allowed_categories = 5;
  string access_level = 6;
  
  // Slice context (if applicable)
  SliceContext slice = 7;
}

message SliceContext {
  string slice_type = 1;       // "product", "department", etc.
  map<string, string> filters = 2;  // e.g., {"product_id": "PROD-001"}
}

message PermissionOptions {
  bool strict_mode = 1;
  int32 timeout_ms = 2;
  string output_format = 3;
  bool include_audit_info = 4;
  string aggregation_level = 5;  // "none", "department", "demographic"
}

message PermissionResponse {
  string request_id = 1;
  Status status = 2;
  
  // Permission-filtered records
  repeated DataRecord filtered_records = 3;
  
  // Decisions made
  PermissionDecisions decisions = 4;
  
  // Audit information
  AuditInfo audit = 5;
  
  float processing_time_ms = 6;
  
  enum Status {
    UNKNOWN = 0;
    SUCCESS = 1;
    PARTIAL = 2;      // Some records filtered out
    DENIED = 3;       // No access
    GENERALIZED = 4;  // K-anonymity applied
    AGGREGATED = 5;   // Converted to aggregates
  }
}

message PermissionDecisions {
  int32 total_input_records = 1;
  int32 records_returned = 2;
  int32 records_anonymized = 3;
  int32 records_filtered_out = 4;
  int32 records_aggregated = 5;
  string access_level_applied = 6;
  string strategy = 7;
}

message AuditInfo {
  string audit_log_id = 1;
  repeated FieldDisclosure disclosures = 2;
  repeated string warnings = 3;
}

message FieldDisclosure {
  string field_name = 1;
  string original_state = 2;   // "anonymized", "encrypted"
  string final_state = 3;      // "original", "kept_anonymized", "masked"
  string reason = 4;
}

// Token resolution
message TokenResolutionRequest {
  string request_id = 1;
  repeated string tokens = 2;
  UserContext user = 3;
  string purpose = 4;          // Required for audit
}

message TokenResolutionResponse {
  string request_id = 1;
  map<string, ResolutionResult> results = 2;
  string audit_log_id = 3;
}

message ResolutionResult {
  string status = 1;           // "resolved", "denied", "not_found"
  string resolved_value = 2;   // Only if resolved
  string masked_value = 3;     // Alternative for partial access
  string denial_reason = 4;
}

// Inference attack detection
message InferenceRequest {
  string user_id = 1;
  string current_query = 2;
  repeated string current_results = 3;
}

message InferenceResponse {
  bool potential_inference = 1;
  float risk_score = 2;
  string explanation = 3;
  string recommended_action = 4;  // "allow", "warn", "block"
}
```

### 3.3 REST API

**New Endpoints:**

```
# Phase 2 (Output) operations
POST /v1/vault/apply-permissions       # Main permission application
POST /v1/vault/resolve-tokens          # Selective deanonymization
POST /v1/vault/verify-access           # Pre-check access
POST /v1/vault/validate-output-set     # K-anonymity check
POST /v1/vault/detect-inference        # Inference attack check

# Combined endpoint
POST /v1/vault/process                 # Phase 1 or 2 via mode
```

**Request Example - Apply Permissions:**

```http
POST /v1/vault/apply-permissions HTTP/1.1
Host: vault.bastion.local
Content-Type: application/json
Authorization: Bearer <jwt-token>

{
  "request_id": "req-vault-out-001",
  "trace_id": "trace-12345",
  "records": [
    {
      "record_id": "rec-001",
      "category": "customer_data",
      "tenant_id": "tenant-acme",
      "fields": {
        "name": {
          "value": "KR_NAME_8f3d2a",
          "type": "token",
          "pii_type": "korean_name"
        },
        "email": {
          "value": "XXXXX@naver.com",
          "type": "masked",
          "pii_type": "email"
        },
        "purchase_amount": {
          "value": "5000000",
          "type": "original",
          "pii_type": null
        },
        "purchase_date": {
          "value": "2026-03-15",
          "type": "original",
          "pii_type": null
        }
      }
    }
  ],
  "user": {
    "user_id": "user-alice",
    "tenant_id": "tenant-acme",
    "department": "marketing",
    "roles": ["marketing_analyst"],
    "allowed_categories": ["customer_data"],
    "access_level": "k_anonymized"
  },
  "options": {
    "strict_mode": true,
    "aggregation_level": "demographic"
  }
}
```

**Response Example (K-anonymized):**

```json
{
  "request_id": "req-vault-out-001",
  "status": "GENERALIZED",
  "filtered_records": [
    {
      "record_id": "rec-001",
      "category": "customer_data",
      "fields": {
        "name": {
          "value": "[ANONYMIZED]",
          "type": "anonymized",
          "pii_type": "korean_name"
        },
        "email_domain": {
          "value": "naver.com",
          "type": "generalized",
          "pii_type": "email"
        },
        "purchase_range": {
          "value": "4M-6M won",
          "type": "generalized",
          "pii_type": null
        },
        "purchase_month": {
          "value": "2026-03",
          "type": "generalized",
          "pii_type": null
        },
        "age_group": {
          "value": "30s",
          "type": "generalized",
          "pii_type": null
        }
      }
    }
  ],
  "decisions": {
    "total_input_records": 5,
    "records_returned": 5,
    "records_anonymized": 5,
    "records_filtered_out": 0,
    "records_aggregated": 0,
    "access_level_applied": "k_anonymized",
    "strategy": "generalization"
  },
  "audit": {
    "audit_log_id": "audit-789",
    "disclosures": [
      {
        "field_name": "name",
        "original_state": "token",
        "final_state": "anonymized",
        "reason": "marketing_analyst has k_anonymized access"
      },
      {
        "field_name": "purchase_amount",
        "original_state": "original",
        "final_state": "range_generalized",
        "reason": "k_anonymity preservation"
      }
    ],
    "warnings": []
  },
  "processing_time_ms": 12.5
}
```

**Response Example (Full Access for HR Manager):**

```json
{
  "request_id": "req-vault-out-002",
  "status": "SUCCESS",
  "filtered_records": [
    {
      "record_id": "emp-001",
      "category": "hr_finance_data",
      "fields": {
        "name": {
          "value": "Hong Gildong",
          "type": "original",
          "pii_type": "korean_name"
        },
        "salary": {
          "value": "65000000",
          "type": "original",
          "pii_type": null
        }
      }
    }
  ],
  "decisions": {
    "access_level_applied": "full",
    "strategy": "deanonymization"
  }
}
```

### 3.4 CLI Interface

```bash
# Apply permissions to data
$ vault-cli apply-permissions \
    --records records.json \
    --user-id alice \
    --department marketing \
    --access-level k_anonymized

# Output (text format):
══════════════════════════════════════════════════════
  Vault-OUT Permission Application
══════════════════════════════════════════════════════
User:         alice@tenant-acme (marketing_analyst)
Access Level: k_anonymized
Records:      5 input → 5 output (generalized)

─── Field-by-Field Decisions ─────────────────────────
Field            Original         Final Value         Reason
─────────────────────────────────────────────────────
name             KR_NAME_8f3d2a   [ANONYMIZED]        k_anon required
email            XXXXX@naver.com  naver.com domain    domain only
purchase_amount  5,000,000        4M-6M won           range generalization
purchase_date    2026-03-15       2026-03             month only
age              32               30s                 age group

─── K-anonymity Check ────────────────────────────────
K target:     5
Actual K:     5 ✅
Result:       Valid

─── Audit Log ────────────────────────────────────────
ID: audit-789
Decisions logged: 25 (5 records × 5 fields)
══════════════════════════════════════════════════════

# Resolve specific tokens (with permission)
$ vault-cli resolve-tokens \
    --tokens "KR_NAME_8f3d2a,EMAIL_xyz" \
    --user-id charlie \
    --department hr \
    --role hr_manager \
    --purpose "salary review meeting"

# Output:
Token Resolution Results:
─────────────────────────────────
KR_NAME_8f3d2a → "Hong Gildong"  ✅ resolved
EMAIL_xyz      → "[DENIED]"      ❌ insufficient permission

Audit: audit-790 (saved for compliance)

# Verify access before query
$ vault-cli verify-access \
    --user-id bob \
    --department manufacturing \
    --resource hr_finance_data

🚫 ACCESS DENIED
Reason: manufacturing department cannot access hr_finance_data

# Test K-anonymity
$ vault-cli check-k-anon \
    --records output.json \
    --quasi-identifiers age_group,gender,region \
    --k 5

Result:
✅ K-anonymity satisfied (min group size: 7)
   Groups: 12
   Smallest: ("30s", "M", "Seoul") with 7 records

# Detect inference attacks
$ vault-cli check-inference \
    --user-id alice \
    --query "30s male customers in Gangnam premium tier"

⚠️ INFERENCE RISK DETECTED
Risk Score: 0.78
Reason: Combined with recent queries, this would isolate ~1 person
Recommendation: BLOCK
Recent queries:
  1. "30s male customers" (5 minutes ago)
  2. "Seoul Gangnam customers" (3 minutes ago)
  3. "Premium tier customers" (1 minute ago)

# Interactive mode
$ vault-cli interactive
vault> mode output
Mode: output (Phase 2)

vault> apply-permissions
[loads context from previous query]
✅ Applied k_anonymized view
5 records, 5 fields generalized

vault> stats
Today:
  Permissions applied: 1,234
  Tokens resolved: 89
  Tokens denied: 12
  Inference blocks: 3

vault> exit

# Server mode (handles both phases)
$ vault-cli server
```

### 3.5 Text Output Format (for Demos)

```
═══════════════════════════════════════════════════════
  Vault-OUT: Permission Application Report
═══════════════════════════════════════════════════════
Trace ID:     trace-12345
User:         alice@tenant-acme
Role:         marketing_analyst
Access:       k_anonymized

─── Input Data ──────────────────────────────────────
Record: customer-rec-001 (DC-01: Customer)
Anonymized fields:
  name: KR_NAME_8f3d2a (tokenized)
  email: XXXXX@naver.com (masked)
  rrn: RRN_a1b2c3 (hashed)
  
Original-state fields:
  purchase_amount: 5,000,000
  purchase_date: 2026-03-15

─── Permission Decision ─────────────────────────────
Access Level: k_anonymized
Strategy:     Generalization

─── Field-by-Field Transformations ──────────────────

📊 name: KR_NAME_8f3d2a
   ↓ k_anonymized access
   ✅ Result: [ANONYMIZED]
   📝 Reason: Identifiers hidden for analyst

📊 email: XXXXX@naver.com
   ↓ k_anonymized access  
   ✅ Result: naver.com (domain only)
   📝 Reason: Allow domain analysis, hide individuals

📊 rrn: RRN_a1b2c3
   ↓ k_anonymized access (irreversible hash)
   ✅ Result: [HIDDEN]
   📝 Reason: PIPA requires irreversibility

📊 purchase_amount: 5,000,000
   ↓ k_anonymized access
   ✅ Result: 4M-6M range
   📝 Reason: Range generalization for K-anon

📊 purchase_date: 2026-03-15
   ↓ k_anonymized access
   ✅ Result: 2026-03 (month)
   📝 Reason: Reduce date specificity

─── K-anonymity Verification ────────────────────────
Quasi-identifiers: [age_group, gender, region]
Target K:          5
Actual K:          7 ✅
Status:            PASSED

─── Inference Risk Check ────────────────────────────
Recent queries by alice (last 10 min): 3
Combined specificity: Medium
Risk score: 0.45
Recommendation: ALLOW with monitoring

─── Final Output ────────────────────────────────────
Status: GENERALIZED ✅
Records: 1 returned with k_anonymized view
Fields modified: 5 of 5

─── Tracker Event ───────────────────────────────────
Event: vault.permission_applied
Severity: info
Audit log: audit-789
═══════════════════════════════════════════════════════
```

---

## 4. Functional Requirements

### 4.1 Permission-based View Generation (FR-PV)

**FR-PV-001: Six Access Levels**

| Level | Description | Example User |
|---|---|---|
| `full` | Original data, all fields | HR Manager, Marketing Manager |
| `read` | Original data, read-only | Auditor (within scope) |
| `anonymized` | Tokens preserved, partial masking | Marketing Staff |
| `k_anonymized` | Generalized for K-anon | Marketing Analyst |
| `slice` | Context-limited fields only | Manufacturing → product context |
| `aggregated` | Statistics only, no individuals | Executive |

**FR-PV-002: View Generation Algorithm**
- Determine effective access level from user roles + category
- Apply transformations per field
- Validate K-anonymity if applicable
- Generate audit trail

**FR-PV-003: Field-Level Decisions**
- Each field gets independent decision
- Based on: field PII type + user access + category policy
- Document each decision

**FR-PV-004: Composition Rules**
- Most restrictive rule wins
- Slice + K-anon: both apply
- Aggregation overrides others

### 4.2 Selective Token Resolution (FR-TR)

**FR-TR-001: Token-to-Original Mapping**
- Query Token DB for mapping
- Verify token belongs to user's tenant
- Verify user has resolution permission

**FR-TR-002: Conditional Resolution**

```
HR Manager + employee name token → Full resolution
Marketing Manager + customer name token → Full resolution  
Marketing Staff + customer name token → Keep as token
Marketing Analyst + customer name token → Replace with [ANONYMIZED]
Manufacturing + customer name token → Block entirely
```

**FR-TR-003: Bulk Resolution Optimization**
- Batch lookups to Token DB
- Cache permissions (5 min)
- Async resolution where possible

**FR-TR-004: Purpose Recording**
- Every resolution requires `purpose` field
- Logged for compliance audit
- Reviewable by auditor

### 4.3 K-anonymity Verification on Output (FR-KO)

**FR-KO-001: Pre-release Validation**
- Before returning records, verify K-anonymity
- Use category-specific K values:
  - DC-01 Customer: K=5
  - DC-02 Manufacturing: K=3
  - DC-03 HR: K=10

**FR-KO-002: Auto-generalization**
- If violation detected, apply hierarchical generalization
- Generalize most specific QI first
- Iterate until K satisfied

**FR-KO-003: Suppression Fallback**
- If generalization cannot achieve K, suppress violating records
- Note in audit log

**FR-KO-004: K-anonymity Strategies**

```yaml
strategies:
  customer_data:
    quasi_identifiers: [age_group, gender, region, membership]
    k: 5
    generalization_hierarchy:
      age: [exact, 5_year, 10_year, decade, adult]
      region: [dong, gu, si, province, country]
  
  hr_finance:
    quasi_identifiers: [department, position, age_group, gender, tenure]
    k: 10
    generalization_hierarchy:
      department: [team, division, function, all]
      tenure: [year, 5_year_range, decade]
```

### 4.4 Slice Access Enforcement (FR-SA)

**FR-SA-001: Slice Context Validation**
- Verify user's slice context is valid
- Validate slice fields match policy

**FR-SA-002: Field Filtering by Slice**
- Manufacturing → Customer (product slice):
  - Allow: product_id, complaint_type, region (generalized), date (month)
  - Block: name, contact, address, payment

**FR-SA-003: Cross-Slice Prevention**
- Detect queries spanning multiple slices
- Block aggregation across slices

### 4.5 Aggregation Enforcement (FR-AE)

**FR-AE-001: Aggregation Levels**

```yaml
aggregation_levels:
  none: Individual records
  demographic: Grouped by demographics (age, gender)
  department: Grouped by department
  geographic: Grouped by region
  temporal: Grouped by time period
  
roles_to_aggregation:
  executive: demographic, department, geographic
  marketing_aggregated: demographic
  hr_aggregated: department
```

**FR-AE-002: Statistical Computation**
- Sum, Average, Count, Min, Max
- Distribution metrics
- Hide individuals

**FR-AE-003: Minimum Group Size**
- Aggregations require min N (default 10)
- Smaller groups suppressed
- Privacy preservation

### 4.6 Inference Attack Detection (FR-IA)

**FR-IA-001: Query History Tracking**
- Track last N queries per user (default 50)
- Time window (default 1 hour)
- Persist to Redis cache

**FR-IA-002: Query Combination Analysis**
- Detect intersecting result sets
- Calculate combined specificity
- Identify potential isolation

**FR-IA-003: Risk Scoring**

```
Risk Score Factors:
- Query specificity (more QIs = higher risk)
- Result set size (smaller = higher risk)  
- Query frequency (rapid queries = suspicious)
- User pattern (analyst vs unusual)

Risk Levels:
0.0 - 0.3: Allow
0.3 - 0.6: Warn (log + notify)
0.6 - 0.8: Generalize further
0.8 - 1.0: Block + alert
```

**FR-IA-004: Mitigation Actions**
- Apply additional generalization
- Inject noise into results
- Require break-glass for further queries
- Alert security team

### 4.7 Cross-tenant Verification (FR-CT)

**FR-CT-001: Final Tenant Check**
- Before returning, verify all records belong to user's tenant
- Even if Phase 1 + Navigator passed, re-verify
- Last line of defense

**FR-CT-002: Cross-tenant Block**
- Reject if mismatch found
- Critical security event to Tracker
- Alert security team

### 4.8 Audit Trail Generation (FR-AT)

**FR-AT-001: Per-decision Logging**
- Every field disclosure logged
- Reason captured
- Reproducible chain

**FR-AT-002: Audit Log Schema**

```json
{
  "audit_id": "audit-789",
  "timestamp": "2026-05-17T14:23:45Z",
  "user": "alice@tenant-acme",
  "request_id": "req-vault-out-001",
  "trace_id": "trace-12345",
  
  "input_summary": {
    "records": 5,
    "total_fields": 25
  },
  
  "permissions": {
    "access_level": "k_anonymized",
    "applied_strategy": "generalization"
  },
  
  "decisions": [
    {
      "record_id": "rec-001",
      "field": "name",
      "input_state": "tokenized",
      "output_state": "anonymized",
      "transformation": "token_to_anonymous",
      "reason": "k_anonymized access level"
    }
    // ... more decisions
  ],
  
  "k_anonymity": {
    "verified": true,
    "actual_k": 7,
    "target_k": 5
  },
  
  "inference_check": {
    "score": 0.45,
    "action": "allow"
  }
}
```

**FR-AT-003: Audit Retention**
- 5 years minimum (compliance)
- Immutable storage
- Searchable

### 4.9 Break-Glass Handling (FR-BG)

**FR-BG-001: Override Detection**
- Detect break-glass tokens in request
- Verify approval status
- Time validity check

**FR-BG-002: Enhanced Audit**
- Every action under break-glass logged
- Real-time alerts
- Post-incident review trigger

**FR-BG-003: Restricted Operations**
- Even break-glass has limits
- Cannot cross tenants
- Cannot override certain policies (e.g., RRN exposure)

---

## 5. Non-Functional Requirements

### 5.1 Performance (NFR-PE)

| ID | Item | Target |
|---|---|---|
| NFR-PE-001 | Permission application (p95) | < 50ms |
| NFR-PE-002 | Token resolution (single) | < 10ms |
| NFR-PE-003 | Bulk resolution (100 tokens) | < 100ms |
| NFR-PE-004 | K-anonymity verification | < 30ms (typical set) |
| NFR-PE-005 | Inference detection | < 20ms |
| NFR-PE-006 | Throughput | ≥ 2,000 ops/s |
| NFR-PE-007 | Cache hit rate | > 80% (permissions) |

### 5.2 Reliability (NFR-RE)

| ID | Item | Target |
|---|---|---|
| NFR-RE-001 | Availability | 99.99% |
| NFR-RE-002 | False access grants | 0% (fail-safe) |
| NFR-RE-003 | False access denials | < 1% (operationally) |
| NFR-RE-004 | Data integrity | 100% |

### 5.3 Security (NFR-SE)

| ID | Item | Requirement |
|---|---|---|
| NFR-SE-001 | Cannot bypass Phase 2 | Mandatory for all reads |
| NFR-SE-002 | Audit completeness | 100% of decisions logged |
| NFR-SE-003 | Token DB encryption | At-rest encryption |
| NFR-SE-004 | KMS access | Required for resolution |
| NFR-SE-005 | Inference attack resistance | Detection + mitigation |
| NFR-SE-006 | Side-channel resistance | Constant-time operations |

### 5.4 Compliance (NFR-CO)

| Requirement | Phase 2 Implementation |
|---|---|
| PIPA Art. 21 (Restriction) | Access-level enforcement |
| PIPA Art. 22 (Provision limits) | Permission-based views |
| GDPR Art. 15 (Right of access) | Audit log accessible |
| GDPR Art. 25 (Privacy by design) | Default to most restrictive |
| SOC 2 (Confidentiality) | Comprehensive controls |

---

## 6. System Architecture

### 6.1 Unified Vault Architecture (Updated)

```
┌─────────────────────────────────────────────────────────┐
│                Unified Vault Service                     │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────────┐    ┌──────────────┐                  │
│  │ Phase 1 API  │    │ Phase 2 API  │ ⭐                │
│  │ /anonymize   │    │ /apply-perms │                  │
│  │              │    │ /resolve     │                  │
│  └──────┬───────┘    └──────┬───────┘                  │
│         │                    │                          │
│         └──────────┬─────────┘                         │
│                    ▼                                    │
│      ┌──────────────────────────────────┐              │
│      │     Vault Engine (Shared)        │              │
│      │  Encryption/Decryption           │              │
│      │  Tokenization/Resolution         │              │
│      │  Hashing/HMAC                    │              │
│      └──────────────┬───────────────────┘              │
│                     │                                   │
│      ┌──────────────┴───────────────────┐              │
│      │     Phase 2 Components ⭐        │              │
│      │  ┌──────────────────────────┐   │              │
│      │  │  Permission Resolver      │   │              │
│      │  │  - Role → Access Level    │   │              │
│      │  │  - Category Check         │   │              │
│      │  └──────────────────────────┘   │              │
│      │  ┌──────────────────────────┐   │              │
│      │  │  View Generator           │   │              │
│      │  │  - Field Transformer      │   │              │
│      │  │  - Strategy Selector      │   │              │
│      │  └──────────────────────────┘   │              │
│      │  ┌──────────────────────────┐   │              │
│      │  │  K-anonymity Verifier    │   │              │
│      │  │  - Group Analysis         │   │              │
│      │  │  - Auto-generalization    │   │              │
│      │  └──────────────────────────┘   │              │
│      │  ┌──────────────────────────┐   │              │
│      │  │  Inference Detector       │   │              │
│      │  │  - Query History          │   │              │
│      │  │  - Risk Scoring           │   │              │
│      │  └──────────────────────────┘   │              │
│      │  ┌──────────────────────────┐   │              │
│      │  │  Audit Builder            │   │              │
│      │  │  - Decision Logger        │   │              │
│      │  │  - Trail Generator        │   │              │
│      │  └──────────────────────────┘   │              │
│      └──────────────────────────────────┘              │
│                                                          │
│      ┌──────────────────────────────────┐              │
│      │     Storage & Integration        │              │
│      │  KMS │ Token DB │ Redis │ OPA    │              │
│      └──────────────────────────────────┘              │
└─────────────────────────────────────────────────────────┘
```

### 6.2 Component Description

| Component | Responsibility |
|---|---|
| **Permission Resolver** | Determine user's effective access level |
| **View Generator** | Apply transformations per field |
| **K-anonymity Verifier** | Validate output sets meet K |
| **Inference Detector** | Track and block attack patterns |
| **Audit Builder** | Generate compliance-ready logs |
| **Token Resolver** | Selective deanonymization |

### 6.3 Data Flow

```
Phase 2 (Output) Flow:

[Retrieved Records] → Vault-OUT API
                          ↓
                    Permission Resolver
                          ↓
                    [Determine Access Level]
                          ↓
                    Parallel Processing:
                    ├─ View Generator (per field)
                    ├─ Token Resolver (if authorized)
                    └─ Cross-tenant Check
                          ↓
                    K-anonymity Verifier
                          ↓
                    [Generalize if needed]
                          ↓
                    Inference Detector
                          ↓
                    [Block or warn if risk]
                          ↓
                    Audit Builder
                          ↓
                    [Generate audit log]
                          ↓
                    [Return filtered records]
                          ↓
                    Tracker Event Publishing
```

---

## 7. Implementation Details

### 7.1 Permission Resolution

```go
type PermissionResolver struct {
    opaClient *OPAClient
    cache     *Cache
}

func (r *PermissionResolver) Resolve(
    user UserContext,
    category string,
) (*EffectivePermission, error) {
    // Check cache first
    cacheKey := fmt.Sprintf("%s:%s:%s", user.UserID, category, user.AccessLevel)
    if cached, ok := r.cache.Get(cacheKey); ok {
        return cached.(*EffectivePermission), nil
    }
    
    // Query OPA
    decision, err := r.opaClient.Evaluate(OPAQuery{
        User:     user,
        Category: category,
        Action:   "read",
    })
    
    if err != nil {
        return nil, err
    }
    
    perm := &EffectivePermission{
        AccessLevel:   decision.AccessLevel,
        AllowedFields: decision.AllowedFields,
        Conditions:    decision.Conditions,
    }
    
    r.cache.Set(cacheKey, perm, 5*time.Minute)
    return perm, nil
}
```

### 7.2 View Generation

```go
type ViewGenerator struct {
    tokenResolver *TokenResolver
    config        *Phase2Config
}

func (g *ViewGenerator) Generate(
    record DataRecord,
    permission EffectivePermission,
) (*DataRecord, []FieldDisclosure, error) {
    output := &DataRecord{
        RecordID: record.RecordID,
        Category: record.Category,
        Fields:   make(map[string]FieldValue),
    }
    
    var disclosures []FieldDisclosure
    
    for fieldName, fieldValue := range record.Fields {
        transformed, disclosure := g.transformField(
            fieldName, fieldValue, permission)
        
        output.Fields[fieldName] = transformed
        disclosures = append(disclosures, disclosure)
    }
    
    return output, disclosures, nil
}

func (g *ViewGenerator) transformField(
    name string,
    value FieldValue,
    perm EffectivePermission,
) (FieldValue, FieldDisclosure) {
    switch perm.AccessLevel {
    case "full":
        if value.Type == "token" {
            // Resolve token to original
            original, _ := g.tokenResolver.Resolve(value.Value)
            return FieldValue{
                Value: original,
                Type:  "original",
            }, FieldDisclosure{
                FieldName:    name,
                OriginalState: value.Type,
                FinalState:   "original",
                Reason:       "full access granted",
            }
        }
        return value, ...
    
    case "anonymized":
        // Keep tokens, partial mask
        return value, ...
    
    case "k_anonymized":
        // Apply generalization
        return g.generalize(name, value), ...
    
    case "slice":
        // Check if field allowed in slice
        if !perm.IsFieldAllowed(name) {
            return FieldValue{
                Value: "[NOT_IN_SLICE]",
                Type:  "filtered",
            }, ...
        }
        return value, ...
    
    case "aggregated":
        // This shouldn't happen per-record
        // Aggregation handled at set level
        return FieldValue{Value: "[AGGREGATED]"}, ...
    }
}
```

### 7.3 K-anonymity Verification on Output

```go
type KAnonVerifier struct {
    config *Phase2Config
}

func (v *KAnonVerifier) Verify(
    records []DataRecord,
    quasiIdentifiers []string,
    k int,
) (*KAnonResult, error) {
    // Group records by QI values
    groups := make(map[string][]DataRecord)
    for _, record := range records {
        key := v.buildKey(record, quasiIdentifiers)
        groups[key] = append(groups[key], record)
    }
    
    // Find violating groups
    var violations []ViolatingGroup
    for key, group := range groups {
        if len(group) < k {
            violations = append(violations, ViolatingGroup{
                Key:     key,
                Size:    len(group),
                Records: group,
            })
        }
    }
    
    if len(violations) == 0 {
        return &KAnonResult{
            Valid:    true,
            ActualK:  v.minGroupSize(groups),
        }, nil
    }
    
    // Auto-generalize
    generalized, err := v.autoGeneralize(records, violations, k)
    if err != nil {
        return nil, err
    }
    
    return &KAnonResult{
        Valid:           false,
        OriginalRecords: records,
        FixedRecords:    generalized,
        Action:          "generalized",
    }, nil
}
```

### 7.4 Inference Attack Detection

```go
type InferenceDetector struct {
    history *QueryHistory
}

func (d *InferenceDetector) Check(
    user string,
    currentQuery string,
    currentResults []DataRecord,
) (*InferenceResult, error) {
    // Get recent queries
    recent := d.history.GetRecent(user, 1*time.Hour, 50)
    
    // Calculate intersection specificity
    intersectionSize := d.estimateIntersection(currentResults, recent)
    
    // Calculate combined specificity
    combinedQIs := d.extractAllQIs(currentQuery, recent)
    
    // Risk scoring
    score := d.calculateRiskScore(
        intersectionSize,
        len(combinedQIs),
        len(currentResults),
        recent.QueryRate(),
    )
    
    var action string
    switch {
    case score < 0.3:
        action = "allow"
    case score < 0.6:
        action = "warn"
    case score < 0.8:
        action = "generalize"
    default:
        action = "block"
    }
    
    // Record this query
    d.history.Record(user, currentQuery, currentResults)
    
    return &InferenceResult{
        RiskScore:          score,
        Action:             action,
        Explanation:        d.buildExplanation(score),
    }, nil
}
```

---

## 8. Configuration Schema

```yaml
# /etc/bastion-vault/phase2-config.yaml
version: 1.0

mode: output  # phase 2

# Permission system
permissions:
  opa:
    endpoint: http://opa:8181
    policy_package: bastion.vault.access
  
  cache:
    ttl: 5m
    max_size: 10000

# Access level definitions
access_levels:
  full:
    description: "Original data, full access"
    can_resolve_tokens: true
    can_view_pii: true
  
  read:
    description: "Read-only original data"
    can_resolve_tokens: true
    can_view_pii: true
    can_modify: false
  
  anonymized:
    description: "Tokens preserved"
    can_resolve_tokens: false
    can_view_pii: false
    show_tokens_as_is: true
  
  k_anonymized:
    description: "K-anonymity applied"
    can_resolve_tokens: false
    can_view_pii: false
    apply_generalization: true
  
  slice:
    description: "Context-limited access"
    fields_allowed_by_context: true
    apply_k_anonymity: true
  
  aggregated:
    description: "Statistics only"
    individual_records: false
    minimum_group_size: 10

# Category-specific rules
categories:
  customer_data:
    quasi_identifiers:
      - age_group
      - gender
      - region
      - membership_grade
    k_value: 5
    
    field_policies:
      name:
        full: "original"
        anonymized: "token_kept"
        k_anonymized: "[ANONYMIZED]"
        slice: "[ANONYMIZED]"
        aggregated: "[NOT_RELEASED]"
      
      email:
        full: "original"
        anonymized: "masked"
        k_anonymized: "domain_only"
        slice: "domain_only"
        aggregated: "[NOT_RELEASED]"
      
      purchase_amount:
        full: "original"
        anonymized: "original"
        k_anonymized: "range_5m"
        slice: "range_5m"
        aggregated: "sum_only"
      
      purchase_date:
        full: "original"
        anonymized: "original"
        k_anonymized: "month_only"
        slice: "month_only"
        aggregated: "year_only"

  manufacturing_data:
    quasi_identifiers:
      - product_category
      - factory_id
      - shift
      - production_month
    k_value: 3
    # ... similar structure
  
  hr_finance_data:
    quasi_identifiers:
      - department
      - position_level
      - age_group
      - gender
      - tenure_years
    k_value: 10
    # ... similar structure

# Generalization hierarchies
generalization:
  age:
    levels:
      - name: exact
        example: "32"
      - name: 5_year
        example: "30-34"
      - name: 10_year
        example: "30-39"
      - name: decade
        example: "30s"
      - name: category
        example: "adult"
  
  region:
    levels:
      - name: dong
        example: "Yeoksam-dong"
      - name: gu
        example: "Gangnam-gu"
      - name: si
        example: "Seoul"
      - name: country
        example: "Korea"
  
  date:
    levels:
      - name: day
        example: "2026-03-15"
      - name: month
        example: "2026-03"
      - name: quarter
        example: "2026-Q1"
      - name: year
        example: "2026"
  
  amount:
    levels:
      - name: exact
        example: "5,000,000"
      - name: range_5m
        example: "5M-10M"
      - name: range_10m
        example: "0-10M"
      - name: category
        example: "high"

# Inference attack detection
inference_detection:
  enabled: true
  
  history:
    window: 1h
    max_queries: 50
    storage: redis
  
  scoring:
    weights:
      query_specificity: 0.3
      result_set_size: 0.3
      query_frequency: 0.2
      pattern_anomaly: 0.2
  
  thresholds:
    warn: 0.3
    generalize: 0.6
    block: 0.8

# Token resolution
token_resolution:
  cache_ttl: 5m
  require_purpose: true
  audit_all_resolutions: true
  
  batch_optimization:
    max_batch_size: 100
    parallel_lookups: true

# Cross-tenant verification
cross_tenant:
  always_verify: true
  block_on_mismatch: true
  alert_security_team: true

# Audit
audit:
  log_all_decisions: true
  retention_days: 1825  # 5 years
  storage: postgresql
  immutable: true

# Break-glass
break_glass:
  enhanced_audit: true
  real_time_alerts: true
  
  even_break_glass_blocks:
    - cross_tenant_access
    - rrn_exposure  # PIPA requires irreversibility
```

---

## 9. Standalone Testing

### 9.1 Operation Modes

```bash
# Test mode without external dependencies
$ vault-cli server --standalone --mode output

# Mock OPA decisions
$ vault-cli server --mock-opa

# Demo mode
$ vault-cli demo --scenario permission-application
```

### 9.2 Test Data

```
tests/output/
├── fixtures/
│   ├── records_customer.jsonl
│   ├── records_hr.jsonl
│   ├── records_manufacturing.jsonl
│   └── access_scenarios.jsonl
├── permissions/
│   ├── marketing_analyst.json
│   ├── hr_manager.json
│   └── manufacturing_qc.json
├── inference/
│   └── attack_scenarios.jsonl
└── golden/
    └── expected_outputs/
```

### 9.3 Demo Scenarios (for Tracker)

```yaml
scenario: "Permission Application by Role"
description: "Same data, different views for different users"
steps:
  - Setup: Single customer record (anonymized in Phase 1)
  - User 1: Marketing Manager
    → View: Full original data
  - User 2: Marketing Staff
    → View: Tokens preserved, email masked
  - User 3: Marketing Analyst
    → View: K-anonymized, generalized
  - User 4: Manufacturing Engineer (slice access)
    → View: Product-related fields only
  - User 5: Executive
    → View: Aggregated statistics only
visualization:
  - Show same source data
  - Show 5 different views side-by-side
  - Highlight differences

scenario: "Inference Attack Detection"
description: "Detect query combination attacks"
steps:
  - User issues query 1: "30s male customers" → 50 results
  - User issues query 2: "Seoul Gangnam customers" → 80 results
  - User issues query 3: "Premium tier customers" → 30 results
  - Intersection estimated: ~3 people
  - Vault-OUT: Risk score 0.85 → BLOCK
  - User receives: "Query pattern flagged, please contact admin"
visualization:
  - Show query history accumulation
  - Show intersection visualization (Venn diagram)
  - Show risk score increasing
  - Show final block

scenario: "K-anonymity Auto-generalization"
description: "Result set fails K-anon, auto-fixes"
steps:
  - Query returns 15 records
  - K-anon check (K=5): 3 groups violate
  - Auto-generalize age: "32" → "30s"
  - Re-check: 2 groups still violate
  - Auto-generalize region: "Gangnam-dong" → "Gangnam-gu"
  - Re-check: All groups ≥ 5 ✅
  - Return generalized results
```

---

## 10. Tracker Integration

### 10.1 Events Published

```yaml
events:
  - vault.permissions_applied
  - vault.token_resolved
  - vault.token_denied
  - vault.k_anonymity_enforced
  - vault.k_anonymity_failed_to_satisfy
  - vault.inference_attack_detected
  - vault.inference_attack_blocked
  - vault.cross_tenant_violation_prevented
  - vault.aggregation_applied
  - vault.slice_access_filtered
```

### 10.2 Visualization

```
In Tracker Live Flow:

[LLM] → [Anchor-OUT] → [Vault-OUT] → [Sentinel-OUT] → [User]
                          ⭐
                          Detail Panel:
                          - Records in: 5
                          - Records out: 5 (generalized)
                          - Tokens resolved: 0
                          - K-anon: applied (K=5)
                          - Inference risk: 0.45 (allow)
                          - Audit log: audit-789
```

---

## 11. Migration Path

### 11.1 Phased Deployment

**Phase 1: Logging Mode**
- Vault-OUT operates in shadow mode
- Logs decisions without enforcing
- Identify policy gaps

**Phase 2: Soft Enforcement**
- Apply permissions but log violations
- Don't block, but transform
- Monitor user feedback

**Phase 3: Full Enforcement**
- Block unauthorized access
- Enforce K-anonymity strictly
- Production-ready

---

## 12. Appendix

### 12.1 Common Scenarios

**Scenario: HR Manager Reviews Salary**
```
Request:
  user: HR Manager
  record: Employee with anonymized salary
  action: View

Vault-OUT Decision:
  Access level: full
  Strategy: deanonymization
  
Token Resolution:
  salary token → 65,000,000 (decrypted from KMS)
  name token → "Hong Gildong"
  
Audit: Logged for compliance
```

**Scenario: Marketing Analyst Reviews Customer Data**
```
Request:
  user: Marketing Analyst
  records: 100 customers (K-anonymized in storage)
  action: View

Vault-OUT Decision:
  Access level: k_anonymized
  Strategy: keep generalization
  
View:
  Names: [ANONYMIZED]
  Emails: domain only
  Purchase: range 5M-10M
  Age: 30s
  
K-anon: 8 (above K=5) ✅
```

**Scenario: Manufacturing → Customer Complaints (Slice)**
```
Request:
  user: Manufacturing Engineer
  records: Customer complaints for PROD-001
  action: View

Vault-OUT Decision:
  Access level: slice
  Allowed fields: [complaint_type, date, region (city), age_group]
  
View:
  All other fields: [NOT_IN_SLICE]
  Generalized region: "Seoul"
  Generalized date: "2026-03"
  
K-anon: 5 (at minimum) ✅
```

### 12.2 Edge Cases

**Mixed Permission in Single Query**
```
Marketing Manager queries customer + product data:
- Customer fields: full access
- Product fields: read access

Vault-OUT handles each category independently
```

**Empty Result After Filtering**
```
User queries returns 10 records
After permission filter: 0 records (none allowed)

Vault-OUT response:
  status: "no_authorized_records"
  message: "No records match your permissions"
```

**Configuration Mismatch**
```
User has K-anonymized access
Query returns 1 record only

Cannot satisfy K=5 with 1 record
Action: Suppress record
Return: empty set with explanation
```

### 12.3 Demo Walkthrough (3 minutes)

```
0:00-0:30  Setup
  - Show anonymized customer record in DB
  - "Name: KR_NAME_8f3d2a, Salary: encrypted"

0:30-1:00  Marketing Manager View
  - Apply Phase 2 with full access
  - Show original data revealed

1:00-1:30  Marketing Analyst View  
  - Same data, different user
  - Show K-anonymized view
  - Highlight transformations

1:30-2:00  Manufacturing Engineer View
  - Slice access scenario
  - Show only product-related fields

2:00-2:30  Inference Attack
  - Multiple queries
  - Risk score rising
  - Final block

2:30-3:00  Audit Trail
  - Show comprehensive log
  - Compliance ready
```

### 12.4 Roadmap

- v1.1: Differential privacy integration
- v1.2: ML-based permission recommendations
- v1.3: Self-service access requests
- v2.0: Federated permission management

### 12.5 Change History

| Version | Date | Changes |
|---|---|---|
| 0.1 | 2026-05-17 | Initial draft |
| 1.0 | 2026-05-17 | Phase 2 specification complete |

---

## 13. Summary

### What This SRS Adds

```
Before (Phase 1 only):
Vault = Data anonymization at storage
Output = Always anonymized (no differentiation by user)

After (Phase 1 + Phase 2):
Vault = Bidirectional data protection
- Phase 1: Storage anonymization
- Phase 2: Permission-based view generation ⭐

Result: Same data, multiple authorized views
```

### Symmetry with Sentinel

```
Module      Phase 1                  Phase 2
─────────   ──────────────────       ──────────────────
Sentinel    Input validation         Output validation
Vault       Anonymization            Permission re-application
Anchor      Embedding noise          Bias check (planned)
```

### Value Delivered

- ✅ True data utility with security
- ✅ Permission-based access control
- ✅ K-anonymity at output level
- ✅ Inference attack detection
- ✅ Compliance-ready audit trail
- ✅ Operational flexibility

---

**End of Document**
