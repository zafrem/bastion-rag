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
