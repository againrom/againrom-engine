package reg_test

import (
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/reg"
)

// accessorsTree is the parsed tree TestAccessors runs every case against: a
// directory section ("Global") holding one key of each of the four types plus
// a nested directory (to stand in for "a key that is itself a directory"),
// and a value node directly under the root (to stand in for "a non-directory
// used as a section").
func accessorsTree(t *testing.T) *reg.Reg {
	t.Helper()
	r, err := reg.Parse(synth.Reg(17, []synth.RegNode{
		{Name: "Global", Kind: 0x01, Children: []synth.RegNode{
			{Name: "Title", Kind: 0x00, Str: "rage"},
			{Name: "UnitCount", Kind: 0x02, Int: 34},
			{Name: "Scale", Kind: 0x04, Float: 1.5},
			{Name: "Frames", Kind: 0x06, Ints: []int32{1, 2, 3}},
			{Name: "Inner", Kind: 0x01, Children: []synth.RegNode{
				{Name: "Depth", Kind: 0x02, Int: -1},
			}},
		}},
		{Name: "Version", Kind: 0x02, Int: 7}, // a value node directly under root
	}))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return r
}

func TestAccessors(t *testing.T) {
	r := accessorsTree(t)

	// --- hits, case-insensitive over ASCII only -----------------------------

	t.Run("GetString hit, ASCII case folded", func(t *testing.T) {
		got, ok := r.GetString("global", "TITLE")
		if !ok || got != "rage" {
			t.Errorf("GetString(\"global\", \"TITLE\") = %q, %v, want \"rage\", true", got, ok)
		}
	})
	t.Run("GetInt hit, ASCII case folded", func(t *testing.T) {
		got, ok := r.GetInt("GLOBAL", "unitcount")
		if !ok || got != 34 {
			t.Errorf("GetInt(\"GLOBAL\", \"unitcount\") = %d, %v, want 34, true", got, ok)
		}
	})
	t.Run("GetFloat hit, ASCII case folded", func(t *testing.T) {
		got, ok := r.GetFloat("Global", "scale")
		if !ok || got != 1.5 {
			t.Errorf("GetFloat(\"Global\", \"scale\") = %v, %v, want 1.5, true", got, ok)
		}
	})
	t.Run("GetIntArray hit, ASCII case folded", func(t *testing.T) {
		got, ok := r.GetIntArray("Global", "FRAMES")
		want := []int32{1, 2, 3}
		if !ok || len(got) != len(want) {
			t.Fatalf("GetIntArray(\"Global\", \"FRAMES\") = %v, %v, want %v, true", got, ok, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("GetIntArray(...)[%d] = %d, want %d", i, got[i], want[i])
			}
		}
	})

	// --- misses, none of which panics ----------------------------------------

	t.Run("wrong-type key misses", func(t *testing.T) {
		if _, ok := r.GetInt("Global", "Title"); ok {
			t.Error("GetInt on a string key returned true, want false")
		}
		if _, ok := r.GetString("Global", "UnitCount"); ok {
			t.Error("GetString on an int key returned true, want false")
		}
	})
	t.Run("missing key misses", func(t *testing.T) {
		if _, ok := r.GetString("Global", "NoSuchKey"); ok {
			t.Error("GetString on a missing key returned true, want false")
		}
	})
	t.Run("missing section misses", func(t *testing.T) {
		if _, ok := r.GetString("NoSuchSection", "Title"); ok {
			t.Error("GetString on a missing section returned true, want false")
		}
	})
	t.Run("a value node used as a section misses, not panics", func(t *testing.T) {
		got, ok := r.GetInt("Version", "Anything")
		if ok {
			t.Errorf("GetInt(\"Version\", ...) = %d, true; want false — Version is a value, not a directory", got)
		}
	})
	t.Run("a directory used as a key misses, not panics", func(t *testing.T) {
		got, ok := r.GetInt("Global", "Inner")
		if ok {
			t.Errorf("GetInt(\"Global\", \"Inner\") = %d, true; want false — Inner is a directory", got)
		}
	})

	// --- the ASCII-only fold boundary ----------------------------------------

	t.Run("the fold boundary: A-Z folds, bytes >= 0x80 do not", func(t *testing.T) {
		// "S" + 0x80, holding key "K" = 5. Built from bytes, never as literal
		// non-ASCII text (golden rule 2).
		hiSection := string([]byte{0x53, 0x80})
		r2, err := reg.Parse(synth.Reg(17, []synth.RegNode{
			{Name: hiSection, Kind: 0x01, Children: []synth.RegNode{
				{Name: "K", Kind: 0x02, Int: 5},
			}},
		}))
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}

		// Exact match hits.
		if got, ok := r2.GetInt(hiSection, "K"); !ok || got != 5 {
			t.Errorf("GetInt(exact) = %d, %v, want 5, true", got, ok)
		}
		// Lower-casing the ASCII byte still hits: the fold applies to the rest
		// of the name even though the high byte is present.
		asciiFolded := string([]byte{0x73, 0x80}) // "s" + 0x80
		if got, ok := r2.GetInt(asciiFolded, "K"); !ok || got != 5 {
			t.Errorf("GetInt(ASCII-folded) = %d, %v, want 5, true — only the ASCII byte differs", got, ok)
		}
		// Changing only the byte >= 0x80 must not match: this is the distinction
		// pinned so it is not merely assumed. strings.EqualFold would decode both
		// invalid-UTF-8 bytes to utf8.RuneError and wrongly call them equal.
		hiDiffer := string([]byte{0x53, 0x81}) // "S" + 0x81
		if got, ok := r2.GetInt(hiDiffer, "K"); ok {
			t.Errorf("GetInt(high-byte differs) = %d, true; want false — 0x80 and 0x81 are not ASCII case variants", got)
		}
	})

	// --- GetIntArray returns a copy -------------------------------------------

	t.Run("GetIntArray returns a copy", func(t *testing.T) {
		got, ok := r.GetIntArray("Global", "Frames")
		if !ok {
			t.Fatal("GetIntArray(\"Global\", \"Frames\") missed, want a hit")
		}
		got[0] = -999
		again, ok := r.GetIntArray("Global", "Frames")
		if !ok || again[0] != 1 {
			t.Errorf("second GetIntArray call = %v, %v; want [1 2 3], true — mutating the first result must not leak", again, ok)
		}
	})
}
