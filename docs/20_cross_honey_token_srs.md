# Bastion Honey-Token Cross-Cutting SRS

**Project:** Bastion - RAG Security Governance Framework  
**Document Type:** Cross-Cutting SRS (Tier 3)  
**Document ID:** 20-honey-token-srs  
**Feature:** Honey-Token (Intrusion Detection)  
**Version:** 1.0  
**Date:** 2026-05-17  
**Status:** Draft

**Foundation References:**
- 01-architecture-principles
- 02-event-schema-standard
- 03-module-interaction-map

**Participating Modules:**
- Vault (create/inject/identify/data-detect) ⭐ Owner
- Sentinel (input/output detection)
- Navigator (search detection)
- Tracker (aggregate/alert) ⭐ Coordinator

---

## 1. Introduction

### 1.1 Purpose

This document redefines the **Honey-Token** capability within the Bastion framework. Originally conceived as a Tracker-only feature, analysis reveals that honey-tokens are a **cross-cutting concern** spanning multiple modules.

This document:
1. Explains why honey-tokens cannot be a single-module responsibility
2. Defines the honey-token lifecycle
3. Distributes responsibilities across modules
4. Specifies detection points throughout the pipeline
5. Provides SRS amendment guidance

### 1.2 What is a Honey-Token?

A honey-token (or honeypot data) is **decoy data** intentionally planted within real datasets to detect unauthorized access or data breaches.

```
Examples:
- Fake customer record: "John Honeypot, ssn-fake-001"
- Fake credential: "api_key_TRAP_xyz123"
- Fake email: "ceo-decoy@company-trap.com"
- Fake employee: "Honey Trap, Salary: 999,999,999"

Principle:
- No legitimate user should ever access these
- Any access = potential breach
- Triggers immediate alert
```

### 1.3 Background: The Original Misconception

#### Original Design (Incorrect)

```
Tracker = Honey-token creation + injection + detection + alerting
```

#### Why This is Wrong

```
Problem 1: Tracker doesn't handle data
- Tracker is an observer (cross-cutting)
- It cannot inject tokens into datasets
- It cannot identify which data is a honey-token

Problem 2: Detection happens at multiple points
- Input queries (Sentinel)
- Search results (Navigator)
- Data access (Vault)
- Output responses (Sentinel-OUT)

Problem 3: Only Vault knows the data
- Vault manages anonymization mappings
- Vault knows real vs. fake data
- Vault should mark honey-tokens
```

### 1.4 Key Insight

```
Honey-tokens are NOT a single-module feature.

They are a CROSS-CUTTING CONCERN:
- Creation/Injection/Identification → Vault (data owner)
- Detection → Multiple modules (each at their layer)
- Aggregation/Alerting → Tracker (observer)
```

---

## 2. Honey-Token Lifecycle

### 2.1 Complete Lifecycle

```
┌─────────────────────────────────────────────────────┐
│  Honey-Token Lifecycle                              │
├─────────────────────────────────────────────────────┤
│                                                     │
│  1. Creation      → Generate fake data              │
│         ↓                                           │
│  2. Injection     → Insert into real dataset        │
│         ↓                                           │
│  3. Identification→ Mark/tag as honey-token         │
│         ↓                                           │
│  4. Storage       → Store alongside real data       │
│         ↓                                           │
│  ─────────────────────────────────────             │
│         ↓ (runtime)                                 │
│  5. Detection     → Catch access attempts           │
│     ├─ Input layer (query references)              │
│     ├─ Search layer (in results)                   │
│     ├─ Data layer (direct access)                  │
│     └─ Output layer (in response)                  │
│         ↓                                           │
│  6. Attribution   → Identify who/when/how           │
│         ↓                                           │
│  7. Alerting      → Notify security team            │
│         ↓                                           │
│  8. Incident Mgmt → Track and respond               │
│                                                     │
└─────────────────────────────────────────────────────┘
```

### 2.2 Lifecycle Phases by Module

| Phase | Responsible Module | Rationale |
|---|---|---|
| **1. Creation** | Vault | Owns data generation |
| **2. Injection** | Vault | Manages data storage |
| **3. Identification** | Vault | Knows real vs. fake |
| **4. Storage** | Vault | Controls data layer |
| **5a. Input Detection** | Sentinel-IN | Sees queries |
| **5b. Search Detection** | Navigator | Sees search results |
| **5c. Data Detection** | Vault | Sees data access |
| **5d. Output Detection** | Sentinel-OUT | Sees responses |
| **6. Attribution** | Tracker | Aggregates events |
| **7. Alerting** | Tracker | Central notifications |
| **8. Incident Mgmt** | Tracker | Tracks incidents |

---

## 3. Responsibility Distribution

### 3.1 Vault: The Honey-Token Owner

Vault is the **primary owner** of honey-tokens because it controls the data layer.

**Responsibilities:**
```
Creation:
- Generate realistic fake data
- Multiple token types (email, credential, identity, etc.)
- Configurable templates

Injection:
- Insert honey-tokens into datasets during indexing
- Distribute strategically (not clustered)
- Maintain ratio (e.g., 1 honey-token per 1000 records)

Identification:
- Tag honey-tokens with metadata
- Maintain honey-token registry
- Mark in storage

Data-level Detection:
- Detect when honey-token data is accessed
- Flag deanonymization attempts on honey-tokens
- Immediate event on access
```

**Storage Schema:**
```json
{
  "record_id": "rec-12345",
  "data": {
    "name": "Honey Trap",
    "email": "decoy@honeypot.local",
    "ssn": "FAKE-000-00-0001"
  },
  "is_honey_token": true,
  "honey_token_id": "HT-001",
  "honey_token_type": "fake_identity",
  "created_at": "2026-04-15T00:00:00Z",
  "trap_metadata": {
    "expected_access": "never",
    "severity_on_access": "critical"
  }
}
```

### 3.2 Sentinel: Input/Output Detection

**Sentinel-IN Responsibilities:**
```
Input Query Detection:
- Detect honey-token references in user queries
- Example: User asks "Tell me about decoy@honeypot.local"
- This means: someone knows about the honey-token!
- Likely: data was already breached elsewhere

Detection Method:
- Maintain honey-token reference list (from Vault)
- Pattern match against query
- Immediate flag on match
```

**Sentinel-OUT Responsibilities:**
```
Output Leakage Detection:
- Detect honey-tokens in LLM responses
- Example: LLM response contains "Honey Trap" record
- This means: honey-token was retrieved and exposed

Detection Method:
- Check response against honey-token list
- Flag any honey-token in output
- Block response + alert
```

### 3.3 Navigator: Search-Layer Detection

**Navigator Responsibilities:**
```
Search Result Detection:
- Detect honey-tokens in search results
- Example: Vector search returns honey-token document
- This means: someone's query matched honey-token

Detection Method:
- Check returned documents for honey-token flag
- Flag if honey-token appears in results
- Continue search but emit event

Why Navigator?
- Honey-token might be semantically similar to malicious query
- Attacker searching for sensitive data might hit honey-token
- Vector space detection is unique to Navigator
```

### 3.4 Tracker: Aggregation & Response

**Tracker Responsibilities (Revised):**
```
NOT responsible for (removed):
❌ Creation
❌ Injection
❌ Identification
❌ Direct detection

Responsible for (retained):
✅ Event Aggregation
   - Collect honey-token events from all modules
   - Correlate related events

✅ Attribution Analysis
   - Who accessed (user_id)
   - When (timestamp)
   - Where (IP, location)
   - How (input/search/data/output)

✅ Alerting
   - Immediate notifications
   - Multi-channel (Slack, email)
   - Severity-based escalation

✅ Incident Management
   - Create incidents from honey-token triggers
   - Track investigation
   - Coordinate response

✅ Visualization
   - Honey-token dashboard
   - Trigger timeline
   - Attribution maps
```

---

## 4. Detection Points (Multi-Layer)

### 4.1 The Four Detection Layers

```
User Query
    ↓
┌─────────────────┐
│ Layer 1: INPUT  │ ← Sentinel-IN
│ Query references│   "User mentions honey-token"
│ honey-token?    │
└────────┬────────┘
         ↓
┌─────────────────┐
│ Layer 2: DATA   │ ← Vault
│ Accessing       │   "Honey-token data accessed"
│ honey-token?    │
└────────┬────────┘
         ↓
┌─────────────────┐
│ Layer 3: SEARCH │ ← Navigator
│ Honey-token in  │   "Search returned honey-token"
│ results?        │
└────────┬────────┘
         ↓
        LLM
         ↓
┌─────────────────┐
│ Layer 4: OUTPUT │ ← Sentinel-OUT
│ Honey-token in  │   "Response contains honey-token"
│ response?       │
└────────┬────────┘
         ↓
    All events → Tracker
```

### 4.2 Detection Scenarios

**Scenario A: Direct Reference (Input Layer)**
```
Attacker knows honey-token (from prior breach):
Query: "What's the status of decoy@honeypot.local?"

Sentinel-IN detects:
- Query references known honey-token
- HIGH severity: attacker has prior knowledge
- Action: Block + immediate alert
```

**Scenario B: Data Access (Data Layer)**
```
Insider tries to access all customer data:
Query: "Export all customer records"

Vault detects:
- Honey-token record included in access
- Honey-token should never be accessed
- Action: Flag access + alert
```

**Scenario C: Search Match (Search Layer)**
```
Attacker searches for sensitive patterns:
Query: "high value customers with full SSN"

Navigator detects:
- Honey-token (fake high-value customer) in results
- Suspicious search pattern
- Action: Emit event + continue
```

**Scenario D: Response Leak (Output Layer)**
```
LLM includes honey-token in response:
Response: "...including Honey Trap (decoy@honeypot.local)..."

Sentinel-OUT detects:
- Honey-token in response
- Data was retrieved and about to leak
- Action: Block response + alert
```

### 4.3 Severity by Layer

| Layer | Severity | Reasoning |
|---|---|---|
| **Input** | 🔴 Critical | Attacker has prior knowledge |
| **Data** | 🟠 High | Unauthorized data access |
| **Search** | 🟡 Medium | Suspicious search pattern |
| **Output** | 🔴 Critical | Active data exfiltration |

---

## 5. Event Schema

### 5.1 Honey-Token Event Structure

```protobuf
message HoneyTokenEvent {
  string event_id = 1;
  string trace_id = 2;
  
  string honey_token_id = 3;
  string honey_token_type = 4;
  
  // Detection details
  string detection_layer = 5;     // "input", "data", "search", "output"
  string detecting_module = 6;    // "sentinel", "vault", "navigator"
  
  // Attribution
  string user_id = 7;
  string tenant_id = 8;
  string source_ip = 9;
  google.protobuf.Timestamp timestamp = 10;
  
  // Context
  string query = 11;
  string access_method = 12;
  
  // Severity
  string severity = 13;           // "critical", "high", "medium"
  string recommended_action = 14;
}
```

### 5.2 Events by Module

```yaml
# Vault events
vault.honey_token_created       # Token generated
vault.honey_token_injected      # Token inserted
vault.honey_token_accessed      # Data-layer detection ⭐

# Sentinel events
sentinel.honey_token_referenced # Input-layer detection ⭐
sentinel.honey_token_leaked     # Output-layer detection ⭐

# Navigator events
navigator.honey_token_retrieved # Search-layer detection ⭐

# Tracker events (aggregation)
tracker.honey_token_incident    # Incident created
tracker.honey_token_alert       # Alert sent
```

---

## 6. Data Flow

### 6.1 Injection Flow (Indexing Time)

```
Document Ingestion:
[Real Documents] + [Honey-Tokens]
         ↓
      [Vault]
      ├─ Generate honey-tokens
      ├─ Mix with real data
      ├─ Tag: {is_honey: true, honey_id: HT-001}
      └─ Apply anonymization (real data only)
         ↓
   [Embedding Service (BGE-M3)]
      └─ Generate embeddings for all documents
         (same service Navigator uses at query time)
         ↓
   [Vector DB / Indexer]
   (honey-tokens stored alongside real data)

Note: Anchor is NOT involved in document indexing.
Anchor Phase 1 operates on query embeddings at request time
(before the LLM), not on document embeddings at index time.
```

### 6.2 Detection Flow (Runtime)

```
User Query
    ↓
[Sentinel-IN] ──── Layer 1: Reference check
    │              └─→ event: honey_token_referenced
    ↓
[Vault] ────────── Layer 2: Data access check
    │              └─→ event: honey_token_accessed
    ↓
[Navigator] ────── Layer 3: Search result check
    │              └─→ event: honey_token_retrieved
    ↓
   LLM
    ↓
[Anchor-OUT]
    ↓
[Vault-OUT]
    ↓
[Sentinel-OUT] ─── Layer 4: Output leak check
    │              └─→ event: honey_token_leaked
    ↓
User Response
    
All events ──────→ [Tracker]
                    ├─ Aggregate
                    ├─ Attribute
                    ├─ Alert
                    └─ Create incident
```

---

## 7. Module SRS Amendments

### 7.1 Vault SRS Amendments

**Add to Functional Requirements:**

```
FR-HT-001: Honey-Token Creation
- Generate realistic fake data
- Support types: email, credential, identity, financial
- Template-based generation
- Configurable per category

FR-HT-002: Honey-Token Injection
- Insert during indexing
- Strategic distribution
- Configurable ratio (default 1:1000)
- Avoid clustering

FR-HT-003: Honey-Token Identification
- Tag with metadata
- Maintain registry
- Mark in storage schema

FR-HT-004: Honey-Token Access Detection
- Detect data-layer access
- Flag deanonymization attempts
- Emit immediate event
```

**Add to Configuration:**
```yaml
honey_tokens:
  enabled: true
  
  generation:
    types:
      - fake_identity
      - fake_credential
      - fake_financial
    ratio: 0.001  # 1 per 1000 records
  
  injection:
    distribution: random
    avoid_clustering: true
  
  detection:
    on_access: alert
    severity: critical
```

### 7.2 Sentinel SRS Amendments

**Add to Input Validation (Sentinel-IN):**
```
FR-HT-IN-001: Honey-Token Reference Detection
- Maintain honey-token reference list (synced from Vault)
- Pattern match against incoming queries
- Detect direct references
- Severity: CRITICAL (prior knowledge implies breach)
```

**Add to Output Validation (Sentinel-OUT):**
```
FR-HT-OUT-001: Honey-Token Leak Detection
- Check responses for honey-tokens
- Block response if honey-token present
- Severity: CRITICAL (active exfiltration)
```

### 7.3 Navigator SRS Amendments

**Add to Functional Requirements:**
```
FR-HT-NAV-001: Honey-Token Search Detection
- Check search results for honey-token flag
- Emit event if honey-token in results
- Continue search (don't block, but log)
- Severity: MEDIUM (suspicious pattern)
```

### 7.4 Tracker SRS Amendments

**Revise Honey-Token Section:**
```
REMOVE:
- FR: Honey-token creation
- FR: Honey-token injection
- FR: Direct detection

REVISE TO:
FR-HT-TRK-001: Event Aggregation
- Collect honey-token events from all modules
- Correlate related events by trace_id

FR-HT-TRK-002: Attribution Analysis
- Build attacker profile
- Track access patterns
- Geographic analysis

FR-HT-TRK-003: Alerting
- Immediate notifications
- Severity-based routing
- Escalation workflows

FR-HT-TRK-004: Incident Management
- Auto-create incidents
- Investigation tracking
- Response coordination

FR-HT-TRK-005: Visualization
- Honey-token dashboard
- Trigger timeline
- Attribution maps
```

---

## 8. Visualization (Tracker)

### 8.1 Honey-Token Dashboard

```
┌──────────────────────────────────────────────────────┐
│  🍯 Honey-Token Monitoring                            │
├──────────────────────────────────────────────────────┤
│  Active Tokens: 47  |  Triggers (24h): 3             │
│                                                       │
│  ─── Multi-Layer Detection Status ──────────         │
│                                                       │
│  Layer          Triggers   Last Event                │
│  ─────────────────────────────────────               │
│  Input          1          14:23 (critical)          │
│  Data           2          14:22 (high)              │
│  Search         0          -                         │
│  Output         0          -                         │
│                                                       │
│  ─── Active Incident ─────────────────               │
│                                                       │
│  🚨 INC-001: Multi-layer honey-token trigger         │
│  ├ HT-001 referenced in input (14:23)               │
│  ├ HT-001 accessed in data (14:22)                  │
│  ├ Same user: suspicious-user@external              │
│  ├ Source IP: 192.168.1.123                         │
│  └ Pattern: User knows token + tried access         │
│                                                       │
│  Attribution:                                         │
│  - User: suspicious-user@external                    │
│  - First seen: 14:20                                 │
│  - Tokens triggered: HT-001                          │
│  - Layers: input, data                               │
│  - Verdict: LIKELY BREACH                            │
│                                                       │
│  [Investigate] [Block User] [Escalate]               │
└──────────────────────────────────────────────────────┘
```

### 8.2 Detection Flow Visualization

```
Live honey-token detection across layers:

[User: suspicious-user]
       ↓
[Sentinel-IN] 🚨 ← "References HT-001!"
       ↓        (event sent)
[Vault] 🚨 ← "Accessing HT-001 data!"
       ↓        (event sent)
[Navigator]
       ↓
      LLM
       ↓
[Sentinel-OUT]
       ↓
[User]

[Tracker] ← Correlates: same user, multiple layers
         → INCIDENT: Likely breach
         → ALERT: Security team
```

---

## 9. Implementation Examples

### 9.1 Vault: Honey-Token Generation

```go
type HoneyTokenManager struct {
    registry  *HoneyTokenRegistry
    generator *FakeDataGenerator
}

func (h *HoneyTokenManager) Create(
    tokenType string,
    category string,
) (*HoneyToken, error) {
    // Generate realistic fake data
    fakeData := h.generator.Generate(tokenType)
    
    token := &HoneyToken{
        ID:        generateID("HT"),
        Type:      tokenType,
        Category:  category,
        Data:      fakeData,
        CreatedAt: time.Now(),
        Metadata: map[string]string{
            "expected_access": "never",
            "severity": "critical",
        },
    }
    
    // Register
    h.registry.Add(token)
    
    return token, nil
}

func (h *HoneyTokenManager) Inject(
    dataset []Record,
    ratio float64,
) []Record {
    count := int(float64(len(dataset)) * ratio)
    
    for i := 0; i < count; i++ {
        token, _ := h.Create("fake_identity", "customer_data")
        
        record := Record{
            Data:          token.Data,
            IsHoneyToken:  true,
            HoneyTokenID:  token.ID,
        }
        
        // Insert at random position
        pos := rand.Intn(len(dataset))
        dataset = insert(dataset, pos, record)
    }
    
    return dataset
}

func (h *HoneyTokenManager) CheckAccess(
    record Record,
    user UserContext,
) *HoneyTokenEvent {
    if record.IsHoneyToken {
        return &HoneyTokenEvent{
            HoneyTokenID:    record.HoneyTokenID,
            DetectionLayer:  "data",
            DetectingModule: "vault",
            UserID:          user.UserID,
            Severity:        "high",
            Timestamp:       time.Now(),
        }
    }
    return nil
}
```

### 9.2 Sentinel: Reference Detection

```go
func (s *Sentinel) checkHoneyTokenReference(
    query string,
) *HoneyTokenEvent {
    // Get honey-token references from cache (synced from Vault)
    references := s.honeyTokenCache.GetReferences()
    
    for _, ref := range references {
        if strings.Contains(query, ref.Value) {
            // Query references a honey-token!
            return &HoneyTokenEvent{
                HoneyTokenID:    ref.ID,
                DetectionLayer:  "input",
                DetectingModule: "sentinel",
                Severity:        "critical",  // Prior knowledge!
                Query:           query,
            }
        }
    }
    return nil
}
```

### 9.3 Navigator: Search Detection

```go
func (n *Navigator) checkHoneyTokenInResults(
    results []SearchResult,
) []*HoneyTokenEvent {
    var events []*HoneyTokenEvent
    
    for _, result := range results {
        if result.Metadata["is_honey_token"] == "true" {
            events = append(events, &HoneyTokenEvent{
                HoneyTokenID:    result.Metadata["honey_token_id"],
                DetectionLayer:  "search",
                DetectingModule: "navigator",
                Severity:        "medium",
            })
        }
    }
    
    return events
}
```

---

## 10. Summary

### 10.1 Before vs. After

```
BEFORE (Incorrect):
Tracker = Everything (creation + detection + alerting)
Problem: Tracker doesn't touch data!

AFTER (Correct):
Vault     = Create, inject, identify, data-detect
Sentinel  = Input/output detection
Navigator = Search detection
Tracker   = Aggregate, attribute, alert, manage
```

### 10.2 Responsibility Matrix

| Capability | Vault | Sentinel | Navigator | Tracker |
|---|:---:|:---:|:---:|:---:|
| Creation | ✅ | | | |
| Injection | ✅ | | | |
| Identification | ✅ | | | |
| Input detection | | ✅ | | |
| Data detection | ✅ | | | |
| Search detection | | | ✅ | |
| Output detection | | ✅ | | |
| Aggregation | | | | ✅ |
| Attribution | | | | ✅ |
| Alerting | | | | ✅ |
| Incident mgmt | | | | ✅ |

### 10.3 Key Principles

```
1. Vault owns the data → owns honey-token lifecycle
2. Detection is multi-layer → each module detects at its layer
3. Tracker observes → aggregates and responds
4. Honey-token is cross-cutting → not a single module
```

### 10.4 Value of This Design

- ✅ Correct separation of concerns
- ✅ Multi-layer detection (defense in depth)
- ✅ Each module detects what it can see
- ✅ Tracker focuses on its strength (observation)
- ✅ More robust breach detection

---

## 11. Detection Effectiveness

### 11.1 Why Multi-Layer is Better

```
Single-layer (old):
Only Tracker → misses context

Multi-layer (new):
- Input: catches attackers with prior knowledge
- Data: catches unauthorized access
- Search: catches suspicious patterns
- Output: catches exfiltration attempts

→ Higher detection rate
→ Better attribution
→ Defense in depth
```

### 11.2 Correlation Power

```
Tracker correlates across layers:

If same user triggers:
- Input layer (knows token)
+ Data layer (accessed it)
= VERY HIGH confidence breach

vs.

Single trigger:
- Just search layer
= Possibly innocent
```

---

## 12. Appendix

### 12.1 Honey-Token Types

| Type | Example | Use Case |
|---|---|---|
| **Fake Identity** | "Honey Trap, fake-ssn" | Customer DB monitoring |
| **Fake Credential** | "api_key_TRAP_xyz" | Credential theft detection |
| **Fake Financial** | "Account: 0000-TRAP" | Financial data monitoring |
| **Fake Email** | "decoy@honeypot.local" | Contact list monitoring |
| **Fake Document** | "CONFIDENTIAL-DECOY.pdf" | Document access monitoring |

### 12.2 Best Practices

```
1. Realistic but identifiable
   - Look real to attackers
   - Clearly fake to systems

2. Strategic placement
   - Mix with real data
   - Don't cluster

3. Never legitimate access
   - No business reason to access
   - Any access = alert

4. Regular rotation
   - Update tokens periodically
   - Avoid pattern recognition
```

### 12.3 Change History

| Version | Date | Changes |
|---|---|---|
| 0.1 | 2026-05-17 | Initial honey-token redesign |
| 1.0 | 2026-05-17 | Cross-cutting design finalized |

---

**End of Document**

---

## Note on SRS Updates Required

This design document requires amendments to the following SRS documents:

1. **Vault SRS** - Add honey-token creation, injection, identification, data-detection
2. **Sentinel SRS (Input)** - Add reference detection
3. **Sentinel Output SRS** - Add leak detection
4. **Navigator SRS** - Add search-layer detection
5. **Tracker SRS** - Revise to aggregation/attribution/alerting only

These amendments transform honey-tokens from a single-module feature into a properly distributed cross-cutting concern, significantly improving detection effectiveness through defense-in-depth.
