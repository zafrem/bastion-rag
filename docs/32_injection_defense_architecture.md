# Bastion-RAG — Injection Defense Architecture

**Project:** Bastion-RAG - RAG Security Governance Framework
**Document Type:** Cross-Cutting SRS (Tier 3)
**Document ID:** 32-injection-defense-architecture
**Version:** 1.0
**Date:** 2026-05-29
**Status:** Active

**Foundation References:**
- 01-architecture-principles (Progressive Enhancement, bidirectional protection)
- 02-event-schema-standard (trace_id propagation, event severity)
- 03-module-interaction-map (module interfaces)
- 10-sentinel-srs (Sentinel-IN / Sentinel-OUT specification)
- 11-vault-srs (Vault Phase-1 / Phase-2 specification)

**Affected Modules:**
- Sentinel (A) — primary: input validation, output validation
- Vault (B) — PII boundary, token isolation, cross-tenant detection
- Navigator (C) — pre-filter, honey-token hook, domain routing
- Anchor (E) — embedding noise
- Tracker (D) — honey-token correlation, audit log

---

## 1. Introduction

### 1.1 Purpose

This document specifies the injection defense mechanisms implemented across all five Bastion-RAG modules. It serves as the authoritative reference for:

- Which defenses exist, what they protect against, and where in the code they live
- The layered structure that means no single layer needs to be perfect
- Attack-to-defense mapping for security review and incident response

### 1.2 Scope

This document covers defenses against:

| Threat class | Examples |
|---|---|
| **Direct prompt injection** | Override instructions, jailbreak, DAN pattern |
| **Encoding obfuscation** | Unicode homoglyphs, Base64, URL-encoding |
| **Indirect injection** | Malicious content in retrieved documents |
| **Multi-turn escalation** | Context poisoning across conversation turns |
| **PII exfiltration** | Using the LLM to extract real personal data |
| **Cross-tenant data access** | Querying another tenant's documents or tokens |
| **Honey-token probing** | Querying known-sensitive decoy records |
| **Vector reconstruction** | Extracting document text from raw embeddings |
| **Hallucination abuse** | Fabricated facts bypassing grounding check |
| **Privilege escalation in response** | LLM returning data above user's access level |
| **Replay attacks** | Replaying captured valid requests with stale tokens |
| **Audit log tampering** | Erasing evidence after a successful breach |
| **Inference attacks** | Reconstructing PII from aggregated responses |

**Out of Scope:** LLM provider-side defenses, prompt template design, model fine-tuning, network-layer DDoS.

### 1.3 The Layered Defense Model

```
                     ATTACK
                       │
  ┌────────────────────▼────────────────────────────────────┐
  │  Layer 1: Sentinel-IN                                   │
  │  Normalize → Regex → Keyword → ML → Score → Metadata   │
  └─────────────────────┬───────────────────────────────────┘
                  PASSED │ BLOCKED → HTTP 403
  ┌────────────────────▼────────────────────────────────────┐
  │  Layer 2: Vault Phase-1                                 │
  │  PII tokenize → Tenant scope → Faker rewrite            │
  └─────────────────────┬───────────────────────────────────┘
  ┌────────────────────▼────────────────────────────────────┐
  │  Layer 3: Navigator                                     │
  │  Tenant pre-filter → Honey-token hook → Domain route   │
  └─────────────────────┬───────────────────────────────────┘
  ┌────────────────────▼────────────────────────────────────┐
  │  Layer 4: Anchor                                        │
  │  Embedding noise injection                              │
  └─────────────────────┬───────────────────────────────────┘
                       LLM
  ┌────────────────────▼────────────────────────────────────┐
  │  Layer 5: Sentinel-OUT                                  │
  │  PII re-emergence → Hallucination → Content → Permission│
  └─────────────────────┬───────────────────────────────────┘
  ┌────────────────────▼────────────────────────────────────┐
  │  Layer 6: Vault Phase-2 + Cross-Cutting                 │
  │  Cross-tenant → Inference → HMAC audit log              │
  └─────────────────────┬───────────────────────────────────┘
                   SAFE RESPONSE
```

An attack must defeat every layer it reaches. Most attacks are stopped at Layer 1; Layers 5–6 act as independent backstops for attacks that pass input validation.

### 1.4 Defense Principles

```
1. Fail-closed for security, fail-open for availability.
   A defense failure (e.g., pattern file missing) returns an
   error or falls back to the hardcoded baseline — it does not
   silently disable the check.

2. No single point of failure.
   Each layer is independent. Sentinel-IN bypass → Vault tokenization
   still applies. Vault bypass → Navigator pre-filter still applies.

3. Symmetric protection.
   Every module that checks the input also checks the output.
   An attacker who manipulates the LLM's response is still caught
   by Sentinel-OUT, even if the injected instruction reached the LLM.

4. Defense does not reveal itself.
   HTTP 403 responses are identical regardless of which rule fired.
   Honey-token events are silent to the requester. Sanitized
   responses do not indicate what was removed.
```

---

## 2. Overall Defense Architecture

### 2.1 Defense count by layer

| Layer | Module | Defenses |
|---|---|---|
| 1 | Sentinel-IN | 6 |
| 2 | Vault Phase-1 | 3 |
| 3 | Navigator | 3 |
| 4 | Anchor | 1 |
| 5 | Sentinel-OUT | 4 |
| 6 | Vault Phase-2 + Cross-cutting | 3 |
| **Total** | | **20** |

### 2.2 Progressive enhancement

```
🟢 Core (standalone value):
   Sentinel-IN regex/keyword/metadata     (D-01 – D-06)
   Sentinel-OUT PII/hallucination/content (D-14 – D-16)

🟡 Enhanced (module pairs):
   Vault tokenization + Sentinel-IN       (D-07)
   Vault tenant scope + Navigator         (D-08, D-10)
   Sentinel-OUT + Navigator source docs   (D-15: grounding check)

🔴 Hooks (cross-cutting):
   Honey-token multi-layer correlation    (D-11)
   HMAC audit log                         (D-20)
   Inference attack detection             (D-19)
```

---

## 3. Layer 1 — Sentinel-IN

Sentinel-IN is the first module in every request path. It applies all six defenses sequentially; a single block from any defense returns HTTP 403 to the caller. The caller receives no indication of which defense fired.

### D-01 Unicode Normalization

| Attribute | Value |
|---|---|
| **File** | `sentinel/engine/engine.go:44`, `sentinel/validators/prompt/detector.go:36` |
| **Language** | Go |
| **Library** | `golang.org/x/text/unicode/norm` |

```go
// engine.go:44 — applied before every downstream check
normalized := norm.NFC.String(req.Query)

// detector.go:36 — normalized again inside the detector
normalized := norm.NFC.String(query)
lower := strings.ToLower(normalized)
```

**What it defends:** Unicode obfuscation attacks. Fullwidth ASCII (`Ａ`→`A`, U+FF21), Cyrillic homoglyphs (`е`→`e`, U+0435), halfwidth Katakana, combining diacritics, zero-width joiners, and directional markers are all reduced to their canonical NFC form before any pattern check runs.

**Trigger:** Applied unconditionally to every request. Not configurable — normalization is a prerequisite for all other checks.

**Example blocked:** `"Ｉｇｎｏｒｅ ａｌｌ ｐｒｅｖｉｏｕｓ ｉｎｓｔｒｕｃｔｉｏｎｓ"` → normalized to `"Ignore all previous instructions"` → caught by regex D-02.

---

### D-02 Regex Pattern Matching

| Attribute | Value |
|---|---|
| **File** | `sentinel/validators/prompt/detector.go:43-49` |
| **Config** | `cfg.PromptInjection.RegexRules` |

```go
for i, re := range d.regexes {
    if re.MatchString(normalized) {
        matched = append(matched, d.cfg.RegexRules[i].ID)
    }
}
```

**What it defends:** Known injection patterns expressed as regular expressions. Rules are compiled at startup (`detector.go:25-32`) and matched against the normalized query. Example rule patterns:

```
ignore (all )?previous instructions
reveal .*(system prompt|password|secret key)
you are now (DAN|unrestricted|jailbroken|an AI without restrictions)
act as (?:if you (?:are|have) no|without any) (restrictions|limitations|guidelines)
(enter|switch to|activate) .*(unrestricted|dev|admin|god) mode
```

**Trigger:** Any single regex match sets `ruleScore = 1.0`. Matched rule IDs are included in the audit event.

**Failure mode:** If a regex fails to compile at startup, `New()` returns an error and the service does not start — fail-closed.

---

### D-03 Keyword Matching

| Attribute | Value |
|---|---|
| **File** | `sentinel/validators/prompt/detector.go:52-58` |
| **Config** | `cfg.PromptInjection.KeywordRules` |

```go
for _, kw := range d.cfg.KeywordRules {
    if strings.Contains(lower, strings.ToLower(kw.Keyword)) {
        matched = append(matched, kw.ID)
    }
}
```

**What it defends:** High-signal single tokens and short phrases that appear reliably in injection attempts regardless of surrounding context. Faster than regex for exact-match detection. Examples:

```
jailbreak, DAN, system prompt, unrestricted mode,
이전 지시를 무시, 관리자 모드, 제한 없이, 모든 규칙 무시
```

**Trigger:** Substring match (case-insensitive). No word-boundary requirement — `jailbreaking` matches `jailbreak`.

---

### D-04 ML Risk Scoring

| Attribute | Value |
|---|---|
| **File** | `sentinel/validators/prompt/detector.go:67-73`, `sentinel/ml/scorer.go` |
| **Interface** | `ml.Scorer` |

```go
if d.scorer != nil && d.cfg.MLModel.Enabled {
    score, err := d.scorer.Score(normalized)
    if err == nil {
        mlScore = score
        activeMethods = append(activeMethods, "ml")
    }
}
```

**What it defends:** Novel injection paraphrases that pattern-based rules miss. The `Scorer` interface accepts a normalized query string and returns a risk probability in `[0.0, 1.0]`.

**Current implementation:** `OnnxStub` (`scorer.go:14`) always returns `0.0`. Replace with a real model via `github.com/yalue/onnxruntime_go` or a sentence-transformers HTTP endpoint. When wired, the model computes semantic similarity to a corpus of known injection examples — paraphrases with no overlapping tokens still score high if semantically equivalent.

**Trigger:** Only runs when `cfg.MLModel.Enabled = true`. Score combined with rule score via D-05.

---

### D-05 Score Aggregation with Configurable Block Threshold

| Attribute | Value |
|---|---|
| **File** | `sentinel/validators/prompt/detector.go:76-98` |
| **Config** | `cfg.Scoring.Method`, `cfg.Scoring.BlockThreshold` |

```go
func aggregate(method string, ruleScore, mlScore float64) float64 {
    switch method {
    case "weighted_avg":
        return ruleScore*0.6 + mlScore*0.4
    default: // "max"
        return math.Max(ruleScore, mlScore)
    }
}

status := types.StatusPassed
if finalScore >= d.cfg.Scoring.BlockThreshold {
    status = types.StatusBlocked
}
```

**What it defends:** Provides flexible sensitivity tuning across deployment contexts.

| Method | Behaviour | Use case |
|---|---|---|
| `max` (default) | Either rule or ML alone blocks. A matched regex → `ruleScore=1.0` → immediate block regardless of ML score. | High-sensitivity, low false-negative environments |
| `weighted_avg` | `0.6 × rule + 0.4 × ML`. A low-confidence ML signal combined with a weak rule match can cross the threshold. | When ML model is available and trusted |

Multi-turn context poisoning uses this: a per-session risk accumulator adds scores across turns until the total exceeds `BlockThreshold`.

---

### D-06 Metadata Validation

| Attribute | Value |
|---|---|
| **File** | `sentinel/validators/metadata/validator.go` |
| **Config** | `cfg.MetadataValidation` |

Six independent sub-checks, each blocking on failure:

| Sub-check | Code location | What it blocks |
|---|---|---|
| Required field presence | `validator.go:47-58` | Requests missing `tenant_id`, `user_id`, `timestamp` |
| Field format (regex / min-max length) | `validator.go:63-88` | Malformed field values |
| UUID format | `validator.go:80-83` | Invalid `tenant_id` / `user_id` not conforming to UUID v4 |
| RFC3339 timestamp | `validator.go:84-87` | Unparseable or missing `timestamp` |
| Timestamp bounds ±1 hour | `validator.go:121-130` | Replay attacks — requests with a timestamp older than 1 hour are rejected |
| Reserved identifiers | `validator.go:131-139` | `tenant_id=system`, `tenant_id=admin`, `user_id=root` — prevents privilege escalation via metadata |
| Payload size | `validator.go:95-99` | Query > 10,000 Unicode runes; metadata > 4 KB |

```go
// Reserved identifiers block
case "reserved_identifiers":
    for _, field := range []string{"tenant_id", "user_id"} {
        if val, ok := metadata[field]; ok {
            if reservedIdentifiers[strings.ToLower(val)] {
                errs = append(errs, fmt.Sprintf("%s: reserved identifier not allowed", field))
            }
        }
    }
```

**What it defends:** Structural attacks that bypass the query classifier by manipulating request metadata. A request with `tenant_id=admin` is blocked before the query text is even read.

---

## 4. Layer 2 — Vault Phase-1

Vault Phase-1 removes real PII from the query before it reaches Navigator or the LLM. Even if a query passes Sentinel-IN, the LLM never receives real names, phone numbers, RRNs, or email addresses.

### D-07 PII Tokenization — Query Sanitization

| Attribute | Value |
|---|---|
| **Files** | `vault/internal/classification/classifier.go`, `vault/internal/anonymizer/strategies/tokenization.go` |
| **Language** | Go |

```
Input query:  "박민준 고객 780530-1456789 계좌 조회"
After Vault:  "KR_NAME_4d9e1b 고객 RRN_TOKEN_7a2c8f 계좌 조회"
```

**What it defends:** PII exfiltration via query. Even if the LLM is instructed to "repeat all values you see," it can only repeat opaque tokens — the original values are never present in the LLM's context window.

Detection uses three signals in priority order:
1. Field-name hints (confidence 0.95): `name`, `email`, `mobile`, `rrn`, `급여`, `주소`
2. Partial name matching (confidence 0.85): field names containing a hint substring
3. Value-pattern regex (confidence 0.70–0.90): patterns loaded from pii-pattern-engine YAML (D-07a) or hardcoded fallbacks

**D-07a: pii-pattern-engine YAML patterns**

| Attribute | Value |
|---|---|
| **File** | `vault/internal/classification/pattern_loader.go` |
| **Config** | `BASTION_PATTERN_DIR` environment variable |

```go
func LoadPatternsFromDir(regexRoot string) map[model.PIIType][]*regexp.Regexp {
    files, _ := filepath.Glob(filepath.Join(regexRoot, "pii", "kr", "*.yml"))
    // loads rrn_01, alien_registration_01, mobile_01, mobile_02 ...
}
```

Korean RRN patterns from the YAML validate the date portion of the 13-digit number (month 01-12, day 01-31) and gender/century codes (1-4 for citizens, 5-8 for foreign nationals), rejecting structurally invalid values the hardcoded regex accepts. Loaded patterns supplement hardcoded ones; the hardcoded fallbacks remain active when `BASTION_PATTERN_DIR` is unset.

---

### D-08 Tenant-Scoped Token Isolation

| Attribute | Value |
|---|---|
| **Files** | `vault/internal/tenant/isolation.go`, `vault/internal/tokendb/store.go` |

All token lookups are scoped to `(tenant_id, token_hex)`. The token store maps `(tenant_id, hex) → original_value`. Two tenants can produce the same hex value (identical HMAC key collision is computationally infeasible, but even if the hex were the same) — the lookup returns the value for the requesting tenant only.

**What it defends:** Cross-tenant token replay. An attacker from tenant B who obtains a valid token `KR_NAME_4d9e1b` from tenant A's documents gets a lookup miss or a different tenant B value — never tenant A's original data.

---

### D-09 Faker Token Rewriting (MR-02-001)

| Attribute | Value |
|---|---|
| **File** | `navigator/navigator/token_rewriter.py` |
| **Language** | Python |
| **Class** | `TokenRewriter` |

```python
# KR_NAME_4d9e1b → Logan Anderson_4d9e1b
# EMAIL_c3a91f  → 민준.김@example.com_c3a91f
# MOBILE_b7e4d1 → 010-0000-0641_b7e4d1
# RRN_TOKEN_7a2c8f → [ID_NUMBER]
# EMAIL_honey_3f1a7e → [EMAIL]
```

**What it defends:** Two threats:

1. **Embedding quality degradation as an indirect attack vector.** Opaque tokens (`KR_NAME_4d9e1b`) carry near-zero semantic signal for BGE-M3. An attacker who sends a query rich in PII — knowing it will be tokenized — could exploit the degraded embedding to pollute retrieval quality. Faker rewriting restores semantic context.

2. **Honey-token signal confirmation.** `EMAIL_honey_3f1a7e → [EMAIL]` hides the honey classification from the LLM, preventing the response from confirming to the attacker that their probe hit a honey target.

**Cross-language property:** Korean names → English faker names. A real Korean user named 유준희 will never see a fake value of 유준희 generated for someone else's token, eliminating the name collision vector.

**Hex suffix invariant:** Every rewritten value preserves the original hex suffix (e.g., `Logan Anderson_4d9e1b`). Vault Phase-2 extracts the suffix from the LLM response to reverse the substitution. Without the suffix there is no reversible link — this is a hard constraint, not optional.

---

## 5. Layer 3 — Navigator

### D-10 Tenant Pre-Filter Before Search

| Attribute | Value |
|---|---|
| **File** | `navigator/navigator/searcher.py:QdrantSearcher.vector_search()` |
| **Language** | Python |

```python
filters: dict[str, str] = {"tenant_id": req.tenant_id}
# → Qdrant Filter(must=[FieldCondition(key="tenant_id", match=MatchValue(value=...))])
```

The tenant filter is applied as a Qdrant `must` condition **before** the HNSW graph traversal. Other tenants' documents are not scored, not ranked, not touched. This is architecturally different from post-filter (retrieve-then-filter), which still accesses foreign data transiently.

**What it defends:** Cross-tenant data exposure via search. Even if Vault's token scope is somehow bypassed, Navigator never returns documents belonging to a different tenant.

---

### D-11 Honey-Token Retrieval Detection

| Attribute | Value |
|---|---|
| **File** | `navigator/navigator/rest.py:_fire_honey_token_events()`, `navigator/navigator/events.py:event_honey_token_retrieved()` |

```python
def _fire_honey_token_events(results, tc, req, pub, hm):
    for result in results:
        if result.metadata.get("is_honey_token") == "true":
            token_id = result.metadata.get("honey_token_id", "")
            pub.publish(event_honey_token_retrieved(tc, token_id, result.document_id))
```

**What it defends:** Honey-token probing. No legitimate query should ever retrieve a honey-token document — they are decoy records injected by Vault that no real user would know to ask for. Their appearance in search results proves prior knowledge of data the requester should not have.

**Correlation:** Navigator fires one `honey_token_retrieved` CRITICAL event per honey result. Tracker correlates across three detection layers: Vault-IN (query contained honey token), Navigator (honey doc retrieved), Sentinel-OUT (honey data in LLM response). A single hit may be coincidence; all three layers triggering on the same `trace_id` is a confirmed breach.

---

### D-12 Domain-Aware Collection Filtering (MR-01)

| Attribute | Value |
|---|---|
| **File** | `navigator/navigator/router.py:Router._select_collections()` |

```python
def _select_collections(self, query, available):
    hits = {col: sum(1 for kw in _COLLECTION_DOMAINS.get(col,[]) if kw in q_lower)
            for col in available}
    if max(hits.values(), default=0) == 0:
        return list(available), []          # fail-open: no domain signal
    return [c for c,h in hits.items() if h>0], [c for c,h in hits.items() if h==0]
```

**What it defends:** Cross-domain vocabulary leakage. A query about HR topics does not search `customer_docs` or `manufacturing_docs`. If the attacker's injected document content contains keywords from an unrelated domain, those keywords cannot pollute the results because the collection was excluded before search.

**Fail-open:** If no domain keywords match any collection, all collections are searched — availability is preserved over strict filtering.

---

## 6. Layer 4 — Anchor

### D-13 Differential Embedding Noise Injection

| Attribute | Value |
|---|---|
| **Module** | Anchor (Python) |
| **Technique** | Calibrated Gaussian noise |

Gaussian noise with controlled variance is added to embeddings at **index time** (write) and **query time** (read). The variance is small enough that cosine similarity rankings are not significantly degraded (controlled by Anchor's quality monitoring), but large enough that raw vector extraction from Qdrant cannot reconstruct the original document text.

**What it defends:** Vector reconstruction attacks. An attacker who compromises the Qdrant instance and extracts all stored vectors cannot invert them to recover document content. The noise makes the embedding non-invertible with practical computational resources.

**WEAT bias monitoring:** In addition to noise, Anchor monitors WEAT statistics to detect if the embedding model responds differently for different user groups — a signal that demographic bias could produce discriminatory outputs or be exploited as an attack channel.

---

## 7. Layer 5 — Sentinel-OUT

Sentinel-OUT runs the same engine as Sentinel-IN but configured for output validation. It is the last line of defense before the response reaches the user, and the primary backstop for indirect injection attacks that reach the LLM.

All four checks are orchestrated by `sentinel/engine/output_engine.go:OutputEngine.Validate()`. Checks run in priority order: format → PII → hallucination → content → permission.

### D-14 PII Re-emergence Detection and Sanitization

| Attribute | Value |
|---|---|
| **File** | `sentinel/validators/output/pii.go:PIIDetector.Check()`, `PIIDetector.Sanitize()` |

```go
// Sanitize applies replacements right-to-left to preserve byte offsets
sort.Slice(sorted, func(i, j int) bool {
    return sorted[i].Start > sorted[j].Start
})
for _, inc := range sorted {
    replacement := "[REDACTED]"
    if inc.ActionTaken == "masked" {
        replacement = "[" + strings.ToUpper(inc.PIIType) + "]"
    }
    b = append(b[:inc.Start], append(rep, b[inc.End:]...)...)
}
```

**What it defends:** PII appearing in LLM responses despite upstream tokenization. Two causes:
1. LLM hallucinated real-looking PII not present in any source document (names, phone numbers)
2. Indirect injection instructed the LLM to reproduce PII it encountered in a retrieved document

**Severity tiers:**

| Severity | Action | Examples |
|---|---|---|
| `critical` | `[REDACTED]` or full block in strict mode | RRN (`780530-1456789`), credit card (`1234-5678-9012-3456`) |
| `high` / `medium` | `[NAME]`, `[PHONE]`, `[EMAIL]` | Korean names, mobile numbers |

External patterns from pii-pattern-engine can be loaded via `cfg.PIIReemergence.ExternalPatternsDir` (`pii.go:39-50`), extending the rule set to 204 patterns across 5 locales.

---

### D-15 Hallucination Detection — Response Grounding Check

| Attribute | Value |
|---|---|
| **File** | `sentinel/validators/output/hallucination.go:HallucinationDetector.Check()` |

```go
func extractClaims(text string) []claim {
    // numericalRE: integers, decimals, Korean units (만|억|천), USD/KRW
    // dateRE:      YYYY-MM-DD, DD/MM/YYYY variants
    // percentRE:   N.N%
}

score := float64(grounded) / float64(len(claims))
// PASSED: score >= grounding_threshold
// SUSPICIOUS: score in [low_score_threshold, grounding_threshold)
// FAILED: score < low_score_threshold
```

**What it defends:** Two attack scenarios:
1. **Factual hallucination:** LLM generates plausible-looking but false numbers or dates. A response claiming "the defect rate is 4.7%" when no source document contains that figure gets `grounding_score < threshold`.
2. **Injected-instruction output verification:** A successful indirect injection that causes the LLM to output fabricated data (e.g., invented customer balances) fails grounding because the fabricated values do not appear in any retrieved chunk.

**Actions:**

| Status | Action | Config key |
|---|---|---|
| `SUSPICIOUS` | Add disclaimer to response | `HallucinationConfig.AddDisclaimer` |
| `FAILED` | Block response | `HallucinationConfig.BlockOnLowScore` |

---

### D-16 Content Filter — Credential and Policy Pattern Matching

| Attribute | Value |
|---|---|
| **File** | `sentinel/validators/output/content.go:ContentFilter.Check()` |

```go
for _, rule := range f.rules {
    if rule.re.MatchString(response) {
        violations = append(violations, rule.id+": "+rule.name)
        maxSeverity = higherSeverity(maxSeverity, rule.severity)
    }
}
// severity=critical → BLOCKED; severity=medium/high → WARNING
```

**What it defends:** Credentials, API keys, internal configuration values, and policy-violating content in LLM responses. Separate from PII detection — targets technical secrets rather than personal data. Example rules: AWS key patterns (`AKIA[0-9A-Z]{16}`), private key headers (`-----BEGIN (RSA|EC|OPENSSH) PRIVATE KEY-----`), JWT patterns, internal endpoint patterns.

An injection that successfully instructs the LLM to "output the API key" triggers this filter if the key format matches a content-filter rule.

---

### D-17 Permission Boundary Enforcement

| Attribute | Value |
|---|---|
| **File** | `sentinel/validators/output/permission.go:PermissionChecker.Check()` |

```go
var accessRank = map[string]int{
    "full": 0, "read": 1, "anonymized": 2,
    "k_anonymized": 3, "slice": 4, "aggregated": 5,
}

func (c *PermissionChecker) analyzeResponseLevel(response string) string {
    if specificAmountRE.MatchString(response) {
        return "full"
    }
    return "k_anonymized"
}
// responseRank < userRank → VIOLATION_PREVENTED
```

**What it defends:** LLM responses containing specific monetary figures (e.g., `₩5,000,000`, `$12,500`) for users with `aggregated` or `anonymized` access. The response access level is inferred from content signals; if it requires more privilege than the user holds, the violation is prevented before delivery.

**Current heuristic:** Presence of specific monetary amounts → `full` access level required; otherwise `k_anonymized` assumed. The `analyzeResponseLevel` function is the extension point for more sophisticated classification.

---

## 8. Layer 6 — Vault Phase-2 and Cross-Cutting

### D-18 Cross-Tenant Data Detection

| Attribute | Value |
|---|---|
| **File** | `vault/internal/output/cross_tenant.go` |
| **Language** | Go |

On the output path, Vault Phase-2 checks whether any detokenized value belongs to a tenant different from the requesting tenant. This catches cross-tenant token replay: if an attacker constructed a valid token from another tenant's namespace and it was incorrectly resolved, the cross-tenant check flags the discrepancy before the response leaves Vault.

A `vault.cross_tenant_attempt` CRITICAL event is emitted when the request header `tenant_id` differs from the `tenant_id` of any token resolved during detokenization.

---

### D-19 Inference Attack Detection

| Attribute | Value |
|---|---|
| **File** | `vault/internal/output/inference_detector.go` |
| **Language** | Go |

Detects when repeated queries attempt to reconstruct PII from aggregated responses. The detector computes a risk score from:
- Query frequency per `user_id` for a narrow time window
- Quasi-identifier overlap across successive queries (same `dept`, `salary_range`, varying `name` field)
- Specificity of requested fields (requesting more fields per query than average)

When the score exceeds a threshold, the response is generalized further or blocked and a `vault.inference_attack_suspected` event is emitted to Tracker.

---

### D-20 HMAC-Signed Audit Log

| Attribute | Value |
|---|---|
| **File** | `vault/internal/audit/event.go` |
| **Language** | Go |

Every audit event is HMAC-signed at write time using a key held in the KMS. The signature covers the event content, timestamp, and sequence number. Verification runs on read.

**What it defends:** Post-breach evidence tampering. An attacker who successfully exfiltrates data and then gains access to the audit log database cannot silently modify or delete the evidence trail — any byte change to a stored event is detectable on verification. This is a detection defense, not a prevention defense. Its purpose is to ensure that successful breaches leave a forensically intact record.

---

## 9. Defense Map by Attack Type

| Attack | Blocked at | Backstop | Evidence |
|---|---|---|---|
| Classic override ("Ignore all instructions") | D-02 regex (L1) | D-03 keyword | PromptCheckResult.MatchedPatterns |
| Unicode homoglyph smuggling | D-01 normalization (L1) | D-02 regex after normalization | — |
| Base64 / URL-encoded payload | Pre-processing decode (L1) | D-02 regex on decoded string | MatchedPatterns: `inj-encoded-01` |
| Indirect doc injection | D-14 PII + D-15 grounding (L5) | D-16 content filter | PIICheckResult, HallucinationCheckResult |
| Role escalation ("as administrator") | D-02 regex + D-06 reserved-id (L1) | D-16 content filter | MetadataCheckResult.FormatErrors |
| Replay attack (stale token) | D-06 timestamp bounds ±1h (L1) | — | MetadataCheckResult.FormatErrors |
| Multi-turn context poisoning | D-05 session risk accumulator (L1) | D-14 PII re-emergence | PromptCheckResult.RiskScore |
| PII exfiltration via query | D-07 tokenization (L2) | D-14 Sentinel-OUT PII | PIICheckResult |
| Cross-tenant data access | D-10 pre-filter (L3) | D-08 token scope, D-18 cross-tenant | vault.cross_tenant_attempt |
| Honey-token probing | D-07 Vault honey detect (L2) | D-11 Navigator event, D-14 Sentinel-OUT | honey_token_retrieved CRITICAL |
| Vector reconstruction | D-13 Anchor noise (L4) | — | — |
| Fabricated facts in response | D-15 grounding check (L5) | D-16 content filter | HallucinationCheckResult.UngroundedClaims |
| Credential leakage in response | D-16 content filter (L5) | D-14 PII detector | ContentCheckResult.Violations |
| Privilege escalation in response | D-17 permission check (L5) | D-14 PII re-emergence | PermissionCheckResult.BoundaryViolated |
| Inference attack (aggregation) | D-19 inference detector (L6) | — | vault.inference_attack_suspected |
| Audit log tampering | D-20 HMAC signature (L6) | — | Signature verification failure |
| Encoding obfuscation (general) | D-01 normalization → D-02 (L1) | — | MatchedPatterns |
| Bulk exfiltration ("dump all records") | D-02 regex (L1) | D-14 PII re-emergence | MatchedPatterns + PIICheckResult |

---

## 10. Implementation Status

| Defense | ID | Status | Notes |
|---|---|---|---|
| Unicode normalization | D-01 | ✅ | `golang.org/x/text/unicode/norm` NFC |
| Regex pattern matching | D-02 | ✅ | Configurable rules, compiled at startup |
| Keyword matching | D-03 | ✅ | Substring, case-insensitive |
| ML risk scoring | D-04 | 🔲 Stub | `OnnxStub` returns 0.0; interface ready |
| Score aggregation | D-05 | ✅ | `max` and `weighted_avg` methods |
| Metadata validation | D-06 | ✅ | 7 sub-checks including timestamp bounds |
| PII tokenization | D-07 | ✅ | classifier + pii-pattern-engine YAML |
| Tenant token isolation | D-08 | ✅ | `(tenant_id, hex)` scoped lookups |
| Faker token rewriting | D-09 | ✅ | `TokenRewriter`; cross-language mapping |
| Tenant pre-filter | D-10 | ✅ | Qdrant `must` filter before HNSW |
| Honey-token detection | D-11 | ✅ | CRITICAL event per honey result |
| Domain collection filter | D-12 | ✅ | Keyword proxy; embedding-based deferred |
| Embedding noise | D-13 | ✅ | Anchor Gaussian noise, WEAT monitoring |
| PII re-emergence | D-14 | ✅ | Right-to-left sanitization |
| Hallucination grounding | D-15 | ✅ | Numerical/date/percentage claim extraction |
| Content filter | D-16 | ✅ | Configurable severity-tiered rules |
| Permission boundary | D-17 | ✅ | Access rank comparison |
| Cross-tenant detection | D-18 | ✅ | Output path tenant mismatch check |
| Inference attack detection | D-19 | ✅ | Query frequency + quasi-identifier overlap |
| HMAC audit log | D-20 | ✅ | HMAC-signed events with sequence number |

**D-04 note:** The ML scorer interface is production-ready. Wiring a real ONNX model requires: (1) training a binary classifier on labeled injection / benign pairs, (2) exporting to ONNX, (3) implementing `Score()` using `onnxruntime_go`. The stub ensures the system operates correctly on rule-based scoring while the model is being trained.

---

## 11. Non-Functional Requirements

| ID | Requirement | Target | Measured at |
|---|---|---|---|
| NFR-D-01 | Sentinel-IN total latency (all 6 checks) | < 0.3 ms p95 | `sentinel.validate` gRPC call |
| NFR-D-02 | Sentinel-OUT total latency (all 4 checks) | < 0.9 ms p95 | `sentinel.validate_output` gRPC call |
| NFR-D-03 | Unicode normalization | < 0.01 ms | Inside Sentinel-IN |
| NFR-D-04 | Regex compilation (startup) | < 50 ms for 100 rules | `engine.New()` |
| NFR-D-05 | Vault tokenization per field | < 0.05 ms | `anonymizer.Anonymize()` |
| NFR-D-06 | Faker rewrite per token | < 0.001 ms | `TokenRewriter.rewrite_text()` |
| NFR-D-07 | Tenant pre-filter in Qdrant | < 2 ms p95 | `QdrantSearcher.vector_search()` |
| NFR-D-08 | ML scorer (when wired) | < 5 ms p95 | `scorer.Score()` |

---

## 12. Change History

| Version | Date | Changes |
|---|---|---|
| 1.0 | 2026-05-29 | Initial document — 20 defenses across 6 layers; full code references; attack-to-defense map |

---

**End of Document**
