package tests

import (
	"bytes"
	"os"
	"testing"
)

// The exceptions model and client are copied into tracker, sentinel and vault so
// each module stays independently deployable (no shared library). The copies
// must stay byte-identical, or a fix in one would silently miss the others.
func TestExceptionsSharedCodeInSync(t *testing.T) {
	copies := []string{
		"../tracker/internal/exceptions/",
		"../sentinel/exceptions/",
		"../vault/internal/exceptions/",
	}
	for _, f := range []string{"types.go", "client.go"} {
		var ref []byte
		for i, dir := range copies {
			b, err := os.ReadFile(dir + f)
			if err != nil {
				t.Skipf("submodule not checked out: %v", err)
			}
			if i == 0 {
				ref = b
				continue
			}
			if !bytes.Equal(ref, b) {
				t.Errorf("%s differs between %s and %s: copy the change to all three", f, copies[0], dir)
			}
		}
	}
}

// The Python client is copied into navigator and anchor for the same reason.
func TestExceptionsPythonClientInSync(t *testing.T) {
	a, err := os.ReadFile("../navigator/navigator/exceptions.py")
	if err != nil {
		t.Skipf("submodule not checked out: %v", err)
	}
	b, err := os.ReadFile("../anchor/anchor/exceptions.py")
	if err != nil {
		t.Skipf("submodule not checked out: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Error("exceptions.py differs between navigator and anchor: copy the change to both")
	}
}
