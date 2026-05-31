# Bastion-Tracker Module SRS

**Project:** Bastion-RAG - RAG Security Governance Framework  
**Document Type:** Module SRS (Tier 2)  
**Document ID:** 14-tracker-srs  
**Module:** D - Tracker (Observability & Visualization)  
**Version:** 2.0 (Foundation-aligned)  
**Date:** 2026-05-17  
**Status:** Draft

**Foundation References:**
- 01-architecture-principles
- 02-event-schema-standard
- 03-module-interaction-map

---

## 1. Introduction

### 1.1 Purpose

This document specifies the **Tracker** module, the observability and visualization layer of Bastion-RAG. Unlike data-path modules (Sentinel/Vault/Navigator/Anchor), Tracker is a **cross-cutting observer** — it watches the entire pipeline without touching data flow.

### 1.2 Module Identity

```
Module: D - Tracker
Role: Observability & Visualization
Position: Cross-cutting (observes all)
Direction: N/A (observer, not in data path)

Standalone value:
"Attach Tracker → see what your LLM system is doing"
```

### 1.3 Why Tracker Has No IN/OUT Distinction

```
Data-path modules (Sentinel, Vault, Anchor):
- Data flows THROUGH them
- IN/OUT = different operations
- Bidirectional makes sense

Tracker:
- Data does NOT flow through it
- It OBSERVES events from all modules
- Input events and output events:
  processed identically (collect → store → visualize)
- IN/OUT distinction is meaningless

Tracker is the "CCTV" of the system.
Whether watching input or output stage,
it does the same thing: record and display.
```

### 1.4 The Standalone Test (Foundation Litmus)

```
Question: "If only Tracker is attached,
          does it provide meaningful value?"

Answer: YES
- Basic logging of LLM interactions
- Self-monitoring
- Request visualization (even single module)

→ Tracker passes the standalone test ✅
(Value increases with more modules emitting events)
```

### 1.5 Scope

**In Scope:**
- 🟢 Core: Event collection
- 🟢 Core: Basic logging
- 🟢 Core: Real-time visualization
- 🟢 Core: Self-monitoring
- 🟢 Core: Request tracing
- 🟡 Enhanced: Multi-module metrics (with events)
- 🟡 Enhanced: Pipeline flow visualization (with multiple modules)
- 🔴 Orchestrated: Honey-token aggregation/alerting
- 🔴 Orchestrated: Data lineage reconstruction
- 🔴 Orchestrated: Cross-module correlation
- Web UI, REST, gRPC, CLI

**Out of Scope:**
- Data validation (Sentinel)
- Data transformation (Vault)
- Search (Navigator)
- Honey-token creation/injection (Vault owns - see Honey-Token SRS)
- Honey-token detection (modules detect - Tracker aggregates)

### 1.6 Definitions

| Term | Definition |
|---|---|
| **Observer** | Watches without modifying |
| **Trace** | Complete request record |
| **Span** | Single operation in trace |
| **Lineage** | Data's journey through pipeline |
| **Correlation** | Linking related events |
| **Pipeline variation** | Full/Lite/Minimal/Blocked |

---

## 2. Overall Description

### 2.1 Observer Architecture

```
┌─────────────────────────────────────────────┐
│            Tracker Service                   │
├─────────────────────────────────────────────┤
│                                              │
│  ┌──────────────────────────────┐            │
│  │  Event Collector (NATS sub)  │            │
│  │  Subscribes: bastion-rag.events.>│            │
│  └────────────┬─────────────────┘            │
│               ▼                              │
│  ┌──────────────────────────────┐            │
│  │  Event Processor             │            │
│  │  - Validate, enrich, route   │            │
│  └────────────┬─────────────────┘            │
│               ▼                              │
│      ┌────────┼─────────┐                    │
│      ▼        ▼         ▼                    │
│  ┌───────┐┌───────┐┌──────────┐              │
│  │Storage││Realtime││Orchestr. │              │
│  │       ││  (WS)  ││(optional)│              │
│  └───────┘└───────┘└──────────┘              │
│               ▼                              │
│  ┌──────────────────────────────┐            │
│  │  Web UI / API                │            │
│  └──────────────────────────────┘            │
│                                              │
└─────────────────────────────────────────────┘

Tracker observes; never blocks data flow.
```

### 2.2 Position: Cross-Cutting Observer

```
        Data Pipeline (Tracker does NOT touch):
User → Sentinel → Vault → Navigator → Anchor → LLM → ... → User
         │          │         │          │
         │ events   │ events  │ events   │ events
         ▼          ▼         ▼          ▼
        ┌────────────────────────────────┐
        │          NATS Event Bus         │
        └────────────────┬────────────────┘
                         ▼
                   ┌──────────┐
                   │ Tracker  │ ← observes all
                   └──────────┘

If Tracker dies: data pipeline continues
(Foundation: graceful degradation)
```

### 2.3 Layer Classification

```
🟢 CORE (Standalone):
   - Event collection
   - Basic logging
   - Self-monitoring
   - Single-request tracing
   - Basic visualization

🟡 ENHANCED (Composition - with events from modules):
   - Multi-module metrics
   - Pipeline flow visualization
   - Per-module dashboards

🔴 ORCHESTRATED (Cross-cutting coordination):
   - Honey-token aggregation
   - Data lineage reconstruction
   - Cross-module correlation
   - Incident management
```

### 2.4 Constraints

```
Language: Go (backend) + React/TS (frontend)
Event bus: NATS
Storage: PostgreSQL + Prometheus + Loki + Jaeger
Real-time: WebSocket
Memory: ≤ 8GB (backend)
```

### 2.5 Dependencies

```
Core dependencies:
- NATS (event subscription)
- PostgreSQL (storage)

Optional (richer features):
- Prometheus, Loki, Jaeger (observability stack)
- Module events (more modules = richer view)

Note: Tracker's failure must NOT affect data path
(Foundation independence requirement)
```

---

## 3. Core Functions (🟢 Standalone)

### 3.1 Event Collection (FR-CORE-EC)

**FR-CORE-EC-001: NATS Subscription**
```
Subscribe: bastion-rag.events.>
Handle: 10,000+ events/s
Per Foundation event schema (02)
Dependency: NATS only
```

**FR-CORE-EC-002: Event Validation**
```
Validate against Foundation schema
Reject malformed (dead letter)
```

**FR-CORE-EC-003: Idempotent Processing**
```
Per Foundation: dedupe by event_id
At-least-once delivery handling
```

### 3.2 Storage (FR-CORE-ST)

**FR-CORE-ST-001: Multi-Backend**
```
- PostgreSQL: recent events, metadata
- Prometheus: metrics (optional)
- Loki: logs (optional)
- Jaeger: traces (optional)

Core needs only PostgreSQL.
```

### 3.3 Basic Visualization (FR-CORE-VZ)

**FR-CORE-VZ-001: Request Flow Display**
```
Real-time request visualization
Even single module shows activity
WebSocket live updates
```

**FR-CORE-VZ-002: Self-Monitoring**
```
Tracker monitors itself:
- Event ingestion rate
- Processing latency
- Storage usage
```

### 3.4 Request Tracing (FR-CORE-RT)

**FR-CORE-RT-001: Single Request Trace**
```
Using trace_id (Foundation):
- Collect events with same trace_id
- Show request timeline
- Per-module breakdown
```

### 3.5 Core Summary

```
Standalone (NATS + PostgreSQL):
✅ Event collection
✅ Basic logging
✅ Request tracing
✅ Real-time visualization
✅ Self-monitoring

Works even observing a single module.
```

---

## 4. Enhanced Functions (🟡 Composition)

### 4.1 Multi-Module Metrics (FR-ENH-MM)

**Requires: Events from multiple modules**

**FR-ENH-MM-001: Aggregated Metrics**
```
When multiple modules emit events:
- Cross-module performance metrics
- Pipeline-wide throughput
- Latency breakdown by module

Graceful degradation:
- Few modules: fewer metrics
- Core single-module tracking works
```

### 4.2 Pipeline Flow Visualization (FR-ENH-PF)

**Requires: Multiple modules in pipeline**

**FR-ENH-PF-001: Full Pipeline Animation**
```
Visualize request flowing through modules:
- Animated dots between modules
- Pipeline variations (Full/Lite/Minimal/Blocked)
- Module bypass visualization

Graceful degradation:
- Single module: simple flow
- More modules: richer animation
```

**FR-ENH-PF-002: Pipeline Variation Display**
```
Show which pipeline each request used:
- Full: A→B→C→E→LLM
- Lite: A→C→E→LLM (Vault bypassed)
- Minimal: A→C→LLM
- Blocked: stopped at module

(Bidirectional flow shown:
 input modules + output modules)
```

### 4.3 Enhanced Summary

```
🟡 Multi-module metrics (+ module events)
🟡 Pipeline flow visualization (+ multiple modules)

Without: core single-module tracking works
```

---

## 5. Orchestrated Functions (🔴 Cross-Cutting)

Tracker is the **aggregation point** for cross-cutting features. Unlike other modules that expose hooks, Tracker **consumes** cross-cutting events and coordinates response.

### 5.1 Honey-Token Aggregation

**Role: Aggregator (NOT creator/detector)**

```
Per Honey-Token SRS, Tracker:
- Aggregates honey-token events from all modules
- Correlates multi-layer detections
- Attribution analysis (who/when/how)
- Alerting
- Incident management

Tracker does NOT:
- Create honey-tokens (Vault does)
- Inject honey-tokens (Vault does)
- Detect at data layer (modules do)

Detail: see Honey-Token SRS (Tier 3).
```

**Brief Contract:**
```
Subscribes: bastion-rag.events.*.honey_token_*

On correlated detection:
- Same trace_id across layers
- = High confidence breach
→ Create incident
→ Send alert

Full logic: Honey-Token SRS.
```

### 5.2 Data Lineage Reconstruction

**Role: Lineage Coordinator**

```
Per Data Lineage SRS, Tracker:
- Subscribes to all events
- Groups by trace_id
- Reconstructs request path
- Builds lineage graph

Detail: see Data Lineage SRS (Tier 3).
```

**Brief Contract:**
```
Subscribes: bastion-rag.events.>

For each trace_id:
- Collect all events
- Order by span/timestamp
- Reconstruct: input → ... → output
→ Provide lineage query

Full logic: Data Lineage SRS.
```

### 5.3 Cross-Module Correlation

```
Correlate events across modules:
- Security event patterns
- Anomaly clusters
- Attack detection

Example:
sentinel.injection + navigator.honey_token
+ same user = coordinated attack
```

### 5.4 Incident Management

```
Auto-create incidents from:
- Correlated security events
- Honey-token triggers
- Anomaly thresholds

Track: investigation, resolution
```

### 5.5 Orchestrated Summary

```
🔴 Honey-token aggregation (→ Honey-Token SRS)
🔴 Data lineage (→ Data Lineage SRS)
🔴 Cross-module correlation
🔴 Incident management

Tracker is the CONSUMER/AGGREGATOR
for cross-cutting features.
Other modules emit; Tracker correlates.
```

---

## 6. External Interfaces

### 6.1 Input: Event Subscription (NATS)

```
Subscribe: bastion-rag.events.>
Per Foundation event schema (02)

All modules publish; Tracker consumes.
```

### 6.2 gRPC Interface

```protobuf
service TrackerService {
  // Event ingestion (fallback to NATS)
  rpc SubmitEvent(BastionEvent) returns (Ack);
  
  // Query (Core)
  rpc GetTrace(TraceRequest) returns (TraceResponse);
  rpc QueryEvents(QueryRequest) returns (stream BastionEvent);
  
  // Orchestrated
  rpc GetLineage(LineageRequest) returns (LineageResponse);
  rpc GetIncidents(IncidentRequest) returns (IncidentResponse);
  
  rpc Health(HealthRequest) returns (HealthResponse);
}
```

### 6.3 REST Interface

```
# Events (Core)
GET  /v1/events
GET  /v1/events/{id}
GET  /v1/traces/{trace_id}

# System (Core)
GET  /v1/topology
GET  /v1/health

# Pipeline (Enhanced)
GET  /v1/pipelines/stats

# Orchestrated
GET  /v1/lineage/{trace_id}
GET  /v1/security/incidents
GET  /v1/honey-tokens/triggers
```

### 6.4 Web UI

```
Pages:
/                  Dashboard (Core)
/flow              Live Flow (Enhanced) ⭐
/topology          System Topology
/traces/{id}       Trace Detail (Core)
/security          Security Events (Orchestrated)
/honey-tokens      Honey-token Monitor (Orchestrated)
/lineage           Data Lineage (Orchestrated)
/demo              PoC Demo Mode
```

### 6.5 CLI Interface

```bash
# Core: stream events
$ tracker-cli stream --tail

# Core: get trace
$ tracker-cli trace trace-12345

# Orchestrated: lineage
$ tracker-cli lineage trace-12345

# Standalone
$ tracker-cli server
```

---

## 7. Non-Functional Requirements

### 7.1 Performance

| ID | Item | Target |
|---|---|---|
| NFR-PE-001 | Event ingestion | ≥ 10,000/s |
| NFR-PE-002 | Processing latency (p95) | < 100ms |
| NFR-PE-003 | UI load | < 2s |
| NFR-PE-004 | Real-time update | < 500ms |
| NFR-PE-005 | Memory | ≤ 8GB |

### 7.2 Independence (Foundation - CRITICAL)

```
NFR-IND-001: Tracker failure does NOT affect data path
- Most critical for Tracker
- Data modules continue if Tracker down
- Events buffer at source

NFR-IND-002: Observer only
- Never blocks data flow
- Pure observation

NFR-IND-003: Core works standalone (NATS+PostgreSQL)
```

### 7.3 Reliability

| ID | Item | Target |
|---|---|---|
| NFR-RE-001 | Availability | 99.5% (lower OK - not data path) |
| NFR-RE-002 | Event loss | < 0.01% |
| NFR-RE-003 | General event retention | 30d hot, 1yr cold |
| NFR-RE-004 | Lineage data retention | 5yr (compliance — see Data Lineage SRS §5.3) |

---

## 8. System Architecture

```
┌────────────────────────────────────────────┐
│           Tracker Service                   │
├────────────────────────────────────────────┤
│  Frontend (React/TS)                        │
│  - Dashboard, Live Flow, Topology           │
│         ↕ WebSocket/REST                    │
│  Backend (Go)                               │
│  ┌─────────────────────────────────┐        │
│  │  Event Collector (NATS sub)     │        │
│  │  Event Processor                │        │
│  │  WebSocket Hub                  │        │
│  │  Orchestration (optional)       │        │
│  │   - Lineage, Honey-token, Incid │        │
│  └─────────────────────────────────┘        │
│         ↓                                   │
│  Storage:                                   │
│  PostgreSQL + Prometheus + Loki + Jaeger    │
└────────────────────────────────────────────┘
         ↑ events
   ┌──────────┐
   │   NATS   │ ← all modules publish
   └──────────┘
```

---

## 9. Standalone Operation

### 9.1 Standalone Mode

```bash
$ tracker-cli server

🚀 Bastion-Tracker v2.0 starting...
✅ NATS connected
✅ PostgreSQL connected
⚠️  Prometheus: not connected (metrics limited)
⚠️  Jaeger: not connected (distributed trace limited)
⚠️  No module events yet (waiting)
✅ Core: FULLY OPERATIONAL
✅ Web UI on :3000
✨ Ready (standalone)
```

### 9.2 Standalone Test (Litmus)

```bash
# Even with one module, Tracker shows activity
$ tracker-cli stream --tail

14:23:45 [sentinel] input_validated (1ms)
14:23:46 [sentinel] input_validated (1ms)

→ Observability even for single module ✅
```

### 9.3 Independence Test (Critical)

```
Test: Kill Tracker, verify data path continues

1. Full pipeline running
2. Kill Tracker
3. Verify:
   ✅ Sentinel still validates
   ✅ Vault still anonymizes
   ✅ Navigator still searches
   ✅ Events buffer at NATS
4. Restart Tracker
5. Verify: buffered events processed

Result: Data path unaffected by Tracker ✅
(Foundation requirement met)
```

---

## 10. Configuration

```yaml
# /etc/bastion-tracker/config.yaml
version: 2.0

# Core
core:
  nats:
    url: nats://nats:4222
    subjects: ["bastion-rag.events.>"]
  storage:
    postgresql: postgresql://...
    retention_days: 30
  realtime:
    websocket_port: 8084

# Enhanced
enhanced:
  prometheus: http://prometheus:9090
  loki: http://loki:3100
  jaeger: http://jaeger:14268
  pipeline_visualization: true

# Orchestrated
orchestrated:
  honey_token_aggregation: true
  lineage_reconstruction: true
  incident_management: true
  alerting:
    slack_webhook: ${SLACK_WEBHOOK}

# Pipelines (for visualization)
pipelines:
  full: [sentinel, vault, navigator, anchor, llm]
  lite: [sentinel, navigator, anchor, llm]
  minimal: [sentinel, navigator, llm]
```

---

## 11. PoC Demo Mode

### 11.1 Demo Scenarios

```
Pre-defined scenarios for PoC:
1. Normal request flow
2. Prompt injection defense
3. PII anonymization
4. Cross-tenant prevention
5. Honey-token detection (multi-layer)
6. Module bypass (lite pipeline)
7. Bidirectional flow (input + output)

Features:
- Slow-motion playback
- Annotations
- Recording
```

### 11.2 Bidirectional Visualization

```
Live Flow showing both directions:

[User] → [Sentinel-IN] → [Vault] → [Navigator] → [Anchor] → [LLM]
                                                              ↓
[User] ← [Sentinel-OUT] ← [Vault-OUT] ← ─ ─ ─ ─ ← [Anchor-OUT]─┘

Same modules shown at both positions
(Tracker observes both input and output events)
```

---

## 12. Summary

```
🟢 Core (Standalone):
   - Event collection
   - Basic logging, tracing
   - Real-time visualization
   - Self-monitoring

🟡 Enhanced (Composition):
   - Multi-module metrics
   - Pipeline flow visualization

🔴 Orchestrated (Cross-cutting):
   - Honey-token aggregation (→ Honey-Token SRS)
   - Data lineage (→ Data Lineage SRS)
   - Cross-module correlation
   - Incident management

Special nature:
- Cross-cutting OBSERVER
- No IN/OUT distinction (observes both identically)
- Failure must NOT affect data path
- Aggregator for cross-cutting features
```

---

## 13. Change History

| Version | Date | Changes |
|---|---|---|
| 1.0 | 2026-05-17 | Initial |
| 2.0 | 2026-05-17 | Foundation-aligned, observer nature clarified |

---

**End of Document**

## Appendix A: Why Tracker is Different

```
Other modules (data-path):
- Data flows through
- IN/OUT distinct
- Expose hooks for cross-cutting

Tracker (observer):
- Data does NOT flow through
- No IN/OUT (observes both same way)
- CONSUMES cross-cutting events
- Aggregates and correlates

Tracker is the only module that:
- Subscribes to ALL events
- Acts as cross-cutting coordinator
- Can fail without breaking data path
```

## Appendix B: Cross-cutting References

```
Honey-Token (Tier 3):
- Tracker AGGREGATES (not creates/detects)
- Correlates multi-layer detections
- Alerting, incident management

Data Lineage (Tier 3):
- Tracker is the Lineage Coordinator
- Reconstructs from trace_id

These are Tracker's PRIMARY orchestrated roles.
```
