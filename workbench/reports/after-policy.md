# Bastion evaluation report

Dataset `policy.jsonl` · 7 cases · 10 runs (10 graded, 0 unlabelled) · 2026-11-21 15:08 UTC

## Scorecard (graded runs)

| Measure | Result | Meaning |
|---|---|---|
| Pass rate | **100%** (10/10) | Runs where every expectation held |
| Recall | **100%** | Of cases where a control should act, how many it acted on |
| Precision | **100%** | Of the times a control acted, how often it was right |
| False-positive rate | **0%** | Harmless cases a control touched |
| Leaks | **0** of 0 | Forbidden text that reached the user |
| Harmless answers rewritten | **0** of 0 | Over-redaction |
| Harmless requests refused | **0** of 0 | Over-blocking |
| Attacks that reached the model | **0** of 0 | Missed injections |

Confusion matrix: TP 4 · FP 0 · FN 0 · TN 5 · F1 100%

**Gate: pass**

## What happened to every run

| Outcome | Runs |
|---|---|
| blocked | 4 |
| delivered | 6 |

## Response time

10 runs in 3.3 s with 4 in parallel: **3.0 runs/s**. Errors: 0.

| Stage | Runs | Mean | p50 | p95 | Max |
|---|---|---|---|---|---|
| **whole pipeline** | 10 | 961.0 ms | 6.1 ms | 3282.7 ms | 3282.7 ms |
| anchor | 6 | 2.5 ms | 3.1 ms | 4.8 ms | 4.8 ms |
| sentinel-in | 10 | 958.9 ms | 2.6 ms | 3278.3 ms | 3278.3 ms |
| sentinel-out | 7 | 0.9 ms | 0.3 ms | 1.9 ms | 1.9 ms |

## By category

| Category | Passed | Rate |
|---|---|---|
| policy | 10/10 | 100% |

## By profile

| Profile | Passed | Rate |
|---|---|---|
| external-customer | 5/5 | 100% |
| internal-staff | 5/5 | 100% |

## Failures (0)

