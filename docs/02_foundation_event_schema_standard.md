# Bastion Event Schema Standard

**Project:** Bastion - RAG Security Governance Framework
**Document Type:** Foundation (Tier 1)
**Document ID:** 02-event-schema-standard
**Version:** 2.0
**Date:** 2026-05-26
**Status:** Active
**Supersedes:** v1.0 (2026-05-17) — archived at docs/archive/v2/

---

## 1. Introduction

### 1.1 Purpose

This document defines the **standard event schema** that all Bastion modules use to communicate. Events are the **only** mechanism for inter-module communication, enabling loose coupling while supporting cross-cutting features.

### 1.2 Why Events Matter

```
Events are the glue of Bastion:

- Modules stay independent (loose coupling)
- Cross-cutting features observe events
- Tracker aggregates events
- Lineage follows events
- No direct module-to-module calls

If module A needs to "tell" module B something,
it publishes an event. B subscribes if interested.
A doesn't know or care who listens.
```

### 1.3 Design Principles

```
1. Self-describing: Every event has full context
2. Traceable: Every event has trace_id
3. Ordered: Events have timestamps and sequence
4. Typed: Clear event types
5. Extensible: Custom data without schema changes
6. Versioned: Schema can evolve
```

---

## 2. Event Bus: NATS

### 2.1 Why NATS

```
Selected: NATS (lightweight message broker)

Rationale:
- Lightweight (fits SMB scale)
- High performance
- Subject-based routing
- At-least-once delivery option
- Simple operations

Alternatives:
- Kafka (rejected: too heavy for SMB)
- RabbitMQ (rejected: more complex)
- Redis Streams (rejected: less robust)
```

### 2.2 Subject Hierarchy

```
bastion.events.{module}.{event_type}

Examples:
bastion.events.sentinel.input_validated
bastion.events.vault.anonymized
bastion.events.navigator.search_completed
bastion.events.anchor.bias_detected
bastion.events.tracker.incident_created

Wildcard subscriptions:
bastion.events.>              # All events
bastion.events.sentinel.>     # All Sentinel events
bastion.events.*.honey_token_* # Honey-token from any module
```

### 2.3 Subscription Patterns

```
Tracker subscribes to:
bastion.events.>              # Everything

Honey-token Coordinator:
bastion.events.*.honey_token_*

Lineage Coordinator:
bastion.events.>              # All, builds trace

Module-specific:
bastion.events.vault.>        # Vault's own events
```

---

## 3. Base Event Schema

### 3.1 Common Event Structure

Every Bastion event MUST include these fields:

```protobuf
syntax = "proto3";
package bastion.events.v1;

import "google/protobuf/timestamp.proto";
import "google/protobuf/struct.proto";

message BastionEvent {
  // === Identity (required) ===
  string event_id = 1;          // Unique event ID (UUID)
  string event_type = 2;        // e.g., "input_validated"
  string schema_version = 3;    // e.g., "1.0"
  
  // === Tracing (required) ===
  string trace_id = 4;          // Request trace ID
  string span_id = 5;           // This operation's span
  string parent_span_id = 6;    // Parent operation
  
  // === Source (required) ===
  string module = 7;            // "sentinel", "vault", etc.
  string module_version = 8;    // Module version
  
  // === Timing (required) ===
  google.protobuf.Timestamp timestamp = 9;
  int64 duration_ms = 10;       // Operation duration
  
  // === Context (required) ===
  string tenant_id = 11;
  string user_id = 12;
  string request_id = 13;
  
  // === Classification (required) ===
  Severity severity = 14;
  Category category = 15;
  
  // === Payload (optional) ===
  google.protobuf.Struct data = 16;  // Event-specific
  
  // === Outcome (optional) ===
  string status = 17;           // "success", "blocked", etc.
  string action_taken = 18;
  
  enum Severity {
    INFO = 0;
    WARNING = 1;
    ERROR = 2;
    CRITICAL = 3;
  }
  
  enum Category {
    OPERATIONAL = 0;    // Normal operations
    SECURITY = 1;       // Security events
    PERFORMANCE = 2;    // Performance metrics
    AUDIT = 3;          // Audit trail
  }
}
```

### 3.2 Field Requirements

| Field | Required | Purpose |
|---|---|---|
| event_id | ✅ | Unique identification |
| event_type | ✅ | Event classification |
| schema_version | ✅ | Schema evolution |
| trace_id | ✅ | Request tracing |
| span_id | ✅ | Operation tracking |
| parent_span_id | ⚠️ | Causality (if applicable) |
| module | ✅ | Source identification |
| timestamp | ✅ | Temporal ordering |
| duration_ms | ⚠️ | Performance (if applicable) |
| tenant_id | ✅ | Multi-tenancy |
| user_id | ✅ | Attribution |
| request_id | ✅ | Request correlation |
| severity | ✅ | Priority |
| category | ✅ | Classification |
| data | ❌ | Event-specific payload |
| status | ❌ | Outcome |
| action_taken | ❌ | Response |

### 3.3 JSON Representation

```json
{
  "event_id": "evt-550e8400-e29b-41d4",
  "event_type": "input_validated",
  "schema_version": "1.0",
  "trace_id": "trace-12345",
  "span_id": "span-001",
  "parent_span_id": null,
  "module": "sentinel",
  "module_version": "1.0.0",
  "timestamp": "2026-05-17T14:23:45.123Z",
  "duration_ms": 1,
  "tenant_id": "tenant-acme",
  "user_id": "user-alice",
  "request_id": "req-789",
  "severity": "INFO",
  "category": "OPERATIONAL",
  "data": {
    "injection_score": 0.05,
    "metadata_valid": true
  },
  "status": "passed",
  "action_taken": "none"
}
```

---

## 4. Trace Context Propagation

### 4.1 W3C Trace Context Standard

Bastion follows W3C Trace Context for distributed tracing:

```
traceparent header format:
00-{trace_id}-{span_id}-{flags}

Example:
00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01

Components:
- 00: version (2 hex chars)
- trace_id: 32 hex chars (16 bytes, request-wide)
- span_id: 16 hex chars (8 bytes, per operation)
- flags: 2 hex chars (sampling, 01 = sampled)
```

### 4.2 Propagation Flow

```
Request enters → Generate trace_id
       ↓
[Sentinel] span_id=001, parent=null
       ↓ propagate trace_id
[Vault] span_id=002, parent=001
       ↓ propagate trace_id
[Navigator] span_id=003, parent=002
       ↓ propagate trace_id
[Anchor] span_id=004, parent=003
       ↓
[LLM] span_id=005, parent=004
       ↓
All events share trace_id=12345
Lineage reconstructs full path
```

### 4.3 Propagation Rules

```
1. trace_id: Generated once at entry, propagated unchanged
2. span_id: New for each module operation
3. parent_span_id: Previous module's span_id
4. Propagation: Via gRPC metadata / HTTP headers

Implementation:
- Each module receives trace context
- Creates new span
- Links to parent
- Propagates to next
```

---

## 5. Standard Event Types

### 5.1 Operational Events

Events for normal operations:

```yaml
# Lifecycle
{module}.request_received
{module}.request_completed
{module}.request_failed

# Processing
{module}.processing_started
{module}.processing_completed

# Pipeline routing
sentinel.pipeline_routing_decided
```

### 5.2 Security Events

Events for security-relevant actions:

```yaml
# Sentinel
sentinel.injection_detected
sentinel.injection_blocked
sentinel.pii_re_emergence_prevented
sentinel.content_filtered

# Vault
vault.access_denied
vault.cross_tenant_attempt
vault.deanonymization_performed
vault.break_glass_requested

# Navigator
navigator.permission_filtered

# Cross-cutting (any module)
{module}.honey_token_referenced
{module}.honey_token_accessed
{module}.honey_token_retrieved
{module}.honey_token_leaked
{module}.anomaly_detected
```

### 5.3 Performance Events

Events for performance monitoring:

```yaml
{module}.latency_threshold_exceeded
{module}.throughput_degraded
{module}.resource_exhausted
```

### 5.4 Audit Events

Events for compliance audit:

```yaml
vault.data_accessed
vault.k_anonymity_enforced
sentinel.validation_logged
{module}.config_changed
```

---

## 6. Module-Specific Event Payloads

### 6.1 Sentinel Events

```protobuf
// sentinel.input_validated
message SentinelInputValidatedData {
  float injection_score = 1;
  bool metadata_valid = 2;
  string pipeline_decision = 3;  // "full", "lite", etc.
}

// sentinel.injection_blocked
message SentinelInjectionBlockedData {
  string pattern_matched = 1;
  float confidence = 2;
  string query_excerpt = 3;  // Sanitized
}

// sentinel.pii_re_emergence_prevented
message SentinelPIIPreventedData {
  repeated string pii_types = 1;
  int32 redactions = 2;
}
```

### 6.2 Vault Events

```protobuf
// vault.anonymized
message VaultAnonymizedData {
  string category = 1;
  int32 fields_anonymized = 2;
  repeated string strategies_used = 3;
}

// vault.access_denied
message VaultAccessDeniedData {
  string requested_category = 1;
  string user_department = 2;
  string deny_reason = 3;
}

// vault.k_anonymity_enforced
message VaultKAnonData {
  int32 required_k = 1;
  int32 achieved_k = 2;
  string generalization = 3;
}
```

### 6.3 Navigator Events

```protobuf
// navigator.search_completed
message NavigatorSearchData {
  int32 candidates = 1;
  int32 filtered = 2;
  int32 returned = 3;
  string strategy = 4;  // "hybrid", "vector", etc.
}
```

### 6.4 Anchor Events

```protobuf
// anchor.bias_detected
message AnchorBiasData {
  string category = 1;  // "gender", etc.
  float score = 2;
  string severity = 3;
}

// anchor.embedding_secured
message AnchorSecuredData {
  float noise_added = 1;
  float similarity_preserved = 2;
}
```

### 6.5 Cross-Cutting Event Payloads

```protobuf
// {module}.honey_token_* (any module)
message HoneyTokenEventData {
  string honey_token_id = 1;
  string honey_token_type = 2;
  string detection_layer = 3;  // "input", "data", "search", "output"
  string detecting_module = 4;
  string source_ip = 5;
}
```

---

## 7. Event Delivery Guarantees

### 7.1 Delivery Modes

```
Operational events: At-most-once
- Best effort
- OK to lose occasionally
- Low overhead

Security events: At-least-once
- Guaranteed delivery
- May duplicate (idempotent handling)
- Higher overhead

Audit events: At-least-once + persistent
- Guaranteed + stored
- Compliance requirement
- Highest reliability
```

### 7.2 Buffering

```
If subscriber (e.g., Tracker) is down:

Publisher behavior:
- Operational: drop after buffer full
- Security: buffer + retry
- Audit: persist locally + retry

Buffer limits:
- Memory buffer: 10,000 events
- Disk buffer (audit): unlimited
```

### 7.3 Idempotency

```
All event handlers MUST be idempotent:

- Use event_id for deduplication
- Same event processed twice = same result
- Critical for at-least-once delivery

Example:
processedEvents = Set()
onEvent(event):
    if event.event_id in processedEvents:
        return  // Already processed
    process(event)
    processedEvents.add(event.event_id)
```

---

## 8. Hook Integration

### 8.1 Hooks Generate Events

```
Cross-cutting hooks publish standard events:

class HoneyTokenHook implements Hook {
    onDetect(context) {
        event = BastionEvent{
            event_type: "honey_token_referenced",
            module: context.module,
            severity: CRITICAL,
            category: SECURITY,
            data: {...}
        }
        eventBus.publish(event)
    }
}
```

### 8.2 Hook Contract

```
Hooks MUST:
1. Use standard event schema
2. Include trace_id
3. Be non-blocking (async publish)
4. Not affect core function on failure

Hooks MUST NOT:
1. Block the core operation
2. Throw exceptions to core
3. Modify core results (only observe)
```

---

## 9. Event Schema Versioning

### 9.1 Version Strategy

```
schema_version field in every event

Compatibility:
- Minor version: backward compatible (add fields)
- Major version: breaking change (rare)

Consumers:
- Handle unknown fields gracefully
- Check schema_version for major changes
```

### 9.2 Evolution Rules

```
Allowed (minor version):
✅ Add optional fields
✅ Add new event types
✅ Add enum values

Forbidden (requires major version):
❌ Remove fields
❌ Rename fields
❌ Change field types
❌ Change required/optional
```

---

## 10. Implementation Guide

### 10.1 Publishing Events

```go
type EventPublisher struct {
    nats     *nats.Conn
    module   string
    version  string
}

func (p *EventPublisher) Publish(
    eventType string,
    traceCtx TraceContext,
    severity Severity,
    data map[string]interface{},
) error {
    event := &BastionEvent{
        EventID:       uuid.New().String(),
        EventType:     eventType,
        SchemaVersion: "1.0",
        TraceID:       traceCtx.TraceID,
        SpanID:        traceCtx.SpanID,
        ParentSpanID:  traceCtx.ParentSpanID,
        Module:        p.module,
        ModuleVersion: p.version,
        Timestamp:     timestamppb.Now(),
        TenantID:      traceCtx.TenantID,
        UserID:        traceCtx.UserID,
        RequestID:     traceCtx.RequestID,
        Severity:      severity,
        Data:          structpb.NewStruct(data),
    }
    
    subject := fmt.Sprintf("bastion.events.%s.%s", 
        p.module, eventType)
    
    payload, _ := proto.Marshal(event)
    return p.nats.Publish(subject, payload)
}
```

### 10.2 Subscribing to Events

```go
type EventSubscriber struct {
    nats *nats.Conn
}

func (s *EventSubscriber) Subscribe(
    pattern string,
    handler func(*BastionEvent),
) error {
    _, err := s.nats.Subscribe(pattern, func(msg *nats.Msg) {
        event := &BastionEvent{}
        proto.Unmarshal(msg.Data, event)
        
        // Idempotent handling
        if s.alreadyProcessed(event.EventID) {
            return
        }
        
        handler(event)
        s.markProcessed(event.EventID)
    })
    return err
}

// Usage
subscriber.Subscribe("bastion.events.>", func(e *BastionEvent) {
    // Tracker processes all events
})
```

### 10.3 Trace Context Helper

```go
type TraceContext struct {
    TraceID      string
    SpanID       string
    ParentSpanID string
    TenantID     string
    UserID       string
    RequestID    string
}

func NewSpan(parent TraceContext) TraceContext {
    return TraceContext{
        TraceID:      parent.TraceID,        // unchanged
        SpanID:       generateSpanID(),       // new
        ParentSpanID: parent.SpanID,          // link
        TenantID:     parent.TenantID,
        UserID:       parent.UserID,
        RequestID:    parent.RequestID,
    }
}
```

---

## 11. Testing Events

### 11.1 Event Validation

```go
func ValidateEvent(event *BastionEvent) error {
    if event.EventID == "" {
        return errors.New("missing event_id")
    }
    if event.TraceID == "" {
        return errors.New("missing trace_id")
    }
    if event.Module == "" {
        return errors.New("missing module")
    }
    // ... validate all required fields
    return nil
}
```

### 11.2 Mock Event Bus

```go
type MockEventBus struct {
    published []*BastionEvent
}

func (m *MockEventBus) Publish(event *BastionEvent) error {
    ValidateEvent(event)  // Ensure valid
    m.published = append(m.published, event)
    return nil
}

// Test assertion
func TestSentinelPublishesEvent(t *testing.T) {
    bus := &MockEventBus{}
    sentinel := NewSentinel(bus)
    
    sentinel.ValidateInput("malicious query")
    
    assert.Equal(t, "injection_detected", 
        bus.published[0].EventType)
}
```

---

## 12. Summary

### 12.1 Key Points

```
1. Events are the ONLY inter-module communication
2. Standard schema for all events
3. trace_id enables lineage
4. NATS as event bus
5. Hooks publish standard events
6. Idempotent handling required
7. Versioned schema
```

### 12.2 The Contract

```
Every module agrees to:
- Publish events in standard format
- Propagate trace context
- Handle events idempotently
- Never block on event publishing

This contract enables:
- Loose coupling
- Cross-cutting features
- Full observability
- Module independence
```

---

## 13. Polyglot Implementation Note (v3)

```
The event schema is LANGUAGE-AGNOSTIC.

Go modules (Sentinel, Vault, Tracker) publish:
  json.Marshal(BastionEvent{...}) → NATS

Python modules (Navigator, Anchor) publish:
  json.dumps(event_dict) → NATS

The JSON schema is identical regardless of source language.
Tracker and any subscriber receive and parse events the same way.

Python publisher implementation:
  asyncio.get_event_loop().run_until_complete(
      nc.publish(subject, json.dumps(event).encode())
  )

This is why the wire contract is fully polyglot-compatible.
```

---

## 14. Change History

| Version | Date | Changes |
|---|---|---|
| 1.0 | 2026-05-17 | Initial event schema standard |
| 2.0 | 2026-05-26 | Polyglot note (Python modules use same JSON schema); `anchor.response_verified` added to registry |

---

## Appendix A: Complete Event Type Registry

```yaml
# Operational
sentinel.request_received
sentinel.input_validated
sentinel.output_validated
sentinel.pipeline_routing_decided
vault.request_received
vault.anonymized
vault.transform_executed
navigator.search_started
navigator.search_completed
anchor.embedding_secured
anchor.response_analyzed
anchor.response_verified

# Security
sentinel.injection_detected
sentinel.injection_blocked
sentinel.pii_re_emergence_prevented
sentinel.content_filtered
sentinel.honey_token_referenced
sentinel.honey_token_leaked
vault.access_denied
vault.cross_tenant_attempt
vault.deanonymization_performed
vault.break_glass_requested
vault.honey_token_accessed
navigator.permission_filtered
navigator.honey_token_retrieved
anchor.bias_detected
anchor.anomaly_detected

# Performance
{module}.latency_threshold_exceeded
{module}.throughput_degraded
{module}.resource_exhausted

# Audit
vault.data_accessed
vault.k_anonymity_enforced
{module}.config_changed

# Tracker (aggregated)
tracker.incident_created
tracker.honey_token_alert
tracker.lineage_completed
```

---

**End of Document**
