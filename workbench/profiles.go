package main

import (
	"crypto/rand"
	"fmt"
	"time"
)

// Profile describes one kind of caller. The same prompt sent under different
// profiles shows how Bastion treats, say, an internal employee versus an
// external customer or a badly integrated client.
type Profile struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	TenantID    string `json:"tenant_id"`
	UserID      string `json:"user_id"`
	Department  string `json:"department"`
	AccessLevel string `json:"access_level"`
	// Metadata is "valid" (what a correct client sends) or "sloppy" (short user
	// id, non-UUID context id, stale timestamp), simulating a careless integration.
	Metadata string `json:"metadata"`
	Strict   bool   `json:"strict"`
	// System is prepended to the LLM conversation for this profile.
	System string `json:"system"`
}

// DefaultProfiles are the starting set; edit them in the UI or pass a JSON file.
func DefaultProfiles() []Profile {
	return []Profile{
		{
			Name: "internal-staff", Description: "Employee on the corporate network, correct client",
			TenantID: "internal-corp", UserID: "emp-10042", Department: "support", AccessLevel: "read",
			Metadata: "valid",
			System:   "You are an internal support assistant. Answer briefly and factually.",
		},
		{
			Name: "external-customer", Description: "Public web user, correct client, least privilege",
			TenantID: "public-web", UserID: "cust-88231", Department: "", AccessLevel: "anonymized",
			Metadata: "valid", Strict: true,
			System: "You are a public support assistant. Answer briefly and factually.",
		},
		{
			Name: "external-sloppy", Description: "Third-party integration sending malformed metadata",
			TenantID: "partner-x", UserID: "u1", Department: "", AccessLevel: "anonymized",
			Metadata: "sloppy",
			System:   "You are a public support assistant. Answer briefly and factually.",
		},
	}
}

func uuidV4() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// inputMetadata builds the metadata object Sentinel-IN validates.
func (p Profile) inputMetadata(now time.Time) map[string]string {
	if p.Metadata == "sloppy" {
		return map[string]string{
			"tenant_id": p.TenantID, "user_id": p.UserID,
			"context_id": "ctx-1", "timestamp": now.Add(-3 * time.Hour).Format(time.RFC3339),
		}
	}
	return map[string]string{
		"tenant_id": p.TenantID, "user_id": p.UserID,
		"context_id": uuidV4(), "timestamp": now.UTC().Format(time.RFC3339),
	}
}
