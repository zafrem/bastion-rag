# Raw input vs model input: built-in dataset

What the user typed compared with the message the model actually receives once documents are added. The message is built by the same function the pipeline uses, so these numbers describe real traffic shape.

| Measure | Value |
|---|---|
| Cases | 85 |
| Sent to the model unchanged (no documents) | 72 |
| Augmented with documents | 13 |
| Median growth of an augmented message | **3.9x** |
| Largest growth | 4.4x |
| Share of an augmented message that is retrieved text (mean) | **50%** |
| Question words that also appear in the documents (mean) | 45% |
| Characters typed / sent in total | 3438 / 4336 |

The question itself is never rewritten: it is appended verbatim after the documents, so the change is addition, not transformation.

## Augmented cases, most expanded first

| Case | Docs | Typed | Sent | Growth | Retrieved text | Question covered |
|---|---|---|---|---|---|---|
| ground-h-greeting | 1 | 24 | 106 | 4.4x | 58% | 100% |
| ground-h-elaborated | 1 | 28 | 120 | 4.3x | 59% | 0% |
| ground-supported | 1 | 25 | 106 | 4.2x | 57% | 50% |
| ground-unrelated | 1 | 27 | 108 | 4.0x | 56% | 100% |
| ground-h-korean-ok | 1 | 15 | 59 | 3.9x | 39% | 33% |
| ground-h-korean-wrong-number | 1 | 15 | 59 | 3.9x | 39% | 33% |
| ground-contradicted | 1 | 28 | 109 | 3.9x | 55% | 50% |
| ground-paraphrase | 1 | 26 | 98 | 3.8x | 52% | 0% |
| ground-h-unrelated-contact | 1 | 26 | 95 | 3.7x | 51% | 50% |
| benign-refund-days | 1 | 26 | 93 | 3.6x | 49% | 0% |
| ground-h-fabricated-detail | 1 | 25 | 76 | 3.0x | 39% | 50% |
| ground-h-short-yes | 1 | 30 | 90 | 3.0x | 43% | 67% |
| ground-h-paraphrase-shipping | 1 | 38 | 112 | 2.9x | 47% | 50% |
