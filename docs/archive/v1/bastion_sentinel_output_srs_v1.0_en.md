# Bastion-Sentinel Output Validation SRS v1.0

**Project:** Bastion-RAG - RAG Security Governance Framework  
**Module:** Module A - Sentinel (Output Validation Extension)  
**Document Version:** 1.0  
**Date:** 2026-05-17  
**Status:** Draft  
**Parent Document:** Bastion-Sentinel SRS v1.0 (Input Validation)  
**Scope:** Output Pipeline Security

---

## 1. Introduction

### 1.1 Purpose

This document defines the **Output Validation** extension of the Bastion-Sentinel module. Sentinel has evolved from a single-direction input gateway to a **bidirectional security guard**, mirroring Vault's two-phase architecture.

This SRS focuses specifically on:
1. **PII Re-emergence Prevention** - Detect data leakage from LLM
2. **Hallucination Detection** - Identify ungrounded responses
3. **Content Filtering** - Block harmful or inappropriate output
4. **Permission Boundary Enforcement** - Verify output matches user's authority
5. **Response Format Validation** - Ensure proper output structure

### 1.2 Background: Why Bidirectional Sentinel?

#### The Asymmetric Problem
The original design protected only the **input path**:
```
User → Sentinel → Vault → Navigator → Anchor → LLM → ??? → User
       (protected)                                  (vulnerable!)
```

This left a critical gap: **LLM responses could undo all input protections**.

#### Real-world Attack Vectors

```
Example 1: PII Re-emergence
─────────────────────────────
Vault anonymizes: "Hong Gildong" → "USER_8f3d2a"
LLM has context with USER_8f3d2a
LLM response: "Hong Gildong's purchase pattern..."
                ↑ LLM inferred or hallucinated original name

Example 2: Hallucination
─────────────────────────────
Search results: (No price information for PROD-001)
LLM response: "PROD-001 costs approximately $500"
                ↑ Fabricated information

Example 3: Permission Bypass via Response
─────────────────────────────
User permission: marketing_analyst (K-anonymized data only)
Vault returned: Aggregated statistics
LLM response: "Customer Kim purchased $5,000 worth..."
                ↑ Detailed info exceeding user's authority
```

#### Bidirectional Design (New)

```
User → Sentinel-IN → Vault-IN → Navigator → Anchor-IN → LLM
                                                          ↓
User ← Sentinel-OUT ← Vault-OUT ← ────────── ← Anchor-OUT
```

This SRS specifies the **Sentinel-OUT** functionality.

### 1.3 Symmetry with Vault

Sentinel-OUT mirrors Vault's Two-Phase design:

| Module | Phase 1 (Input) | Phase 2 (Output) |
|---|---|---|
| **Sentinel** | Input validation | Output validation (this doc) |
| **Vault** | Anonymization (storage) | Re-application (use) |
| **Anchor** | Embedding noise | Bias check on response |

This creates a consistent **bidirectional security pattern** across the entire Bastion-RAG framework.

### 1.4 Scope

**In Scope:**
- LLM response validation
- PII re-emergence detection
- Hallucination detection (basic heuristics)
- Content filtering (blacklist + patterns)
- Permission boundary verification
- Format validation
- Citation enforcement (optional)
- Integration with Vault for PII mapping lookup
- Integration with Navigator for context verification
- Streaming response handling
- Multiple interfaces (gRPC, REST, CLI)

**Out of Scope:**
- Input validation (Parent SRS - Sentinel Input)
- LLM-based deep semantic analysis (future)
- Multi-modal output (images, audio - future)
- Advanced adversarial detection (future)

### 1.5 Design Philosophy

```
Principle 1: Same Engine, Different Configurations
- Single validation engine processes both input and output
- Mode determined by configuration
- Code reuse, operational simplicity

Principle 2: Defense in Depth
- Output validation is the LAST line of defense
- Even if other modules fail, Sentinel-OUT catches issues
- "Fail safe" by default

Principle 3: Performance Aware
- Output is naturally longer than input
- Multiple parallel checks where possible
- Streaming-aware processing

Principle 4: Operational Visibility
- All decisions logged
- Events sent to Tracker
- Clear feedback to user when content is filtered
```

### 1.6 Definitions and Acronyms

| Term | Definition |
|---|---|
| **Sentinel-IN** | Sentinel in input validation mode |
| **Sentinel-OUT** | Sentinel in output validation mode (this doc) |
| **PII Re-emergence** | LLM response containing original PII |
| **Hallucination** | LLM generating ungrounded content |
| **Grounding** | Response based on retrieved context |
| **Citation** | Source attribution for claims |
| **Streaming** | Token-by-token response delivery |
| **Output Boundary** | Last verifiable boundary before user |

### 1.7 References

- Parent: Bastion-Sentinel SRS v1.0 (Input)
- Related: Bastion-Vault SRS v1.0
- Related: Bastion-Navigator SRS v1.0
- Related: Bastion-Tracker SRS v1.0
- External: [pii-pattern-engine](https://github.com/zafrem/pii-pattern-engine) (Pattern Repository)

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
                    [...pipeline...]
                         ▼
                        LLM
                         ▼
                    [...pipeline...]
                         ▼
        ┌────────────────────────────────────┐
        │   Vault-OUT (Permission Re-apply)  │
        └────────────────┬───────────────────┘
                         ▼
        ┌────────────────────────────────────┐
        │   Sentinel-OUT  ◄── (This doc)     │
        │   - PII Re-emergence Check         │
        │   - Hallucination Detection        │
        │   - Content Filtering              │
        │   - Permission Verification        │
        │   - Format Validation              │
        └────────────────┬───────────────────┘
                         ▼
        ┌────────────────────────────────────┐
        │           User Response            │
        └────────────────────────────────────┘
                         │
                         ▼ (async)
                  Tracker (audit)
```

### 2.2 Unified Sentinel Architecture

Sentinel operates as a single service with bidirectional capability:

```
┌─────────────────────────────────────────────────────┐
│             Unified Sentinel Service                │
├─────────────────────────────────────────────────────┤
│                                                     │
│  ┌─────────────────────────────────────────────┐    │
│  │     Single Validation Engine                │    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────────┐ │    │
│  │  │ Pattern  │ │  Rule    │ │  ML Model    │ │    │
│  │  │ Matcher  │ │ Engine   │ │  Inference   │ │    │
│  │  └──────────┘ └──────────┘ └──────────────┘ │    │
│  └─────────────────┬───────────────────────────┘    │
│                    │                                │
│  ┌─────────────────┼───────────────────────────┐    │
│  │   Configuration Manager                     │    │
│  │  ┌──────────────────────────────────────┐   │    │
│  │  │ Input Config (Phase 1)               │   │    │
│  │  │ - Prompt Injection Rules             │   │    │
│  │  │ - Metadata Schema                    │   │    │
│  │  └──────────────────────────────────────┘   │    │
│  │  ┌──────────────────────────────────────┐   │    │
│  │  │ Output Config (Phase 2) ⭐           │   │    │
│  │  │ - PII Re-emergence Rules             │   │    │
│  │  │ - Hallucination Heuristics           │   │    │
│  │  │ - Content Filter Lists               │   │    │
│  │  │ - Permission Rules                   │   │    │
│  │  └──────────────────────────────────────┘   │    │
│  └─────────────────────────────────────────────┘    │
│                                                     │
└─────────────────────────────────────────────────────┘
                          │
                          ▼
                 Single Service Endpoint
                 (Mode determined by API call)
```

### 2.3 Operating Modes

**Mode 1: Input Validation (Sentinel-IN)**
- Triggered by: `POST /v1/validate` with `mode=input`
- Or: `POST /v1/validate/input`
- Activates: Input Configuration

**Mode 2: Output Validation (Sentinel-OUT)**
- Triggered by: `POST /v1/validate` with `mode=output`
- Or: `POST /v1/validate/output`
- Activates: Output Configuration

**Mode 3: Auto-detect**
- Sentinel infers mode from request structure
- Input request → Input mode
- LLM response → Output mode

### 2.4 Functions (Output-Specific)

1. **F1: PII Re-emergence Detection** ⭐ (Most Important)
   - Detect original PII in LLM responses
   - Cross-reference Vault's anonymization mappings
   - Pattern-based detection for un-anonymized data

2. **F2: Hallucination Detection**
   - Verify response is grounded in retrieved context
   - Detect fabricated facts (numbers, dates, names)
   - Confidence scoring

3. **F3: Content Filtering**
   - Block harmful content
   - Filter inappropriate language
   - Detect policy violations

4. **F4: Permission Boundary Check**
   - Verify response matches user's data category access
   - Detect inference attacks
   - Block over-disclosure

5. **F5: Format Validation**
   - Verify response structure
   - Length limits
   - Required fields presence

6. **F6: Citation Enforcement** (Optional)
   - Require source attribution
   - Add citations automatically
   - Validate citation accuracy

7. **F7: Streaming Support**
   - Token-by-token validation
   - Buffer management
   - Late detection handling

### 2.5 Constraints

- **Language:** Go (same as parent Sentinel)
- **Memory:** Shared with input validation (≤ 1GB total)
- **Latency:** <100ms p95 (longer than input due to larger payload)
- **Throughput:** ≥ 1,000 responses/s
- **Dependencies:** Vault (for PII mapping), Navigator (for context)

### 2.6 Assumptions

- LLM responses are received as text (not embeddings)
- Vault provides PII mapping lookup API
- Navigator provides retrieval context API
- User permissions are available in request context
- Tracker is available for event publishing (graceful degradation if not)

---

## 3. External Interface Requirements

### 3.1 Interface Overview

Sentinel-OUT uses the same interfaces as Sentinel-IN, with **mode parameter** distinguishing direction.

| Category | Interface | Purpose |
|---|---|---|
| **Input** | gRPC | Internal validation calls |
| **Input** | REST API | External integrations |
| **Input** | CLI | Manual testing, demos |
| **Output** | Validated response | To user |
| **Output** | Events to Tracker | Audit trail |

### 3.2 gRPC Interface

```protobuf
// sentinel.proto (extended)
syntax = "proto3";
package bastion-rag.sentinel.v1;

service SentinelService {
  // Existing input methods
  rpc ValidateInput(InputRequest) returns (InputResponse);
  
  // NEW: Output validation
  rpc ValidateOutput(OutputRequest) returns (OutputResponse);
  rpc ValidateOutputStream(stream OutputChunk) returns (stream OutputAck);
  
  // Health
  rpc Health(HealthRequest) returns (HealthResponse);
}

message OutputRequest {
  string request_id = 1;
  string trace_id = 2;
  
  // The LLM response to validate
  string llm_response = 3;
  
  // Context for validation
  UserContext user = 4;
  RetrievalContext retrieval = 5;
  
  // Options
  OutputValidationOptions options = 6;
}

message UserContext {
  string user_id = 1;
  string tenant_id = 2;
  string department = 3;
  repeated string roles = 4;
  repeated string allowed_categories = 5;
  string access_level = 6;  // "full", "anonymized", "k_anonymized", etc.
}

message RetrievalContext {
  repeated string source_documents = 1;
  repeated DocumentSource sources = 2;
  string query = 3;
  string anonymized_query = 4;
}

message DocumentSource {
  string doc_id = 1;
  string snippet = 2;
  float relevance_score = 3;
  map<string, string> metadata = 4;
}

message OutputValidationOptions {
  bool check_pii_reemergence = 1;
  bool check_hallucination = 2;
  bool check_content = 3;
  bool check_permission = 4;
  bool enforce_citation = 5;
  bool strict_mode = 6;
  int32 timeout_ms = 7;
}

message OutputResponse {
  string request_id = 1;
  Status status = 2;
  string validated_response = 3;  // May be modified (sanitized)
  
  ValidationResults checks = 4;
  ModificationLog modifications = 5;
  float processing_time_ms = 6;
  
  enum Status {
    UNKNOWN = 0;
    PASSED = 1;       // Response OK, unchanged
    SANITIZED = 2;    // Response modified (PII redacted, etc.)
    BLOCKED = 3;      // Response blocked, user gets generic message
    WARNING = 4;      // Response passed but with warnings
  }
}

message ValidationResults {
  PIICheck pii_check = 1;
  HallucinationCheck hallucination_check = 2;
  ContentCheck content_check = 3;
  PermissionCheck permission_check = 4;
  FormatCheck format_check = 5;
}

message PIICheck {
  string status = 1;
  repeated PIIIncident incidents = 2;
  int32 redactions_applied = 3;
}

message PIIIncident {
  string pii_type = 1;        // "email", "phone", "korean_rrn", etc.
  string original_value = 2;
  string position = 3;        // Range in response
  string action_taken = 4;    // "redacted", "blocked", "warned"
}

message HallucinationCheck {
  string status = 1;
  float grounding_score = 2;  // 0-1
  repeated string ungrounded_claims = 3;
  string method = 4;          // "heuristic", "ml", "fact_check"
}

message ContentCheck {
  string status = 1;
  repeated string violations = 2;
  string severity = 3;
}

message PermissionCheck {
  string status = 1;
  string user_access_level = 2;
  string response_access_level = 3;
  bool boundary_violated = 4;
}

message FormatCheck {
  string status = 1;
  bool length_ok = 2;
  bool structure_ok = 3;
  repeated string issues = 4;
}

message ModificationLog {
  repeated Modification changes = 1;
}

message Modification {
  string type = 1;            // "redacted", "added_citation", "filtered"
  string position = 2;
  string original = 3;
  string replacement = 4;
}

// Streaming support
message OutputChunk {
  string request_id = 1;
  string token = 2;
  bool is_final = 3;
  UserContext user = 4;
  RetrievalContext retrieval = 5;
}

message OutputAck {
  string request_id = 1;
  string action = 2;          // "allow", "stop", "redact"
  string modified_token = 3;
  string warning = 4;
}
```

### 3.3 REST API

**New Endpoints:**

```
# Output validation
POST /v1/validate/output                    # Validate LLM response
POST /v1/validate/output/stream             # Streaming validation
POST /v1/validate/output/batch              # Batch validation

# Combined endpoint (mode-based)
POST /v1/validate                           # Mode in body
```

**Request Example:**

```http
POST /v1/validate/output HTTP/1.1
Host: sentinel.bastion-rag.local
Content-Type: application/json
Authorization: Bearer <jwt-token>

{
  "request_id": "req-sentinel-out-001",
  "trace_id": "trace-12345",
  "llm_response": "Hong Gildong's purchase history shows...",
  "user": {
    "user_id": "user-alice",
    "tenant_id": "tenant-acme",
    "department": "marketing",
    "roles": ["marketing_analyst"],
    "allowed_categories": ["customer_data"],
    "access_level": "k_anonymized"
  },
  "retrieval": {
    "query": "customer purchase patterns",
    "source_documents": [
      "USER_8f3d2a purchased product X in March...",
      "USER_8f3d2a is in age group 30s..."
    ]
  },
  "options": {
    "check_pii_reemergence": true,
    "check_hallucination": true,
    "check_content": true,
    "check_permission": true,
    "strict_mode": true
  }
}
```

**Response Example (PII Re-emergence Detected):**

```json
{
  "request_id": "req-sentinel-out-001",
  "status": "SANITIZED",
  "validated_response": "[USER_8f3d2a]'s purchase history shows...",
  "processing_time_ms": 45.2,
  "checks": {
    "pii_check": {
      "status": "VIOLATIONS_DETECTED",
      "incidents": [
        {
          "pii_type": "korean_name",
          "original_value": "Hong Gildong",
          "position": "0-12",
          "action_taken": "redacted"
        }
      ],
      "redactions_applied": 1
    },
    "hallucination_check": {
      "status": "PASSED",
      "grounding_score": 0.85,
      "ungrounded_claims": []
    },
    "content_check": {
      "status": "PASSED",
      "violations": []
    },
    "permission_check": {
      "status": "VIOLATION_PREVENTED",
      "user_access_level": "k_anonymized",
      "response_access_level": "full",
      "boundary_violated": true
    },
    "format_check": {
      "status": "PASSED",
      "length_ok": true,
      "structure_ok": true
    }
  },
  "modifications": {
    "changes": [
      {
        "type": "redacted",
        "position": "0-12",
        "original": "Hong Gildong",
        "replacement": "[USER_8f3d2a]"
      }
    ]
  }
}
```

### 3.4 CLI Interface

```bash
# Validate single response
$ sentinel-cli validate-output \
    --response "Hong Gildong's salary is 80M won" \
    --user-id alice \
    --department marketing \
    --access-level k_anonymized

# Output (text format):
════════════════════════════════════════════════════
  Sentinel-OUT Validation Report
════════════════════════════════════════════════════
Status:           SANITIZED 🟡
Processing:       42ms

─── PII Re-emergence Check ─────────────────────────
🚫 VIOLATION DETECTED
  - Korean name detected: "Hong Gildong"
  - Korean salary pattern: "80M won"
  - Action: Both redacted

─── Hallucination Check ────────────────────────────
✅ PASSED (grounding: 0.85)

─── Content Filter ──────────────────────────────────
✅ PASSED

─── Permission Check ───────────────────────────────
🚫 BOUNDARY VIOLATION
  - User access: k_anonymized
  - Response would expose: full names + exact figures

─── Modified Response ──────────────────────────────
"[REDACTED]'s salary is [PRIVATE]"

─── Recommendation ─────────────────────────────────
Original response blocked, user receives:
"This information requires additional permissions."
════════════════════════════════════════════════════

# Batch validation
$ sentinel-cli validate-output \
    --input-file responses.jsonl \
    --output-file results.jsonl

# Streaming mode (test)
$ sentinel-cli stream-test \
    --simulate-llm \
    --user-context user.json

# Configuration management
$ sentinel-cli config show --mode output
$ sentinel-cli config reload --mode output

# Interactive
$ sentinel-cli interactive
sentinel> mode output
Mode set to: output

sentinel> validate "Hong Gildong's email is hong@naver.com"
🚫 BLOCKED: Multiple PII detected (name, email)

sentinel> stats --mode output
Today:
  Validated: 1,234
  Sanitized: 89 (7%)
  Blocked: 12 (1%)

sentinel> exit
```

### 3.5 Text Output Format (for Demos)

```
════════════════════════════════════════════════════
  Sentinel-OUT: Response Validation
════════════════════════════════════════════════════
Trace ID:     trace-12345
User:         alice@tenant-acme (marketing_analyst)
Access:       k_anonymized

─── Original LLM Response ──────────────────────────
"Hong Gildong (hong@naver.com) purchased Product X 
 on March 15, 2026. His total purchases this quarter 
 amount to 5,000,000 won."

─── Validation Results ─────────────────────────────

🔍 PII Re-emergence Check:
  🚫 Korean name detected: "Hong Gildong"
     ↳ Cross-referenced Vault: token USER_8f3d2a exists
     ↳ Action: Redact to token
  🚫 Email detected: "hong@naver.com"
     ↳ Action: Mask
  🚫 Exact amount: "5,000,000 won"
     ↳ User has K-anonymized access, expose ranges only
     ↳ Action: Generalize

🔍 Hallucination Check:
  ✅ "purchased Product X" - grounded in source docs
  ✅ "March 15, 2026" - grounded in source docs
  ⚠️ "5,000,000 won total" - partially grounded

🔍 Permission Check:
  🚫 Response level: detailed (full access required)
  🚫 User level: k_anonymized
  → Boundary violation prevented

🔍 Content Filter:
  ✅ No inappropriate content
  ✅ No harmful information

─── Sanitized Response ─────────────────────────────
"[USER_8f3d2a] purchased Product X on March 15, 2026.
 Total purchases this quarter: in the range of 
 4M-6M won."

─── Tracker Event Published ────────────────────────
event: sentinel.pii_re_emergence_prevented
severity: warning
trace_id: trace-12345
incidents: 3

════════════════════════════════════════════════════
```

---

## 4. Functional Requirements

### 4.1 PII Re-emergence Detection (FR-PR)

**FR-PR-001: Pattern-based Detection**
- Detect common PII patterns in response text
- **Engine Integration:** Support loading rules from `pii-pattern-engine` YAML format.
- Korean: RRN, mobile, email, name patterns
- English: SSN, phone, email, name patterns
- Multilingual support

**FR-PR-002: Vault Mapping Cross-reference**
- Query Vault for known anonymization mappings
- If response contains data that maps to existing token: violation
- Cache mappings for performance (5 min TTL)

**FR-PR-003: Token Pattern Detection**
- Identify Sentinel/Vault tokens that shouldn't appear: leak detection
- Pattern: `^[A-Z]+_[A-Z]+_[a-z0-9]{16}$`
- These should be invisible to end-users

**FR-PR-004: Sanitization Actions**
- Redact: Replace with `[REDACTED]`
- Mask: `john@***.com`
- Tokenize: Replace with corresponding token from Vault
- Block: Reject entire response

**FR-PR-005: Configurable Strictness**
- Strict mode: Block entire response on any PII
- Standard mode: Sanitize and continue
- Lenient mode: Warn but pass

### 4.2 Hallucination Detection (FR-HD)

**FR-HD-001: Grounding Score Calculation**
- Compare response against retrieved context
- Methods:
  - Lexical overlap (simple)
  - Semantic similarity (advanced)
  - Citation matching (when citations present)

**FR-HD-002: Claim Extraction**
- Identify specific factual claims:
  - Numerical values
  - Dates
  - Names
  - Quotes
- Verify each claim against context

**FR-HD-003: Heuristic Detection**
- Common hallucination patterns:
  - Specific numbers not in source
  - Exact dates not in source
  - Names not anonymized in source
  - Detailed quotes not verbatim

**FR-HD-004: Confidence Scoring**
- Overall grounding score: 0.0 - 1.0
- Per-claim confidence
- Threshold for "ungrounded": < 0.5

**FR-HD-005: Mitigation Strategies**
- Add disclaimer: "Information not verified"
- Remove ungrounded claims
- Request LLM to re-generate (advanced)

### 4.3 Content Filtering (FR-CF)

**FR-CF-001: Blacklist Filtering**
- Profanity, hate speech
- Multilingual lists
- Configurable per tenant

**FR-CF-002: Category Filtering**
- Harmful content: violence, self-harm, illegal acts
- Inappropriate: explicit content
- Sensitive: medical/legal advice (with disclaimer)

**FR-CF-003: Pattern-based Filtering**
- API keys, secrets, credentials
- System paths, internal URLs
- IP addresses (internal)

**FR-CF-004: Action Per Violation**
- Block: Entire response rejected
- Replace: Sanitize specific phrases
- Warn: Pass with warning

### 4.4 Permission Boundary Check (FR-PB)

**FR-PB-001: Access Level Comparison**
- Determine response's information level
- Compare with user's access level
- Detect boundary violations

**FR-PB-002: Category Verification**
- Identify data categories referenced in response
- Verify all categories in user's allowed list
- Block if unauthorized category present

**FR-PB-003: Inference Attack Prevention**
- Detect responses that combine allowed data to reveal restricted data
- Example: K-anonymized data + LLM inference = re-identification

**FR-PB-004: Slice Boundary Enforcement**
- For slice-access users (e.g., manufacturing→customer):
- Verify response stays within slice context
- Block cross-context information leakage

### 4.5 Format Validation (FR-FV)

**FR-FV-001: Length Constraints**
- Minimum: 10 characters
- Maximum: Configurable (default 10,000)
- Per-endpoint configurable

**FR-FV-002: Structure Validation**
- Required sections for certain query types
- Forbidden patterns (markdown injection, etc.)

**FR-FV-003: Encoding Validation**
- UTF-8 validity
- No null bytes
- No control characters in output

### 4.6 Citation Enforcement (FR-CE) - Optional

**FR-CE-001: Citation Detection**
- Identify factual claims requiring citation
- Check if citations present

**FR-CE-002: Automatic Citation Addition**
- Pull from retrieval context
- Format: "[Source: doc-123, p.45]"
- Configurable format

**FR-CE-003: Citation Validation**
- Verify cited documents exist
- Verify content matches citation

### 4.7 Streaming Support (FR-ST)

**FR-ST-001: Token-by-Token Processing**
- Process LLM response as it streams
- Buffer for context-dependent checks

**FR-ST-002: Late Detection Handling**
- If PII detected after partial stream:
  - Stop stream
  - Cancel response
  - Send error to user

**FR-ST-003: Buffering Strategy**
- Buffer N tokens (default 50) before validation
- Larger buffer for accuracy, smaller for latency

**FR-ST-004: Early Termination**
- Detect critical violations early
- Cancel LLM generation if possible
- Save resources

### 4.8 Configuration Management (FR-CM)

**FR-CM-001: Mode-specific Configuration**
- Input config (existing)
- Output config (this SRS) ⭐
- Shared rules where applicable

**FR-CM-002: Per-tenant Output Rules**
- Different strictness per tenant
- Custom PII patterns per region
- Industry-specific rules

**FR-CM-003: Hot Reload**
- Reload output rules without restart
- API or signal-based

---

## 5. Non-Functional Requirements

### 5.1 Performance (NFR-PE)

| ID | Item | Target |
|---|---|---|
| NFR-PE-001 | Output validation latency (p95) | < 100ms |
| NFR-PE-002 | Streaming token latency | < 10ms per token |
| NFR-PE-003 | PII pattern matching | < 20ms |
| NFR-PE-004 | Hallucination check | < 50ms |
| NFR-PE-005 | Throughput | ≥ 1,000 responses/s |
| NFR-PE-006 | Memory overhead | ≤ 200MB (in addition to input mode) |

### 5.2 Reliability (NFR-RE)

| ID | Item | Target |
|---|---|---|
| NFR-RE-001 | Availability | 99.99% (same as Sentinel-IN) |
| NFR-RE-002 | False positive rate | < 5% (sanitization actions) |
| NFR-RE-003 | False negative rate | < 1% (missed violations) |
| NFR-RE-004 | Vault integration uptime | Graceful degradation |

### 5.3 Security (NFR-SE)

| ID | Item | Requirement |
|---|---|---|
| NFR-SE-001 | Cannot be bypassed | Required for all LLM responses |
| NFR-SE-002 | Audit completeness | Every decision logged |
| NFR-SE-003 | Fail-safe behavior | Block on uncertainty |
| NFR-SE-004 | Tamper resistance | Configuration integrity |

---

## 6. System Architecture

### 6.1 Unified Sentinel Architecture (Updated)

```
┌─────────────────────────────────────────────────────────┐
│                 Sentinel Service (Unified)               │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌────────────────┐    ┌────────────────┐              │
│  │   Input API    │    │   Output API   │              │
│  │  (/validate/   │    │  (/validate/   │              │
│  │     input)     │    │     output)    │              │
│  └────────┬───────┘    └────────┬───────┘              │
│           │                     │                       │
│           └─────────┬───────────┘                      │
│                     ▼                                   │
│      ┌──────────────────────────────────┐              │
│      │     Mode Dispatcher              │              │
│      │  - Input mode                    │              │
│      │  - Output mode                   │              │
│      │  - Auto-detect                   │              │
│      └──────────────┬───────────────────┘              │
│                     ▼                                   │
│      ┌──────────────────────────────────┐              │
│      │     Validation Engine (Shared)   │              │
│      │  ┌──────┐ ┌──────┐ ┌──────────┐ │              │
│      │  │Regex │ │ Rule │ │   ML     │ │              │
│      │  └──────┘ └──────┘ └──────────┘ │              │
│      └──────┬───────────────────────┬───┘              │
│             │                       │                   │
│      ┌──────▼──────┐         ┌─────▼──────┐            │
│      │Input Config │         │Output Config│ ⭐         │
│      │             │         │             │            │
│      │-Injection   │         │-PII reemerge│            │
│      │ rules       │         │-Hallucination│           │
│      │-Metadata    │         │-Content filt│            │
│      │ schema      │         │-Permission  │            │
│      └─────────────┘         │-Citation    │            │
│                              └─────────────┘            │
│                                     │                   │
│      ┌──────────────────────────────┴───────┐          │
│      │     External Integrations            │          │
│      │  ┌──────────┐ ┌──────────┐ ┌──────┐ │          │
│      │  │  Vault   │ │Navigator │ │NATS  │ │          │
│      │  │ (PII map)│ │(context) │ │(audit│ │          │
│      │  └──────────┘ └──────────┘ └──────┘ │          │
│      └──────────────────────────────────────┘          │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### 6.2 Components

| Component | Responsibility |
|---|---|
| **Mode Dispatcher** | Route to input/output mode |
| **Validation Engine** | Shared logic for both modes |
| **Input Config** | Input-specific rules (existing) |
| **Output Config** | Output-specific rules (new) ⭐ |
| **Vault Integration** | PII mapping lookup |
| **Navigator Integration** | Context verification |
| **NATS Publisher** | Events to Tracker |

### 6.3 Data Flow

```
Output Validation Flow:

[LLM Response]
     ↓
[Sentinel-OUT API]
     ↓
[Mode Dispatcher] → "output mode"
     ↓
[Apply Output Config]
     ↓
[Parallel Checks]:
├─ PII Re-emergence Check
│    └─ Query Vault for mappings
├─ Hallucination Check
│    └─ Query Navigator for context
├─ Content Filter
├─ Permission Check
└─ Format Validation
     ↓
[Aggregate Results]
     ↓
[Apply Modifications]:
├─ Redact PII
├─ Add disclaimers
├─ Remove violations
└─ Add citations
     ↓
[Generate Response]
     ↓
[Publish Event to Tracker]
     ↓
[Return to User]
```

---

## 7. Implementation Details

### 7.1 PII Re-emergence Algorithm

```go
type PIIReEmergenceDetector struct {
    patterns       map[string]*regexp.Regexp
    vaultClient    *VaultClient
    cache          *Cache
}

func (d *PIIReEmergenceDetector) Check(
    response string,
    user UserContext,
) *PIICheckResult {
    result := &PIICheckResult{}
    
    // 1. Pattern-based detection
    patternMatches := d.findPatternMatches(response)
    
    // 2. Cross-reference with Vault
    for _, match := range patternMatches {
        // Query Vault for known mappings
        mapping, err := d.vaultClient.LookupMapping(
            user.TenantID, match.Value)
        
        if mapping != nil {
            // This PII was previously anonymized!
            // LLM should not have used original
            result.AddIncident(PIIIncident{
                Type:           match.Type,
                OriginalValue:  match.Value,
                Position:       match.Position,
                MappedToken:    mapping.Token,
                ActionTaken:    "redact_to_token",
            })
        } else {
            // PII not in mappings, but pattern matches
            result.AddIncident(PIIIncident{
                Type:           match.Type,
                OriginalValue:  match.Value,
                Position:       match.Position,
                ActionTaken:    "mask",
            })
        }
    }
    
    // 3. Detect leaked tokens (Vault tokens shouldn't appear to user)
    tokenMatches := d.findTokenMatches(response)
    for _, match := range tokenMatches {
        result.AddIncident(PIIIncident{
            Type:           "leaked_token",
            OriginalValue:  match.Value,
            ActionTaken:    "redact",
        })
    }
    
    return result
}

func (d *PIIReEmergenceDetector) Sanitize(
    response string,
    incidents []PIIIncident,
) string {
    sanitized := response
    
    // Sort incidents by position (descending) to apply from end
    sort.Slice(incidents, func(i, j int) bool {
        return incidents[i].Position.Start > incidents[j].Position.Start
    })
    
    for _, incident := range incidents {
        replacement := d.getReplacement(incident)
        sanitized = applyReplacement(sanitized, incident.Position, replacement)
    }
    
    return sanitized
}
```

### 7.2 Hallucination Detection Algorithm

```go
type HallucinationDetector struct {
    nlpClient *NLPClient
}

func (d *HallucinationDetector) Check(
    response string,
    context RetrievalContext,
) *HallucinationResult {
    // 1. Extract claims from response
    claims := d.extractClaims(response)
    
    // 2. Verify each claim
    var groundedCount int
    var ungroundedClaims []string
    
    for _, claim := range claims {
        if d.isGrounded(claim, context.SourceDocuments) {
            groundedCount++
        } else {
            ungroundedClaims = append(ungroundedClaims, claim)
        }
    }
    
    // 3. Calculate grounding score
    score := float64(groundedCount) / float64(len(claims))
    
    return &HallucinationResult{
        Status:           getStatus(score),
        GroundingScore:   score,
        UngroundedClaims: ungroundedClaims,
        Method:           "heuristic",
    }
}

func (d *HallucinationDetector) extractClaims(response string) []Claim {
    claims := []Claim{}
    
    // Extract numerical claims
    for _, match := range numericalPattern.FindAllString(response, -1) {
        claims = append(claims, Claim{
            Type:  "numerical",
            Value: match,
        })
    }
    
    // Extract date claims
    for _, match := range datePattern.FindAllString(response, -1) {
        claims = append(claims, Claim{
            Type:  "date",
            Value: match,
        })
    }
    
    // Extract name claims, etc.
    
    return claims
}

func (d *HallucinationDetector) isGrounded(
    claim Claim,
    sources []string,
) bool {
    for _, source := range sources {
        if strings.Contains(source, claim.Value) {
            return true
        }
    }
    return false
}
```

### 7.3 Permission Boundary Check

```go
func (s *Sentinel) checkPermissionBoundary(
    response string,
    user UserContext,
) *PermissionResult {
    // Determine response's information level
    responseLevel := s.analyzeResponseLevel(response)
    
    // Compare with user's access
    if responseLevel.RequiresFullAccess() && 
       user.AccessLevel != "full" {
        return &PermissionResult{
            Status:              "VIOLATION_PREVENTED",
            UserAccessLevel:     user.AccessLevel,
            ResponseAccessLevel: "full",
            BoundaryViolated:    true,
        }
    }
    
    // Check K-anonymity preservation
    if user.AccessLevel == "k_anonymized" {
        if s.detectKAnonymityViolation(response) {
            return &PermissionResult{
                Status:           "VIOLATION_PREVENTED",
                BoundaryViolated: true,
                Reason:           "Specific values exposed for k-anonymized user",
            }
        }
    }
    
    return &PermissionResult{Status: "PASSED"}
}
```

---

## 8. Standalone Testing

### 8.1 Standalone Modes

```bash
# Test mode (no Vault/Navigator dependency)
$ sentinel-cli server --standalone --mode output

# Mock external services
$ sentinel-cli server --mock-vault --mock-navigator

# Demo mode
$ sentinel-cli demo --scenario output-validation
```

### 8.2 Test Data

```
tests/output/
├── fixtures/
│   ├── llm_responses_clean.jsonl       # Clean responses
│   ├── llm_responses_pii.jsonl         # PII re-emergence cases
│   ├── llm_responses_hallucination.jsonl
│   ├── llm_responses_inappropriate.jsonl
│   └── llm_responses_permission_violation.jsonl
├── golden/
│   └── expected_validations/
└── streaming/
    └── streaming_scenarios.jsonl
```

### 8.3 Demo Scenarios (for Tracker)

```yaml
scenario: "PII Re-emergence Prevention"
description: "LLM tries to leak original name, Sentinel-OUT catches it"
steps:
  - User asks: "What is customer's purchase pattern?"
  - Vault anonymizes: Hong Gildong → USER_8f3d2a
  - Navigator searches: returns context with token
  - LLM responds: "Hong Gildong purchased..." [violation!]
  - Sentinel-OUT detects: PII re-emergence
  - Response sanitized: "[USER_8f3d2a] purchased..."
  - User sees: Safe response
  - Tracker logs: Security event

scenario: "Hallucination Detection"
description: "LLM fabricates information, Sentinel-OUT flags it"
steps:
  - User asks: "What is PROD-001 price?"
  - Navigator returns: PROD-001 specs (no price)
  - LLM responds: "PROD-001 costs $500" [hallucination!]
  - Sentinel-OUT detects: ungrounded claim
  - Response modified: "Price information not available. Please contact sales."

scenario: "Permission Boundary Enforcement"
description: "K-anonymized user gets too-detailed answer, Sentinel-OUT prevents"
steps:
  - User permission: marketing_analyst (k_anonymized)
  - LLM response: "Customer Kim purchased $5,000 worth"
  - Sentinel-OUT detects: specific values for k-anon user
  - Response modified: "Customers in this segment purchased in the $4K-$6K range"
```

---

## 9. Configuration Schema

```yaml
# /etc/bastion-sentinel/output-config.yaml
version: 1.0

mode: output

# PII Re-emergence detection
pii_reemergence:
  enabled: true
  
  patterns:
    korean_name:
      regex: '[가-힣]{2,4}(?:님|씨)?'
      severity: high
    korean_rrn:
      regex: '\d{6}-\d{7}'
      severity: critical
    korean_mobile:
      regex: '01[0-9]-?\d{3,4}-?\d{4}'
      severity: high
    email:
      regex: '[\w.+-]+@[\w-]+\.[\w.-]+'
      severity: high
    credit_card:
      regex: '\d{4}-?\d{4}-?\d{4}-?\d{4}'
      severity: critical
    leaked_token:
      regex: '^[A-Z]+_[A-Z]+_[a-z0-9]{16}$'
      severity: high
  
  vault_integration:
    enabled: true
    endpoint: http://vault:8080
    cache_ttl: 5m
  
  actions:
    redact: true
    mask_partial: true
    block_on_critical: true

# Hallucination detection
hallucination:
  enabled: true
  
  navigator_integration:
    enabled: true
    endpoint: http://navigator:8080
  
  grounding_threshold: 0.5
  
  methods:
    - lexical_overlap
    - claim_extraction
    # - semantic_similarity (future)
  
  actions:
    add_disclaimer: true
    remove_ungrounded: false
    block_on_low_score: true
    low_score_threshold: 0.3

# Content filter
content_filter:
  enabled: true
  
  blacklists:
    profanity:
      path: /etc/sentinel/blacklist-profanity.txt
    harmful_content:
      categories: [violence, self_harm, illegal]
    
  patterns:
    api_keys:
      regex: 'sk-[a-zA-Z0-9]{32,}'
      severity: critical
    internal_paths:
      regex: '/var/.*|/etc/.*'
      severity: medium

# Permission check
permission_check:
  enabled: true
  
  access_levels:
    - full
    - read
    - anonymized
    - k_anonymized
    - slice
    - aggregated
  
  k_anonymity_enforcement:
    strict: true
    detect_specific_values: true

# Format validation
format:
  min_length: 10
  max_length: 10000
  utf8_required: true
  no_control_chars: true

# Citation
citation:
  enabled: false  # Optional
  format: "[Source: {doc_id}]"

# Streaming
streaming:
  enabled: true
  buffer_size: 50
  validate_chunks: true
  early_termination: true

# Performance
performance:
  parallel_checks: true
  timeout_ms: 100
  cache_results: true
  cache_ttl: 1m

# Tracker
tracker:
  enabled: true
  publish_violations: true
```

---

## 10. Deployment

### 10.1 Same Service, Extended Configuration

Sentinel runs as a single service handling both input and output:

```yaml
# Docker Compose - no change to deployment structure
sentinel:
  image: bastion-rag/sentinel:1.1.0  # Version bumped for output support
  ports:
    - "8080:8080"    # REST (both input and output)
    - "9090:9090"    # gRPC
  environment:
    - INPUT_CONFIG=/etc/sentinel/input-config.yaml
    - OUTPUT_CONFIG=/etc/sentinel/output-config.yaml
  resources:
    limits:
      memory: 1G     # Same as input-only (shared engine)
      cpu: '2'
```

### 10.2 Backward Compatibility

- Existing `/v1/validate` endpoint continues to work (input mode)
- New `/v1/validate/output` endpoint for output validation
- Combined `/v1/validate` with `mode` parameter

---

## 11. Tracker Integration

### 11.1 New Events

```yaml
# Events published by Sentinel-OUT
events:
  - sentinel.output_validated         # Successful validation
  - sentinel.pii_re_emergence_prevented   # PII caught
  - sentinel.hallucination_detected        # Hallucination flagged
  - sentinel.content_filtered              # Content blocked
  - sentinel.permission_violation_prevented # Access boundary enforced
  - sentinel.response_blocked              # Full response rejected
  - sentinel.response_sanitized            # Response modified
```

### 11.2 Visualization in Tracker

```
Live Flow (Updated):

[User] → [Sentinel-IN] → [Vault] → [Navigator] → [Anchor] → [LLM]
                                                              ↓
[User] ← [Sentinel-OUT] ← [Vault-OUT] ← ─ ─ ─ ─ ← [Anchor-OUT] ─┘

Both Sentinel nodes shown - same module, different positions
Same color/style (unified module)
```

---

## 12. Migration Path

### 12.1 Phased Rollout

**Phase 1: Deploy Output Mode (Detection Only)**
- Sentinel-OUT logs violations but doesn't block
- Gather data on false positives
- Tune thresholds

**Phase 2: Sanitization Mode**
- Apply redactions and modifications
- Don't block full responses
- Monitor user experience

**Phase 3: Full Enforcement**
- Block on critical violations
- Sanitize on warnings
- Production-ready

### 12.2 Configuration Versioning

```yaml
config_version: 1.0
features:
  output_validation: enabled
  output_blocking: disabled  # Phase 1
  output_sanitization: enabled  # Phase 2
  output_full_enforcement: disabled  # Phase 3
```

---

## 13. Appendix

### 13.1 Common Patterns

**Pattern: LLM Tries to Reveal Anonymized Data**
```
Input: "Tell me about USER_8f3d2a's history"
Context: User has K-anonymized access
LLM Response: "USER_8f3d2a is actually Hong Gildong, who..."
              ↑ Re-identification attempt

Sentinel-OUT Action: Redact "Hong Gildong" → "[USER_8f3d2a]"
```

**Pattern: Hallucination of Specific Numbers**
```
Context: "Product PROD-001 specifications..."
LLM Response: "PROD-001 has 89% efficiency rating"
              ↑ Specific number not in context

Sentinel-OUT Action: Flag as ungrounded, modify or remove
```

**Pattern: Cross-Permission Leak**
```
Marketing user asks about products
Search returns: Product info + Customer purchase data
LLM Response: Includes customer names and purchases
              ↑ Customer data is for marketing OK, but specifics?

Sentinel-OUT Action: Generalize customer-specific details
```

### 13.2 Edge Cases

**Streaming + Late PII Detection**
```
Tokens already sent: "Hong Gildong's purchase..."
PII detected at token 5
Action: 
  - Stop further tokens
  - Send error/correction to user
  - Note: cannot un-send already-streamed tokens
  
Mitigation: Increase buffer size for sensitive contexts
```

**Multilingual PII**
```
Response: "홍길동(Hong Gildong)의 이메일은..."
Detection: Both Korean and English name
Action: Redact both
```

### 13.3 Demo Walkthrough (3 minutes)

```
0:00-0:30  Setup: Show user query "Customer purchase summary"
0:30-1:00  Show Vault anonymization (PII → tokens)
1:00-1:30  Show Navigator finding documents (with tokens)
1:30-2:00  Show LLM response (containing original name!)
2:00-2:30  Sentinel-OUT detection (red highlight on PII)
2:30-3:00  Sanitized response delivered to user
```

### 13.4 Roadmap

- v1.1: LLM-based semantic analysis
- v1.2: Real-time grounding verification
- v1.3: Multi-modal output (images)
- v2.0: Adversarial response detection
- v2.1: Self-supervised hallucination detection

### 13.5 Change History

| Version | Date | Changes |
|---|---|---|
| 0.1 | 2026-05-17 | Initial draft based on output gap analysis |
| 1.0 | 2026-05-17 | First release - bidirectional Sentinel |

---

## 14. Summary

### What This SRS Adds

```
Before:
Sentinel = Input gateway only
Output = Unprotected ❌

After:
Sentinel = Bidirectional gateway
- Sentinel-IN: Input validation (existing)
- Sentinel-OUT: Output validation (this SRS) ✅
```

### Key Design Decisions

1. **Unified Service**: Single Sentinel handles both directions
2. **Shared Engine**: Same validation engine, different configurations
3. **Symmetric Architecture**: Matches Vault's Two-Phase pattern
4. **Independent Configuration**: Output rules separate from input
5. **External Integration**: Leverages Vault and Navigator for context
6. **Streaming Support**: Handles real-time LLM responses

### Value Delivered

- ✅ Closes the output security gap
- ✅ Maintains 5-module architecture
- ✅ Consistent design pattern
- ✅ Enables complete PoC demonstration
- ✅ Production-ready security model

---

**End of Document**
