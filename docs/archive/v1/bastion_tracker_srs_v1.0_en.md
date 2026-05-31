# Bastion-Tracker System Requirements Specification (SRS) v1.0

**Project:** Bastion-RAG - RAG Security Governance Framework  
**Module:** Module D - Tracker (Observability & Visualization)  
**Document Version:** 1.0  
**Date:** 2026-05-17  
**Status:** Draft  
**Purpose:** PoC Demonstration & Operations Monitoring  
**Primary Users:** Operations Team

---

## 1. Introduction

### 1.1 Purpose

This document defines the functional and non-functional requirements for the **Bastion-Tracker** module. Unlike traditional audit logging modules, Tracker serves as the **central observability platform** and **visual control center** of the entire Bastion-RAG framework, providing:

1. **Real-time Request Flow Visualization** (Primary Focus)
2. **System Topology Display** with health status
3. **Audit Logging** and security event monitoring
4. **Honey-token** intrusion detection
5. **Operations Dashboard** for SRE/Ops teams
6. **PoC Demonstration** capabilities

### 1.2 Scope

**In Scope:**
- Real-time visualization of request flow through Bastion-RAG modules
- Support for multiple pipeline variations (full, lite, bypass)
- Event collection from all Bastion-RAG modules (A, B, C, E)
- Audit log storage and querying
- Honey-token management and triggering detection
- Operations dashboard with key metrics
- Security event alerting and incident management
- Standalone execution
- Web-based UI for operations team
- Demo scenarios for PoC showcase

**Out of Scope:**
- Input validation (Module A - Sentinel)
- Data anonymization (Module B - Vault)
- Vector search (Module C - Navigator)
- Embedding security (Module E - Anchor)
- LLM response generation
- Long-term archival (use external systems for >1 year retention)

### 1.3 Design Philosophy

```
Traditional Tracker:  "Log everything, query later"
Bastion-RAG Tracker:      "Show everything happening NOW + log for later"

Focus Priorities:
1. ⭐ Real-time visualization (primary)
2. ⭐ Operations team usability
3. ⭐ Demo/PoC clarity
4. Comprehensive logging (secondary)
5. Long-term analytics (tertiary)
```

### 1.4 Target Users

| User Type | Primary Needs | Interface |
|---|---|---|
| **Operations Team** | System health, incident response | Web Dashboard ⭐ |
| **Security Team** | Threat monitoring, investigation | Web Dashboard, CLI |
| **DevOps** | Performance metrics, debugging | Web Dashboard, Grafana |
| **Developers** | Request tracing, debugging | Web Dashboard, CLI |
| **PoC Demo Audience** | System understanding | Web Dashboard ⭐ |
| **Bastion-RAG Modules** | Event submission | gRPC, NATS |

### 1.5 Definitions and Acronyms

| Term | Definition |
|---|---|
| **Tracker** | Module D - Observability and visualization |
| **Pipeline** | The sequence of modules a request flows through |
| **Full Pipeline** | A → B → C → E → LLM (all modules) |
| **Lite Pipeline** | A → C → E → LLM (Vault bypass) |
| **Bypass** | Skipping a module in the pipeline |
| **Trace** | Complete record of a request through modules |
| **Span** | Single operation within a trace |
| **Honey-token** | Decoy data to detect intrusion |
| **NATS** | Lightweight message broker |
| **Loki** | Log aggregation system |
| **Jaeger** | Distributed tracing system |
| **SSE** | Server-Sent Events |
| **PoC** | Proof of Concept |

### 1.6 References

- OpenTelemetry Specification
- Prometheus Best Practices
- W3C Trace Context Specification
- NIST Cybersecurity Framework
- PIPA Audit Log Requirements

---

## 2. Overall Description

### 2.1 Product Perspective

Tracker is the **observability hub** that all other Bastion-RAG modules feed into.

```
┌─────────────────────────────────────────────────────────┐
│              User Request                                │
└────────────────────────┬────────────────────────────────┘
                         ▼
        ┌────────────────────────────────────┐
        │   Module A: Sentinel                │ ─┐
        └────────────────┬───────────────────┘  │
                         │                       │
              ┌──────────┴──────────┐           │
              ▼                     ▼           │
   ┌─────────────────┐    ┌────────────────┐   │
   │  Module B:      │    │  (bypass)      │   │
   │  Vault          │    │   ↓            │   │ events
   └────────┬────────┘    │                │   │   ↓
            ▼              │                │   │
        ┌───────────────────────────────┐  │   │
        │   Module C: Navigator         │ ─┤   │
        └────────────────┬──────────────┘  │   │
                         ▼                  │   │
        ┌────────────────────────────────┐ │   │
        │   Module E: Anchor             │─┤   │
        └────────────────┬───────────────┘ │   │
                         ▼                  │   │
        ┌────────────────────────────────┐ │   │
        │           LLM                   │─┘   │
        └────────────────────────────────┘     │
                                                ▼
                    ┌──────────────────────────────────┐
                    │   Module D: TRACKER  ◄── (This)  │
                    │   - Event Collection             │
                    │   - Real-time Visualization      │
                    │   - Operations Dashboard         │
                    │   - Audit Storage                │
                    │   - Honey-token Detection        │
                    └──────────────────────────────────┘
```

### 2.2 Pipeline Variations

Tracker must visualize and track **multiple pipeline configurations**:

#### Pipeline 1: Full (Production)
```
A (Sentinel) → B (Vault) → C (Navigator) → E (Anchor) → LLM
```
- Most secure
- All security layers active
- Used for: customer data, PII-containing data

#### Pipeline 2: Lite (Public Data)
```
A (Sentinel) → C (Navigator) → E (Anchor) → LLM
            ↑
        Vault bypassed
```
- For PII-free data (docs, FAQs)
- Lower latency
- Used for: public documentation

#### Pipeline 3: Minimal (Dev/Test)
```
A (Sentinel) → C (Navigator) → LLM
            ↑              ↑
       Vault skip    Anchor skip
```
- Development mode
- Maximum performance
- Used for: testing, PoC

#### Pipeline 4: Custom (Configurable)
```
Any combination based on:
- Tenant policy
- Data category
- Explicit request
- System availability
```

**Tracker must clearly show which pipeline each request used.**

### 2.3 Product Functions

1. **F1: Event Collection**
   - Receive events from all Bastion-RAG modules via NATS
   - Standardized event schema
   - Handle high throughput (10,000+ events/s)

2. **F2: Real-time Visualization** ⭐ (Primary)
   - Live request flow animation
   - System topology with health status
   - Pipeline variation highlighting
   - Bypass route visualization

3. **F3: Operations Dashboard** ⭐
   - Health overview
   - Active incidents
   - Performance metrics
   - Recent events

4. **F4: Request Tracing**
   - End-to-end request tracking
   - Per-module breakdown
   - Decision audit trail

5. **F5: Audit Logging**
   - Structured event storage
   - Searchable archive
   - Compliance reports

6. **F6: Honey-token Management**
   - Token creation and deployment
   - Trigger detection
   - Incident response

7. **F7: Alerting**
   - Configurable thresholds
   - Multi-channel notifications (Slack, Email)
   - Escalation workflows

8. **F8: PoC Demo Mode** ⭐
   - Replay scenarios
   - Slow-motion request flow
   - Annotated explanations
   - Recording for presentations

9. **F9: Multiple Interfaces**
   - Web UI (primary)
   - REST API (data access)
   - gRPC (event ingestion)
   - CLI (admin)

### 2.4 Constraints

- **Language:** Go (backend) + HTML5/SVG/Vanilla JS (PoC UI) / TypeScript/React (Target UI)
- **Message Queue:** NATS (lightweight, fits SMB)
- **Storage (PoC):** In-memory Ring Buffer (10,000 events)
- **Storage (Target):** PostgreSQL + Prometheus + Loki + Jaeger
- **Memory:** 8GB per pod (backend), 2GB (frontend)
- **Real-time:** WebSocket for live updates
- **Browser Support:** Chrome 100+, Firefox 100+, Safari 16+
- **Mobile:** Responsive (tablet minimum, phone optional)

### 2.5 Assumptions and Dependencies

**Assumptions:**
- All Bastion-RAG modules publish standardized events
- NATS is the central event bus
- Operations team uses modern browsers
- PoC demos run in controlled environments

**External Dependencies:**
- NATS 2.10+ (message broker)
- PostgreSQL 15+ (metadata, audit logs)
- Prometheus 2.40+ (metrics)
- Loki 2.9+ (logs)
- Jaeger 1.50+ (traces)
- Grafana 10+ (optional embedded dashboards)
- Redis 7+ (real-time state cache)

---

## 3. External Interface Requirements

### 3.1 Interface Overview

| Category | Interface | Target Users |
|---|---|---|
| **Input** | NATS (event subscription) | Bastion-RAG modules |
| **Input** | gRPC (event submission) | Modules (fallback) |
| **Input** | REST API | External integrations |
| **Input** | CLI | Admins, operators |
| **Output** | Web UI (Primary) | Operations team |
| **Output** | REST API (JSON) | External tools |
| **Output** | gRPC streaming | Live event consumers |
| **Output** | WebSocket | Real-time dashboard |

### 3.2 Input Interface 1: NATS Event Subscription

**Subjects (Topics):**
```
bastion-rag.events.sentinel.>     # All Sentinel events
bastion-rag.events.vault.>         # All Vault events
bastion-rag.events.navigator.>     # All Navigator events
bastion-rag.events.anchor.>        # All Anchor events
bastion-rag.events.security.>      # Cross-module security events
bastion-rag.events.system.>        # System-level events
```

**Event Schema (Common):**
```protobuf
message BastionEvent {
  string event_id = 1;
  string trace_id = 2;          // For request tracing
  string span_id = 3;
  string parent_span_id = 4;
  
  string module = 5;             // "sentinel", "vault", etc.
  string event_type = 6;         // "request_started", "blocked", etc.
  string severity = 7;           // "info", "warning", "error", "critical"
  
  google.protobuf.Timestamp timestamp = 8;
  
  string tenant_id = 9;
  string user_id = 10;
  string request_id = 11;
  
  map<string, string> labels = 12;
  google.protobuf.Struct data = 13;  // Event-specific payload
  
  // Pipeline context
  string pipeline_type = 14;     // "full", "lite", "minimal", "custom"
  repeated string modules_used = 15;
  repeated string modules_skipped = 16;
  
  // Performance
  int64 duration_ms = 17;
  
  // Outcome
  string status = 18;            // "passed", "blocked", "error"
  string action_taken = 19;
}
```

**Event Types Per Module:**

```yaml
# Sentinel events
sentinel:
  - request_received
  - validation_started
  - prompt_injection_detected
  - metadata_validation_failed
  - validation_passed
  - validation_blocked
  - pipeline_routing_decided   # NEW: which pipeline to use

# Vault events
vault:
  - request_received
  - pii_detected
  - anonymization_applied
  - access_denied
  - access_granted
  - cross_tenant_attempt
  - break_glass_requested
  - k_anonymity_validated

# Navigator events
navigator:
  - search_started
  - search_completed
  - permission_filtered
  - reranking_applied
  - zero_results

# Anchor events
anchor:
  - embedding_secured
  - bias_detected
  - noise_injected

# Security events (cross-module)
security:
  - honey_token_triggered
  - suspicious_pattern
  - anomaly_detected
  - incident_created

# System events
system:
  - module_started
  - module_stopped
  - module_degraded
  - config_changed
  - deployment_completed
```

### 3.3 Input Interface 2: gRPC (Fallback)

```protobuf
service TrackerService {
  rpc SubmitEvent(BastionEvent) returns (SubmitResponse);
  rpc SubmitBatchEvents(BatchEventRequest) returns (BatchResponse);
  rpc StreamEvents(stream BastionEvent) returns (stream Ack);
  
  // Query
  rpc QueryEvents(QueryRequest) returns (stream BastionEvent);
  rpc GetTrace(TraceRequest) returns (TraceResponse);
  
  // Health
  rpc Health(HealthRequest) returns (HealthResponse);
}
```

### 3.4 Output Interface 1: Web UI (Primary)

**URL Structure:**
```
https://tracker.bastion-rag.local/

Pages:
├── /                          # Main Dashboard
├── /topology                  # System Topology
├── /flow                      # Live Request Flow ⭐
├── /traces                    # Request Traces
├── /traces/{trace_id}         # Trace Detail
├── /security                  # Security Events
├── /honey-tokens              # Honey-token Management
├── /metrics                   # Performance Metrics
├── /audit                     # Audit Log Search
├── /demo                      # PoC Demo Mode
├── /settings                  # Configuration
└── /alerts                    # Alert Management
```

**Real-time Updates:**
- WebSocket connection to `/ws/events`
- Server-Sent Events for less critical updates
- Polling fallback if WebSocket fails

### 3.5 Output Interface 2: REST API

```
# Events
GET    /v1/events                       # Recent events
GET    /v1/events/{event_id}            # Event detail
GET    /v1/events/search                # Search events

# Traces
GET    /v1/traces                       # Recent traces
GET    /v1/traces/{trace_id}            # Full trace
GET    /v1/traces/{trace_id}/timeline   # Visualization data

# System
GET    /v1/topology                     # Current system topology
GET    /v1/topology/health              # Health of each module
GET    /v1/pipelines/stats              # Pipeline usage statistics

# Security
GET    /v1/security/events              # Security events
GET    /v1/security/incidents           # Active incidents
POST   /v1/security/incidents/{id}/resolve

# Honey-tokens
GET    /v1/honey-tokens                 # List tokens
POST   /v1/honey-tokens                 # Create token
DELETE /v1/honey-tokens/{id}            # Remove token
GET    /v1/honey-tokens/{id}/triggers   # Trigger history

# Alerts
GET    /v1/alerts                       # Active alerts
POST   /v1/alerts/{id}/acknowledge      # Acknowledge alert

# Demo
GET    /v1/demo/scenarios               # Available demo scenarios
POST   /v1/demo/replay                  # Replay a scenario
POST   /v1/demo/inject                  # Inject demo event

# Admin
GET    /v1/health                       # Tracker health
GET    /v1/metrics                      # Prometheus metrics
POST   /v1/config/reload                # Reload configuration
```

### 3.6 Output Interface 3: CLI

```bash
# View live event stream
$ tracker-cli stream --tail
14:23:45 [Sentinel] request_received user=alice@acme
14:23:45 [Sentinel] validation_passed (0.8ms)
14:23:45 [Vault] anonymization_applied fields=3 (4.2ms)
14:23:45 [Navigator] search_completed results=10 (78ms)
...

# Query traces
$ tracker-cli traces \
    --user alice@acme \
    --since 1h \
    --limit 10

# Show specific trace
$ tracker-cli trace req-12345

Output:
═══════════════════════════════════════════
  Request Trace: req-12345
═══════════════════════════════════════════
Total Time: 1,320ms
Pipeline: full (A→B→C→E→LLM)

Sentinel    [0.8ms]  ✅ validated
Vault       [4.2ms]  ✅ anonymized (3 fields)
Navigator   [78ms]   ✅ found 10 results
Anchor      [3.5ms]  ✅ secured
LLM         [1234ms] ✅ generated

# Security events
$ tracker-cli security --severity critical --since 24h

# Honey-token management
$ tracker-cli honey-token list
$ tracker-cli honey-token create --type email --location customer_db
$ tracker-cli honey-token triggers HT-001

# Demo mode
$ tracker-cli demo list
$ tracker-cli demo replay scenario-prompt-injection --speed 0.5x
$ tracker-cli demo inject --type cross_tenant_attempt
```

---

## 4. Visualization Requirements (Primary Focus)

### 4.1 Main Dashboard (Operations Overview)

**Purpose:** Quick health overview for operations team

```
┌─────────────────────────────────────────────────────────────┐
│  🏛️ Bastion-RAG Control Center          ⚠️ 2 Alerts  👤 ops    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐          │
│  │ Throughput  │ │ Avg Latency │ │  Pipeline   │          │
│  │  89 req/s   │ │   1.2s      │ │  Full: 65%  │          │
│  │   ↑ 5%      │ │   ↓ 0.1s    │ │  Lite: 30%  │          │
│  └─────────────┘ └─────────────┘ └─────────────┘          │
│                                                             │
│  ╔═══════════════════════════════════════════════════════╗ │
│  ║  Live System Status                                    ║ │
│  ║                                                        ║ │
│  ║  [Mini topology view with health indicators]          ║ │
│  ║  Sentinel 🟢 → Vault 🟢 → Navigator 🟢 → Anchor 🟡  ║ │
│  ║                                                        ║ │
│  ║  [View Full Topology →]                                ║ │
│  ╚═══════════════════════════════════════════════════════╝ │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │  ⚠️ Active Alerts (2)                                │  │
│  ├─────────────────────────────────────────────────────┤  │
│  │ 🔴 CRITICAL: Anchor latency > 200ms (5m ago)         │  │
│  │ 🟡 WARNING: Cross-tenant attempts (10m ago)          │  │
│  └─────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │  Recent Security Events                              │  │
│  ├─────────────────────────────────────────────────────┤  │
│  │ 14:23 🚨 Prompt injection blocked (user-xyz)        │  │
│  │ 14:22 ⚠️ Cross-tenant attempt (denied)              │  │
│  │ 14:20 ℹ️ Honey-token HT-001 accessed                │  │
│  │ [View All →]                                         │  │
│  └─────────────────────────────────────────────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 4.2 Live Request Flow ⭐ (Most Important Screen)

**Purpose:** Animated visualization of requests flowing through modules

```
┌─────────────────────────────────────────────────────────────┐
│  🌊 Live Request Flow            [Speed: 1x ▼] [⏸ Pause]   │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   Throughput: 89/s  |  Active: 23 requests in flight       │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │                                                       │  │
│  │   [User]                                             │  │
│  │      ↓ ●●●  (animated request dots)                 │  │
│  │   [Sentinel] 🟢                                       │  │
│  │      │                                                │  │
│  │      ├─→●● (to Vault, full pipeline)                │  │
│  │      └─→○ (bypass to Navigator, lite pipeline)       │  │
│  │      ↓                                                │  │
│  │   [Vault] 🟢                                          │  │
│  │      ↓ ●●                                            │  │
│  │   [Navigator] 🟢 ←──── ○ (lite pipeline arrives)    │  │
│  │      ↓ ●●○                                           │  │
│  │   [Anchor] 🟡 (degraded)                             │  │
│  │      ↓ ●●○                                           │  │
│  │   [LLM]                                              │  │
│  │      ↓ ●●○                                           │  │
│  │   [Response]                                          │  │
│  │                                                       │  │
│  │   Legend:                                             │  │
│  │   ● Full Pipeline   ○ Lite Pipeline   △ Blocked     │  │
│  │                                                       │  │
│  └─────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │  Recent Requests (Click to trace)                    │  │
│  ├─────────────────────────────────────────────────────┤  │
│  │ 14:23:45.123 ● req-12345 alice@acme  1.3s ✅ Full  │  │
│  │ 14:23:45.089 ○ req-12344 bob@acme   780ms ✅ Lite  │  │
│  │ 14:23:45.012 △ req-12343 xyz@acme    0.3s 🚫 Block │  │
│  │ 14:23:44.998 ● req-12342 alice@acme  1.2s ✅ Full  │  │
│  └─────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

**Key Features:**
- ⭐ Animated dots showing requests moving between modules
- ⭐ Different colors/shapes for different pipelines
- ⭐ Pipeline branching visible (Vault bypass clearly shown)
- ⭐ Module health indicated by color
- ⭐ Speed control (0.5x, 1x, 2x for demo)
- ⭐ Pause/play for explanation during demos
- ⭐ Click on request to see full trace

### 4.3 Pipeline Variation Display

When showing system flow, the UI must clearly distinguish pipelines:

```
Pipeline Visualization:

Full Pipeline (Security-critical data):
[User] ●→ [Sentinel] ●→ [Vault] ●→ [Navigator] ●→ [Anchor] ●→ [LLM]
       Solid line, blue dots

Lite Pipeline (Public data, Vault bypassed):
[User] ○→ [Sentinel] ○─────────→ [Navigator] ○→ [Anchor] ○→ [LLM]
       Dashed line, white dots
       (Vault box shown with dotted outline = bypassed)

Minimal Pipeline (Dev/test):
[User] ◇→ [Sentinel] ◇─────────→ [Navigator] ◇─────────→ [LLM]
       Dotted line, diamond shapes

Blocked Request:
[User] △→ [Sentinel] △─🚫
       Red, stops at blocking module
```

**Pipeline Statistics Panel:**
```
┌─────────────────────────────────────┐
│ Pipeline Usage (Last 1 hour)         │
├─────────────────────────────────────┤
│ Full:    65% ████████████░░         │
│ Lite:    30% █████░░░░░░░░░         │
│ Minimal:  3% ░░░░░░░░░░░░░          │
│ Blocked:  2% ░░░░░░░░░░░░░          │
└─────────────────────────────────────┘
```

### 4.4 System Topology (Static View)

**Purpose:** Detailed system architecture with module status

```
┌─────────────────────────────────────────────────────────────┐
│  🗺️ System Topology                  [Layout: Hierarchical ▼]│
├─────────────────────────────────────────────────────────────┤
│                                                             │
│                     ┌──────────┐                            │
│                     │   User   │                            │
│                     └────┬─────┘                            │
│                          │ 89 req/s                         │
│                          ▼                                  │
│                     ┌──────────┐                            │
│                ┌────│ Sentinel │────┐                       │
│                │    │   🟢      │    │                       │
│                │    │  0.8ms   │    │                       │
│                │    └──────────┘    │                       │
│           full │                    │ lite                  │
│                ▼                    ▼                       │
│           ┌──────────┐         (skip Vault)                │
│           │  Vault   │              │                       │
│           │   🟢      │              │                       │
│           │  4.2ms   │              │                       │
│           └────┬─────┘              │                       │
│                │                    │                       │
│                ▼                    ▼                       │
│           ┌────────────────────────────┐                   │
│           │       Navigator             │                   │
│           │         🟢                   │                   │
│           │         87ms                │                   │
│           └────────────┬───────────────┘                   │
│                        ▼                                    │
│                  ┌──────────┐                              │
│                  │  Anchor  │ ⚠️ Degraded                  │
│                  │   🟡      │   145ms (normal: 3.5ms)     │
│                  └────┬─────┘                              │
│                       ▼                                    │
│                  ┌──────────┐                              │
│                  │   LLM    │                              │
│                  │  1234ms  │                              │
│                  └──────────┘                              │
│                                                             │
│  Click any module for details ↑                            │
└─────────────────────────────────────────────────────────────┘
```

**Module Detail Modal (on click):**
```
┌─────────────────────────────────────────┐
│  Anchor Module                       [×] │
├─────────────────────────────────────────┤
│  Status: 🟡 Degraded                     │
│                                         │
│  Current Performance:                   │
│  - Latency p95: 145ms (target: <5ms)   │
│  - Error rate: 0.3%                     │
│  - Throughput: 89 req/s                 │
│                                         │
│  Recent Issues:                         │
│  - 14:18: Latency spike detected        │
│  - 14:15: Memory usage 85%             │
│                                         │
│  Actions:                               │
│  [View Logs] [View Metrics] [Restart]  │
└─────────────────────────────────────────┘
```

### 4.5 Request Trace Detail

**Purpose:** Deep dive into a single request

```
┌─────────────────────────────────────────────────────────────┐
│  🔍 Request Trace: req-12345                         [Back] │
├─────────────────────────────────────────────────────────────┤
│  Time: 2026-05-17 14:23:45.123                              │
│  User: alice@tenant-acme                                    │
│  Pipeline: Full (A→B→C→E→LLM)                              │
│  Total: 1,320ms                                             │
│                                                             │
│  ─── Timeline ──────────────────────────────────────        │
│                                                             │
│  Sentinel  ▓ 0.8ms                                          │
│  Vault       ▓ 4.2ms                                        │
│  Navigator     ▓▓▓▓▓ 78ms                                   │
│  Anchor              ▓ 3.5ms                                │
│  LLM                   ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓ 1234ms          │
│                                                             │
│  ─── Module Details ──────────────────────                  │
│                                                             │
│  ┌─ Sentinel (0.8ms) ─────────────────────────┐ ✓          │
│  │ Validation: PASSED                          │            │
│  │ - Prompt injection score: 0.12 (safe)       │            │
│  │ - Metadata: valid                           │            │
│  │ - Pipeline decision: full                   │            │
│  └─────────────────────────────────────────────┘            │
│                                                             │
│  ┌─ Vault (4.2ms) ────────────────────────────┐ ✓          │
│  │ Anonymization: SUCCESS                      │            │
│  │ - PII detected in query: 0                  │            │
│  │ - Permission: marketing_analyst             │            │
│  │ - Allowed categories: customer_data         │            │
│  │ - K-anonymity required: K=5                 │            │
│  └─────────────────────────────────────────────┘            │
│                                                             │
│  ┌─ Navigator (78ms) ─────────────────────────┐ ✓          │
│  │ Search: SUCCESS                             │            │
│  │ - Strategy: hybrid + rerank                 │            │
│  │ - Initial candidates: 50                    │            │
│  │ - After permission filter: 38               │            │
│  │ - Final results: 10                         │            │
│  │ - Documents used:                           │            │
│  │   📄 doc-001 (score: 0.92)                 │            │
│  │   📄 doc-045 (score: 0.88)                 │            │
│  │   📄 doc-123 (score: 0.85)                 │            │
│  └─────────────────────────────────────────────┘            │
│                                                             │
│  ┌─ Anchor (3.5ms) ───────────────────────────┐ ✓          │
│  │ Security: APPLIED                           │            │
│  │ - Noise injection: enabled                  │            │
│  │ - Bias check: passed                        │            │
│  └─────────────────────────────────────────────┘            │
│                                                             │
│  ┌─ LLM (1234ms) ─────────────────────────────┐ ✓          │
│  │ Generation: SUCCESS                         │            │
│  │ - Model: claude-sonnet-4                    │            │
│  │ - Tokens: 1,234 in, 567 out                │            │
│  └─────────────────────────────────────────────┘            │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 4.6 Security Events Dashboard

```
┌─────────────────────────────────────────────────────────────┐
│  🚨 Security Events                                          │
├─────────────────────────────────────────────────────────────┤
│  [Filter: All ▼] [Severity: All ▼] [Time: 24h ▼] [🔍]     │
│                                                             │
│  🚨 Critical: 12  |  ⚠️ Warning: 45  |  ℹ️ Info: 234       │
│                                                             │
│  ─── Event Timeline ───────────────                         │
│                                                             │
│  [Bar chart showing events over time]                       │
│                                                             │
│  ─── Active Incidents ─────────────                         │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │ 🚨 INC-001: Honey-token triggered (Active)           │  │
│  │ Started: 14:22:10 (1 minute ago)                     │  │
│  │ Token: HT-001 (Fake CEO email)                       │  │
│  │ Source: 192.168.1.123                                │  │
│  │ Status: Investigating                                │  │
│  │ [Resolve] [Escalate] [View Details]                  │  │
│  └─────────────────────────────────────────────────────┘  │
│                                                             │
│  ─── Recent Events ────────────────                         │
│                                                             │
│  14:23:45 🚨 Prompt injection blocked                       │
│  ├ Module: Sentinel                                         │
│  ├ User: xyz@tenant-acme                                    │
│  ├ Pattern: "ignore all previous instructions"             │
│  └ Action: Request rejected, user warned                    │
│                                                             │
│  14:22:10 🚨 Cross-tenant access attempt                    │
│  ├ Module: Vault                                            │
│  ├ User: bob@tenant-globex                                  │
│  ├ Target: tenant-acme.customer_data                       │
│  └ Action: Access denied, alert sent                        │
│                                                             │
│  14:20:33 ℹ️ Honey-token accessed                          │
│  ├ Token: HT-001                                            │
│  └ → Promoted to incident INC-001                          │
└─────────────────────────────────────────────────────────────┘
```

### 4.7 Honey-token Management

```
┌─────────────────────────────────────────────────────────────┐
│  🍯 Honey-Tokens                                             │
├─────────────────────────────────────────────────────────────┤
│  Total: 47 active  |  Triggers (24h): 3                    │
│                                                             │
│  [+ New Honey-Token]                                        │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │ HT-001: Fake CEO Email                               │  │
│  │ Type: 📧 Email                                       │  │
│  │ Location: customer_database                          │  │
│  │ Created: 2026-04-15                                  │  │
│  │ Last Trigger: 1 minute ago 🚨                        │  │
│  │ Total Triggers: 3                                    │  │
│  │ Status: 🚨 ACTIVE INCIDENT                           │  │
│  │ [View Details] [Disable] [View Incident]             │  │
│  └─────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │ HT-002: Fake API Key                                 │  │
│  │ Type: 🔑 Credential                                  │  │
│  │ Location: hr_database                                │  │
│  │ Last Trigger: never                                  │  │
│  │ Status: ✅ Healthy                                   │  │
│  └─────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │ HT-003: Fake Customer Record                         │  │
│  │ Type: 👤 Identity (Honggildong with fake SSN)       │  │
│  │ Location: customer_database                          │  │
│  │ Last Trigger: 2 weeks ago                            │  │
│  │ Status: ⚠️ Monitor                                   │  │
│  └─────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 4.8 PoC Demo Mode ⭐

**Purpose:** Controlled demonstration of Bastion-RAG capabilities

```
┌─────────────────────────────────────────────────────────────┐
│  🎬 PoC Demo Mode                          [Recording: 🔴]   │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ─── Available Scenarios ────────                           │
│                                                             │
│  📋 Scenario 1: Normal Request Flow                         │
│  Shows: Full pipeline operation, normal data                │
│  Duration: 30 seconds                                       │
│  [▶ Play] [Edit]                                            │
│                                                             │
│  📋 Scenario 2: Prompt Injection Defense                    │
│  Shows: Sentinel blocking malicious input                   │
│  Duration: 20 seconds                                       │
│  [▶ Play] [Edit]                                            │
│                                                             │
│  📋 Scenario 3: PII Anonymization                           │
│  Shows: Vault detecting and masking customer data           │
│  Duration: 40 seconds                                       │
│  [▶ Play] [Edit]                                            │
│                                                             │
│  📋 Scenario 4: Cross-tenant Prevention                     │
│  Shows: Vault blocking unauthorized access                  │
│  Duration: 25 seconds                                       │
│  [▶ Play] [Edit]                                            │
│                                                             │
│  📋 Scenario 5: Honey-token Detection                       │
│  Shows: Attacker accessing fake data, alert flow            │
│  Duration: 35 seconds                                       │
│  [▶ Play] [Edit]                                            │
│                                                             │
│  📋 Scenario 6: Module Bypass (Lite Pipeline)              │
│  Shows: Public data using lite pipeline                     │
│  Duration: 25 seconds                                       │
│  [▶ Play] [Edit]                                            │
│                                                             │
│  ─── Demo Controls ──────────────                           │
│                                                             │
│  [▶ Play All] [⏸ Pause] [⏹ Stop]                            │
│  Speed: [0.25x] [0.5x] [1x] [2x]                           │
│                                                             │
│  Narration: ☑ Show explanations                            │
│  Highlights: ☑ Show key events                             │
│  Annotations: ☑ Show captions                              │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Demo Mode Features:**
- Pre-defined scenarios with realistic data
- Slow-motion playback for explanation
- Annotated highlights at key moments
- Pause for Q&A
- Recording capability for offline demos
- Reset button to start fresh

---

## 5. Module Bypass Handling

### 5.1 Bypass Detection

Tracker must accurately detect and visualize when modules are bypassed.

```
Event from Sentinel:
{
  "event_type": "pipeline_routing_decided",
  "module": "sentinel",
  "data": {
    "pipeline_type": "lite",
    "modules_used": ["sentinel", "navigator", "anchor", "llm"],
    "modules_skipped": ["vault"],
    "bypass_reason": "public_data_category"
  }
}

Visualization:
[Sentinel] ──→ [Vault (skipped)] ──→ [Navigator]
                     ↓ dotted outline
                     "bypassed"
```

### 5.2 Pipeline Statistics

Track and display pipeline usage:

```
Last Hour Pipeline Distribution:

Full Pipeline (A→B→C→E→LLM):     65%
├─ Customer data queries:        45%
├─ HR data queries:              15%
└─ Manufacturing queries:         5%

Lite Pipeline (A→C→E→LLM):       30%
├─ Public docs queries:          25%
└─ FAQ queries:                   5%

Minimal Pipeline (A→C→LLM):       3%
└─ Development/testing             3%

Blocked at Sentinel:              2%
└─ Prompt injection attempts      2%
```

### 5.3 Bypass Validation

Tracker monitors bypass decisions for anomalies:

```
Alerts on:
- Unexpected bypass patterns
- High-risk data using lite pipeline
- Tenant violation of bypass policies

Example Alert:
⚠️ Pipeline Anomaly Detected
Tenant: tenant-acme
Issue: Customer data using lite pipeline (Vault skipped)
Action: Verify pipeline configuration
```

### 5.4 Bypass Visualization Modes

**Mode 1: Show All Pipelines (Default)**
- All paths visible
- Active paths animated
- Inactive paths dimmed

**Mode 2: Filter by Pipeline**
- Show only full pipeline traffic
- Show only lite pipeline traffic
- Useful for focused analysis

**Mode 3: Highlight Bypass**
- Emphasize bypassed modules
- Show bypass reasons
- Audit-focused view

---

## 6. Functional Requirements

### 6.1 Event Collection (FR-EC)

**FR-EC-001: NATS Subscription**
- Subscribe to all `bastion-rag.events.>` subjects
- Handle 10,000+ events/s
- Buffer for slow consumers

**FR-EC-002: Event Validation**
- Validate against schema
- Reject malformed events (log to dead letter)
- Track validation metrics

**FR-EC-003: Event Storage**
- Store in PostgreSQL (recent)
- Forward to Loki (long-term logs)
- Forward to Prometheus (metrics)
- Forward to Jaeger (traces)

**FR-EC-004: Event Enrichment**
- Add timestamp metadata
- Calculate derived fields
- Link to traces

### 6.2 Real-time Visualization (FR-RV)

**FR-RV-001: WebSocket Streaming**
- Push events to UI in real-time
- Filter per subscriber preferences
- Handle 1,000+ concurrent connections

**FR-RV-002: Animation Engine**
- Render request flow animations
- Smooth 60fps animations
- Configurable speed (0.1x - 10x)

**FR-RV-003: Pipeline Visualization**
- Distinguish pipeline types visually
- Show bypassed modules
- Highlight active routes

**FR-RV-004: System Topology**
- Real-time module health
- Connection status
- Performance indicators

### 6.3 Request Tracing (FR-RT)

**FR-RT-001: Distributed Tracing**
- Use Jaeger for trace storage
- W3C Trace Context standard
- Cross-module trace correlation

**FR-RT-002: Trace Visualization**
- Timeline view (span-based)
- Module breakdown
- Decision audit trail

**FR-RT-003: Trace Search**
- By trace_id, user, tenant
- Time range filtering
- Status filtering

### 6.4 Security Monitoring (FR-SM)

**FR-SM-001: Real-time Threat Detection**
- Subscribe to security events
- Aggregate related events
- Create incidents from patterns

**FR-SM-002: Incident Management**
- Auto-create incidents from critical events
- Manual incident creation
- Investigation workflow
- Resolution tracking

**FR-SM-003: Honey-token Management**
- CRUD operations for tokens
- Multiple token types
- Trigger detection
- Auto-create incidents on triggers

### 6.5 Alerting (FR-AL)

**FR-AL-001: Alert Rules**
- Configurable thresholds
- Composite conditions
- Per-tenant rules

**FR-AL-002: Notification Channels**
- Slack
- Email
- Webhook
- In-app notifications

**FR-AL-003: Alert Lifecycle**
- Triggered → Acknowledged → Resolved
- Escalation after timeout
- Auto-resolve when condition clears

### 6.6 Demo Mode (FR-DM)

**FR-DM-001: Scenario Recording**
- Capture event sequences
- Save as replayable scenarios
- Include all module events

**FR-DM-002: Scenario Replay**
- Replay at any speed
- Pause/resume capability
- Step-by-step mode

**FR-DM-003: Event Injection**
- Manual event injection
- For scenario creation
- Sandbox isolation (doesn't affect production)

**FR-DM-004: Annotations**
- Add explanatory captions
- Highlight key moments
- Educational walkthroughs

### 6.7 Operations Features (FR-OP)

**FR-OP-001: Health Dashboard**
- Module health overview
- Active alerts summary
- Quick metrics

**FR-OP-002: Quick Actions**
- Acknowledge alerts
- Resolve incidents
- Restart suggestions

**FR-OP-003: Run Books**
- Linked playbooks for common issues
- Step-by-step guidance
- Action automation (where safe)

### 6.8 Module Bypass Tracking (FR-MB)

**FR-MB-001: Bypass Detection**
- Identify pipeline from events
- Track skipped modules
- Validate bypass reasons

**FR-MB-002: Bypass Statistics**
- Per-pipeline usage rates
- Bypass reasons breakdown
- Trend analysis

**FR-MB-003: Bypass Anomaly Detection**
- Unexpected bypass patterns
- High-risk data using lite pipeline
- Alert on violations

---

## 7. Non-Functional Requirements

### 7.1 Performance (NFR-PE)

| ID | Item | Target |
|---|---|---|
| NFR-PE-001 | Event ingestion rate | ≥ 10,000 events/s |
| NFR-PE-002 | Event processing latency (p95) | < 100ms |
| NFR-PE-003 | UI page load time | < 2s |
| NFR-PE-004 | Real-time update latency | < 500ms |
| NFR-PE-005 | Query response (p95) | < 1s |
| NFR-PE-006 | Concurrent UI users | ≥ 50 |
| NFR-PE-007 | Memory usage | ≤ 8GB (backend) |
| NFR-PE-008 | Trace search response | < 3s |

### 7.2 Reliability (NFR-RE)

| ID | Item | Target |
|---|---|---|
| NFR-RE-001 | Availability | 99.5% (lower than data path modules) |
| NFR-RE-002 | Event loss rate | < 0.01% |
| NFR-RE-003 | Data retention | 30 days hot, 1 year cold |
| NFR-RE-004 | MTTR | < 15 minutes |

**Note:** Tracker's failure should not affect data path modules. They can continue operating without it.

### 7.3 Scalability (NFR-SC)

| ID | Item | Target |
|---|---|---|
| NFR-SC-001 | Events stored | 100M+ |
| NFR-SC-002 | Traces per day | 10M+ |
| NFR-SC-003 | Concurrent UI users | 50+ |
| NFR-SC-004 | Honey-tokens managed | 1,000+ |

### 7.4 Security (NFR-SE)

| ID | Item | Requirement |
|---|---|---|
| NFR-SE-001 | Authentication | JWT-based |
| NFR-SE-002 | Authorization | Role-based (admin, ops, viewer) |
| NFR-SE-003 | Encryption in transit | TLS 1.3 |
| NFR-SE-004 | Audit log integrity | HMAC-signed |
| NFR-SE-005 | PII in logs | Anonymized |

### 7.5 Usability (NFR-US)

| ID | Item | Target |
|---|---|---|
| NFR-US-001 | Time to identify incident | < 30s |
| NFR-US-002 | Time to view trace | < 10s |
| NFR-US-003 | Browser compatibility | Modern browsers (Chrome 100+, Firefox 100+, Safari 16+) |
| NFR-US-004 | Mobile responsive | Tablet minimum |
| NFR-US-005 | Documentation in UI | Contextual help available |

---

## 8. System Architecture

### 8.1 High-Level Architecture

```
┌────────────────────────────────────────────────────────────┐
│                    Tracker System                           │
├────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │  Frontend (React + TypeScript)                      │  │
│  │  ├─ Dashboard                                       │  │
│  │  ├─ Live Flow Visualization                         │  │
│  │  ├─ System Topology                                 │  │
│  │  ├─ Security Events                                 │  │
│  │  ├─ Honey-token Management                          │  │
│  │  ├─ Demo Mode                                       │  │
│  │  └─ Embedded Grafana                                │  │
│  └─────────────────────┬───────────────────────────────┘  │
│                        │                                    │
│                  WebSocket/REST                             │
│                        │                                    │
│  ┌─────────────────────▼───────────────────────────────┐  │
│  │  Backend (Go)                                       │  │
│  │  ┌─────────────┐ ┌──────────────┐ ┌─────────────┐ │  │
│  │  │ API Server  │ │ WebSocket    │ │ Event       │ │  │
│  │  │ (REST/gRPC) │ │ Hub          │ │ Processor   │ │  │
│  │  └─────────────┘ └──────────────┘ └──────┬──────┘ │  │
│  │                                          │         │  │
│  │  ┌─────────────┐ ┌──────────────┐       │         │  │
│  │  │ Demo        │ │ Alert        │       │         │  │
│  │  │ Engine      │ │ Manager      │       │         │  │
│  │  └─────────────┘ └──────────────┘       │         │  │
│  └──────────────────────────────────────────┼─────────┘  │
│                                              │             │
└──────────────────────────────────────────────┼─────────────┘
                                               │
                          ┌────────────────────┴───────────┐
                          │                                │
                    ┌─────▼─────┐                ┌─────────▼──────┐
                    │   NATS    │                │  Storage Layer │
                    │           │                │                │
                    │ Event Bus │                │ - PostgreSQL   │
                    └─────┬─────┘                │ - Prometheus   │
                          │                      │ - Loki         │
                          │ events from          │ - Jaeger       │
                          │                      │ - Redis        │
                    ┌─────▼──────────┐           └────────────────┘
                    │ Bastion-RAG Modules │
                    │ A, B, C, E      │
                    └─────────────────┘
```

### 8.2 Component Description

| Component | Responsibility |
|---|---|
| **Frontend** | Web UI for operations team |
| **API Server** | REST/gRPC endpoints |
| **WebSocket Hub** | Real-time event streaming to clients |
| **Event Processor** | Process NATS events, enrich, route |
| **Demo Engine** | Manage scenarios, replay, injection |
| **Alert Manager** | Evaluate rules, send notifications |
| **NATS** | Central event bus |
| **PostgreSQL** | Recent events, metadata, configuration |
| **Prometheus** | Metrics time-series |
| **Loki** | Log aggregation |
| **Jaeger** | Distributed traces |
| **Redis** | Real-time state cache |

### 8.3 Data Flow

```
Event Ingestion:
[Modules] → NATS → Event Processor → Routing:
                                      ├─→ PostgreSQL (recent)
                                      ├─→ Loki (logs)
                                      ├─→ Prometheus (metrics)
                                      ├─→ Jaeger (traces)
                                      └─→ WebSocket Hub → UI

Query Flow:
[UI] → API Server → Storage Query → Response
                                 ├─ PostgreSQL
                                 ├─ Loki (full-text)
                                 └─ Jaeger (trace)
```

### 8.4 Frontend Architecture

```
src/
├── pages/
│   ├── Dashboard.tsx
│   ├── LiveFlow.tsx           ⭐ Primary
│   ├── Topology.tsx
│   ├── TraceDetail.tsx
│   ├── Security.tsx
│   ├── HoneyTokens.tsx
│   ├── Demo.tsx
│   └── Settings.tsx
│
├── components/
│   ├── flow/                  ⭐ Animation components
│   │   ├── PipelineFlow.tsx
│   │   ├── RequestDot.tsx
│   │   └── ModuleNode.tsx
│   ├── topology/
│   │   ├── SystemGraph.tsx
│   │   └── ModuleDetail.tsx
│   ├── timeline/
│   │   └── SpanTimeline.tsx
│   └── shared/
│       ├── EventList.tsx
│       └── HealthIndicator.tsx
│
├── services/
│   ├── api.ts                 # REST client
│   ├── websocket.ts           # WebSocket connection
│   └── grafana.ts             # Grafana embed
│
├── store/
│   ├── events.ts              # Zustand store
│   ├── topology.ts
│   └── alerts.ts
│
└── utils/
    ├── animation.ts           # Animation helpers
    └── trace.ts               # Trace utilities
```

---

## 9. Standalone Testing Environment

### 9.1 Independence

Tracker operates independently of other modules:
- Mock event sources for testing
- Pre-recorded scenarios for demos
- Synthetic data generation

### 9.2 Standalone Modes

**Mode 1: Server Mode**
```bash
$ tracker-cli server

🚀 Bastion-Tracker v1.0 starting...
✅ Config loaded
✅ NATS connected: nats://localhost:4222
✅ PostgreSQL connected
✅ Prometheus scraping enabled
✅ Loki client ready
✅ Jaeger client ready
✅ WebSocket Hub started
✅ REST API on :8080
✅ Frontend served on :3000
⚠️  No event sources detected (waiting for modules)
✨ Ready
```

**Mode 2: Demo Mode (no real modules)**
```bash
$ tracker-cli demo-server --scenario all

🎬 Starting demo mode...
✅ Generating synthetic events
✅ Running scenario: normal-flow
✅ UI available at http://localhost:3000

Demo will cycle through all scenarios automatically.
```

**Mode 3: Event Injection**
```bash
# Inject test events
$ tracker-cli inject \
    --module sentinel \
    --event-type prompt_injection_detected \
    --user test-user

# Inject pre-defined scenario
$ tracker-cli inject-scenario \
    --name prompt-injection-attempt \
    --speed 1x
```

### 9.3 Demo Scenarios

```
scenarios/
├── 01-normal-flow.json          # Normal request flow
├── 02-prompt-injection.json     # Sentinel blocks injection
├── 03-pii-anonymization.json    # Vault anonymizes data
├── 04-cross-tenant.json         # Vault blocks cross-tenant
├── 05-honey-token.json          # Honey-token triggered
├── 06-module-bypass.json        # Lite pipeline (Vault bypass)
├── 07-anchor-degraded.json      # Anchor performance issue
└── 08-incident-response.json    # Full incident workflow
```

### 9.4 Synthetic Data Generation

```bash
# Generate continuous test traffic
$ tracker-cli generate \
    --rate 50/s \
    --duration 5m \
    --pipelines full,lite,minimal \
    --include-security-events
```

---

## 10. Data Requirements

### 10.1 Configuration Schema

```yaml
# /etc/bastion-tracker/config.yaml
version: 1.0

server:
  rest_port: 8080
  grpc_port: 9090
  websocket_port: 8081
  frontend_port: 3000

# NATS event bus
nats:
  url: nats://nats:4222
  subjects:
    - "bastion-rag.events.>"
  max_pending: 100000

# Storage
storage:
  postgresql:
    url: ${POSTGRES_URL}
    max_connections: 50
    retention_days: 30
  
  prometheus:
    url: http://prometheus:9090
    retention: 90d
  
  loki:
    url: http://loki:3100
    retention: 30d
  
  jaeger:
    collector_url: http://jaeger-collector:14268
    query_url: http://jaeger-query:16686
    retention: 30d
  
  redis:
    url: redis://redis:6379

# Real-time
realtime:
  websocket:
    max_connections: 1000
    heartbeat_interval: 30s
    buffer_size: 1000

# Alerting
alerting:
  enabled: true
  channels:
    slack:
      webhook_url: ${SLACK_WEBHOOK}
    email:
      smtp_host: smtp.example.com
      smtp_port: 587
      from: tracker@bastion-rag.local
  
  rules:
    - name: high_latency
      condition: "module.latency_p95 > 200ms"
      severity: warning
      channels: [slack]
    
    - name: prompt_injection_spike
      condition: "sentinel.blocks_per_minute > 50"
      severity: critical
      channels: [slack, email]
    
    - name: honey_token_triggered
      condition: "honey_token.triggered == true"
      severity: critical
      channels: [slack, email]

# Demo mode
demo:
  enabled: true
  scenarios_path: /scenarios
  default_speed: 1.0

# Pipeline tracking
pipelines:
  full: ["sentinel", "vault", "navigator", "anchor", "llm"]
  lite: ["sentinel", "navigator", "anchor", "llm"]
  minimal: ["sentinel", "navigator", "llm"]
  custom: dynamic

# Authentication
auth:
  type: jwt
  jwt_secret_file: /etc/secrets/jwt
  roles:
    - admin
    - operator
    - viewer

# Logging
logging:
  level: info
  format: json

# Metrics
metrics:
  enabled: true
  port: 9091
```

---

## 11. Deployment and Operations

### 11.1 Docker Compose (Recommended for PoC)

```yaml
version: '3.8'

services:
  tracker-backend:
    image: bastion-rag/tracker-backend:1.0.0
    ports:
      - "8080:8080"
      - "9090:9090"
      - "8081:8081"
    environment:
      - CONFIG_PATH=/etc/tracker/config.yaml
    depends_on:
      - nats
      - postgres
      - redis
  
  tracker-frontend:
    image: bastion-rag/tracker-frontend:1.0.0
    ports:
      - "3000:3000"
    environment:
      - REACT_APP_API_URL=http://localhost:8080
      - REACT_APP_WS_URL=ws://localhost:8081
  
  nats:
    image: nats:2.10-alpine
    ports:
      - "4222:4222"
      - "8222:8222"  # Monitoring
  
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: tracker
    volumes:
      - tracker_data:/var/lib/postgresql/data
  
  prometheus:
    image: prom/prometheus:v2.45.0
    ports:
      - "9092:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
  
  loki:
    image: grafana/loki:2.9.0
    ports:
      - "3100:3100"
  
  jaeger:
    image: jaegertracing/all-in-one:1.50
    ports:
      - "16686:16686"  # UI
      - "14268:14268"  # Collector
  
  grafana:
    image: grafana/grafana:10.0.0
    ports:
      - "3001:3000"
    volumes:
      - grafana_data:/var/lib/grafana
  
  redis:
    image: redis:7-alpine

volumes:
  tracker_data:
  grafana_data:
```

### 11.2 Resource Requirements

**PoC Minimum:**
- 4 CPU cores
- 16GB RAM
- 100GB SSD

**Production:**
- 8 CPU cores
- 32GB RAM
- 500GB SSD
- Replicate for HA

### 11.3 Monitoring

**Self-monitoring (Tracker monitors itself):**
- Event ingestion rate
- Processing latency
- Storage usage
- WebSocket connections
- API response times

---

## 12. Appendix

### 12.1 PoC Demo Walkthrough

**Recommended Demo Flow (10 minutes):**

```
0:00-1:00  Introduction & System Topology
           - Show overall architecture
           - Explain 5 modules

1:00-3:00  Normal Request Flow
           - Live request visualization
           - Show full pipeline (A→B→C→E→LLM)
           - Highlight each module's role

3:00-4:00  Prompt Injection Defense
           - Inject malicious query
           - Show Sentinel blocking
           - Display security event

4:00-5:30  Multi-tenant + Anonymization
           - Customer data request
           - Vault PII detection
           - K-anonymity in action

5:30-6:30  Cross-tenant Prevention
           - Attempted cross-tenant access
           - Vault denial
           - Alert generation

6:30-7:30  Honey-token Detection
           - Attacker accesses fake data
           - Real-time alert
           - Incident creation

7:30-8:30  Module Bypass (Lite Pipeline)
           - Public docs request
           - Vault bypassed
           - Show different routing

8:30-9:30  Operations Dashboard
           - Health overview
           - Performance metrics
           - Alert management

9:30-10:00 Q&A
```

### 12.2 Operations Team Workflows

**Workflow 1: Alert Response**
```
1. Receive alert (Slack)
2. Open Tracker dashboard
3. View alert details
4. Click into related trace
5. Investigate root cause
6. Acknowledge or escalate
7. Resolve when fixed
```

**Workflow 2: Investigating Performance Issue**
```
1. Notice latency spike on dashboard
2. View topology to find slow module
3. Click module for details
4. Drill into recent traces
5. Identify common pattern
6. Check logs in Grafana
7. Take action (scaling, restart, etc.)
```

**Workflow 3: Security Incident Response**
```
1. Critical alert triggered
2. Open security events page
3. View incident details
4. Trace attacker actions
5. Identify scope of access
6. Initiate response (block IP, etc.)
7. Document in incident report
```

### 12.3 Roadmap

- v1.1: ML-based anomaly detection
- v1.2: Automated remediation actions
- v1.3: Custom dashboard builder
- v2.0: Multi-cluster federation
- v2.1: Long-term analytics & reports
- v2.2: AI assistant for investigations

### 12.4 Change History

| Version | Date | Changes |
|---|---|---|
| 0.1 | 2026-05-15 | Initial draft |
| 0.5 | 2026-05-16 | Visualization focus added |
| 1.0 | 2026-05-17 | PoC-ready release |

---

**End of Document**
