# Bastion Module Interaction Map

**Project:** Bastion - RAG Security Governance Framework  
**Document Type:** Foundation (Tier 1)  
**Document ID:** 03-module-interaction-map  
**Version:** 1.0  
**Date:** 2026-05-17  
**Status:** Draft

---

## 1. Introduction

### 1.1 Purpose

This document maps **how modules interact** within Bastion. It defines:
- Module interfaces (what each exposes)
- Hooks (cross-cutting extension points)
- Composition relationships (which modules enhance each other)
- Data flow between modules

This is the "wiring diagram" for the entire framework.

### 1.2 Relationship to Other Foundation Docs

```
01-architecture-principles: WHAT and WHY
02-event-schema-standard: HOW modules communicate (events)
03-module-interaction-map: WHO connects to WHOM (this doc)
```

---

## 2. Module Interface Summary

### 2.1 Each Module's Public Interface

```
┌─────────────────────────────────────────────────┐
│ Module     │ Provides         │ Consumes         │
├─────────────────────────────────────────────────┤
│ Sentinel   │ Validation       │ (Navigator ctx)  │
│ Vault      │ Anonymization    │ (none core)      │
│            │ Permission       │                  │
│ Navigator  │ Search           │ (Vault perms)    │
│ Anchor     │ Embedding sec    │ (none core)      │
│ Tracker    │ Observability    │ All events       │
└─────────────────────────────────────────────────┘
```

### 2.2 Core vs Enhanced Interfaces

```
Core interfaces (always available):
- Sentinel.ValidateInput()
- Sentinel.ValidateOutput()
- Vault.Anonymize()
- Vault.Transform()
- Navigator.Search()
- Anchor.SecureEmbedding()
- Anchor.AnalyzeResponse()

Enhanced interfaces (composition):
- Sentinel.ValidateWithContext(navigatorCtx)
- Vault.TransformWithSearch(navigatorResults)
- Navigator.SearchWithPermissions(vaultPerms)
```

---

## 3. Data Flow Map

### 3.1 Input Pipeline Data Flow

```
User Query
    │
    ▼
┌─────────────┐
│  Sentinel   │ Input: raw query
│             │ Output: validated query + pipeline decision
└──────┬──────┘
       │ validated query
       ▼
┌─────────────┐
│   Vault     │ Input: validated query
│  (Phase 1)  │ Output: anonymized query + permissions
└──────┬──────┘
       │ anonymized query + user permissions
       ▼
┌─────────────┐
│  Navigator  │ Input: anonymized query + permissions
│             │ Output: search results (filtered)
└──────┬──────┘
       │ search results
       ▼
┌─────────────┐
│   Anchor    │ Input: embeddings
│  (Phase 1)  │ Output: secured embeddings
└──────┬──────┘
       │ secured context
       ▼
      LLM
```

### 3.2 Output Pipeline Data Flow

```
      LLM
       │ response
       ▼
┌─────────────┐
│   Anchor    │ Input: LLM response
│  (Phase 2)  │ Output: analyzed response + bias flags
└──────┬──────┘
       │ analyzed response
       ▼
┌─────────────┐
│   Vault     │ Input: response + user permissions
│  (Phase 2)  │ Output: permission-transformed response
└──────┬──────┘
       │ transformed response
       ▼
┌─────────────┐
│  Sentinel   │ Input: transformed response
│  (Phase 2)  │ Output: validated final response
└──────┬──────┘
       │ safe response
       ▼
     User
```

### 3.3 Observability Flow (Cross-cutting)

```
All modules ──events──► NATS ──────► Tracker
                                       │
                                       ├─ Lineage reconstruction
                                       ├─ Honey-token aggregation
                                       ├─ Metrics collection
                                       └─ Visualization

Tracker observes but does NOT block data flow.
```

---

## 4. Composition Relationships

### 4.1 Module Pairing Matrix

Which modules enhance each other:

```
            Sentinel  Vault  Navigator  Anchor  Tracker
Sentinel       -       ✓        ✓         -       →
Vault          ✓       -        ✓✓        ✓       →
Navigator      ✓       ✓✓       -         ✓       →
Anchor         -       ✓        ✓         -       →
Tracker        →       →        →         →       -

Legend:
✓✓ = Strong composition (important)
✓  = Optional enhancement
→  = Sends events to
-  = No direct relationship
```

### 4.2 Key Compositions

**Vault + Navigator (Strong)**
```
Purpose: Permission-aware search

Vault provides: user permissions, category access
Navigator uses: to filter search results

Without composition:
- Navigator searches all (less secure)
- Vault permissions unused in search

With composition:
- Navigator pre-filters by permission
- Cross-tenant isolation in search
```

**Sentinel + Navigator (Optional)**
```
Purpose: Indirect injection defense

Navigator provides: search results
Sentinel uses: to check for indirect injection

Without composition:
- Direct injection only (Sentinel-IN)

With composition:
- Indirect injection caught (Sentinel-OUT)
```

**Vault + Anchor (Optional)**
```
Purpose: Layered data protection

Vault provides: anonymized data
Anchor adds: embedding noise

Combined:
- Text-level protection (Vault)
- Embedding-level protection (Anchor)
```

---

## 5. Hook Registry

### 5.1 What are Hooks?

```
Hooks are extension points for cross-cutting features.

- Modules expose hooks
- Cross-cutting coordinators register handlers
- Core function works with or without hooks

Hook execution:
1. Core function runs
2. Registered hooks fire (if any)
3. Core result returned (unaffected by hooks)
```

### 5.2 Sentinel Hooks

```go
type SentinelHooks struct {
    OnInputValidate  []Hook  // After input validation
    OnOutputValidate []Hook  // After output validation
}

// Hook points:
sentinel.input.validated     // Input checked
sentinel.input.honey_check    // Honey-token reference check
sentinel.output.validated     // Output checked
sentinel.output.honey_check   // Honey-token leak check
```

### 5.3 Vault Hooks

```go
type VaultHooks struct {
    OnAnonymize    []Hook  // After anonymization
    OnTransform    []Hook  // After permission transform
    OnDataAccess   []Hook  // On data access
}

// Hook points:
vault.anonymize.completed
vault.transform.completed
vault.data.accessed          // Honey-token data check
vault.honey.inject            // Honey-token injection
```

### 5.4 Navigator Hooks

```go
type NavigatorHooks struct {
    OnSearchComplete []Hook  // After search
    OnResultFilter   []Hook  // After filtering
}

// Hook points:
navigator.search.completed
navigator.results.scanned    // Honey-token in results
```

### 5.5 Anchor Hooks

```go
type AnchorHooks struct {
    OnSecure  []Hook  // After embedding security
    OnAnalyze []Hook  // After response analysis
}

// Hook points:
anchor.embedding.secured
anchor.response.analyzed
```

### 5.6 Hook Summary Table

| Module | Hook Point | Used By Cross-cutting |
|---|---|---|
| Sentinel | input.honey_check | Honey-token |
| Sentinel | output.honey_check | Honey-token |
| Vault | data.accessed | Honey-token |
| Vault | honey.inject | Honey-token |
| Navigator | results.scanned | Honey-token |
| All | *.completed | Lineage |
| All | * | Tracker (events) |

---

## 6. Cross-Cutting Coordinators

### 6.1 Coordinator Pattern

```
A Coordinator manages a cross-cutting feature:

┌──────────────────────────────────┐
│  Cross-Cutting Coordinator       │
│  - Registers hooks in modules    │
│  - Subscribes to events          │
│  - Aggregates/correlates         │
│  - Manages feature lifecycle     │
└──────────────────────────────────┘
         │ registers hooks
         ▼
   Module Hook Points

If Coordinator absent:
- Hooks not registered
- Modules work normally
- Feature simply inactive
```

### 6.2 Honey-Token Coordinator

```
Registers hooks in:
- Sentinel (input/output honey check)
- Vault (data access, injection)
- Navigator (result scanning)

Subscribes to events:
- *.honey_token_*

Responsibilities:
- Inject tokens (via Vault hook)
- Aggregate detections
- Correlate multi-layer triggers
- Generate alerts
```

### 6.3 Multi-Tenancy Coordinator

```
Registers hooks in:
- Vault (tenant key isolation)
- Navigator (pre-filter enforcement)
- Sentinel (tenant validation)

Responsibilities:
- Enforce tenant_id propagation
- Verify isolation at each layer
- Detect cross-tenant attempts
```

### 6.4 Data Lineage Coordinator

```
Registers hooks in:
- All modules (operation tracking)

Subscribes to events:
- bastion.events.> (everything)

Responsibilities:
- Reconstruct request path via trace_id
- Build lineage graph
- Provide trace queries
```

---

## 7. Interface Contracts

### 7.1 Sentinel Interface

```protobuf
service SentinelService {
  // Core
  rpc ValidateInput(InputRequest) returns (InputResponse);
  rpc ValidateOutput(OutputRequest) returns (OutputResponse);
  
  // Enhanced (with context)
  rpc ValidateInputWithContext(ContextualRequest) returns (InputResponse);
  
  // Standard
  rpc Health(HealthRequest) returns (HealthResponse);
}

// Provides to others:
// - Validation results
// - Pipeline routing decisions
```

### 7.2 Vault Interface

```protobuf
service VaultService {
  // Core - Phase 1
  rpc Anonymize(AnonymizeRequest) returns (AnonymizeResponse);
  
  // Core - Phase 2
  rpc Transform(TransformRequest) returns (TransformResponse);
  rpc CheckAccess(AccessRequest) returns (AccessResponse);
  
  // Provides to Navigator (composition)
  rpc GetPermissions(PermRequest) returns (PermResponse);
}

// Provides to others:
// - Anonymized data
// - User permissions (to Navigator)
// - Access decisions
```

### 7.3 Navigator Interface

```protobuf
service NavigatorService {
  // Core
  rpc Search(SearchRequest) returns (SearchResponse);
  
  // Enhanced (with permissions from Vault)
  rpc SearchWithPermissions(PermSearchRequest) returns (SearchResponse);
}

// Provides to others:
// - Search results
// - Context (to Sentinel for indirect injection)
```

### 7.4 Anchor Interface

```protobuf
service AnchorService {
  // Core - Phase 1
  rpc SecureEmbedding(SecureRequest) returns (SecureResponse);
  
  // Core - Phase 2
  rpc AnalyzeResponse(AnalyzeRequest) returns (AnalyzeResponse);
}

// Provides to others:
// - Secured embeddings
// - Response analysis
```

### 7.5 Tracker Interface

```protobuf
service TrackerService {
  // Event ingestion (from all)
  rpc SubmitEvent(BastionEvent) returns (Ack);
  
  // Query (for users)
  rpc GetTrace(TraceRequest) returns (TraceResponse);
  rpc QueryEvents(QueryRequest) returns (stream BastionEvent);
}

// Consumes from others:
// - All events
// Provides to users:
// - Traces, visualizations, alerts
```

---

## 8. Dependency Rules

### 8.1 Allowed Dependencies

```
Core functions: NO dependencies
- Each module's core works alone

Enhanced functions: SOFT dependencies
- Optional, graceful degradation
- Via interface, not direct coupling

Examples:
✅ Navigator.SearchWithPermissions(perms)
   - perms passed in, not Navigator calling Vault
✅ Sentinel checks Navigator context
   - context passed in request
```

### 8.2 Forbidden Dependencies

```
❌ Module directly instantiating another:
   class Navigator {
       vault = new Vault()  // FORBIDDEN
   }

❌ Module calling another's API directly:
   navigator.search() {
       perms = vault.getPermissions()  // FORBIDDEN
   }

✅ Instead, pass via request:
   navigator.searchWithPermissions(perms) {
       // perms provided by caller
   }
```

### 8.3 Communication Rules

```
Synchronous (in request path):
- Via gRPC interface
- Data passed in request
- No hidden calls

Asynchronous (observability):
- Via events (NATS)
- Fire-and-forget
- No blocking
```

---

## 9. Deployment Topology

### 9.1 Standard Topology

```
┌──────────────────────────────────────────────┐
│                Load Balancer                  │
└────────────────────┬─────────────────────────┘
                     │
    ┌────────────────┼────────────────┐
    ▼                ▼                ▼
┌─────────┐    ┌─────────┐     ┌─────────┐
│Sentinel │    │  Vault  │     │Navigator│
│ (pods)  │    │ (pods)  │     │ (pods)  │
└────┬────┘    └────┬────┘     └────┬────┘
     │              │               │
     └──────────────┼───────────────┘
                    │ events
                    ▼
              ┌──────────┐
              │   NATS   │
              └─────┬────┘
                    │
              ┌─────▼────┐
              │ Tracker  │
              └──────────┘
```

### 9.2 Communication Channels

```
Request path (synchronous):
LB → Sentinel → Vault → Navigator → Anchor → LLM
(gRPC between modules)

Event path (asynchronous):
All modules → NATS → Tracker
(event publishing)

Storage:
Vault → PostgreSQL (tokens)
Navigator → Qdrant (vectors)
Tracker → PostgreSQL/Loki/Jaeger
```

---

## 10. Interaction Scenarios

### 10.1 Scenario: Full Pipeline Request

```
1. User query arrives
2. Sentinel validates (core)
   → event: input_validated
3. Sentinel passes to Vault
4. Vault anonymizes (core)
   → event: anonymized
5. Vault provides permissions to Navigator
6. Navigator searches with permissions (enhanced)
   → event: search_completed
7. Anchor secures embeddings (core)
   → event: embedding_secured
8. LLM generates
9. Anchor analyzes response (core)
   → event: response_analyzed
10. Vault transforms by permission (core)
    → event: transform_executed
11. Sentinel validates output (core)
    → event: output_validated
12. Response to user

All events → Tracker (builds lineage)
```

### 10.2 Scenario: Module Failure

```
Navigator fails mid-request:

Without graceful degradation:
❌ Entire request fails

With graceful degradation:
1. Sentinel: works (validated)
2. Vault: works (anonymized)
3. Navigator: FAILS
   → Fallback: return cached/empty results
   → event: navigator.request_failed
4. Pipeline continues with degraded results
5. Tracker logs the failure

System remains operational.
```

### 10.3 Scenario: Honey-Token Trigger

```
Cross-cutting feature in action:

1. Attacker query references honey-token
2. Sentinel core validates (passes)
3. Sentinel honey hook fires
   → event: honey_token_referenced
4. Request continues (not blocked yet)
5. Vault data access
6. Vault honey hook fires (token accessed)
   → event: honey_token_accessed
7. Honey-token Coordinator correlates:
   - Same trace_id
   - Input reference + data access
   - = High confidence breach
8. Coordinator → Tracker
   → incident created
   → alert sent

Each module did its core job + fired hooks.
Coordinator correlated across modules.
```

---

## 11. Module Independence Verification

### 11.1 Standalone Tests

Each module must pass standalone tests:

```
Sentinel standalone:
- Deploy only Sentinel
- Attach to LLM
- Verify: input validation works
- Verify: no errors from missing modules

Vault standalone:
- Deploy only Vault
- Verify: anonymization works
- Verify: graceful without Navigator

(Similar for all modules)
```

### 11.2 Composition Tests

```
Vault + Navigator:
- Deploy both
- Verify: permission-aware search activates
- Verify: each still works if other removed

Sentinel + Navigator:
- Verify: indirect injection defense activates
```

### 11.3 Degradation Tests

```
Full deployment, then kill modules one by one:

Kill Tracker:
- Verify: data path continues
- Verify: events buffer

Kill Anchor:
- Verify: pipeline continues
- Verify: less embedding security

Kill Navigator:
- Verify: graceful fallback
```

---

## 12. Summary

### 12.1 Key Interactions

```
Strong compositions:
- Vault ↔ Navigator (permission-aware search)

Optional enhancements:
- Sentinel ↔ Navigator (indirect injection)
- Vault ↔ Anchor (layered protection)

Cross-cutting (via hooks + events):
- Honey-token (Vault, Sentinel, Navigator)
- Multi-tenancy (Vault, Navigator, Sentinel)
- Lineage (all modules)

Observability:
- All → Tracker (events)
```

### 12.2 Golden Rules

```
1. Core functions: zero dependencies
2. Enhanced functions: data passed in, not fetched
3. Cross-cutting: hooks + events only
4. Communication: gRPC (sync) + NATS (async)
5. Failures: graceful degradation
6. Tracker: observes, never blocks
```

### 12.3 Interaction Principles

```
Loose coupling everywhere:
- No direct module instantiation
- No hidden API calls
- Data flows through interfaces
- Events for observability
- Hooks for cross-cutting

Result:
- Modules independent
- Features composable
- System resilient
```

---

## 13. Change History

| Version | Date | Changes |
|---|---|---|
| 1.0 | 2026-05-17 | Initial interaction map |

---

## Appendix A: Quick Reference

### Module Provides/Consumes

```
Sentinel:
  Provides: validation, routing decision
  Consumes: (optional) Navigator context

Vault:
  Provides: anonymization, permissions, access decisions
  Consumes: (none in core)

Navigator:
  Provides: search results, context
  Consumes: (optional) Vault permissions

Anchor:
  Provides: secured embeddings, response analysis
  Consumes: (none in core)

Tracker:
  Provides: traces, visualizations, alerts
  Consumes: all events
```

### Hook Quick Reference

```
For Honey-Token, register hooks at:
- sentinel.input.honey_check
- sentinel.output.honey_check
- vault.data.accessed
- vault.honey.inject
- navigator.results.scanned

For Lineage, hook at:
- *.completed (all modules)

For Tracker, subscribe to:
- bastion.events.> (all)
```

---

**End of Document**
