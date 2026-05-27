# Bastion Data Lineage Cross-Cutting SRS

**Project:** Bastion - RAG Security Governance Framework  
**Document Type:** Cross-Cutting SRS (Tier 3)  
**Document ID:** 22-data-lineage-srs  
**Feature:** Data Lineage Tracking  
**Version:** 1.0  
**Date:** 2026-05-17  
**Status:** Draft

**Foundation References:**
- 01-architecture-principles
- 02-event-schema-standard (trace_id propagation)
- 03-module-interaction-map

**Participating Modules:**
- ALL modules (emit lineage events)
- Tracker (reconstruct & visualize) ⭐ Coordinator

---

## 1. Feature Overview

### 1.1 Purpose

Data Lineage tracks the **complete journey** of a request/data through the Bastion pipeline. It answers:
- Where did this data come from?
- How was it transformed at each stage?
- Where did it go?
- What decisions were made along the way?

### 1.2 Why Lineage Matters

```
Use cases:
1. Compliance: prove data handling (GDPR, PIPA)
2. Debugging: trace a problematic request
3. Audit: who accessed what, when
4. Security: reconstruct attack path
5. Transparency: explain LLM responses
```

### 1.3 Why This is Cross-Cutting

```
Lineage requires EVERY module to participate:

- Each module knows only its own operation
- No single module sees the full journey
- Tracker aggregates via trace_id

Single module = partial view
All modules + Tracker = complete lineage
```

### 1.4 Scope

**In Scope:**
- Lineage event emission (all modules)
- Trace reconstruction (Tracker)
- Lineage graph building
- Lineage query API
- Visualization
- Retention for compliance

**Out of Scope:**
- Module core functions (their SRS)
- Detailed metrics (Tracker core)
- Real-time blocking (lineage is observational)

---

## 2. Module Responsibilities

### 2.1 Responsibility Distribution

| Module | Responsibility |
|---|---|
| **All modules** | Emit lineage event on operation |
| **Tracker** | Reconstruct, store, visualize ⭐ |

### 2.2 All Modules' Role

```
Each module emits lineage events:
- On operation start
- On operation complete
- With transformation details

Per Foundation event schema:
- trace_id (links events)
- span_id (this operation)
- parent_span_id (causality)
```

### 2.3 Tracker's Role (Coordinator)

```
Lineage Coordinator:
- Subscribe to all events
- Group by trace_id
- Order by span hierarchy
- Reconstruct full path
- Build lineage graph
- Provide queries
```

---

## 3. Hook Usage (Detailed)

### 3.1 Universal Lineage Hook

```
Every module exposes:
{module}.{operation}.completed

This hook emits a lineage event.

Implementation (shared pattern):
func lineageHook(operation, traceContext, details) {
    event := LineageEvent{
        trace_id:    traceContext.TraceID,
        span_id:     traceContext.SpanID,
        parent_span: traceContext.ParentSpanID,
        module:      thisModule,
        operation:   operation,
        input_ref:   details.InputRef,
        output_ref:  details.OutputRef,
        transformation: details.What,
        timestamp:   now(),
    }
    publish("bastion.events." + module + ".lineage", event)
}
```

### 3.2 Per-Module Lineage Events

```
Sentinel:
- sentinel.input.validated → lineage
- sentinel.output.validated → lineage
  Details: validation decision

Vault:
- vault.anonymize.completed → lineage
- vault.transform.completed → lineage
  Details: what fields transformed, how

Navigator:
- navigator.search.completed → lineage
  Details: query → documents found

Anchor:
- anchor.embedding.secured → lineage
- anchor.response.analyzed → lineage
  Details: noise added, analysis results
```

### 3.3 Lineage Event Schema

```protobuf
message LineageEvent {
  // Foundation base
  string trace_id = 1;
  string span_id = 2;
  string parent_span_id = 3;
  string module = 4;
  google.protobuf.Timestamp timestamp = 5;
  
  // Lineage-specific
  string operation = 6;
  string input_ref = 7;       // What came in (reference)
  string output_ref = 8;      // What went out (reference)
  string transformation = 9;  // What happened
  map<string, string> metadata = 10;
}
```

---

## 4. Data Flow

### 4.1 Lineage Reconstruction Flow

```
Request flows through pipeline:

[Sentinel] span=001, parent=null
   → emit lineage: "validated input"
        ↓
[Vault] span=002, parent=001
   → emit lineage: "anonymized 3 fields"
        ↓
[Navigator] span=003, parent=002
   → emit lineage: "found 10 docs"
        ↓
[Anchor] span=004, parent=003
   → emit lineage: "noise injected"
        ↓
      LLM
        ↓
[Anchor] span=005, parent=004
   → emit lineage: "analyzed response"
        ↓
[Vault] span=006, parent=005
   → emit lineage: "transformed by permission"
        ↓
[Sentinel] span=007, parent=006
   → emit lineage: "validated output"

All events → Tracker
Tracker groups by trace_id
Reconstructs: 001→002→003→004→005→006→007
```

### 4.2 Lineage Graph

```
Tracker builds graph from spans:

trace-12345:
  span-001 (Sentinel: input validated)
    └─ span-002 (Vault: anonymized)
        └─ span-003 (Navigator: searched)
            └─ span-004 (Anchor: secured)
                └─ span-005 (Anchor: analyzed)
                    └─ span-006 (Vault: transformed)
                        └─ span-007 (Sentinel: output validated)

Parent-child via parent_span_id.
```

### 4.3 Data Transformation Trail

```
For sensitive data, track transformations:

Original: "Hong Gildong"
  ↓ [Vault span-002] anonymized
Token: "KR_NAME_8f3d2a"
  ↓ [Navigator span-003] used in search
Found in: doc-001
  ↓ [Vault span-006] permission transform
Output: "KR_NAME_8f3d2a" (analyst, kept anon)

Full transformation history preserved.
```

---

## 5. Coordinator Design

### 5.1 Lineage Coordinator (Tracker)

```
┌────────────────────────────────────┐
│  Lineage Coordinator (in Tracker)  │
│  - Subscribe: bastion.events.>     │
│  - Group by trace_id               │
│  - Build span tree                 │
│  - Store lineage graph             │
│  - Provide query API               │
└────────────────────────────────────┘
```

### 5.2 Reconstruction Algorithm

```
func reconstructLineage(trace_id) Graph {
    // Collect all events for trace
    events := getEventsByTrace(trace_id)
    
    // Build span map
    spans := map[span_id]Span{}
    for event in events {
        spans[event.span_id] = Span{
            module: event.module,
            operation: event.operation,
            parent: event.parent_span_id,
            details: event.transformation,
        }
    }
    
    // Build tree via parent links
    graph := buildTree(spans)
    
    return graph
}
```

### 5.3 Storage

```
Lineage storage:
- Jaeger: distributed traces (span tree)
- PostgreSQL: lineage metadata
- Retention: 5 years (compliance)

Query by:
- trace_id
- user_id
- time range
- data element
```

---

## 6. Lineage Queries

### 6.1 Query Types

```
1. Request trace:
   "Show full path of request X"
   → span tree for trace_id

2. Data provenance:
   "Where did this output come from?"
   → backward trace from output

3. User activity:
   "What did user Y access?"
   → all traces for user

4. Compliance audit:
   "Prove data X was handled correctly"
   → full lineage with transformations
```

### 6.2 Query API

```
GET /v1/lineage/{trace_id}
  → Full lineage graph

GET /v1/lineage/data/{data_ref}
  → Provenance of data element

GET /v1/lineage/user/{user_id}
  → User's data access history

GET /v1/lineage/audit?from=&to=
  → Compliance audit trail
```

### 6.3 Query Example

```bash
$ tracker-cli lineage trace-12345

Lineage for trace-12345:
═══════════════════════════════════════
Request: "customer purchase summary"
User: alice@tenant-acme

Timeline:
14:23:45.001 [Sentinel] Input validated
             → injection check passed
14:23:45.005 [Vault] Anonymized
             → name, email tokenized
14:23:45.083 [Navigator] Searched
             → 10 docs found (tenant-acme)
14:23:45.087 [Anchor] Embeddings secured
             → noise σ=0.01
14:23:46.300 [Anchor] Response analyzed
             → bias 0.08 (low)
14:23:46.310 [Vault] Transformed
             → K-anon applied (analyst)
14:23:46.315 [Sentinel] Output validated
             → no PII leak

Total: 1,314ms, 7 stages
═══════════════════════════════════════
```

---

## 7. Visualization

### 7.1 Lineage Graph View

```
Tracker UI - Lineage:

[Input] ──→ [Sentinel] ──→ [Vault] ──→ [Navigator]
                                            │
[User] ←── [Sentinel] ←── [Vault] ←── [Anchor]
                                            │
                                          [LLM]

Click any node → transformation details
Hover edge → data passed
```

### 7.2 Data Provenance View

```
For a specific output, show origin:

Output: "4M-6M won range"
   ↑ generalized by Vault (K-anon)
   ↑ from "5,000,000" 
   ↑ found in doc-045
   ↑ matched query "purchase amount"

Backward trace visualization.
```

---

## 8. Compliance Use

### 8.1 GDPR/PIPA Support

```
Lineage proves:
- Data minimization (transformations shown)
- Purpose limitation (access reasons)
- Right to access (user's data history)
- Accountability (full audit trail)
```

### 8.2 Audit Report

```
Generate compliance report:
- All accesses to specific data
- All transformations applied
- All users who accessed
- Full timeline

Export: PDF, JSON, CSV
```

---

## 9. Performance Considerations

### 9.1 Overhead

```
Lineage adds:
- Event emission per operation (async)
- Storage of events
- Reconstruction on query (not real-time)

Minimal impact:
- Events async (non-blocking)
- Reconstruction on-demand
- Per Foundation: never blocks data path
```

### 9.2 Optimization

```
- Sampling for high volume (configurable)
- Async event emission
- Lazy reconstruction (on query)
- Indexed by trace_id
```

---

## 10. Summary

### 10.1 Responsibility Matrix

| Capability | All Modules | Tracker |
|---|:---:|:---:|
| Emit lineage events | ✅ | |
| Propagate trace_id | ✅ | |
| Reconstruct path | | ✅ |
| Build graph | | ✅ |
| Store lineage | | ✅ |
| Query/visualize | | ✅ |

### 10.2 Key Points

```
1. Lineage is CROSS-CUTTING (all modules emit)
2. Tracker is Coordinator (reconstructs)
3. trace_id is the linchpin (Foundation)
4. Async, non-blocking (observational)
5. Compliance-grade (5yr retention)
6. Provenance + audit support
```

### 10.3 The Foundation Connection

```
Lineage depends entirely on Foundation's
trace context propagation (doc 02):

- trace_id: same across request
- span_id: per operation
- parent_span_id: causality

Without consistent trace propagation,
lineage is impossible.
This is why Foundation event schema is critical.
```

---

## 11. Change History

| Version | Date | Changes |
|---|---|---|
| 1.0 | 2026-05-17 | Initial data lineage cross-cutting SRS |

---

**End of Document**

## Appendix: Module SRS Cross-References

```
Referenced briefly in:

Sentinel SRS (10) §5.3: lineage hooks
Vault SRS (11) §5.3: lineage hooks
Navigator SRS (12) §5.3: lineage hook
Anchor SRS (13) §5.2: lineage hooks
Tracker SRS (14) §5.2: lineage coordinator

Detailed logic HERE.
Modules just emit; Tracker reconstructs.
```
