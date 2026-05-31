# Bastion-Tracker Module SRS

**Project:** Bastion-RAG - RAG Security Governance Framework
**Document Type:** Module SRS (Tier 2)
**Document ID:** 14-tracker-srs
**Module:** D - Tracker (Observability & Visualization)
**Version:** 3.1
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

This document specifies the **Tracker** module, the observability and visualization layer of Bastion-RAG. Unlike data-path modules (Sentinel/Vault/Navigator/Anchor), Tracker is a **cross-cutting observer** — it watches the entire pipeline without touching data flow.

### 1.2 What Changed in v3

```
Tracker backend is UNCHANGED.
Language: Go 1.22+ (was 1.21+) — backend
Frontend: React/TypeScript (unchanged)

Polyglot context:
- Tracker (D): Go — unchanged
- Navigator (C): now Python (was Go)
- Anchor (E): now Python (was Go)

Tracker observes events from ALL modules identically.
Navigator and Anchor publish the same NATS event schema
(JSON, same subjects) whether Go or Python.
Tracker receives them unchanged.

Pipeline visualization: module language labels updated.
```

### 1.3 Module Identity

```
Module: D - Tracker
Language: Go 1.22+ (backend) + React/TypeScript (frontend)
Role: Observability & Visualization
Position: Cross-cutting (observes all)
Direction: N/A (observer, not in data path)

Standalone value:
"Attach Tracker → see what your LLM system is doing"
```

### 1.4 Why Tracker Has No IN/OUT Distinction

```
Data-path modules (Sentinel, Vault, Anchor):
- Data flows THROUGH them
- IN/OUT = different operations

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

### 1.5 The Standalone Test (Foundation Litmus)

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

### 1.6 Scope

**In Scope:**
- 🟢 Core: Event collection
- 🟢 Core: Basic logging
- 🟢 Core: Real-time visualization
- 🟢 Core: Self-monitoring
- 🟢 Core: Request tracing
- 🟡 Enhanced: Multi-module metrics (with events from all modules)
- 🟡 Enhanced: Pipeline flow visualization (with multiple modules)
- 🔴 Orchestrated: Honey-token aggregation/alerting
- 🔴 Orchestrated: Data lineage reconstruction
- 🔴 Orchestrated: Cross-module correlation
- Web UI, REST, gRPC, CLI

**Out of Scope:**
- Data validation (Sentinel)
- Data transformation (Vault)
- Search (Navigator)
- Honey-token creation/injection (Vault owns)
- Honey-token detection at data layer (modules detect; Tracker aggregates)

### 1.7 Definitions

| Term | Definition |
|---|---|
| **Observer** | Watches without modifying |
| **Trace** | Complete request record (by trace_id) |
| **Span** | Single operation within a trace |
| **Lineage** | Data's journey through pipeline |
| **Pipeline variation** | Full/Lite/Minimal/Blocked |

---

## 2. Overall Description

### 2.1 Observer Architecture

```
┌─────────────────────────────────────────────┐
│            Tracker Service (Go)              │
├─────────────────────────────────────────────┤
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
│  Web UI (React/TS)                           │
└─────────────────────────────────────────────┘

Tracker observes; never blocks data flow.
```

### 2.2 Position: Cross-Cutting Observer

```
Data Pipeline (Tracker does NOT touch):
User → Sentinel(Go) → Vault(Go) → Navigator(Py) → Anchor(Py) → LLM → ... → User
         │                │              │               │
         │ events          │ events        │ events         │ events
         ▼                ▼              ▼               ▼
        ┌─────────────────────────────────────────────────┐
        │               NATS Event Bus                     │
        │   bastion-rag.events.{module}.{event_type}           │
        │   JSON payload — same format regardless of lang  │
        └──────────────────────┬──────────────────────────┘
                               ▼
                         ┌──────────┐
                         │ Tracker  │ ← observes all
                         └──────────┘

If Tracker dies: data pipeline continues (Foundation: graceful degradation)
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
   - Per-module dashboards (Go + Python modules)

🔴 ORCHESTRATED (Cross-cutting coordination):
   - Honey-token aggregation
   - Data lineage reconstruction
   - Cross-module correlation
   - Incident management
```

### 2.4 Constraints

```
Language: Go 1.22+ (backend) + React/TypeScript (frontend)
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

Events received from Go modules (Sentinel, Vault)
and Python modules (Navigator, Anchor) — identical format.
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
- Per-module breakdown (Go and Python modules shown uniformly)
```

### 3.5 Core Summary

```
Standalone (NATS + PostgreSQL):
✅ Event collection (Go and Python module events)
✅ Basic logging
✅ Request tracing
✅ Real-time visualization
✅ Self-monitoring
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

v3: Includes Python module events from Navigator and Anchor.
Dashboard labels show language (Go/Python) next to module name.
```

### 4.2 Pipeline Flow Visualization (FR-ENH-PF)

**FR-ENH-PF-001: Full Pipeline Animation**
```
Animated request flow through modules:

Input:  [Sentinel-Go] → [Vault-Go] → [Navigator-Py] → [Anchor-Py] → LLM
Output: LLM → [Anchor-Py] → [Vault-Go] → [Sentinel-Go] → User

Module language labels shown in UI.
Animation data-source: events from NATS (language-agnostic).
```

**FR-ENH-PF-002: Pipeline Variation Display**
```
Pipeline configurations:
- Full:      A(Go)→B(Go)→C(Py)→E(Py)→LLM
- Lite:      A(Go)→C(Py)→E(Py)→LLM
- Minimal:   A(Go)→C(Py)→LLM
- Basic:     A(Go)→B(Go)→LLM
- Blocked:   stopped at module
```

---

## 5. Orchestrated Functions (🔴 Cross-Cutting)

### 5.1 Honey-Token Aggregation

**Role: Aggregator (NOT creator/detector)**
```
Per Honey-Token SRS, Tracker:
- Aggregates honey-token events from all modules
- Correlates multi-layer detections
- Attribution analysis (who/when/how)
- Alerting and incident management

Tracker does NOT:
- Create honey-tokens (Vault does)
- Inject honey-tokens (Vault does)
- Detect at data layer (modules detect)

Detail: see Honey-Token SRS (Tier 3).
```

**Brief Contract:**
```
Subscribes: bastion-rag.events.*.honey_token_*

On correlated detection (same trace_id across layers):
→ High confidence breach
→ Create incident
→ Send alert

Full logic: Honey-Token SRS.
```

### 5.2 Data Lineage Reconstruction

**Role: Lineage Coordinator**
```
Subscribes to all events.
Groups by trace_id.
Reconstructs request path through pipeline.
Builds lineage graph.

v3: Lineage spans Go and Python modules.
trace_id / span_id propagated by all modules identically.

Detail: see Data Lineage SRS (Tier 3).
```

### 5.3 Cross-Module Correlation

```
Correlate events across modules:
- Security event patterns
- Anomaly clusters
- Attack detection

Example:
sentinel.injection_detected + navigator.honey_token_retrieved
+ anchor.bias_detected + same trace_id = sophisticated attack
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

Tracker is the CONSUMER/AGGREGATOR.
Other modules emit; Tracker correlates.
```

---

## 6. External Interfaces

### 6.1 Input: Event Subscription (NATS)

```
Subscribe: bastion-rag.events.>
Per Foundation event schema (02)
All modules publish (Go and Python); Tracker consumes.
```

### 6.2 gRPC Interface

```
Wire format: JSON-over-gRPC (Go encoding.RegisterCodec JSONCodec)
Service: bastion-rag.tracker.v1.TrackerService

Methods:
  SubmitEvent(BastionEvent) → Ack
  GetTrace(TraceRequest) → TraceResponse
  QueryEvents(QueryRequest) → stream BastionEvent
  GetLineage(LineageRequest) → LineageResponse
  GetIncidents(IncidentRequest) → IncidentResponse
  Health(HealthRequest) → HealthResponse
```

### 6.3 REST Interface

```
# Auth
POST /v1/auth/login          — issue JWT (admin/operator/viewer)
POST /v1/auth/refresh         — extend session without re-login (v3.1)
GET  /v1/auth/login-audit     — login attempt trail, admin only (v3.1)

# Events (Core)
GET  /v1/events               — recent events with field filters
GET  /v1/events/{id}          — single event with signature status
GET  /v1/events/search        — full-text keyword search
GET  /v1/events/page          — cursor-based pagination (v3.1)
GET  /v1/events/export        — JSONL streaming download, admin only (v3.1)

# Traces (Core)
GET  /v1/traces/{trace_id}
GET  /v1/traces/{trace_id}/timeline

# System (Core)
GET  /v1/topology
GET  /v1/topology/health
GET  /v1/health
GET  /v1/metrics

# Pipeline (Enhanced)
GET  /v1/pipelines/stats

# Dashboard (v3.1)
GET  /v1/dashboard/summary        — 1h/24h counts, active incidents, top modules
GET  /v1/dashboard/pipeline-health — per-module latency, error rate, events/min
GET  /v1/dashboard/recent-activity — last 20 events ≥ warning severity
GET  /v1/dashboard/tenant/{id}    — per-tenant event/incident/blocked breakdown

# Anomaly detection (v3.1)
GET  /v1/anomaly/baselines    — statistical baselines per module×event_type
GET  /v1/anomaly/events       — detected anomalies from the last 24h

# Orchestrated
GET  /v1/lineage/{trace_id}
GET  /v1/lineage/{trace_id}/sources
GET  /v1/lineage/user/{user_id}
GET  /v1/lineage/data/{data_ref}
GET  /v1/lineage/audit
GET  /v1/security/incidents
GET  /v1/honey-tokens/triggers

# Audit verification
GET  /v1/audit/verify        — HMAC-SHA256 signature scan over all stored events
```

### 6.4 Web UI

```
Pages:
/                       Dashboard summary — event counts, module health,
                          active incidents, honey-token triggers (v3.1)
/flow                   Live Flow (Enhanced) ⭐ — shows Go+Python modules
/topology               System Topology (module language badges)
/traces/{id}            Trace Detail (Core)
/security               Security Events (Orchestrated)
/honey-tokens           Honey-token Monitor (Orchestrated)
/lineage                Data Lineage (Orchestrated)
/logs                   Log Browser — time-range filter, cursor pagination,
                          severity/module/tenant filter, JSONL export (v3.1)
/anomaly                Anomaly Events — detected patterns, baselines (v3.1)
/admin/login-audit      Login Audit Trail — success/fail per user (v3.1)
/demo                   PoC Demo Mode

WebSocket: /ws/events         — live event stream (filterable per-client)
WebSocket: /ws/dashboard      — 30s periodic dashboard summary push (v3.1)
```

### 6.5 CLI Interface

```bash
$ tracker-cli stream --tail
$ tracker-cli trace trace-12345
$ tracker-cli lineage trace-12345
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
- Most critical requirement for Tracker
- Data modules continue if Tracker down
- Events buffer at NATS source

NFR-IND-002: Observer only — never blocks data flow

NFR-IND-003: Core works standalone (NATS+PostgreSQL)
```

### 7.3 Reliability

| ID | Item | Target |
|---|---|---|
| NFR-RE-001 | Availability | 99.5% (lower OK — not data path) |
| NFR-RE-002 | Event loss | < 0.01% |
| NFR-RE-003 | General event retention | 30d hot, 1yr cold |
| NFR-RE-004 | Lineage data retention | 5yr (compliance — see Data Lineage SRS) |

---

## 8. System Architecture

```
┌────────────────────────────────────────────┐
│           Tracker Service (Go 1.22+)        │
├────────────────────────────────────────────┤
│  Frontend: React/TypeScript                 │
│  - Dashboard, Live Flow, Topology           │
│         ↕ WebSocket/REST                    │
│  Backend (Go):                              │
│  - Event Collector (NATS subscription)      │
│  - Event Processor                          │
│  - WebSocket Hub                            │
│  - Orchestration (optional)                 │
│    └─ Lineage, Honey-token, Incidents       │
│         ↓                                   │
│  Storage:                                   │
│  PostgreSQL + Prometheus + Loki + Jaeger    │
└────────────────────────────────────────────┘
         ↑ events (JSON, language-agnostic)
  ┌──────────────────────┐
  │    NATS Event Bus     │
  │  bastion-rag.events.>     │
  └────┬──────────────────┘
       │
  ┌────┴───────────────────────────────┐
  │  Sentinel(Go)  Vault(Go)           │
  │  Navigator(Py) Anchor(Py)          │
  │  (all publish same JSON schema)    │
  └────────────────────────────────────┘

Ports: REST :8084 | gRPC :9094 | WebSocket :8084/ws
```

---

## 9. Standalone Operation

### 9.1 Startup Log

```
[tracker] starting v3.0 (REST :8084, gRPC :9094)
[tracker] NATS connected — subscribing bastion-rag.events.>
[tracker] PostgreSQL connected
[tracker] no module events yet (waiting)
[tracker] core: FULLY OPERATIONAL
[tracker] web UI ready
[tracker] ready
```

### 9.2 Standalone Test (Litmus)

```bash
$ tracker-cli stream --tail

14:23:45 [sentinel/Go]   input_validated  (1ms)
14:23:45 [vault/Go]      anonymized       (3ms)
14:23:45 [navigator/Py]  search_completed (95ms)
14:23:45 [anchor/Py]     embedding_secured (2ms)

→ Observability across Go and Python modules ✅
```

### 9.3 Independence Test (Critical)

```
Test: Kill Tracker, verify data path continues

1. Full pipeline running
2. Kill Tracker process
3. Verify:
   ✅ Sentinel (Go) still validates
   ✅ Vault (Go) still anonymizes
   ✅ Navigator (Python) still searches
   ✅ Anchor (Python) still secures embeddings
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
version: "3.0"

server:
  rest_port: 8084
  grpc_port: 9094

core:
  nats:
    url: nats://nats:4222
    subjects: ["bastion-rag.events.>"]
  storage:
    postgresql: postgresql://postgres:5432/tracker
    retention_days: 30
  realtime:
    websocket_port: 8084

enhanced:
  prometheus: http://prometheus:9090
  loki: http://loki:3100
  jaeger: http://jaeger:14268
  pipeline_visualization: true

orchestrated:
  honey_token_aggregation: true
  lineage_reconstruction: true
  incident_management: true

# Pipeline configurations for visualization
pipelines:
  full:     [sentinel, vault, navigator, anchor, llm]     # A(Go)+B(Go)+C(Py)+E(Py)
  lite:     [sentinel, navigator, anchor, llm]            # A(Go)+C(Py)+E(Py)
  minimal:  [sentinel, navigator, llm]                    # A(Go)+C(Py)
  basic:    [sentinel, vault, llm]                        # A(Go)+B(Go)
```

---

## 11. Summary

```
🟢 Core (Standalone, Go 1.22+):
   - Event collection (from Go and Python modules)
   - Basic logging, tracing
   - Real-time visualization
   - Self-monitoring

🟡 Enhanced (Composition):
   - Multi-module metrics (Go + Python modules)
   - Pipeline flow visualization (polyglot labels)

🔴 Orchestrated (Cross-cutting):
   - Honey-token aggregation (→ Honey-Token SRS)
   - Data lineage (→ Data Lineage SRS)
   - Cross-module correlation
   - Incident management

Special nature:
- Cross-cutting OBSERVER
- No IN/OUT distinction
- Failure must NOT affect data path
- Aggregator for cross-cutting features
- Language-agnostic: receives same JSON event schema from all modules
```

---

## 11b. Management Layer (v3.1)

The management layer adds an operator-facing control plane on top of the existing observer backend.

### Admin Authentication

| Feature | Detail |
|---|---|
| JWT login | `POST /v1/auth/login` — issues HS256 JWT; role hierarchy: admin > operator > viewer |
| Session refresh | `POST /v1/auth/refresh` — extends session from valid token; preserves user ID and role |
| Login audit trail | Every attempt (success + failure) written to ring buffer; queryable at `GET /v1/auth/login-audit` |
| Bcrypt enforcement | `auth.require_bcrypt: true` in config rejects plaintext passwords at startup |
| Configurable TTL | `jwt_expiry` (default 8h) and `refresh_expiry` (default 168h) are independently configurable |

### Dashboard

`GET /v1/dashboard/summary` returns a `DashboardSummary` computed live from the ring buffer:
- `event_count_1h` / `event_count_24h` — event volume over rolling windows
- `active_incidents` — incidents with status ≠ resolved
- `firing_alerts` — alerts in FIRING state
- `honey_token_triggers` — honey-token events in the last 24h
- `blocked_requests_1h` — events with `status=blocked` in the last hour
- `top_modules` — top 5 modules by event volume, sorted descending

`GET /v1/dashboard/pipeline-health` returns per-module: avg latency, error rate, events/min, last seen.

`GET /v1/dashboard/tenant/{tenant_id}` returns per-tenant breakdown (operator+ only).

WebSocket `/ws/dashboard` pushes `DashboardSummary` to all clients every 30 seconds.

### Anomaly Detection

The `anomaly.Detector` runs as a processor hook — every inbound event is inspected. Six pattern rules:

| Pattern | Trigger | Severity |
|---|---|---|
| `statistical_spike` | Event rate for module×type exceeds 3σ above 1h rolling mean | warning |
| `high_freq_user` | Same `user_id` exceeds N requests/minute (default 30) | warning |
| `repeated_block` | Same `user_id` blocked ≥ 3 times within 5 minutes | critical |
| `cross_tenant_signal` | Any `cross_tenant_attempt` event | critical |
| `honey_multi_layer` | Same `trace_id` triggers honey-tokens in ≥ 2 different modules | critical |
| `off_hours_access` | Request outside `active_hours_start`–`active_hours_end` (when configured) | warning |

On detection: `AnomalyEvent` is stored in-memory, broadcast as `WSMessage{Type:"anomaly"}` to WebSocket clients, and synthesised into a `BastionEvent{EventType:"anomaly_detected"}` that passes through the processor (triggering incident auto-creation).

### Enhanced Log Browser

| Feature | Endpoint |
|---|---|
| Cursor pagination | `GET /v1/events/page?cursor={event_id}&limit=50` |
| JSONL export | `GET /v1/events/export?limit=10000` (streams NDJSON, admin only) |
| Time-range filter | `?from=RFC3339&to=RFC3339` on existing `GET /v1/events` |
| Full-text search | `GET /v1/events/search?q={keyword}` |

## 12. Change History

| Version | Date | Changes |
|---|---|---|
| 1.0 | 2026-05-17 | Initial |
| 2.0 | 2026-05-17 | Foundation-aligned, observer nature clarified |
| 3.0 | 2026-05-26 | Go 1.22+; polyglot context (Navigator/Anchor → Python); pipeline visualization updated for language labels; foundation ref v3 |
| 3.1 | 2026-05-31 | Management layer: admin JWT auth + refresh, login audit trail, `DashboardSummary`, pipeline health, tenant activity, `anomaly.Detector` (6 pattern rules + σ baseline), cursor pagination, JSONL export, WebSocket 30s push |

---

**End of Document**

## Appendix A: Why Tracker is Polyglot-Transparent

```
Tracker consumes NATS events.
NATS event schema is JSON (Foundation §7.1).

Go modules publish:  json.Marshal(BastionEvent{...})
Python modules publish: json.dumps(event_dict)

Tracker receives: identical JSON bytes.

The language of the source module is invisible to Tracker.
A "Navigator search_completed" event looks the same
whether Navigator was Go or Python.

This is the power of the polyglot wire contract.
```

## Appendix B: Cross-cutting References

```
Honey-Token (Tier 3): Tracker AGGREGATES
- Correlates multi-layer detections
- Alerting, incident management

Data Lineage (Tier 3): Tracker is Lineage Coordinator
- Reconstructs from trace_id (spans Go + Python modules)

These are Tracker's PRIMARY orchestrated roles.
```
