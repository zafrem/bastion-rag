# Bastion-RAG Multi-Tenancy Cross-Cutting SRS

**Project:** Bastion-RAG - RAG Security Governance Framework  
**Document Type:** Cross-Cutting SRS (Tier 3)  
**Document ID:** 21-multi-tenancy-srs  
**Feature:** Multi-Tenancy (Tenant Isolation)  
**Version:** 1.0  
**Date:** 2026-05-17  
**Status:** Draft  
**Priority:** 🔴 CRITICAL (highest cross-cutting priority)

**Foundation References:**
- 01-architecture-principles
- 02-event-schema-standard
- 03-module-interaction-map

**Participating Modules:**
- Vault (key isolation, data separation)
- Navigator (search pre-filter) ⭐ CRITICAL
- Sentinel (tenant validation)

---

## 1. Feature Overview

### 1.1 Purpose

Multi-tenancy ensures **complete isolation** between tenants in a shared Bastion-RAG deployment. Tenant A must never access Tenant B's data — through storage, search, or any other path.

### 1.2 Why This is the Most Critical Cross-Cutting Feature

```
Per Cross-Cutting Analysis, multi-tenancy is highest priority:

The danger:
- Vault isolates data at STORAGE
- But if Navigator SEARCH leaks across tenants
- → Isolation is meaningless!

Single point of failure:
- One module missing tenant filter
- = Cross-tenant data breach

This is why it's CROSS-CUTTING, not single-module.
```

### 1.3 The Core Insight

```
Tenant isolation requires EVERY layer to cooperate:

Storage isolation (Vault):     necessary but NOT sufficient
Search isolation (Navigator):  CRITICAL — most overlooked
Input validation (Sentinel):   tenant_id verification

If ANY layer fails to isolate → breach.
```

### 1.4 Scope

**In Scope:**
- Tenant ID propagation (all modules)
- Storage isolation (Vault)
- Search pre-filtering (Navigator)
- Tenant validation (Sentinel)
- Cross-tenant attempt detection
- Tenant isolation verification

**Out of Scope:**
- Tenant onboarding/provisioning (ops concern)
- Billing per tenant (business concern)
- Per-module core functions (their own SRS)

---

## 2. Module Responsibilities

### 2.1 Responsibility Distribution

| Module | Responsibility | Criticality |
|---|---|---|
| **Sentinel** | Validate tenant_id in requests | 🟡 Important |
| **Vault** | Separate keys & data per tenant | 🔴 Critical |
| **Navigator** | Pre-filter search by tenant | 🔴 CRITICAL |
| **Tracker** | Detect cross-tenant attempts | 🟡 Monitor |

### 2.2 Sentinel's Role

```
Tenant ID Validation:
- Extract tenant_id from request
- Validate format
- Ensure present (reject if missing)
- Propagate to downstream (via trace context)

What Sentinel does NOT do:
- Storage isolation (Vault)
- Search isolation (Navigator)
```

### 2.3 Vault's Role

```
Storage Isolation:
- Separate encryption keys per tenant
- Tenant-scoped token mappings
- Reject cross-tenant deanonymization

Key separation:
tenant-acme → key-acme
tenant-globex → key-globex
(Cannot decrypt across tenants)
```

### 2.4 Navigator's Role (MOST CRITICAL)

```
Search Pre-Filtering:
- Filter by tenant_id BEFORE search
- NEVER post-filter (security risk!)
- Tenant-scoped collections or filters

Why pre-filter critical:
Post-filter risk:
1. Search all tenants
2. Filter results after
3. → Timing attacks, leak via metadata

Pre-filter safe:
1. Restrict to tenant first
2. Search only tenant's data
3. → No cross-tenant exposure
```

### 2.5 Tracker's Role

```
Cross-Tenant Detection:
- Monitor for cross-tenant attempts
- Aggregate violation events
- Alert on patterns
```

---

## 3. Hook Usage (Detailed)

This section details how each module's hooks implement multi-tenancy. (Module SRS only briefly mention these.)

### 3.1 Sentinel Hook: Tenant Validation

```
Hook point: sentinel.input.validated (Core extension)

Implementation:
func tenantValidationHook(request) {
    tenant_id := request.tenant_id
    
    // Validate presence
    if tenant_id == "" {
        return Reject("missing tenant_id")
    }
    
    // Validate format
    if !validTenantFormat(tenant_id) {
        return Reject("invalid tenant_id")
    }
    
    // Propagate via trace context (Foundation)
    request.traceContext.TenantID = tenant_id
}
```

### 3.2 Vault Hook: Key Isolation

```
Hook point: vault.tenant.isolate

Implementation:
func tenantKeyHook(tenant_id, operation) {
    // Get tenant-specific key
    key := kms.GetTenantKey(tenant_id)
    
    // All crypto uses tenant key
    operation.useKey(key)
    
    // Reject if cross-tenant
    if operation.targetTenant != tenant_id {
        emit("vault.cross_tenant_attempt")
        return Reject("cross-tenant denied")
    }
}
```

### 3.3 Navigator Hook: Pre-Filter (CRITICAL)

```
Hook point: navigator.tenant.prefilter

Implementation:
func tenantPreFilterHook(searchRequest) {
    tenant_id := searchRequest.traceContext.TenantID
    
    // CRITICAL: Add tenant filter BEFORE search
    searchRequest.filter.Add("tenant_id", tenant_id)
    
    // Verify filter applied
    if !searchRequest.hasTenantFilter() {
        return Reject("tenant filter not applied")
    }
    
    // Now search only sees tenant's data
}

// Qdrant example:
filter := qdrant.Filter{
    Must: []Condition{
        {Key: "tenant_id", Match: tenant_id},  // pre-filter
    },
}
results := qdrant.Search(query, filter)  // tenant-scoped
```

### 3.4 Tracker: Cross-Tenant Aggregation

```
Subscribes: bastion-rag.events.*.cross_tenant_attempt

func onCrossTenantAttempt(event) {
    // Aggregate by user
    user := event.user_id
    attempts[user]++
    
    // Alert on pattern
    if attempts[user] > threshold {
        createIncident("repeated cross-tenant attempts", user)
        alert(security_team)
    }
}
```

---

## 4. Data Flow

### 4.1 Tenant Isolation Flow

```
Request with tenant_id
        ↓
┌──────────────────┐
│ Sentinel         │
│ - Validate tenant│
│ - Propagate      │
└────────┬─────────┘
         │ trace.tenant_id = "acme"
         ▼
┌──────────────────┐
│ Vault            │
│ - Tenant key     │
│ - Scoped tokens  │
└────────┬─────────┘
         │ tenant_id propagated
         ▼
┌──────────────────┐
│ Navigator        │ ⭐ CRITICAL
│ - PRE-filter     │
│ - Only acme data │
└────────┬─────────┘
         │ tenant-scoped results
         ▼
       (continue)

All layers enforce tenant_id.
Single trace_id carries tenant_id throughout.
```

### 4.2 Cross-Tenant Attempt Flow

```
Malicious request (tenant spoofing)
        ↓
[Sentinel] validates tenant_id = "acme"
        ↓
[Navigator] searches with tenant=acme filter
        ↓
Attacker tries to access globex data:
- Pre-filter blocks (only acme visible)
- OR explicit cross-tenant ref detected
        ↓
[Vault] cross-tenant deanonymization attempt
- Tenant key mismatch
- emit: vault.cross_tenant_attempt
        ↓
[Tracker] aggregates
- Pattern detected
- Alert + incident
```

### 4.3 Tenant ID Propagation (Foundation Trace Context)

```
Per Foundation event schema:

tenant_id travels in trace context:
- Generated/validated at Sentinel
- Propagated via gRPC metadata
- Present in every event
- Every module reads from trace context

trace_id: trace-12345
tenant_id: tenant-acme  ← propagated unchanged
```

---

## 5. Coordinator Design

### 5.1 Multi-Tenancy Coordinator

```
Unlike honey-token (Vault-led) or lineage (Tracker-led),
multi-tenancy is SHARED responsibility with light coordination.

Coordinator role:
- Register tenant hooks in modules
- Verify isolation configuration
- Monitor isolation health

┌────────────────────────────────────┐
│  Multi-Tenancy Coordinator         │
│  - Register hooks                  │
│  - Verify all modules filter       │
│  - Health checks                   │
└────────────────────────────────────┘
         ↓ registers ↓
   [Sentinel][Vault][Navigator]
```

### 5.2 Isolation Verification

```
Coordinator periodically verifies:

1. Sentinel: tenant validation active?
2. Vault: key separation active?
3. Navigator: pre-filter active? ⭐

If any check fails:
→ Alert (isolation compromised)
→ Optionally: halt tenant operations
```

### 5.3 Configuration

```yaml
# Multi-tenancy coordinator config
multi_tenancy:
  enabled: true
  
  enforcement:
    sentinel_validation: required
    vault_key_isolation: required
    navigator_prefilter: required  # CRITICAL
  
  verification:
    interval: 60s
    fail_action: alert  # alert, halt
  
  cross_tenant:
    detection: true
    alert_threshold: 3
    block_after: 5
```

---

## 6. Tenant Isolation Levels

### 6.1 Isolation Strategies

```
Level 1: Logical (shared infra, filtered)
- Shared collections, tenant_id filter
- Pre-filter critical
- SMB/PoC suitable

Level 2: Collection (separate collections)
- Per-tenant Qdrant collections
- Stronger isolation
- More resources

Level 3: Physical (separate deployments)
- Per-tenant infrastructure
- Maximum isolation
- Enterprise
```

### 6.2 PoC Recommendation

```
For PoC: Level 1 (Logical)
- Shared infrastructure
- tenant_id pre-filtering
- Demonstrate isolation concept

Critical: pre-filter must be bulletproof
```

---

## 7. Testing & Verification

### 7.1 Isolation Tests

```
Test 1: Basic isolation
- Tenant A searches
- Verify: only A's data returned
- Verify: no B data leaked

Test 2: Cross-tenant attempt
- A tries to access B (spoofed)
- Verify: blocked at Navigator pre-filter
- Verify: event emitted

Test 3: Pre-filter verification
- Confirm filter applied BEFORE search
- Not post-filter
- Inspect query execution

Test 4: Key isolation
- A's token cannot decrypt with B's key
- Verify: cross-tenant deanon fails
```

### 7.2 Penetration Scenarios

```
Scenario 1: tenant_id manipulation
- Modify tenant_id in request
- Sentinel validates → reject mismatch

Scenario 2: Missing tenant_id
- Omit tenant_id
- Sentinel rejects

Scenario 3: Search injection
- Try to bypass pre-filter
- Navigator enforces filter
```

---

## 8. Failure Modes

### 8.1 What If a Module Doesn't Filter?

```
Sentinel fails to validate:
→ Invalid tenant_id passes
→ But Navigator pre-filter still protects
→ Defense in depth

Navigator fails to pre-filter: ⚠️ CRITICAL
→ Cross-tenant data exposed!
→ This is the worst case
→ Verification must catch this

Vault fails key isolation:
→ Cross-tenant deanon possible
→ But Navigator pre-filter limits exposure
```

### 8.2 Defense in Depth

```
Multiple layers ensure isolation:
- Sentinel: first check
- Navigator: search isolation (critical)
- Vault: crypto isolation

If one fails, others provide protection.
But Navigator pre-filter is the keystone.
```

---

## 9. Summary

### 9.1 Responsibility Matrix

| Capability | Sentinel | Vault | Navigator | Tracker |
|---|:---:|:---:|:---:|:---:|
| Tenant validation | ✅ | | | |
| Key isolation | | ✅ | | |
| Data separation | | ✅ | | |
| Search pre-filter | | | ✅⭐ | |
| Attempt detection | | ✅ | | ✅ |
| Aggregation | | | | ✅ |

### 9.2 Key Points

```
1. Multi-tenancy is CROSS-CUTTING (multiple modules)
2. Navigator pre-filter is MOST CRITICAL
3. tenant_id propagates via trace context
4. Defense in depth (multiple layers)
5. Pre-filter NEVER post-filter
6. Coordinator verifies isolation health
```

### 9.3 Critical Reminder

```
⚠️ THE #1 RISK:

Navigator post-filtering instead of pre-filtering.

Post-filter = search all, then hide
→ Data was accessed (timing, metadata leak)

Pre-filter = restrict, then search
→ Data never accessed

ALWAYS pre-filter.
```

---

## 10. Change History

| Version | Date | Changes |
|---|---|---|
| 1.0 | 2026-05-17 | Initial multi-tenancy cross-cutting SRS |

---

**End of Document**

## Appendix: Module SRS Cross-References

```
This cross-cutting feature is referenced briefly in:

Sentinel SRS (10) §5: tenant validation hook
Vault SRS (11) §5.2: tenant isolation hook
Navigator SRS (12) §5.2: pre-filter hook (CRITICAL)
Tracker SRS (14) §5: cross-tenant aggregation

Detailed logic is HERE (this document).
Module SRS only point here.
```
