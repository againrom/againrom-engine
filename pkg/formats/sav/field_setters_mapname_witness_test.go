//go:build savdocumentaudit

package sav

import (
	"bytes"
	"crypto/sha256"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetMapNameCorpusByteLevelShiftNotScatter(t *testing.T) {
	root := os.Getenv("AGAINROM_DOCUMENT_CORPUS")
	if root == "" {
		t.Fatal("set AGAINROM_DOCUMENT_CORPUS to the explicit read-only SAV directory")
	}
	var paths []string
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(path), ".sav") {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("selected directory tree contains no SAV subjects")
	}

	var filesOpened, calls, prefixMovedBytes, suffixChangedBytes, regionChanges, lengthAnomalies int
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if len(raw) > maxArchiveOutput {
			continue // outside what this codec accepts as input at all
		}
		digest := sha256.Sum256(raw)

		f, err := Open(raw)
		if err != nil {
			// Not every .sav under these trees is a well-formed city SAV
			// this codec opens (forged/malformed review fixtures among
			// them); this witness measures only ones that do, exactly as
			// the sibling corpus tests do.
			continue
		}
		filesOpened++

		labelBefore := append([]byte(nil), f.LabelRegion...)
		storeBefore := append([]byte(nil), f.Store...)
		tailBefore := append([]byte(nil), f.TailRest...)
		mapNameOff := f.Head.MapNameOff

		cur := f.Head.MapName
		var names []string
		for _, d := range []int{1, 2, 3, 1} {
			cur = cur + strings.Repeat("x", d)
			names = append(names, cur)
		}
		names = append(names, "") // the shrink

		for _, name := range names {
			before := append([]byte(nil), f.Body...)
			beforeName := f.Head.MapName
			beforeDoc, _, err := f.exactDocument()
			if err != nil {
				t.Fatalf("%s: pre-edit exactDocument: %v", path, err)
			}
			hadPad := beforeDoc.end&1 != 0
			oldEnd := mapNameOff + 1 + len(beforeName)
			if oldEnd > len(before) {
				t.Fatalf("%s: old name span [%d,%d) outside a %d-byte body", path, mapNameOff, oldEnd, len(before))
			}

			if err := f.SetMapName(name); err != nil {
				t.Fatalf("%s: SetMapName(%q -> %q): %v", path, beforeName, name, err)
			}
			calls++

			if f.Head.MapNameOff != mapNameOff {
				t.Fatalf("%s: MapNameOff moved from %d to %d", path, mapNameOff, f.Head.MapNameOff)
			}

			// Prefix: everything before MapNameOff.
			if len(f.Body) < mapNameOff {
				t.Fatalf("%s: edited body is %d bytes, shorter than MapNameOff %d", path, len(f.Body), mapNameOff)
			}
			if n := byteDiffOverlap(before[:mapNameOff], f.Body[:mapNameOff]); n != 0 {
				prefixMovedBytes += n
				t.Errorf("%s: %d prefix bytes moved for %q -> %q", path, n, beforeName, name)
			}

			// Suffix: the old suffix reappears unchanged, minus/plus exactly
			// one trailing alignment byte on an odd-parity delta.
			oldSuffix := before[oldEnd:]
			newEnd := mapNameOff + 1 + len(name)
			if newEnd > len(f.Body) {
				t.Fatalf("%s: new name span [%d,%d) outside a %d-byte body", path, mapNameOff, newEnd, len(f.Body))
			}
			newSuffix := f.Body[newEnd:]
			delta := len(name) - len(beforeName)
			wantLen := len(oldSuffix)
			if delta%2 != 0 {
				if hadPad {
					wantLen--
				} else {
					wantLen++
				}
			}
			if len(newSuffix) != wantLen {
				lengthAnomalies++
				t.Errorf("%s: suffix is %d bytes for %q -> %q, want %d", path, len(newSuffix), beforeName, name, wantLen)
			}
			overlap := len(oldSuffix)
			if len(newSuffix) < overlap {
				overlap = len(newSuffix)
			}
			if n := byteDiffOverlap(oldSuffix[:overlap], newSuffix[:overlap]); n != 0 {
				suffixChangedBytes += n
				t.Errorf("%s: %d suffix bytes changed for %q -> %q", path, n, beforeName, name)
			}

			// The three carried regions sit outside Body entirely.
			if !bytes.Equal(f.LabelRegion, labelBefore) || !bytes.Equal(f.Store, storeBefore) || !bytes.Equal(f.TailRest, tailBefore) {
				regionChanges++
				t.Errorf("%s: a carried region moved for %q -> %q", path, beforeName, name)
			}

			back, err := Open(f.Marshal())
			if err != nil {
				t.Fatalf("%s: reopen after SetMapName(%q): %v", path, name, err)
			}
			if back.Head.MapName != name {
				t.Fatalf("%s: reopened MapName = %q, want %q", path, back.Head.MapName, name)
			}
			f = back
		}

		still, err := os.ReadFile(path)
		if err != nil || sha256.Sum256(still) != digest {
			t.Fatalf("%s: read-only corpus changed", path)
		}
	}

	t.Logf("files opened: %d, SetMapName calls: %d, prefix bytes moved: %d, suffix bytes changed: %d, carried-region changes: %d, suffix length anomalies: %d",
		filesOpened, calls, prefixMovedBytes, suffixChangedBytes, regionChanges, lengthAnomalies)
	if filesOpened == 0 {
		t.Fatal("no subject under the selected tree opened as a SAV file")
	}
}

// byteDiffOverlap counts differing bytes over the shared prefix of a and b.
// A caller that needs the length difference counted too (as
// field_setters_corpus_test.go's diffCount does) is asking a different
// question; this one is asked only over slices already sliced to a matching
// or deliberately overlapping length.
func byteDiffOverlap(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	diff := 0
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			diff++
		}
	}
	return diff
}
