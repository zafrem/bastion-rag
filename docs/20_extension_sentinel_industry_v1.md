# Bastion-Sentinel Industry Filter Extension SRS

**Project:** Bastion-RAG - RAG Security Governance Framework
**Document Type:** Module Extension SRS (Tier 2.5)
**Document ID:** 20-sentinel-industry-ext
**Module:** A - Sentinel (extension)
**Version:** 1.0
**Date:** 2026-05-27
**Status:** Draft

**Base Module Reference:** 10-sentinel-srs (v3.0)
**Foundation References:**
- 01-architecture-principles (v3)
- 02-event-schema-standard

---

## 1. Introduction

### 1.1 Purpose

This document specifies the **Industry Filter Extension** for Sentinel. It enables industry-specific validation rules to be layered on top of Sentinel's Core without modifying core logic.

### 1.2 Problem

Sentinel's Core handles general-purpose validation: prompt injection, PII patterns, content policy. Different industries (healthcare, defense, finance) require additional domain-specific checks that:
- Must sometimes **block** the request synchronously (result affects the response)
- Sometimes only need to **audit** asynchronously (no latency impact needed)

Modifying Core for each industry creates a monolithic, tightly-coupled Sentinel that is hard to maintain and slow for tenants that don't need that industry's rules.

### 1.3 Design Principle

Industry filters are **additive**. The Core always runs first. Industry filters extend, never replace, core validation.

```
Request
  ↓
Core Filter Chain          ← always runs, always first (unchanged)
  ↓
Industry Blocking Filters  ← sync, opt-in, inserted into critical path
  ↓
Response
  ↑ (fire-and-forget, parallel)
Industry Async Hooks       ← zero critical-path latency
```

Strong coupling (blocking mode) is **opt-in per filter**. Latency cost is paid only for the specific checks that require a synchronous decision.

---

## 2. Filter Modes

### 2.1 Blocking Mode (sync)

Inserted into the filter chain after Core. Runs synchronously. Can block, redact, or flag the request before a response is returned.

**Use when:** the filter's result must affect whether the request is allowed to proceed.

| Action | Effect |
|---|---|
| `block` | Reject request immediately (HTTP 422), no further processing |
| `redact` | Replace matched content with a placeholder, continue pipeline |
| `flag` | Allow request, add warning to `ValidateResponse.flags` |

**Examples:**
- HIPAA: PHI term detected → block
- GDPR: EU personal data in query → redact before forwarding to LLM
- ITAR: export-controlled technical term → block and emit alert event

**Latency impact:** proportional to the filter's own execution time. Regex-based filters add ~0.1–1ms. ML inference filters add 5–200ms depending on hardware.

### 2.2 Async Mode (fire-and-forget)

Registered as a hook handler via `HookManager`. Runs in a background goroutine. No return value, no effect on the response path.

**Use when:** the filter's result is for audit, telemetry, or downstream notification only — the request proceeds regardless.

**Examples:**
- Log that a finance-domain query was processed
- Notify a SIEM system of a sensitive-domain access
- Collect industry-specific metrics for compliance dashboards

**Latency impact:** zero. Fire-and-forget is always available as the low-cost alternative to blocking mode.

---

## 3. Interfaces

### 3.1 IndustryFilter (Blocking)

```go
// IndustryFilter runs synchronously in the filter chain.
// Priority >= 100 ensures it runs after Core (Core uses 0–99).
type IndustryFilter interface {
    ID() string
    Priority() int
    Filter(ctx context.Context, req *FilterRequest) (FilterDecision, error)
}

type FilterRequest struct {
    Text         string
    TenantID     string
    UserID       string
    DataCategory string
    Metadata     map[string]string
}

type FilterDecision struct {
    Allowed  bool
    Action   FilterAction
    Reason   string
    Modified string  // populated if Action == FilterRedact
}

type FilterAction int

const (
    FilterBlock  FilterAction = iota  // reject; return HTTP 422
    FilterRedact                       // replace matched text; continue
    FilterFlag                         // allow; append to response flags
)
```

### 3.2 IndustryHookHandler (Async)

```go
// IndustryHookHandler runs asynchronously via HookManager.
type IndustryHookHandler interface {
    ID() string
    EventTypes() []string  // hook event names to subscribe to
    Handle(evt hooks.Event)
}
```

### 3.3 IndustryFilterRegistry

```go
type IndustryFilterRegistry interface {
    RegisterFilter(f IndustryFilter) error
    RegisterHookHandler(h IndustryHookHandler) error
    FiltersForTenant(tenantID string) []IndustryFilter
    HandlersForTenant(tenantID string) []IndustryHookHandler
}
```

Tenant resolution order: `tenant_overrides[tenantID]` → `filters` (global list) → empty (no industry filters).

---

## 4. Built-in Filter Sets

| ID | Domain | Default Mode | Default Action |
|---|---|---|---|
| `hipaa` | Healthcare (HIPAA PHI patterns) | blocking | block |
| `gdpr` | EU personal data | blocking | redact |
| `itar` | Defense / export-controlled terms | blocking | block |
| `pci` | Payment card data (PCI DSS) | blocking | redact |
| `custom_regex` | User-defined regex patterns | configurable | configurable |

Each built-in is a compiled-in implementation of `IndustryFilter`. Custom filters can be loaded as Go plugins (`.so`) or registered at compile time for static builds.

---

## 5. Execution Flow

```
POST /v1/sentinel/validate/input
  ↓
1. Core filter chain (injection, PII, content policy)  — always, unmodified
2. Load tenant's blocking IndustryFilters (registry.FiltersForTenant)
3. For each blocking filter (sorted by Priority):
     a. result = filter.Filter(ctx, req)
     b. if !result.Allowed && result.Action == FilterBlock:
            emit bastion-rag.events.sentinel.industry_filter_blocked
            return HTTP 422 immediately
     c. if result.Action == FilterRedact:
            req.Text = result.Modified
            continue to next filter
     d. if result.Action == FilterFlag:
            append result.Reason to response.Flags
            continue to next filter
4. Fire async hook handlers (hm.Fire — non-blocking, goroutine)
5. Return ValidateResponse
```

---

## 6. Configuration

```yaml
sentinel:
  industry:
    enabled: true

    filters:
      - id: hipaa_phi
        builtin: hipaa
        mode: blocking
        position: after_core      # after_core | before_core
        action_on_match: block

      - id: gdpr_personal
        builtin: gdpr
        mode: blocking
        position: after_core
        action_on_match: redact

      - id: itar_alert
        builtin: itar
        mode: async               # audit only; zero latency impact

      - id: custom_finance_log
        plugin_path: /etc/sentinel/plugins/finance_logger.so
        mode: async

    # Per-tenant overrides: only these filters run for the given tenant.
    # Omit a tenant to use the global filter list.
    tenant_overrides:
      tenant-hospital:
        filters: [hipaa_phi, itar_alert]
      tenant-eu:
        filters: [gdpr_personal]
      tenant-default:
        filters: []               # no industry filters
```

---

## 7. Performance

| Scenario | Critical-path latency impact |
|---|---|
| Industry extension disabled | Zero |
| All filters async | Zero |
| 1 blocking filter (regex, `hipaa`) | +0.1–1ms |
| 1 blocking filter (ML inference, CPU) | +50–200ms |
| 1 blocking filter (ML inference, GPU) | +5–30ms |
| Tenant has no matching filters | O(1) map lookup only |

ML-based blocking filters must use GPU or a pre-warmed batch queue to satisfy Sentinel's p95 latency targets. Async mode is always available as an alternative when latency cannot be sacrificed.

---

## 8. Events

All use Foundation BastionEvent schema (`module: "sentinel"`, `module_version: "3.0.0"`).

```
bastion-rag.events.sentinel.industry_filter_blocked
  → filter_id, tenant_id, reason

bastion-rag.events.sentinel.industry_filter_redacted
  → filter_id, tenant_id, fields_redacted

bastion-rag.events.sentinel.industry_filter_flagged
  → filter_id, tenant_id, flag_reason
```

---

## 9. Non-Functional Requirements

| ID | Requirement |
|---|---|
| NFR-IND-001 | Core filter chain runs regardless of industry filter state |
| NFR-IND-002 | Async filters add zero critical-path latency |
| NFR-IND-003 | Tenant override takes precedence over the global filter list |
| NFR-IND-004 | Plugin load failure is logged as ERROR but does not prevent Sentinel startup |
| NFR-IND-005 | Each IndustryFilter instance is stateless per request |
| NFR-IND-006 | `before_core` position is available but discouraged; requires explicit acknowledgment in config |

---

## 10. Change History

| Version | Date | Changes |
|---|---|---|
| 1.0 | 2026-05-27 | Initial draft |

---

**End of Document**
