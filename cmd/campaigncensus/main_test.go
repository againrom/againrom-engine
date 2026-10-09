package main

import (
	"io"
	"strings"
	"testing"
)

func TestRunNamesTheMissingRoot(t *testing.T) {
	t.Setenv("AGAINROM_ASSETS", "")
	err := run(nil, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "AGAINROM_ASSETS") {
		t.Fatalf("run without a root = %v, want the missing-root error", err)
	}
}
