# Bastion-Vault System Requirements Specification (SRS) v1.0

**Project:** Bastion - RAG Security Governance Framework  
**Module:** Module B - Vault (Data Isolation & Anonymization)  
**Document Version:** 1.0  
**Date:** 2026-05-17  
**Status:** Draft

---

## 1. Introduction

### 1.1 Purpose

This document defines the functional and non-functional requirements for the **Bastion-Vault** module. Vault serves as the data protection core of the RAG (Retrieval-Augmented Generation) pipeline and is responsible for the following four security functions:

1. **Multi-Tenancy Data Isolation** - Preventing data leakage between tenants
2. **Deterministic Anonymization** - PII transformation with consistent tokenization
3. **Access Control** - Department-based RBAC with policy enforcement
4. **Data Residue Prevention** - Clearing PII from memory, logs, and caches

This document provides a reference baseline for development, operations, security, compliance, and QA teams to design, implement, test, and operate the Vault module.

### 1.2 Scope

**In Scope:**
- Data classification into three categories (Customer, Manufacturing, HR/Finance)
- PII detection and anonymization (hybrid approach)
- Multi-KMS abstraction (AWS KMS, HashiCorp Vault, Local Dev)
- Department-based RBAC with OPA policy engine
- K-anonymity validation and automatic generalization
- Break-glass emergency access with audit enforcement
- Auditor role integration for compliance
- Standalone execution and testing environment
- API-based input/output (gRPC, REST)
- Manual input and text output (CLI)

**Out of Scope:**
- Input validation (Module A - Sentinel responsibility)
- Vector search and ranking (Module C - Navigator responsibility)
- Audit log persistent storage (Module D - Tracker responsibility)
- Embedding security (Module E - Anchor responsibility)
- KMS implementation (uses external services)

### 1.3 Definitions and Acronyms

| Term | Definition |
|---|---|
| **Vault** | Module B of the Bastion framework (data isolation layer) |
| **PII** | Personally Identifiable Information |
| **KMS** | Key Management Service |
| **DEK** | Data Encryption Key |
| **KEK** | Key Encryption Key |
| **HMAC** | Hash-based Message Authentication Code |
| **FPE** | Format-Preserving Encryption |
| **RBAC** | Role-Based Access Control |
| **OPA** | Open Policy Agent |
| **K-anonymity** | Privacy protection where each record is indistinguishable from at least K-1 others |
| **Quasi-Identifier** | Field that combined with others enables identification |
| **Break-Glass** | Emergency access mechanism bypassing normal controls |
| **DC** | Data Category (DC-01, DC-02, DC-03) |
| **PIPA** | Personal Information Protection Act (South Korea) |
| **GDPR** | General Data Protection Regulation (EU) |
| **PCI DSS** | Payment Card Industry Data Security Standard |
| **HIPAA** | Health Insurance Portability and Accountability Act |
| **SRS** | Software Requirements Specification |

### 1.4 References

- ISO/IEC/IEEE 29148:2018 - Systems and software engineering — Life cycle processes — Requirements engineering
- NIST Special Publication 800-188 - De-Identifying Government Datasets
- NIST Special Publication 800-57 - Recommendation for Key Management
- ISO/IEC 27001:2022 - Information security management systems
- PIPA (Personal Information Protection Act, South Korea)
- GDPR (General Data Protection Regulation, EU)
- PCI DSS v4.0
- OWASP Top 10 for LLM Applications (2025)

### 1.5 Document Overview

This document is organized as follows:
- Section 2: Overall System Description
- Section 3: External Interfaces
- Section 4: Functional Requirements
- Section 5: Non-Functional Requirements
- Section 6: System Architecture
- Section 7: Data Classification and Categories
- Section 8: KMS Abstraction Layer
- Section 9: Anonymization Strategies
- Section 10: Access Control Model
- Section 11: K-anonymity Implementation
- Section 12: Standalone Testing Environment
- Section 13: Data Requirements
- Section 14: Deployment and Operations
- Section 15: Compliance
- Section 16: Appendix

---

## 2. Overall Description

### 2.1 Product Perspective

Vault is the second module in the Bastion framework's defense pipeline.

```
┌──────────────────────────────────────────────────────┐
│              User / Client Application                │
└────────────────────────┬─────────────────────────────┘
                         │
                         ▼
        ┌────────────────────────────────────┐
        │   Module A: Sentinel                │
        │   - Prompt Injection Detection      │
        │   - Metadata Validation             │
        └────────────────┬───────────────────┘
                         │ (if PASSED)
                         ▼
        ┌────────────────────────────────────┐
        │   Module B: VAULT  ◄── (This doc)   │
        │   - Data Classification             │
        │   - PII Anonymization               │
        │   - Multi-tenant Isolation          │
        │   - Access Control                  │
        │   - K-anonymity Enforcement         │
        └────────────────┬───────────────────┘
                         │
                         ▼
        ┌────────────────────────────────────┐
        │   Module C: Navigator (Search)      │
        └────────────────┬───────────────────┘
                         │
                         ▼
        ┌────────────────────────────────────┐
        │   Module E: Anchor (Embedding Sec.) │
        └────────────────┬───────────────────┘
                         │
                         ▼
        ┌────────────────────────────────────┐
        │              LLM                    │
        └────────────────────────────────────┘
                         │
                         ▼ (async)
        ┌────────────────────────────────────┐
        │   Module D: Tracker (Audit Logs)   │
        └────────────────────────────────────┘
```

**Independence Principle:**
- Vault must operate and be testable independently
- External dependencies (KMS, DB, Module C) failures must not break core functionality
- Local development mode available for all dependencies

### 2.2 Product Functions

Vault's core functions:

1. **F1: Data Classification**
   - Automatic categorization into DC-01, DC-02, DC-03
   - PII detection (hybrid: explicit + ML-assisted)
   - Sensitivity level assignment (L1-L5)

2. **F2: PII Anonymization**
   - Multiple strategies (tokenization, hashing, FPE, masking, generalization)
   - Two-phase execution (at storage + at use)
   - Deterministic mapping for searchability

3. **F3: Multi-tenant Isolation**
   - Logical isolation with strict access boundaries
   - Tenant key separation
   - Cross-tenant access prevention

4. **F4: KMS Abstraction**
   - AWS KMS, HashiCorp Vault, Local provider support
   - Envelope encryption pattern
   - Key rotation and lifecycle management

5. **F5: Access Control**
   - Department-based RBAC
   - OPA policy engine integration
   - Six access levels (Full, Read, Anonymized, K-anonymized, Slice, Aggregated)

6. **F6: K-anonymity Enforcement**
   - Quasi-identifier-based grouping
   - Automatic generalization on K-violation
   - Category-specific K-values (5/3/10)

7. **F7: Break-Glass Access**
   - Emergency override with mandatory audit
   - Dual-approval workflow
   - Time-bounded permissions

8. **F8: Multiple Interfaces**
   - gRPC (system-to-system)
   - REST API (external clients)
   - CLI (manual operations)
   - File I/O (batch processing)

### 2.3 User Characteristics

| User Type | Department | Primary Access | Interface |
|---|---|---|---|
| **Marketing Manager** | Marketing | Full customer data | REST/CLI |
| **Marketing Analyst** | Marketing | K-anonymized data | REST/CLI |
| **Manufacturing Manager** | Manufacturing | Full manufacturing data | REST/CLI |
| **HR Manager** | HR | Full HR/Finance data | REST/CLI |
| **HR Payroll** | HR | Salary data | REST/CLI |
| **Executive** | Leadership | Aggregated statistics | REST |
| **Auditor** | Audit/Compliance | Audit logs and metadata | REST/CLI |
| **AI System** | System | Application context | gRPC |
| **Security Team** | Security | Alert-driven access | REST |
| **IT Operations** | IT | Metadata only | CLI |

### 2.4 Constraints

- **Language:** Go 1.21+
- **Container:** Docker, OCI compliant
- **Orchestration:** Kubernetes 1.28+
- **Operating System:** Linux (Ubuntu 22.04+), macOS (development)
- **Memory Usage:** Maximum 1GB per pod (higher than Sentinel due to encryption operations)
- **Network:** TLS 1.3 required
- **Database:** PostgreSQL 15+ (with RLS support)
- **Compliance:** PIPA, GDPR, PCI DSS, ISO 27001

### 2.5 Assumptions and Dependencies

**Assumptions:**
- Sentinel (Module A) has already validated inputs before reaching Vault
- Authentication is handled by upstream layer (JWT/mTLS)
- KMS service is available with appropriate IAM permissions
- Tenant identifiers are pre-established

**External Dependencies:**
- **KMS Provider** (one of):
  - AWS KMS (production)
  - HashiCorp Vault (multi-cloud/on-premise)
  - Local file-based (development only)
- PostgreSQL 15+ (token mapping storage)
- Redis 7.0+ (token cache, optional)
- OPA 0.60+ (policy evaluation)
- Elasticsearch 8.0+ (audit log forwarding, optional)

---

## 3. External Interface Requirements

### 3.1 Interface Overview

Vault supports **4 input methods** and **4 output formats** (consistent with Sentinel).

| Category | Interface | Target Users |
|---|---|---|
| **Input** | gRPC | AI systems, internal modules |
| **Input** | REST API (JSON) | External applications |
| **Input** | CLI | Developers, operators (manual) |
| **Input** | File input (JSONL/CSV) | Batch processing, QA |
| **Output** | Protobuf | gRPC responses |
| **Output** | JSON | REST responses |
| **Output** | Text | Console, reports (human-readable) |
| **Output** | File output (JSONL/CSV) | Batch results |

### 3.2 Input Interface 1: gRPC (AI/System)

**Protocol:** gRPC over HTTP/2  
**Format:** Protocol Buffers  
**Purpose:** Internal module communication, AI system calls

```protobuf
// vault.proto
syntax = "proto3";

package bastion.vault.v1;

service VaultService {
  // Anonymization operations
  rpc Anonymize(AnonymizeRequest) returns (AnonymizeResponse);
  rpc Deanonymize(DeanonymizeRequest) returns (DeanonymizeResponse);
  rpc BatchAnonymize(BatchRequest) returns (BatchResponse);
  
  // Data access with access control
  rpc AccessData(AccessRequest) returns (AccessResponse);
  rpc QueryData(QueryRequest) returns (stream QueryResponse);
  
  // Classification
  rpc ClassifyData(ClassifyRequest) returns (ClassifyResponse);
  rpc DetectPII(DetectRequest) returns (DetectResponse);
  
  // K-anonymity
  rpc ValidateKAnonymity(KAnonRequest) returns (KAnonResponse);
  
  // Health and metrics
  rpc Health(HealthRequest) returns (HealthResponse);
}

message AnonymizeRequest {
  string request_id = 1;
  string tenant_id = 2;
  DataCategory category = 3;  // DC_01, DC_02, DC_03
  map<string, string> data = 4;
  AnonymizeOptions options = 5;
  
  enum DataCategory {
    UNKNOWN = 0;
    CUSTOMER_DATA = 1;        // DC-01
    MANUFACTURING_DATA = 2;   // DC-02
    HR_FINANCE_DATA = 3;      // DC-03
  }
}

message AnonymizeOptions {
  bool include_metadata = 1;
  string output_format = 2;
  bool strict_mode = 3;
  int32 timeout_ms = 4;
}

message AnonymizeResponse {
  string request_id = 1;
  Status status = 2;
  map<string, AnonymizedField> anonymized_data = 3;
  string token_set_id = 4;
  float processing_time_ms = 5;
  
  enum Status {
    UNKNOWN = 0;
    SUCCESS = 1;
    PARTIAL = 2;
    FAILED = 3;
  }
}

message AnonymizedField {
  string original_type = 1;       // PII type (e.g., "korean_rrn")
  string strategy_used = 2;       // "tokenization", "hashing", etc.
  string anonymized_value = 3;
  bool is_reversible = 4;
  string token_id = 5;            // For deanonymization
}

message AccessRequest {
  string request_id = 1;
  UserContext user = 2;
  string resource_id = 3;
  DataCategory category = 4;
  string action = 5;              // "read", "write", "deanonymize"
}

message UserContext {
  string user_id = 1;
  string department = 2;
  repeated string roles = 3;
  string tenant_id = 4;
}

message AccessResponse {
  string request_id = 1;
  bool allowed = 2;
  string access_level = 3;        // "full", "anonymized", "k_anonymized", etc.
  repeated string conditions = 4; // e.g., "k_anonymity_required"
  string deny_reason = 5;
  string audit_log_id = 6;
}

message KAnonRequest {
  repeated Record records = 1;
  repeated string quasi_identifiers = 2;
  int32 k_value = 3;
}

message KAnonResponse {
  bool valid = 1;
  int32 violating_groups = 2;
  repeated Record generalized_records = 3;
}

message Record {
  map<string, string> fields = 1;
}
```

### 3.3 Input Interface 2: REST API (External Systems)

**Protocol:** HTTPS (TLS 1.3)  
**Format:** JSON  
**Purpose:** External applications, web clients

**Endpoints:**

```
# Anonymization
POST /v1/vault/anonymize              # Anonymize data
POST /v1/vault/deanonymize            # Deanonymize (with permissions)
POST /v1/vault/anonymize/batch        # Batch anonymization

# Data Access
GET  /v1/vault/data/{resource_id}     # Access data with permissions check
POST /v1/vault/query                  # Query with access control

# Classification
POST /v1/vault/classify               # Classify data category
POST /v1/vault/detect-pii             # Detect PII in text

# K-anonymity
POST /v1/vault/k-anonymity/validate   # Validate K-anonymity
POST /v1/vault/k-anonymity/generalize # Auto-generalize

# Break-glass
POST /v1/vault/break-glass/request    # Request emergency access
POST /v1/vault/break-glass/approve    # Approve emergency access

# Operations
GET  /v1/health                       # Health check
GET  /v1/metrics                      # Prometheus metrics
GET  /v1/config                       # Current configuration
POST /v1/config/reload                # Hot reload
```

**Request Example - Anonymize:**

```http
POST /v1/vault/anonymize HTTP/1.1
Host: vault.bastion.local
Content-Type: application/json
Authorization: Bearer <jwt-token>

{
  "request_id": "req-vault-001",
  "tenant_id": "tenant-acme",
  "category": "CUSTOMER_DATA",
  "data": {
    "name": "Hong Gildong",
    "korean_rrn": "850315-1234567",
    "mobile": "010-1234-5678",
    "email": "hong@naver.com",
    "address": "Seoul Gangnam-gu",
    "credit_card": "1234-5678-9012-3456"
  },
  "options": {
    "include_metadata": true,
    "output_format": "json",
    "strict_mode": true
  }
}
```

**Response Example:**

```json
{
  "request_id": "req-vault-001",
  "status": "SUCCESS",
  "timestamp": "2026-05-17T10:30:00.005Z",
  "processing_time_ms": 4.2,
  "token_set_id": "ts-abc123",
  "anonymized_data": {
    "name": {
      "original_type": "korean_name",
      "strategy_used": "deterministic_tokenization",
      "anonymized_value": "KR_NAME_8f3d2a",
      "is_reversible": true,
      "token_id": "tok-001"
    },
    "korean_rrn": {
      "original_type": "korean_rrn",
      "strategy_used": "hmac_sha256",
      "anonymized_value": "RRN_a1b2c3d4e5f6...",
      "is_reversible": false,
      "token_id": null
    },
    "mobile": {
      "original_type": "korean_mobile",
      "strategy_used": "partial_masking",
      "anonymized_value": "010-****-5678",
      "is_reversible": false,
      "token_id": null
    },
    "email": {
      "original_type": "email",
      "strategy_used": "fpe_email",
      "anonymized_value": "XXXXX@naver.com",
      "is_reversible": true,
      "token_id": "tok-002"
    },
    "address": {
      "original_type": "korean_address",
      "strategy_used": "generalization",
      "anonymized_value": "Seoul Gangnam-gu",
      "is_reversible": false,
      "token_id": null
    },
    "credit_card": {
      "original_type": "credit_card",
      "strategy_used": "pci_tokenization",
      "anonymized_value": "1234-****-****-3456",
      "is_reversible": false,
      "token_id": "tok-003"
    }
  }
}
```

**Request Example - Access Control:**

```http
POST /v1/vault/data/customer-12345 HTTP/1.1
Authorization: Bearer <jwt-token>

{
  "user": {
    "user_id": "user-alice",
    "department": "marketing",
    "roles": ["marketing_analyst"],
    "tenant_id": "tenant-acme"
  },
  "action": "read"
}
```

**Response (with K-anonymity applied):**

```json
{
  "status": "ALLOWED",
  "access_level": "k_anonymized",
  "k_value": 5,
  "data": {
    "age_group": "30s",
    "gender": "M",
    "region": "Seoul",
    "membership_grade": "Gold"
  },
  "audit_log_id": "audit-789",
  "conditions_applied": [
    "k_anonymity_k5",
    "generalization_applied"
  ]
}
```

### 3.4 Input Interface 3: CLI (Manual Operations)

**Tool Name:** `vault-cli`

**Basic Commands:**

```bash
# Anonymize single record
$ vault-cli anonymize \
    --category customer_data \
    --tenant tenant-acme \
    --data '{"name":"Hong Gildong","email":"hong@naver.com"}'

# Batch anonymization
$ vault-cli anonymize \
    --input-file customers.jsonl \
    --output-file anonymized.jsonl \
    --category customer_data \
    --parallel 10

# Access data with user context
$ vault-cli access \
    --user-id user-alice \
    --department marketing \
    --role marketing_analyst \
    --resource customer-12345

# Detect PII
$ vault-cli detect-pii \
    --text "My phone is 010-1234-5678 and email hong@naver.com"
# Output:
# Detected PII:
#   - Korean Mobile: 010-1234-5678 (position: 12-25)
#   - Email: hong@naver.com (position: 37-50)

# Classify data
$ vault-cli classify \
    --input '{"employee_id":"EMP001","salary":65000000}'
# Output:
# Category: DC-03 (HR_FINANCE_DATA)
# Confidence: 0.98

# K-anonymity validation
$ vault-cli k-anon-check \
    --input-file dataset.csv \
    --quasi-identifiers age,gender,region \
    --k 5

# Interactive mode (REPL)
$ vault-cli interactive
vault> anonymize
Category: customer_data
Tenant: tenant-acme
Data (JSON): {"name":"Hong Gildong"}
✅ Anonymized: KR_NAME_8f3d2a (3.2ms)

vault> stats
Total operations: 1234
Anonymized: 1200
Deanonymized: 34
K-anon validations: 56
Avg latency: 4.5ms

vault> exit

# Server mode
$ vault-cli server --port 8080 --grpc-port 9090
```

**CLI Options:**

| Option | Description | Example |
|---|---|---|
| `--category` | Data category | `customer_data`, `manufacturing_data`, `hr_finance_data` |
| `--tenant` | Tenant ID | `--tenant tenant-acme` |
| `--user-id` | User performing action | `--user-id user-alice` |
| `--department` | User department | `--department marketing` |
| `--role` | User role | `--role marketing_analyst` |
| `--input-file` | Input file (JSONL/CSV) | `--input-file data.jsonl` |
| `--output-file` | Output file | `--output-file result.jsonl` |
| `--output-format` | Output format | `text`, `json`, `compact`, `yaml` |
| `--strict-mode` | Strict validation | `--strict-mode` |
| `--break-glass` | Emergency access | `--break-glass --reason "incident-123"` |
| `--config` | Configuration file | `--config /etc/vault.yaml` |
| `--verbose` | Verbose logging | `-v`, `-vv`, `-vvv` |

### 3.5 Output Format 1: Structured Data (AI/System)

Returns Protobuf (gRPC) or JSON (REST) as defined in 3.2 and 3.3.

### 3.6 Output Format 2: Text (Human-Readable)

**Text format for anonymization result:**

```
════════════════════════════════════════════════════════
  Bastion-Vault Anonymization Report
════════════════════════════════════════════════════════
Request ID:      req-vault-001
Tenant:          tenant-acme
Category:        DC-01 (Customer Data)
Timestamp:       2026-05-17 10:30:00.005
Processing Time: 4.2 ms
Status:          ✅ SUCCESS

─── Anonymized Fields ──────────────────────────────────
Field            Type             Strategy          Reversible
─────────────────────────────────────────────────────────
name             korean_name      Tokenization      ✅
korean_rrn       korean_rrn       HMAC-SHA256       ❌
mobile           korean_mobile    Partial Mask      ❌
email            email            FPE (domain)      ✅
address          korean_address   Generalization    ❌
credit_card      credit_card      PCI Tokenization  ❌

─── Token Set ──────────────────────────────────────────
Token Set ID:    ts-abc123
Mappings stored: 3 (name, email - encrypted in token DB)

─── Compliance Check ───────────────────────────────────
PIPA:            ✅ Compliant (RRN irreversibly hashed)
GDPR:            ✅ Compliant (right to erasure supported)
PCI DSS:         ✅ Compliant (credit card tokenized)

─── Next Steps ─────────────────────────────────────────
→ Data stored in customer_data partition
→ Audit log forwarded to Module D (Tracker)
════════════════════════════════════════════════════════
```

**Text format for access denial:**

```
════════════════════════════════════════════════════════
  Bastion-Vault Access Decision
════════════════════════════════════════════════════════
Request ID:      req-access-789
Requestor:       user-bob (Manufacturing)
Resource:        customer-12345 (DC-01)
Action:          read
Decision:        🚫 DENIED

─── Reason ─────────────────────────────────────────────
Department 'manufacturing' is not authorized to access 
'customer_data' resources.

─── Allowed Categories ─────────────────────────────────
✅ DC-02: Manufacturing Data (Full)
🟣 DC-01: Customer Data (Slice only - product claims)
❌ DC-03: HR/Finance Data (No access)

─── Suggested Action ───────────────────────────────────
Use slice access for product-related customer data:
$ vault-cli access \\
    --resource customer-claims-by-product \\
    --slice-context product_id=PROD-001
════════════════════════════════════════════════════════
```

**Compact Format:**

```
[req-001] anonymized category=customer_data fields=6 time=4.2ms tokens=3
[req-002] DENIED user=bob dept=manufacturing resource=customer-12345
[req-003] k_anonymity_check k=5 violations=0 records=1500
```

### 3.7 Input/Output File Formats

**JSONL Input:**
```jsonl
{"tenant_id":"acme","category":"customer_data","data":{"name":"Hong","email":"hong@naver.com"}}
{"tenant_id":"acme","category":"customer_data","data":{"name":"Kim","email":"kim@gmail.com"}}
```

**CSV Input:**
```csv
tenant_id,category,name,email,mobile
acme,customer_data,Hong Gildong,hong@naver.com,010-1234-5678
acme,customer_data,Kim Cheolsoo,kim@gmail.com,010-9876-5432
```

### 3.8 Monitoring Interface

**Prometheus Metrics:**

```
# Anonymization operations
vault_anonymization_total{category="customer_data",status="success"} 12345
vault_anonymization_total{category="hr_finance_data",status="success"} 567
vault_anonymization_duration_seconds_bucket{le="0.005"} 12000

# Access control
vault_access_decisions_total{decision="allowed",department="marketing"} 8900
vault_access_decisions_total{decision="denied",department="manufacturing"} 234

# K-anonymity
vault_kanon_validations_total{result="valid"} 1234
vault_kanon_violations_total 12
vault_kanon_generalizations_total 45

# KMS operations
vault_kms_calls_total{provider="aws_kms",operation="encrypt"} 5678
vault_kms_call_duration_seconds_bucket{le="0.01"} 5600

# Break-glass
vault_break_glass_requests_total 3
vault_break_glass_approved_total 2
```

---

## 4. Functional Requirements

### 4.1 Data Classification (FR-DC)

**FR-DC-001: Automatic Category Detection**
- Detect data category from structure and content
- Categories: DC-01 (Customer), DC-02 (Manufacturing), DC-03 (HR/Finance)
- Confidence threshold: 0.8

**FR-DC-002: Manual Category Assignment**
- Accept category in request metadata
- Override automatic detection if specified

**FR-DC-003: PII Type Detection (Hybrid)**
- Explicit field mapping (priority)
- Regex pattern detection (secondary)
- ML model detection (tertiary, for unstructured text)

**FR-DC-004: Sensitivity Level Assignment**
- L1: Direct identifier (RRN, SSN, email)
- L2: Indirect identifier (name, phone)
- L3: Sensitive personal (medical, religion)
- L4: Quasi-identifier (age, gender, region)
- L5: Public (employee ID, department)

### 4.2 PII Anonymization (FR-AN)

**FR-AN-001: Tokenization**
- Deterministic tokenization for searchable fields
- Reversible with token DB lookup
- Format: `{PREFIX}_{16_chars}` (e.g., `KR_NAME_8f3d2a4b1c9e5d7f`)

**FR-AN-002: HMAC-SHA256 Hashing**
- Irreversible hashing for RRN, SSN
- Tenant-scoped HMAC keys
- Required by PIPA, HIPAA

**FR-AN-003: Format-Preserving Encryption (FPE)**
- Email: preserve domain (`XXXXX@naver.com`)
- Credit card: preserve format (4 groups of 4 digits)
- Phone: preserve format (`010-****-1234`)

**FR-AN-004: Partial Masking**
- Mobile: `010-****-5678` (first/last 4)
- Credit card: `1234-****-****-3456`
- SSN: `***-**-6789`

**FR-AN-005: Generalization**
- Age: exact → range (e.g., 32 → "30-39")
- Address: detailed → city/district
- Date: exact date → year-month or quarter

**FR-AN-006: Suppression**
- Replace with `[REMOVED]` for high re-identification risk
- Applied to sensitive personal data (race, religion)

**FR-AN-007: Two-Phase Anonymization**
- **Phase 1 (Storage):** Anonymize before persistence
- **Phase 2 (Use):** Re-apply based on requester's permissions
- Token mapping preserved across phases

### 4.3 Multi-Tenancy Isolation (FR-MT)

**FR-MT-001: Logical Isolation**
- Database row-level security (RLS) by tenant_id
- Separate schemas per data category
- Mandatory tenant_id in all queries

**FR-MT-002: Tenant Key Separation**
- Unique encryption keys per tenant
- KMS-based key derivation
- No cross-tenant key access

**FR-MT-003: Cross-Tenant Prevention**
- Reject any request where user.tenant_id != resource.tenant_id
- Audit log all rejection attempts
- Alert security team on repeated violations

**FR-MT-004: Tenant Lifecycle**
- Tenant creation with key generation
- Tenant suspension (read-only mode)
- Tenant deletion (key destruction + data purge)

### 4.4 Access Control (FR-AC)

**FR-AC-001: RBAC Model**
- Roles per department (marketing, manufacturing, hr)
- Role hierarchy (manager > staff > analyst)
- Multiple roles per user supported

**FR-AC-002: OPA Policy Engine**
- Policy-as-code with Rego
- Hot reload of policies
- Decision logging

**FR-AC-003: Access Levels**
- **Full**: Original data access (with deanonymization)
- **Read**: Original data read-only
- **Anonymized**: De-identified data
- **K-anonymized**: Group-protected data
- **Slice**: Context-limited access
- **Aggregated**: Statistics only

**FR-AC-004: Department-Based Access Matrix**

| Department | DC-01 Customer | DC-02 Manufacturing | DC-03 HR/Finance |
|---|---|---|---|
| Marketing | Full/Anonymized/K-anon | Read | Aggregated |
| Manufacturing | Slice (K-anon) | Full | None |
| HR | None | None | Full/Anonymized |

**FR-AC-005: Slice Access (Cross-Department)**
- Manufacturing → Customer (product-claim slice only, K-anonymized)
- Marketing → HR (aggregated statistics only)
- All slice access logged for audit

**FR-AC-006: Self-Access**
- Users can always access their own data
- Applies to HR data primarily
- Logged but allowed

### 4.5 K-anonymity (FR-KA)

**FR-KA-001: Quasi-Identifier Detection**
- Pre-defined QI list per category
- Customer: age_group, gender, region, membership_grade
- Manufacturing: product_category, factory, shift, month
- HR: department, position_level, age_group, gender, tenure_years

**FR-KA-002: K-Value Enforcement**
- DC-01 Customer: K=5
- DC-02 Manufacturing: K=3
- DC-03 HR/Finance: K=10
- Configurable via policy

**FR-KA-003: Automatic Generalization**
- On K-violation, apply generalization hierarchy
- Age: exact → 5-year → 10-year → decade
- Region: dong → gu → si → province
- Date: day → month → quarter → year

**FR-KA-004: K-anonymity Validation API**
- Pre-publish validation
- Identify violating groups
- Suggest generalization paths

### 4.6 KMS Integration (FR-KMS)

**FR-KMS-001: Multi-Provider Support**
- AWS KMS (production)
- HashiCorp Vault (multi-cloud/on-premise)
- Local file-based (development)

**FR-KMS-002: Envelope Encryption**
- Master Key (KEK) in KMS
- Data Encryption Keys (DEK) generated per operation
- DEK encrypted with KEK before storage

**FR-KMS-003: Key Rotation**
- DEK rotation: 30 days
- Tenant key rotation: 90 days
- Master key rotation: 1 year (manual approval)

**FR-KMS-004: Key Caching**
- Local cache for active DEKs (in-memory only)
- TTL: 5 minutes
- Zeroize on eviction

**FR-KMS-005: Failover**
- Primary: configured provider
- Secondary: alternate provider (if configured)
- Last resort: cached keys (read-only)

### 4.7 Break-Glass Access (FR-BG)

**FR-BG-001: Emergency Request**
- User requests break-glass with justification
- Required: incident ID, reason, duration
- Auto-notification to security team

**FR-BG-002: Dual Approval**
- Requires 2 approvers (different roles)
- Approval window: 15 minutes
- Auto-reject if not approved

**FR-BG-003: Time-Bounded Access**
- Maximum duration: 4 hours
- Auto-revoke on expiration
- All actions logged in detail

**FR-BG-004: Mandatory Audit**
- Every action logged with break-glass tag
- Immutable audit trail
- Post-incident review required

### 4.8 Auditor Integration (FR-AU)

**FR-AU-001: Audit Role**
- Auditor role has read access to audit logs
- No access to actual data
- Can request metadata (record counts, access patterns)

**FR-AU-002: Compliance Reports**
- Pre-built reports for PIPA, GDPR, PCI DSS
- Custom report generation
- Export to PDF/CSV

**FR-AU-003: Anomaly Detection**
- Unusual access patterns flagged
- Time-of-day anomalies
- Volume anomalies

### 4.9 Operational Features (FR-OP)

**FR-OP-001: Health Checks**
- `/health/live` - Liveness
- `/health/ready` - Readiness (includes KMS connectivity)
- Kubernetes compatible

**FR-OP-002: Graceful Shutdown**
- Complete in-flight operations
- Maximum wait: 60 seconds (longer than Sentinel due to crypto operations)

**FR-OP-003: Hot Reload**
- OPA policies
- Anonymization strategy mappings
- K-anonymity parameters

**FR-OP-004: Error Handling**
- KMS failure → cached keys + degraded mode
- Token DB failure → reject anonymization (fail-safe)
- OPA failure → strict deny-all mode

---

## 5. Non-Functional Requirements

### 5.1 Performance (NFR-PE)

| ID | Item | Target |
|---|---|---|
| NFR-PE-001 | Anonymization latency (p95) | < 5ms |
| NFR-PE-002 | Anonymized data retrieval (p95) | < 10ms |
| NFR-PE-003 | Deanonymization latency (p95) | < 20ms |
| NFR-PE-004 | K-anonymity validation (p95) | < 50ms |
| NFR-PE-005 | Access control decision (p95) | < 2ms |
| NFR-PE-006 | Throughput (anonymization) | ≥ 10,000 ops/s |
| NFR-PE-007 | Throughput (access decisions) | ≥ 20,000 ops/s |
| NFR-PE-008 | Memory usage | ≤ 1GB/pod |
| NFR-PE-009 | CPU usage | ≤ 2 vCPU/pod (normal load) |

### 5.2 Reliability (NFR-RE)

| ID | Item | Target |
|---|---|---|
| NFR-RE-001 | Availability (SLA) | 99.99% |
| NFR-RE-002 | Error rate | < 0.1% |
| NFR-RE-003 | Data integrity | 100% (no corruption) |
| NFR-RE-004 | Anonymization consistency | 100% (deterministic) |
| NFR-RE-005 | Mean Time To Recovery (MTTR) | < 5 minutes |

### 5.3 Scalability (NFR-SC)

| ID | Item | Target |
|---|---|---|
| NFR-SC-001 | Horizontal scaling | Automatic (HPA) |
| NFR-SC-002 | Tenants supported | 10,000+ |
| NFR-SC-003 | Concurrent users | 50,000+ |
| NFR-SC-004 | Token mappings | 1 billion+ |
| NFR-SC-005 | Data volume | Petabyte scale |

### 5.4 Security (NFR-SE)

| ID | Item | Requirement |
|---|---|---|
| NFR-SE-001 | Encryption in transit | TLS 1.3 required |
| NFR-SE-002 | Encryption at rest | AES-256-GCM |
| NFR-SE-003 | Key management | KMS-based |
| NFR-SE-004 | Authentication | mTLS (system) + JWT (user) |
| NFR-SE-005 | Authorization | RBAC + OPA |
| NFR-SE-006 | Memory protection | Zeroize after use |
| NFR-SE-007 | Side-channel protection | Constant-time operations |
| NFR-SE-008 | Audit log integrity | HMAC-signed, append-only |
| NFR-SE-009 | Secret rotation | Automated (DEK 30d, KEK 90d) |
| NFR-SE-010 | Vulnerability scanning | Weekly (Snyk, Trivy) |

### 5.5 Compliance (NFR-CO)

| ID | Requirement | Implementation |
|---|---|---|
| NFR-CO-001 | PIPA - Personal data masking | Tokenization + Masking |
| NFR-CO-002 | PIPA - RRN irreversibility | HMAC-SHA256 (no reverse) |
| NFR-CO-003 | PIPA - Retention period | Auto-purge after legal period |
| NFR-CO-004 | GDPR - Right to erasure | Token deletion on request |
| NFR-CO-005 | GDPR - Data portability | Export in standard format |
| NFR-CO-006 | PCI DSS - Card data protection | PCI tokenization |
| NFR-CO-007 | SOC 2 Type II | Comprehensive audit logs |
| NFR-CO-008 | Audit log retention | 5 years |
| NFR-CO-009 | Data residency | Korean data in Korea |

### 5.6 Maintainability (NFR-MA)

| ID | Item | Target |
|---|---|---|
| NFR-MA-001 | Code coverage | ≥ 85% (higher than Sentinel) |
| NFR-MA-002 | Static analysis | golangci-lint, gosec pass |
| NFR-MA-003 | Documentation | godoc + architecture docs |
| NFR-MA-004 | API versioning | URL path (/v1, /v2) |
| NFR-MA-005 | Backward compatibility | 12 months minimum |
| NFR-MA-006 | Configuration validation | Pre-deployment checks |

---

## 6. System Architecture

### 6.1 High-Level Architecture

```
┌────────────────────────────────────────────────────────────────┐
│                    Vault Service                                │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐           │
│  │  gRPC API   │  │  REST API   │  │     CLI     │           │
│  │   (:9090)   │  │   (:8080)   │  │  vault-cli  │           │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘           │
│         └─────────────────┴─────────────────┘                  │
│                           │                                    │
│                           ▼                                    │
│         ┌────────────────────────────────────┐                │
│         │      Request Dispatcher             │                │
│         └────────────────┬───────────────────┘                │
│                          │                                     │
│         ┌────────────────┼────────────────┐                   │
│         ▼                ▼                ▼                   │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐         │
│  │ Classification│ │  Anonymizer  │ │  Access      │         │
│  │   Engine     │ │    Engine    │ │  Controller  │         │
│  │              │ │              │ │   (OPA)      │         │
│  │ - PII Detect │ │ - Tokenize   │ │              │         │
│  │ - Category   │ │ - HMAC Hash  │ │ - RBAC       │         │
│  │ - Sensitivity│ │ - FPE        │ │ - Policy     │         │
│  │              │ │ - Mask       │ │   Eval       │         │
│  │              │ │ - Generalize │ │ - Audit      │         │
│  └──────────────┘ └──────┬───────┘ └──────────────┘         │
│                          │                                    │
│                          ▼                                    │
│         ┌────────────────────────────────────┐                │
│         │      KMS Abstraction Layer          │                │
│         │  ┌──────────┐ ┌──────────┐ ┌─────┐│                │
│         │  │ AWS KMS  │ │HashiCorp │ │Local││                │
│         │  └──────────┘ └──────────┘ └─────┘│                │
│         └────────────────┬───────────────────┘                │
│                          │                                    │
│         ┌────────────────┼────────────────┐                   │
│         ▼                ▼                ▼                   │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐         │
│  │  K-anonymity │ │ Break-Glass  │ │   Auditor    │         │
│  │   Validator  │ │   Manager    │ │   Service    │         │
│  └──────────────┘ └──────────────┘ └──────────────┘         │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐   │
│  │   Data Storage Layer                                  │   │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐   │   │
│  │  │ PostgreSQL  │ │ Token DB    │ │   Redis     │   │   │
│  │  │ (with RLS)  │ │             │ │  (Cache)    │   │   │
│  │  │             │ │             │ │             │   │   │
│  │  │ DC-01: cust │ │ Mappings    │ │ Tokens      │   │   │
│  │  │ DC-02: mfg  │ │             │ │ Permissions │   │   │
│  │  │ DC-03: hr   │ │             │ │             │   │   │
│  │  └─────────────┘ └─────────────┘ └─────────────┘   │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                               │
└────────────────────────────────────────────────────────────────┘
                          │
                          ▼ (audit logs)
                  Module D (Tracker)
```

### 6.2 Component Description

| Component | Responsibility |
|---|---|
| **Request Dispatcher** | Route requests to appropriate handler |
| **Classification Engine** | PII detection, category assignment |
| **Anonymizer Engine** | Apply anonymization strategies |
| **Access Controller** | OPA-based policy enforcement |
| **K-anonymity Validator** | Validate and enforce K-anon |
| **Break-Glass Manager** | Emergency access workflow |
| **Auditor Service** | Audit log access for compliance |
| **KMS Abstraction** | Unified interface for multiple KMS |
| **PostgreSQL (RLS)** | Tenant-isolated data storage |
| **Token DB** | PII-to-token mappings |
| **Redis Cache** | Performance optimization |

### 6.3 Data Flow - Anonymization

```
Client          Vault         Classifier    Anonymizer    KMS         TokenDB    Storage
  │              │                │              │           │             │          │
  ├─Anonymize───►│                │              │           │             │          │
  │              ├─Classify──────►│              │           │             │          │
  │              │◄──Category─────┤              │           │             │          │
  │              ├─Detect PII────►│              │           │             │          │
  │              │◄──PII List─────┤              │           │             │          │
  │              │                                                                     │
  │              ├─Anonymize─────────────────────►│                                  │
  │              │                                ├─Get DEK──►│                       │
  │              │                                │◄──DEK─────┤                       │
  │              │                                │                                   │
  │              │                                ├─Tokenize/Hash/FPE/Mask           │
  │              │                                │                                   │
  │              │                                ├─Store Mapping───────────►│       │
  │              │◄──Anonymized Data──────────────┤                          │       │
  │              │                                                                     │
  │              ├─Store Data───────────────────────────────────────────────────────►│
  │              │                                                                     │
  │◄──Response───┤                                                                     │
  │              ├─Async Audit Log ─────────────────────────────────► Module D       │
```

### 6.4 Data Flow - Access with K-anonymity

```
Client          Vault          OPA           Storage      K-anon       
  │              │              │              │             │
  ├─Access Req──►│              │              │             │
  │              ├─Evaluate────►│              │             │
  │              │◄──Decision───┤              │             │
  │              │   (allowed,                                │
  │              │    k_anonymized)                           │
  │              │                                            │
  │              ├─Fetch Data───────────────►│                │
  │              │◄──Records────────────────┤                │
  │              │                                            │
  │              ├─Validate K──────────────────────────────►│
  │              │◄──Valid/Generalized──────────────────────┤
  │              │                                            │
  │◄──Response───┤                                            │
  │              ├─Audit Log ──────────────────► Module D    │
```

---

## 7. Data Classification and Categories

### 7.1 Data Categories Overview

| ID | Name | Department Access | Sensitivity |
|---|---|---|---|
| **DC-01** | Customer Personal Data | Marketing | 🔴 Very High |
| **DC-02** | Manufacturing Data | Manufacturing, Marketing | 🟡 Medium |
| **DC-03** | HR & Finance Data | HR | 🔴 Very High |

### 7.2 DC-01: Customer Personal Data

**Data Types:**
- Identity: name, date of birth, gender
- Contact: email, mobile, address
- Identifiers: RRN, foreign registration number
- Payment: credit card, bank account
- Purchase history, behavior data, membership

**Applicable Regulations:**
- PIPA (South Korea)
- GDPR (international customers)
- PCI DSS (payment data)

**Anonymization Strategy Matrix:**

| Field | Strategy | Reversible | Example |
|---|---|---|---|
| Korean Name | Deterministic Tokenization | ✅ | `KR_NAME_8f3d2a` |
| Korean RRN | HMAC-SHA256 | ❌ | `RRN_a1b2c3...` |
| Mobile (display) | Partial Masking | ❌ | `010-****-5678` |
| Mobile (search) | Deterministic Tokenization | ✅ | `MOBILE_xyz123` |
| Email | FPE (preserve domain) | ✅ | `XXXXX@naver.com` |
| Address (detail) | Suppression | ❌ | `[REMOVED]` |
| Address (city) | Generalization | ❌ | `Seoul Gangnam-gu` |
| Credit Card | PCI Tokenization | ❌ | `1234-****-****-3456` |
| Date of Birth | Generalization | ❌ | `1980-1989` |

### 7.3 DC-02: Manufacturing Data

**Data Types:**
- Products: product code, specifications, BOM
- Production: volume, defect rate, utilization
- Quality: QC results, inspections
- Materials: codes, suppliers
- Equipment: ID, status, runtime
- Worker IDs (separated from HR)

**Anonymization Strategy:**
- Product codes: As-is (internal identifier)
- Production data: As-is (accuracy critical)
- Cost data: Department-dependent (Marketing sees aggregated)
- Worker IDs: Tokenized (decoupled from HR records)

### 7.4 DC-03: HR & Finance Data

**Data Types:**
- Personal: name, RRN, employee ID
- Contact: mobile, address, emergency contact
- Organization: department, position, title
- Employment: hire date, termination date
- Compensation: salary, bonus, allowances
- Evaluation: performance reviews
- Education: school, major, certifications
- Family: dependents, beneficiaries
- Health: medical checkups, disability
- Disciplinary records, awards

**Anonymization Strategy Matrix:**

| Field | Strategy | Reversible | Access Required |
|---|---|---|---|
| Employee ID | As-is | - | All HR roles |
| Name | Tokenization | ✅ | HR Manager+ |
| Korean RRN | HMAC Hashing | ❌ | None |
| Department/Position | As-is | - | All HR roles |
| Salary (exact) | Encryption | ✅ | HR Manager only |
| Salary (range) | Range Generalization | ❌ | HR Staff |
| Performance Review | Encryption | ✅ | HR Manager only |
| Education (school) | Generalization | ❌ | HR Staff |
| Health Data | Encryption | ✅ | HR Manager only |
| Disability | Tokenization | ✅ | HR Manager only |

### 7.5 Cross-Category Access (Extended Permissions)

**Marketing → HR (Aggregated)**
- Demographics for marketing analysis
- Department-level statistics only
- K-anonymity K=10 enforced
- Example: "30s male employees: count by region"

**Manufacturing → Customer (Slice)**
- Product-related customer complaints
- K-anonymity K=5 enforced
- Limited to product context only
- Example: "Complaints for PROD-001 by month"

---

## 8. KMS Abstraction Layer

### 8.1 Provider Interface

```go
type KMSProvider interface {
    GenerateDataKey(ctx context.Context, keyID string) (*DataKey, error)
    Encrypt(ctx context.Context, plaintext []byte, keyID string) ([]byte, error)
    Decrypt(ctx context.Context, ciphertext []byte, keyID string) ([]byte, error)
    HMAC(ctx context.Context, data []byte, keyID string) ([]byte, error)
    CreateKey(ctx context.Context, spec KeySpec) (string, error)
    RotateKey(ctx context.Context, keyID string) error
    GetKeyInfo(ctx context.Context, keyID string) (*KeyInfo, error)
    HealthCheck(ctx context.Context) error
}
```

### 8.2 Provider Configurations

**AWS KMS (Production):**
```yaml
kms:
  provider: aws
  aws:
    region: ap-northeast-2
    master_key_id: alias/bastion-vault-prod
    role_arn: arn:aws:iam::123456789:role/bastion-vault
    endpoint: ""  # default AWS endpoint
```

**HashiCorp Vault (Multi-cloud/On-premise):**
```yaml
kms:
  provider: hashicorp
  hashicorp:
    address: https://vault.example.com:8200
    namespace: bastion
    mount_path: transit/
    role: bastion-vault-role
    auth_method: kubernetes
```

**Local (Development):**
```yaml
kms:
  provider: local
  local:
    master_key_file: /var/lib/vault/master.key
    key_store_dir: /var/lib/vault/keys/
    # WARNING: Development only, not for production
```

### 8.3 Key Hierarchy

```
Master Key (KEK) - Stored in KMS, never leaves
    │
    ├─ Tenant Key (TK) - Per-tenant, derived from Master
    │   │
    │   ├─ Data Encryption Key (DEK) - Per-operation, ephemeral
    │   │   └─ Encrypts actual PII data
    │   │
    │   └─ HMAC Key - For deterministic tokenization
    │       └─ Generates consistent tokens
    │
    └─ Audit Key - For audit log signing
```

### 8.4 Key Rotation Schedule

| Key Type | Rotation | Trigger |
|---|---|---|
| DEK | 30 days | Automatic |
| Tenant Key | 90 days | Automatic + manual override |
| Master Key | 1 year | Manual with dual approval |
| HMAC Key | 90 days | Automatic |
| Audit Key | 180 days | Automatic |

### 8.5 Failover Strategy

```
Primary Provider (AWS KMS)
    │
    ▼ Failure
Secondary Provider (HashiCorp Vault)
    │
    ▼ Failure
Cached Keys (in-memory, read-only mode)
    │
    ▼ Failure
Service Degradation (reject new operations, complete in-flight)
```

---

## 9. Anonymization Strategies

### 9.1 Strategy Catalog

| Strategy | Description | Reversible | Searchable | Use Case |
|---|---|---|---|---|
| **Deterministic Tokenization** | Map to consistent token | ✅ | ✅ | Names, IDs |
| **Random Tokenization** | Map to random token | ✅ | ❌ | Sensitive data |
| **HMAC Hashing** | One-way HMAC | ❌ | ✅ | RRN, SSN |
| **Format-Preserving Encryption** | Preserve format, encrypt content | ✅ | Partial | Email, phone |
| **Partial Masking** | Show partial value | ❌ | Partial | Display-only |
| **Generalization** | Reduce specificity | ❌ | ✅ | Age, location |
| **Range Generalization** | Convert to range | ❌ | ✅ | Salary, scores |
| **Suppression** | Remove entirely | ❌ | ❌ | Re-id risk |
| **Encryption** | Strong encryption | ✅ | ❌ | Highly sensitive |
| **Perturbation** | Add noise | ❌ | Statistical | Statistics |

### 9.2 PII Type to Strategy Mapping

(See Section 7 for category-specific mappings)

### 9.3 Two-Phase Anonymization

**Phase 1: At Storage (Write Path)**
- Applied before data persistence
- Strong anonymization (default secure)
- Token mappings encrypted in Token DB
- KMS-encrypted DEKs for reversible fields

**Phase 2: At Use (Read Path)**
- Applied based on requester's permissions
- May relax anonymization for authorized users
- Audit logged for all deanonymization

```
Write Path:
  Raw Data → Phase 1 (Storage Anonymization) → DB
  
Read Path:
  DB → Phase 2 (Use-time Re-application) → Response
  
Example - HR Manager reading employee:
  DB: name = "KR_NAME_8f3d2a"
  → Phase 2 (HR Manager has full access)
  → Response: name = "Hong Gildong" (deanonymized)
  
Example - HR Staff reading same employee:
  DB: name = "KR_NAME_8f3d2a"  
  → Phase 2 (HR Staff has anonymized access)
  → Response: name = "KR_NAME_8f3d2a" (kept anonymized)
```

---

## 10. Access Control Model

### 10.1 RBAC Roles

```yaml
roles:
  marketing_manager:
    department: marketing
    permissions:
      - customer_data:full
      - customer_data:deanonymize
      - manufacturing_data:read
      - hr_finance_data:aggregated
  
  marketing_staff:
    department: marketing
    permissions:
      - customer_data:anonymized
      - manufacturing_data:read
  
  marketing_analyst:
    department: marketing
    permissions:
      - customer_data:k_anonymized
      - manufacturing_data:k_anonymized
      - hr_finance_data:aggregated
  
  manufacturing_manager:
    department: manufacturing
    permissions:
      - manufacturing_data:full
      - customer_data:slice  # product claims only
  
  manufacturing_staff:
    department: manufacturing
    permissions:
      - manufacturing_data:full
      - customer_data:slice  # product claims only
  
  manufacturing_qc:
    department: manufacturing
    permissions:
      - manufacturing_data:read
      - customer_data:slice
  
  hr_manager:
    department: hr
    permissions:
      - hr_finance_data:full
      - hr_finance_data:deanonymize
  
  hr_staff:
    department: hr
    permissions:
      - hr_finance_data:anonymized
  
  hr_payroll:
    department: hr
    permissions:
      - hr_finance_data:salary
      - hr_finance_data:benefits
  
  executive:
    department: leadership
    permissions:
      - "*:aggregated"
  
  auditor:
    department: compliance
    permissions:
      - "*:audit_log"
      - "*:metadata"
  
  security_team:
    department: security
    permissions:
      - "*:read_on_alert"
    conditions:
      - dual_approval: true
      - time_bounded: 4h
```

### 10.2 OPA Policy Examples

```rego
package bastion.vault.access

import future.keywords.if
import future.keywords.in

default allow = false

# DC-01 Customer Data - Marketing only
allow if {
    input.resource.category == "customer_data"
    input.user.department == "marketing"
}

# DC-02 Manufacturing Data - Manufacturing + Marketing
allow if {
    input.resource.category == "manufacturing_data"
    input.user.department in ["manufacturing", "marketing"]
}

# DC-03 HR/Finance Data - HR only
allow if {
    input.resource.category == "hr_finance_data"
    input.user.department == "hr"
}

# Extended: Marketing → HR aggregated
allow if {
    input.user.department == "marketing"
    input.resource.category == "hr_finance_data"
    input.action == "read_aggregated"
    input.aggregation_level in ["department", "demographic"]
}

# Extended: Manufacturing → Customer slice
allow if {
    input.user.department == "manufacturing"
    input.resource.category == "customer_data"
    input.resource.context == "product_claim"
}

# Self-access: HR data for own records
allow if {
    input.resource.category == "hr_finance_data"
    input.resource.subject_id == input.user.employee_id
    input.action == "read_own"
}

# Executive: aggregated only
allow if {
    input.user.role == "executive"
    input.action == "read_aggregated"
}

# Auditor: audit logs only
allow if {
    input.user.role == "auditor"
    input.action in ["read_audit_log", "read_metadata"]
}

# Break-glass override (with strict conditions)
allow if {
    input.break_glass.active == true
    input.break_glass.approved == true
    input.break_glass.expires_at > time.now_ns()
}

# Determine access level
access_level = "full" if {
    input.user.role in ["marketing_manager", "manufacturing_manager", "hr_manager"]
    allow
}

access_level = "anonymized" if {
    input.user.role in ["marketing_staff", "hr_staff"]
    allow
}

access_level = "k_anonymized" if {
    input.user.role == "marketing_analyst"
    allow
}

access_level = "slice" if {
    input.user.department == "manufacturing"
    input.resource.category == "customer_data"
    allow
}

access_level = "aggregated" if {
    input.action == "read_aggregated"
    allow
}

# Require audit log
require_audit = true if {
    input.action == "deanonymize"
}

require_audit = true if {
    input.resource.category == "hr_finance_data"
}

require_audit = true if {
    input.break_glass.active == true
}
```

---

## 11. K-anonymity Implementation

### 11.1 K-Value Configuration

| Category | K-Value | Rationale |
|---|---|---|
| DC-01 Customer | K=5 | Standard practice, marketing analysis feasible |
| DC-02 Manufacturing | K=3 | Lower identification risk |
| DC-03 HR/Finance | K=10 | Highly sensitive, strong protection |

### 11.2 Quasi-Identifier Definitions

```yaml
quasi_identifiers:
  customer_data:
    - age_group
    - gender
    - region
    - membership_grade
    - registration_year
  
  manufacturing_data:
    - product_category
    - factory_id
    - shift
    - production_date_month
  
  hr_finance_data:
    - department
    - position_level
    - age_group
    - gender
    - tenure_years
```

### 11.3 Generalization Hierarchies

```yaml
generalization:
  age:
    level_0: "exact"      # 32
    level_1: "5_year"     # 30-34
    level_2: "10_year"    # 30-39
    level_3: "decade"     # adult
  
  region:
    level_0: "dong"       # Yeoksam-dong
    level_1: "gu"         # Gangnam-gu
    level_2: "si"         # Seoul
    level_3: "country"    # South Korea
  
  date:
    level_0: "day"        # 2024-03-15
    level_1: "month"      # 2024-03
    level_2: "quarter"    # 2024-Q1
    level_3: "year"       # 2024
  
  salary:
    level_0: "exact"      # 65,000,000
    level_1: "5m_range"   # 60-65M
    level_2: "10m_range"  # 60-70M
    level_3: "category"   # mid-range
```

### 11.4 Auto-Generalization Algorithm

```go
func (v *KAnonymityValidator) AutoGeneralize(
    records []Record,
    qiList []string,
    k int,
) []Record {
    for {
        // Group by quasi-identifiers
        groups := groupBy(records, qiList)
        
        // Check K-anonymity
        violatingGroups := findViolations(groups, k)
        if len(violatingGroups) == 0 {
            return records  // K-anonymity satisfied
        }
        
        // Find QI with highest cardinality
        targetQI := findHighestCardinality(qiList, violatingGroups)
        
        // Apply next generalization level
        currentLevel := v.getCurrentLevel(targetQI)
        if currentLevel >= v.getMaxLevel(targetQI) {
            // Cannot generalize further, suppress
            return v.suppressViolatingRecords(records, violatingGroups)
        }
        
        records = v.generalize(records, targetQI, currentLevel+1)
    }
}
```

---

## 12. Standalone Testing Environment

### 12.1 Independence Principle

Vault must be runnable, testable, and operable independently of other Bastion modules.

### 12.2 Standalone Execution Modes

**Mode 1: Full Server Mode**

```bash
$ vault-cli server --port 8080 --grpc-port 9090

🚀 Bastion-Vault v1.0 starting...
✅ Config loaded from /etc/vault.yaml
✅ KMS provider: local (development)
✅ Token DB connected: postgres://localhost:5432/vault_tokens
✅ Redis cache: localhost:6379
✅ OPA policies loaded: 12 policies
✅ Anonymization strategies: 10 strategies
✅ REST API listening on :8080
✅ gRPC API listening on :9090
✨ Ready to accept requests
```

**Mode 2: One-shot Anonymization**

```bash
$ vault-cli anonymize \
    --category customer_data \
    --tenant tenant-acme \
    --data '{"name":"Hong Gildong","email":"hong@naver.com"}' \
    --output-format text

════════════════════════════════════════════
  Vault Anonymization Result
════════════════════════════════════════════
Status: ✅ SUCCESS
Time:   4.2 ms

Anonymized:
  name:  KR_NAME_8f3d2a (Reversible)
  email: XXXXX@naver.com (FPE)
════════════════════════════════════════════
```

**Mode 3: Interactive Mode (REPL)**

```bash
$ vault-cli interactive
Welcome to Bastion-Vault REPL v1.0

vault> anonymize
Category: customer_data
Data (JSON): {"name":"Kim","mobile":"010-1234-5678"}
✅ Anonymized in 3.8ms

vault> access --user user-bob --dept manufacturing --resource customer-12345
🚫 DENIED - manufacturing cannot access customer_data (except slice)

vault> stats
Anonymizations: 234
Access checks: 567 (allowed: 540, denied: 27)
K-anon validations: 12

vault> exit
```

**Mode 4: Batch Processing**

```bash
$ vault-cli anonymize \
    --input-file employees.csv \
    --output-file anonymized.csv \
    --category hr_finance_data \
    --tenant tenant-acme \
    --parallel 10

Processing: ████████████░░ 80% (8000/10000) | 4.5ms avg

✅ Total: 10000
✅ Success: 9985
❌ Failed: 15 (saved to errors.log)
⏱️  Avg latency: 4.5ms
```

### 12.3 Dependency Fallback

| Dependency | Fallback | Behavior |
|---|---|---|
| AWS KMS | HashiCorp Vault | Try secondary provider |
| HashiCorp Vault | Local KMS | Use local keys (warning) |
| Redis | In-memory cache | Local cache, no sharing |
| Token DB | Read-only mode | Reject new anonymization |
| OPA | Strict deny-all | Fail safely |
| Module D (Tracker) | Local log file | Continue with degraded logging |

### 12.4 Test Data

```
tests/
├── fixtures/
│   ├── customer_data_samples.jsonl      # 1,000 customer records
│   ├── manufacturing_data_samples.jsonl # 500 manufacturing records  
│   ├── hr_data_samples.jsonl            # 200 HR records
│   ├── pii_test_cases.jsonl             # PII detection tests
│   ├── access_test_cases.jsonl          # Access control tests
│   ├── k_anon_test_cases.jsonl          # K-anonymity tests
│   └── break_glass_scenarios.jsonl      # Emergency access tests
├── golden/
│   └── expected_anonymizations/         # Expected outputs
└── benchmarks/
    └── perf_test_cases.jsonl            # Performance scenarios
```

### 12.5 Integration Test with Docker Compose

```yaml
# tests/integration/docker-compose.yml
version: '3.8'
services:
  vault:
    build: ../..
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      - VAULT_MODE=integration
      - KMS_PROVIDER=local
  
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: vault_test
      POSTGRES_USER: vault
      POSTGRES_PASSWORD: testpass
  
  redis:
    image: redis:7-alpine
  
  opa:
    image: openpolicyagent/opa:latest
    command: run --server /policies
    volumes:
      - ./policies:/policies
  
  test-runner:
    build: ./test-runner
    depends_on:
      - vault
      - postgres
      - redis
      - opa
    command: pytest tests/
```

---

## 13. Data Requirements

### 13.1 Input Schema (Anonymize Request)

```yaml
type: object
required:
  - tenant_id
  - category
  - data
properties:
  request_id:
    type: string
    format: uuid
  tenant_id:
    type: string
    pattern: "^[a-z0-9-]+$"
  category:
    type: string
    enum:
      - customer_data
      - manufacturing_data
      - hr_finance_data
  data:
    type: object
    additionalProperties: true
  options:
    type: object
    properties:
      strict_mode: boolean
      timeout_ms: integer
      output_format:
        type: string
        enum: [json, protobuf, text, compact]
```

### 13.2 Configuration Schema (YAML)

```yaml
# /etc/bastion-vault/config.yaml
version: 1.0

server:
  rest_port: 8080
  grpc_port: 9090

# KMS Configuration
kms:
  provider: aws  # aws, hashicorp, local
  fallback: hashicorp
  
  aws:
    region: ap-northeast-2
    master_key_id: alias/bastion-vault-prod
  
  hashicorp:
    address: https://vault.example.com:8200
    mount_path: transit/
  
  local:
    master_key_file: /var/lib/vault/master.key

# Data Categories
data_categories:
  customer_data:
    k_anonymity_value: 5
    storage_schema: customer_data
    retention_days: 1825  # 5 years
    quasi_identifiers:
      - age_group
      - gender
      - region
      - membership_grade
  
  manufacturing_data:
    k_anonymity_value: 3
    storage_schema: manufacturing_data
    retention_days: 2555  # 7 years
    quasi_identifiers:
      - product_category
      - factory_id
      - shift
  
  hr_finance_data:
    k_anonymity_value: 10
    storage_schema: hr_finance_data
    retention_days: 1095  # 3 years after termination
    quasi_identifiers:
      - department
      - position_level
      - age_group
      - gender
      - tenure_years

# PII Detection
pii_detection:
  mode: hybrid  # explicit, auto, hybrid
  
  explicit_mappings:
    customer.name: korean_name
    customer.rrn: korean_rrn
    customer.mobile: korean_mobile
    customer.email: email
    employee.salary: salary
    employee.evaluation: performance_review
  
  auto_detection:
    regex_enabled: true
    ml_enabled: true
    ml_model_path: /models/pii-detector.onnx
    confidence_threshold: 0.85

# Anonymization Strategies
anonymization:
  default_phase: storage_and_use  # storage, use, storage_and_use
  
  strategies:
    korean_name:
      strategy: deterministic_tokenization
      prefix: "KR_NAME_"
      reversible: true
    
    korean_rrn:
      strategy: hmac_sha256
      reversible: false
    
    korean_mobile:
      strategy: partial_masking
      pattern: "{first:3}-****-{last:4}"
      reversible: false
    
    email:
      strategy: fpe_email
      preserve_domain: true
      reversible: true
    
    credit_card:
      strategy: pci_tokenization
      reversible: false
    
    salary:
      strategy: encryption
      key_id: salary-encryption-key
      reversible: true
      required_role: hr_manager

# Access Control
access_control:
  provider: opa
  opa:
    url: http://opa:8181
    policy_path: /v1/data/bastion/vault/access
    
  hot_reload: true
  reload_interval: 60s

# K-anonymity
k_anonymity:
  enabled: true
  auto_generalize: true
  max_generalization_level: 3

# Break-glass
break_glass:
  enabled: true
  approvers_required: 2
  approval_window: 15m
  max_duration: 4h
  mandatory_audit: true

# Token Storage
token_db:
  type: postgresql
  connection_string: ${TOKEN_DB_URL}
  encryption: true
  
cache:
  type: redis
  url: redis://redis:6379
  ttl: 5m

# Logging
logging:
  level: info
  format: json
  destination: stdout

# Metrics
metrics:
  enabled: true
  port: 9091

# Features
features:
  hot_reload: true
  graceful_shutdown: true
  shutdown_timeout: 60s
```

---

## 14. Deployment and Operations

### 14.1 Deployment Environments

| Environment | Replicas | Memory | KMS |
|---|---|---|---|
| dev | 1 | 512Mi | Local |
| staging | 3 | 1Gi | HashiCorp |
| prod | 10+ | 1Gi | AWS KMS |

### 14.2 Container Image

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o vault ./cmd/vault

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/vault /usr/local/bin/
EXPOSE 8080 9090 9091

HEALTHCHECK --interval=10s --timeout=3s \
  CMD wget --quiet --spider http://localhost:8080/health/live || exit 1

ENTRYPOINT ["vault"]
CMD ["server"]
```

### 14.3 Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bastion-vault
  namespace: bastion
spec:
  replicas: 3
  selector:
    matchLabels:
      app: vault
  template:
    metadata:
      labels:
        app: vault
    spec:
      serviceAccountName: vault-service-account
      containers:
      - name: vault
        image: bastion/vault:1.0.0
        ports:
        - containerPort: 8080
          name: rest
        - containerPort: 9090
          name: grpc
        - containerPort: 9091
          name: metrics
        env:
        - name: TOKEN_DB_URL
          valueFrom:
            secretKeyRef:
              name: vault-secrets
              key: token-db-url
        - name: AWS_REGION
          value: ap-northeast-2
        resources:
          requests:
            cpu: 1000m
            memory: 512Mi
          limits:
            cpu: 4000m
            memory: 2Gi
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        volumeMounts:
        - name: config
          mountPath: /etc/vault
        - name: policies
          mountPath: /policies
      volumes:
      - name: config
        configMap:
          name: vault-config
      - name: policies
        configMap:
          name: vault-opa-policies
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: vault-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: bastion-vault
  minReplicas: 3
  maxReplicas: 30
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

### 14.4 Monitoring

**Key Metrics:**
- `vault_anonymization_duration_seconds` (histogram)
- `vault_access_decisions_total` (counter, by department/decision)
- `vault_kanon_violations_total` (counter)
- `vault_kms_call_duration_seconds` (histogram, by provider)
- `vault_break_glass_active` (gauge)

**Alert Thresholds:**
- p95 anonymization > 10ms: Warning
- KMS error rate > 1%: Critical
- Access denial rate > 20%: Warning
- Break-glass active > 1 hour: Critical
- K-anon violations > 100/min: Critical

---

## 15. Compliance

### 15.1 PIPA (South Korea)

| Requirement | Implementation |
|---|---|
| Lawful basis for processing | Consent tracking in metadata |
| Data minimization | Category-based access |
| Purpose limitation | Per-department policies |
| Storage limitation | Auto-purge after retention period |
| Integrity & confidentiality | Encryption + RBAC |
| RRN special protection | Irreversible HMAC hashing |
| Right to access | Self-service API |
| Right to deletion | Token deletion + data purge |

### 15.2 GDPR (EU)

| Article | Implementation |
|---|---|
| Art. 5 - Principles | Comprehensive controls |
| Art. 17 - Right to erasure | Token-based deletion |
| Art. 20 - Data portability | Export API |
| Art. 25 - Privacy by design | Default anonymization |
| Art. 32 - Security of processing | KMS + encryption |
| Art. 33-34 - Breach notification | Tracker integration |

### 15.3 PCI DSS

| Requirement | Implementation |
|---|---|
| Req 3.4 - PAN rendering | PCI tokenization |
| Req 3.5 - Key management | KMS-based |
| Req 3.6 - Key rotation | Automated rotation |
| Req 7 - Access restriction | RBAC + OPA |
| Req 10 - Audit logging | Tracker integration |

---

## 16. Appendix

### 16.1 Usage Scenarios

**Scenario 1: Customer Data Anonymization at Ingestion**

```bash
# Marketing system stores new customer
$ curl -X POST https://vault.bastion.local/v1/vault/anonymize \
    -H "Authorization: Bearer $TOKEN" \
    -d '{
        "tenant_id": "tenant-acme",
        "category": "customer_data",
        "data": {
            "name": "Hong Gildong",
            "rrn": "850315-1234567",
            "email": "hong@naver.com",
            "mobile": "010-1234-5678"
        }
    }'

# Response: Anonymized data stored in DB
```

**Scenario 2: Marketing Analyst Queries Customer Behavior**

```bash
$ vault-cli access \
    --user-id alice \
    --department marketing \
    --role marketing_analyst \
    --resource customer-purchase-data

# K-anonymized data returned (K=5)
{
  "age_group": "30s",
  "gender": "M",
  "region": "Seoul",
  "avg_purchase": 150000
}
```

**Scenario 3: Manufacturing Investigates Product Defect**

```bash
$ vault-cli access \
    --user-id bob \
    --department manufacturing \
    --resource product-claims \
    --slice-context "product_id=PROD-001"

# Slice access: only claims for PROD-001, anonymized
[
  {
    "complaint_type": "defect",
    "customer_age_group": "30s",
    "region": "Seoul",
    "month": "2024-10"
  }
]
```

**Scenario 4: HR Manager Updates Salary**

```bash
$ vault-cli deanonymize \
    --user-id charlie \
    --department hr \
    --role hr_manager \
    --field salary \
    --employee-id EMP-001

# Audit logged
{
  "employee_id": "EMP-001",
  "salary": 65000000,  # decrypted
  "audit_log_id": "audit-12345"
}
```

**Scenario 5: Break-Glass for Security Incident**

```bash
# Request emergency access
$ vault-cli break-glass request \
    --user-id security-lead \
    --reason "Investigating data breach incident INC-789" \
    --duration 2h \
    --resources "customer_data,hr_finance_data"

# Wait for 2-approver approval
# Once approved, access granted with full audit
```

### 16.2 Troubleshooting

| Symptom | Cause | Resolution |
|---|---|---|
| Anonymization > 10ms | KMS latency | Check KMS connectivity, enable caching |
| Access always denied | OPA policy misconfiguration | Validate policies with `opa eval` |
| K-anonymity always fails | Insufficient data volume | Increase data set or lower K-value |
| Token mismatch | Token DB corruption | Restore from backup, re-tokenize |
| Memory leak | Key not zeroized | Audit code for sensitive data handling |

### 16.3 Roadmap

- v1.1: Tenant-specific anonymization policies
- v1.2: Automated PII detection improvements (ML model updates)
- v1.3: L-diversity in addition to K-anonymity
- v2.0: T-closeness support
- v2.1: Differential privacy for analytics
- v2.2: Homomorphic encryption for computation on encrypted data

### 16.4 Change History

| Version | Date | Changes | Author |
|---|---|---|---|
| 0.1 | 2026-05-15 | Initial draft | - |
| 0.5 | 2026-05-16 | KMS abstraction added | - |
| 0.8 | 2026-05-17 | Access control matrix finalized | - |
| 1.0 | 2026-05-17 | Initial release | - |

### 16.5 Approval

| Role | Name | Signature | Date |
|---|---|---|---|
| Project Manager | | | |
| Technical Lead | | | |
| Security Officer | | | |
| Compliance Officer | | | |
| QA Lead | | | |

---

**End of Document**
