package game

// A long corpus instrument runs as several part tests, so the milestone-2
// gate can run them as separate processes at once. Each part takes a fixed
// share of the discovered population and checks it on the whole's terms. A
// check over the whole population runs once, in the part that completes the
// set, from the summaries every part leaves in AGAINROM_M2_SHARE (one
// directory per root and run). Without that variable the parts of one process
// share their summaries in memory.

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
)

// corpusPartOf assigns each item to one of parts shares: heaviest first, each to
// the lightest share so far (the lower share on a tie), equal costs in name
// order. The assignment depends only on the population.
func corpusPartOf(names []string, cost []int64, parts int) []int {
	order := make([]int, len(names))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		if cost[order[a]] != cost[order[b]] {
			return cost[order[a]] > cost[order[b]]
		}
		return names[order[a]] < names[order[b]]
	})
	load := make([]int64, parts)
	out := make([]int, len(names))
	for _, i := range order {
		lightest := 0
		for p := 1; p < parts; p++ {
			if load[p] < load[lightest] {
				lightest = p
			}
		}
		out[i] = lightest
		load[lightest] += cost[i]
	}
	return out
}

var corpusPartsMemory struct {
	sync.Mutex
	sets map[string][][]byte
	done map[string]bool
}

// corpusPartsCollect records part's summary of family and returns every part's
// summary, in part order, when this part completes the set; otherwise nil.
// Exactly one part of a set receives the summaries.
func corpusPartsCollect(t *testing.T, family string, part, parts int, summary any) [][]byte {
	t.Helper()
	raw, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("part summary: %v", err)
	}
	h := fnv.New32a()
	h.Write([]byte(os.Getenv("AGAINROM_ASSETS")))
	key := fmt.Sprintf("%s-%08x", family, h.Sum32())
	dir := os.Getenv("AGAINROM_M2_SHARE")
	if dir == "" {
		corpusPartsMemory.Lock()
		defer corpusPartsMemory.Unlock()
		if corpusPartsMemory.sets == nil {
			corpusPartsMemory.sets, corpusPartsMemory.done = map[string][][]byte{}, map[string]bool{}
		}
		set := corpusPartsMemory.sets[key]
		if set == nil {
			set = make([][]byte, parts)
			corpusPartsMemory.sets[key] = set
		}
		if corpusPartsMemory.done[key] {
			set = make([][]byte, parts)
			corpusPartsMemory.sets[key], corpusPartsMemory.done[key] = set, false
		}
		set[part] = raw
		for _, s := range set {
			if s == nil {
				return nil
			}
		}
		corpusPartsMemory.done[key] = true
		return set
	}
	tmp, err := os.CreateTemp(dir, key+".tmp*")
	if err != nil {
		t.Fatalf("part summary: %v", err)
	}
	_, werr := tmp.Write(raw)
	if cerr := tmp.Close(); werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Rename(tmp.Name(), filepath.Join(dir, fmt.Sprintf("%s.part%d.json", key, part)))
	}
	if werr != nil {
		t.Fatalf("part summary: %v", werr)
	}
	set := make([][]byte, parts)
	for p := range set {
		b, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("%s.part%d.json", key, p)))
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			t.Fatalf("part summary: %v", err)
		}
		set[p] = b
	}
	claim, err := os.OpenFile(filepath.Join(dir, key+".done"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return nil
	}
	if err != nil {
		t.Fatalf("part summary: %v", err)
	}
	claim.Close()
	return set
}

// corpusPartsDecode decodes every part's summary into a T.
func corpusPartsDecode[T any](t *testing.T, set [][]byte) []T {
	t.Helper()
	out := make([]T, len(set))
	for p, raw := range set {
		if err := json.Unmarshal(raw, &out[p]); err != nil {
			t.Fatalf("part %d summary: %v", p+1, err)
		}
	}
	return out
}

// TestCorpusPartOf pins the share rule: every item in exactly one share, the
// heaviest items spread first, and the same population always split alike.
func TestCorpusPartOf(t *testing.T) {
	names := []string{"a", "b", "c", "d", "e", "f"}
	cost := []int64{10, 9, 1, 1, 8, 1}
	got := corpusPartOf(names, cost, 3)
	want := []int{0, 1, 2, 1, 2, 2}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("shares %v, want %v", got, want)
		}
	}
	if again := corpusPartOf(names, cost, 3); fmt.Sprint(again) != fmt.Sprint(got) {
		t.Fatalf("shares changed between calls: %v then %v", got, again)
	}
}

// TestCorpusPartsCollect proves exactly one part receives the set, in memory and
// through a share directory, whatever order the parts finish in.
func TestCorpusPartsCollect(t *testing.T) {
	for _, shared := range []bool{false, true} {
		dir := ""
		if shared {
			dir = t.TempDir()
		}
		t.Setenv("AGAINROM_M2_SHARE", dir)
		family := fmt.Sprintf("probe-%v", shared)
		got := 0
		for _, part := range []int{2, 0, 1} {
			if set := corpusPartsCollect(t, family, part, 3, part); set != nil {
				got++
				if part != 1 {
					t.Errorf("shared=%v: part %d received the set before the last part finished", shared, part+1)
				}
				if parts := corpusPartsDecode[int](t, set); fmt.Sprint(parts) != "[0 1 2]" {
					t.Errorf("shared=%v: set %v", shared, parts)
				}
			}
		}
		if corpusPartsCollect(t, family, 1, 3, 1) != nil {
			got++
		}
		if got != 1 {
			t.Errorf("shared=%v: %d parts received the set, want 1", shared, got)
		}
	}
}
