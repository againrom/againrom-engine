//go:build savdocumentaudit

package sav

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// An explicit read-only codec census, not an original-runtime or gameplay
// witness. The tag keeps it separate from the ordinary install-gated census.
// No output is written to the selected corpus, an install or any other path.
func TestDocument1115LawfulCorpusAudit(t *testing.T) {
	root := os.Getenv("AGAINROM_DOCUMENT_CORPUS")
	if root == "" {
		t.Fatal("set AGAINROM_DOCUMENT_CORPUS to the explicit read-only SAV directory")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".sav") {
			continue
		}
		count++
		t.Run(entry.Name(), func(t *testing.T) {
			info, err := entry.Info()
			if err != nil || info.Size() > maxArchiveOutput {
				t.Fatalf("input size: %v", err)
			}
			path := filepath.Join(root, entry.Name())
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(raw)
			d, err := parseSaveDocument(raw)
			if err != nil {
				t.Fatalf("input %x: %v", digest, err)
			}
			clear(raw)
			out, err := serializeSaveDocument(d)
			if err != nil {
				t.Fatal(err)
			}
			fresh, err := parseSaveDocument(out)
			if err != nil {
				t.Fatal(err)
			}
			again, err := serializeSaveDocument(fresh)
			if err != nil || !bytes.Equal(out, again) {
				t.Fatalf("repeated detached encoding differs: %v", err)
			}
			// A known header field is independently read directly from the new
			// decoded wire. This is a mutation check, not a claimed game action.
			wantClock := d.archive.head.CounterA + 1
			d.archive.head.CounterA = wantClock
			changed, err := serializeSaveDocument(d)
			if err != nil {
				t.Fatal(err)
			}
			opened, err := Open(changed)
			if err != nil || u32(opened.Body, 0) != wantClock {
				t.Fatalf("header mutation not emitted: %v", err)
			}
			still, err := os.ReadFile(path)
			if err != nil || sha256.Sum256(still) != digest {
				t.Fatalf("read-only corpus changed: %v", err)
			}
			t.Logf("source %x; world=%t; detached=%d bytes", digest, d.archive.world != nil, len(out))
		})
	}
	if count == 0 {
		t.Fatal("selected directory contains no SAV subjects")
	}
	t.Logf("read-only codec subjects: %d", count)
}
