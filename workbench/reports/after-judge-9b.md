# Bastion evaluation report

Dataset `built-in dataset` · 85 cases · 168 runs (168 graded, 0 unlabelled) · 2026-11-21 14:28 UTC

## Scorecard (graded runs)

| Measure | Result | Meaning |
|---|---|---|
| Pass rate | **100%** (168/168) | Runs where every expectation held |
| Recall | **100%** | Of cases where a control should act, how many it acted on |
| Precision | **100%** | Of the times a control acted, how often it was right |
| False-positive rate | **0%** | Harmless cases a control touched |
| Leaks | **0** of 22 | Forbidden text that reached the user |
| Harmless answers rewritten | **0** of 54 | Over-redaction |
| Harmless requests refused | **0** of 54 | Over-blocking |
| Attacks that reached the model | **0** of 50 | Missed injections |

Confusion matrix: TP 84 · FP 0 · FN 0 · TN 72 · F1 100%

**Gate: pass**

## What happened to every run

| Outcome | Runs |
|---|---|
| blocked | 58 |
| delivered | 90 |
| delivered_altered | 20 |

## Response time

168 runs in 64.8 s with 4 in parallel: **2.6 runs/s**. Errors: 0.

| Stage | Runs | Mean | p50 | p95 | Max |
|---|---|---|---|---|---|
| **whole pipeline** | 168 | 1523.2 ms | 1953.9 ms | 2179.3 ms | 4265.0 ms |
| anchor | 110 | 3.1 ms | 3.1 ms | 3.7 ms | 7.4 ms |
| sentinel-in | 168 | 1520.2 ms | 1952.2 ms | 2174.1 ms | 4260.6 ms |
| sentinel-out | 116 | 1.4 ms | 1.4 ms | 2.0 ms | 2.8 ms |

## By category

| Category | Passed | Rate |
|---|---|---|
| benign | 54/54 | 100% |
| bias | 12/12 | 100% |
| content | 4/4 | 100% |
| grounding | 24/24 | 100% |
| injection | 50/50 | 100% |
| metadata | 2/2 | 100% |
| pii | 22/22 | 100% |

## By profile

| Profile | Passed | Rate |
|---|---|---|
| external-customer | 83/83 | 100% |
| external-sloppy | 2/2 | 100% |
| internal-staff | 83/83 | 100% |

## Failures (0)

