package game

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/divledger"
)

// Every object of the ROM1 town description that states a value carries a
// cite, and every cited ID is a claim in the pinned knowledge snapshot, a row
// of the divergence ledger, or "owner".
func TestTownDescriptionEveryValueIsCited(t *testing.T) {
	checkDescriptionCites(t, rom1TownJSON)
}

// checkDescriptionCites walks one JSON description: every object stating a
// value carries a cite list, and each cite is a claim heading or claim table
// row in the pinned snapshot, a ledger row, or "owner". A "when" object is a
// state condition of the art entry around it and carries that entry's cite.
func checkDescriptionCites(t *testing.T, data []byte) {
	t.Helper()
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	claims := map[string]bool{}
	files, err := filepath.Glob(filepath.Join("..", "..", "knowledge", "claims", "*.md"))
	if err != nil || len(files) == 0 {
		t.Fatalf("knowledge claims: %v (%d files)", err, len(files))
	}
	for _, name := range files {
		f, err := os.Open(name)
		if err != nil {
			t.Fatal(err)
		}
		s := bufio.NewScanner(f)
		s.Buffer(make([]byte, 1<<20), 1<<20)
		for s.Scan() {
			if id, ok := strings.CutPrefix(s.Text(), "### "); ok {
				claims[strings.TrimSpace(id)] = true
			}
			if row, ok := strings.CutPrefix(s.Text(), "| "); ok {
				if id, _, found := strings.Cut(row, " |"); found && !strings.ContainsAny(id, " `") {
					claims[id] = true
				}
			}
		}
		f.Close()
		if err := s.Err(); err != nil {
			t.Fatal(name, err)
		}
	}
	rows, err := divledger.ParseGlob(filepath.Join("..", "..", "docs", "divergences", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	divs := map[string]bool{}
	for _, r := range rows {
		divs[r.ID()] = true
	}

	objects, cites := 0, 0
	var walk func(path string, v any)
	walk = func(path string, v any) {
		switch x := v.(type) {
		case []any:
			for i, e := range x {
				walk(fmt.Sprintf("%s[%d]", path, i), e)
			}
		case map[string]any:
			objects++
			states := false
			for k, e := range x {
				if k == "cite" || k == "when" {
					continue
				}
				if leafValue(e) {
					states = true
				}
				walk(path+"."+k, e)
			}
			list, _ := x["cite"].([]any)
			if states && len(list) == 0 {
				t.Errorf("%s states a value without a cite", path)
			}
			for _, c := range list {
				id, _ := c.(string)
				cites++
				switch {
				case id == "owner":
				case strings.HasPrefix(id, "DIV-"):
					if !divs[id] {
						t.Errorf("%s cites %s, which no ledger row carries", path, id)
					}
				case !claims[id]:
					t.Errorf("%s cites %q, which is no claim in the pinned snapshot", path, id)
				}
			}
		}
	}
	walk("$", doc)
	if objects == 0 || cites == 0 {
		t.Fatalf("walked %d objects and %d cites", objects, cites)
	}
	t.Logf("%d objects, %d cites checked", objects, cites)
}

// leafValue reports a scalar or a list of scalars: a value an object states
// itself rather than through a nested object.
func leafValue(v any) bool {
	switch x := v.(type) {
	case map[string]any:
		return false
	case []any:
		for _, e := range x {
			if !leafValue(e) {
				return false
			}
		}
		return true
	}
	return true
}
