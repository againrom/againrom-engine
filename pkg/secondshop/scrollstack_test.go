package secondshop

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/random"
)

// scrollStackTables are sortTables with every scroll row of MagicItems priced
// at ten times its row and the six potions at 25.
func scrollStackTables() Tables {
	tb := sortTables()
	rows := map[int][]int32{}
	for row := scrollOffset + 1; row <= superOffset+bookSpells; row++ {
		rows[row] = []int32{int32(10 * row)}
	}
	for row := potionFirst; row <= potionLast; row++ {
		rows[row] = []int32{25}
	}
	tb.MagicItems = fixtureRows{n: 128, params: rows}
	return tb
}

// scrollStackRecord holds a plain armour and shield category drawing
// stackable units at bound 2 and a magic category drawing 20 scrolls priced
// 100 or more at bound 1.
func scrollStackRecord() Record {
	return Record{
		{Min: 0, Max: 100000, Draws: 30, Bound: 2, Mask: kindArmor | kindShield | 1<<classShift | 1<<1},
		3: {Min: 100, Max: 100000, Draws: 20, Bound: 1, Mask: kindMagic},
	}
}

// scrollStackLine is one fill at seed with factor rolls: the shop stream's
// state after it, a digest of every row but the scroll rows, and each scroll
// row's identity and quantity.
func scrollStackLine(tb Tables, seed uint32, factor int) (string, []Item) {
	g := &random.MSVC{State: seed}
	tb.Rolls = Rolls{}
	if factor > 1 {
		tb.Rolls = rollsOf(factor)
	}
	s := tb.Fill(scrollStackRecord(), func() int { return int(g.Rand()) })
	h := sha256.New()
	var scrolls []string
	var rows []Item
	for k, list := range s {
		for _, it := range list {
			if scroll(it) {
				scrolls = append(scrolls, fmt.Sprintf("%d:%d:%d", it.Row, it.Price, it.Count))
				rows = append(rows, it)
				continue
			}
			fmt.Fprintf(h, "%d %+v;", k, it)
		}
	}
	return fmt.Sprintf("seed %d x%d state %d rows %x scrolls %s", seed, factor, g.State, h.Sum(nil)[:8], strings.Join(scrolls, " ")), rows
}

// The record testdata/scroll_stack_record.txt is the fill as drawn before the
// scroll stack range, where a scroll row's quantity is its picks' sum of
// 1..2. Each fill now ends at the same stream state with the same other rows,
// every scroll row holds 51..100, rows of three or more picks included, and
// the rows reach both ends.
func TestSecondShopScrollRowsStackFiftyOneToHundred(t *testing.T) {
	want, err := os.ReadFile(filepath.Join("testdata", "scroll_stack_record.txt"))
	if err != nil {
		t.Fatal(err)
	}
	tb := scrollStackTables()
	least, most, repeated := 1000, 0, 0
	for i, line := range strings.Split(strings.TrimSpace(string(want)), "\n") {
		var seed uint32
		var factor int
		if _, err := fmt.Sscanf(line, "seed %d x%d", &seed, &factor); err != nil {
			t.Fatalf("record line %d: %v", i, err)
		}
		got, scrolls := scrollStackLine(tb, seed, factor)
		head, wantScrolls, _ := strings.Cut(line, " scrolls ")
		if gotHead, _, _ := strings.Cut(got, " scrolls "); gotHead != head {
			t.Fatalf("seed %d x%d: stream or other rows moved:\n got %s\nwant %s", seed, factor, gotHead, head)
		}
		before := strings.Fields(wantScrolls)
		if len(before) != len(scrolls) {
			t.Fatalf("seed %d x%d: %d scroll rows, want %d", seed, factor, len(scrolls), len(before))
		}
		for j, it := range scrolls {
			id := fmt.Sprintf("%d:%d:", it.Row, it.Price)
			q, err := strconv.Atoi(strings.TrimPrefix(before[j], id))
			if err != nil || !strings.HasPrefix(before[j], id) {
				t.Fatalf("seed %d x%d: scroll row %d is %s, want %s", seed, factor, j, id, before[j])
			}
			if it.Count < 51 || it.Count > 100 {
				t.Fatalf("seed %d x%d: scroll row %d of %d units before holds %d", seed, factor, it.Row, q, it.Count)
			}
			if q >= 3 {
				repeated++
			}
			least, most = min(least, int(it.Count)), max(most, int(it.Count))
		}
	}
	if least != 51 || most != 100 || repeated == 0 {
		t.Fatalf("scroll rows reach %d..%d, %d of them repeated picks, want 51..100", least, most, repeated)
	}
}

// A magic category of 20 draws and a multiplier of 3 over one admitted
// scroll picks it 60 times beside the six potions; its row holds one draw
// of 51..100.
func TestSecondShopRepeatedScrollPicksKeepOneStack(t *testing.T) {
	tb := sortTables()
	tb.MagicItems = fixtureRows{n: 128, params: map[int][]int32{scrollOffset + 1: {60}, superOffset + 1: {350}}}
	tb.Rolls = rollsOf(3)
	r := Record{{Min: 0, Max: 1000, Draws: 20, Bound: 1, Mask: kindMagic}}
	least, most := 1000, 0
	for seed := uint32(1); seed <= 60; seed++ {
		g := &random.MSVC{State: seed}
		s := tb.Fill(r, func() int { return int(g.Rand()) })
		var rows []Item
		for _, it := range s[0] {
			if scroll(it) {
				rows = append(rows, it)
			}
		}
		if len(rows) != 1 {
			t.Fatalf("seed %d: scroll rows %+v, want one", seed, rows)
		}
		n := int(rows[0].Count)
		if n < 51 || n > 100 {
			t.Fatalf("seed %d: the scroll row holds %d after 60 picks", seed, n)
		}
		least, most = min(least, n), max(most, n)
	}
	if least > 60 || most < 90 {
		t.Fatalf("scroll rows reach %d..%d over 60 fills", least, most)
	}
}

// A mod's stocked scroll keeps the one unit it places: it takes no draw and
// no stack range.
func TestSecondShopScrollStacksKeepModStockUnits(t *testing.T) {
	tb := scrollStackTables()
	tb.Stocked = []Item{{Table: Magic, Row: scrollOffset + 1, Count: 1, Price: 60}}
	for seed := uint32(1); seed <= 8; seed++ {
		g := &random.MSVC{State: seed}
		s := tb.Fill(scrollStackRecord(), func() int { return int(g.Rand()) })
		found, drawn := false, 0
		for _, it := range s[3] {
			switch {
			case it.Row == scrollOffset+1:
				found = true
				if it.Count != 1 {
					t.Fatalf("seed %d: the mod's scroll holds %d units, want 1", seed, it.Count)
				}
			case scroll(it):
				drawn++
				if it.Count < 51 {
					t.Fatalf("seed %d: scroll row %d holds %d units", seed, it.Row, it.Count)
				}
			}
		}
		if !found || drawn == 0 {
			t.Fatalf("seed %d: mod scroll placed %t, %d drawn scroll rows", seed, found, drawn)
		}
	}
}
