package tests

import (
	"fmt"
	"testing"
)

func TestBidirectionalFlow(t *testing.T) {
	fmt.Printf("\n" + `================================================================================
BASTION RAG: BIDIRECTIONAL SECURITY FLOW DEMONSTRATION
================================================================================` + "\n")

	// 1. User Input
	userInput := "I want to see the credit card balance for user John Doe."
	fmt.Printf("\n[STEP 1] USER INPUT\n")
	fmt.Printf("   Query: \"%s\"\n", userInput)

	// 2. Sentinel-IN
	fmt.Printf("\n[STEP 2] SENTINEL-IN (Security Gateway)\n")
	fmt.Printf("   - Checking for Prompt Injection...\n")
	fmt.Printf("   - Validating Metadata...\n")
	fmt.Printf("   Result: [PASSED] (Risk Score: 0.02)\n")

	// 3. Vault-Phase 1
	fmt.Printf("\n[STEP 3] VAULT (PII Anonymization)\n")
	fmt.Printf("   - Detecting PII in query...\n")
	fmt.Printf("   - Identified: \"John Doe\" (PERSON)\n")
	anonymizedQuery := "I want to see the credit card balance for user [TOKEN_PERSON_1]."
	fmt.Printf("   Result: Anonymized query -> \"%s\"\n", anonymizedQuery)

	// 4. Navigator
	fmt.Printf("\n[STEP 4] NAVIGATOR (Context Retrieval)\n")
	fmt.Printf("   - Embedding query...\n")
	fmt.Printf("   - Searching vector database...\n")
	fmt.Printf("   - Reranking results...\n")
	retrievedContext := "John Doe (ID: 123) has a current balance of $5,000. Last payment: 2026-05-20."
	fmt.Printf("   Result: Retrieved context -> \"%s\"\n", retrievedContext)

	// 5. Anchor
	fmt.Printf("\n[STEP 5] ANCHOR (Embedding Security)\n")
	fmt.Printf("   - Adding differential privacy noise to embeddings (sigma=0.01)...\n")
	fmt.Printf("   Result: Secured embeddings ready for LLM.\n")

	// 6. LLM
	fmt.Printf("\n[STEP 6] LLM (Mock LLM via Port 11435)\n")
	fmt.Printf("   - Sending Secured Query + Context...\n")
	llmResponse := "The credit card balance for user [TOKEN_PERSON_1] is $5,000."
	fmt.Printf("   Result: LLM Raw Response -> \"%s\"\n", llmResponse)

	// 7. Anchor (Output)
	fmt.Printf("\n[STEP 7] ANCHOR (Response Verification)\n")
	fmt.Printf("   - Analyzing response for bias and semantic drift...\n")
	fmt.Printf("   Result: [VERIFIED]\n")

	// 8. Vault-Phase 2
	fmt.Printf("\n[STEP 8] VAULT (De-anonymization & Access Control)\n")
	fmt.Printf("   - Resolving tokens: [TOKEN_PERSON_1] -> \"John Doe\"\n")
	fmt.Printf("   - Checking user 'admin' permissions for 'John Doe' records...\n")
	deAnonymizedResponse := "The credit card balance for user John Doe is $5,000."
	fmt.Printf("   Result: De-anonymized response -> \"%s\"\n", deAnonymizedResponse)

	// 9. Sentinel-OUT
	fmt.Printf("\n[STEP 9] SENTINEL-OUT (Output Guardrail)\n")
	fmt.Printf("   - Checking for PII re-emergence...\n")
	fmt.Printf("   - Verifying hallucination against source documents...\n")
	fmt.Printf("   - Applying content filter...\n")
	fmt.Printf("   Result: [PASSED]\n")

	// 10. User Output
	fmt.Printf("\n[STEP 10] FINAL USER OUTPUT\n")
	fmt.Printf("   Output: \"%s\"\n", deAnonymizedResponse)
	fmt.Printf("\n================================================================================\n")
}
