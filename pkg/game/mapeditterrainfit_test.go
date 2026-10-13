package game

import (
	"os"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
)

// The transition table (pkg/mapedit/terraintable.go) is fitted to the installed
// planes: for each side of a strip and each mask of neighbours of the other
// class, the shape most installed cells show. This file holds the fitting, the
// fallback for masks too rare to fit, and the held-out check.

const (
	tabPrimary = iota
	tabSecondary
	tabWater
)

// fitMinimum is the fewest installed cells a mask needs to be fitted from.
const fitMinimum = 8

type observation struct{ table, mask, code int }

var neighbourSteps = [8][2]int{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}}

// terrainObservations visits the playable cells of the given maps whose word
// shows a transition toward the partner the rule would choose, or is pure
// beside a neighbour of that partner's class.
func terrainObservations(t *testing.T, x *MapInspector, keys []string, visit func(observation)) {
	t.Helper()
	for _, key := range keys {
		d, err := x.Open(key)
		if err != nil {
			t.Fatal(err)
		}
		m, err := alm.Open(d.SourceBytes)
		if err != nil {
			t.Fatal(err)
		}
		cls := make([]uint8, len(m.Tiles))
		for i, w := range m.Tiles {
			cls[i] = mapload.TileClass(w)
		}
		for y := 8; y < m.Height-8; y++ {
			for xx := 8; xx < m.Width-8; xx++ {
				i := y*m.Width + xx
				var around [8]uint8
				for k, o := range neighbourSteps {
					around[k] = cls[(y+o[1])*m.Width+xx+o[0]]
				}
				b := mapedit.BlendOf(cls[i], around)
				if !b.Active {
					continue
				}
				shape, group, ok := mapedit.WordShape(m.Tiles[i])
				if !ok || shape.Kind != mapedit.ShapePure && (group != b.Group || shape.Primary != b.Primary) {
					continue
				}
				table := tabSecondary
				switch {
				case b.Water:
					table = tabWater
				case b.Primary:
					table = tabPrimary
				}
				visit(observation{table, b.Mask, shape.Code()})
			}
		}
	}
}

type fitCounts [3][256][17]int

func fitFrom(t *testing.T, x *MapInspector, keys []string) (c fitCounts) {
	terrainObservations(t, x, keys, func(o observation) { c[o.table][o.mask][o.code]++ })
	return c
}

// codes is the fitted table: the commonest shape per mask, the fallback rule's
// shape where fewer than fitMinimum cells were seen.
func (c *fitCounts) codes() (out [3][256]int) {
	for table := range out {
		for mask := range out[table] {
			total, best := 0, 0
			for code, n := range c[table][mask] {
				total += n
				if n > c[table][mask][best] {
					best = code
				}
			}
			out[table][mask] = best
			if total < fitMinimum {
				out[table][mask] = fallbackCode(table, mask)
			}
		}
	}
	return out
}

// fallbackCode is the shape for a mask the installed planes show too rarely to
// fit: a water cell takes the edge of a lone cardinal and the outer corner of
// two adjacent ones; a land primary cell takes the edge of a full side or of a
// lone cardinal, the inner corner of a diagonal two of whose three cells are
// the other class; a land secondary cell takes a tip.
func fallbackCode(table, mask int) int {
	const (
		nw, n, ne, w, e, sw, s, se = 1, 2, 4, 8, 16, 32, 64, 128
	)
	cards := [4]int{n, e, s, w}
	diags := [4][3]int{{nw, n, w}, {ne, n, e}, {sw, s, w}, {se, s, e}}
	sides := [4]int{nw | n | ne, ne | e | se, sw | s | se, nw | w | sw}
	score := func(d [3]int) (v int) {
		for _, b := range d {
			if mask&b != 0 {
				v++
			}
		}
		return v
	}
	switch table {
	case tabSecondary:
		best, bestScore := 0, 0
		for i, d := range diags {
			if mask&d[0] != 0 && score(d) >= 2 && score(d) > bestScore {
				best, bestScore = 13+i, score(d)
			}
		}
		return best
	case tabWater:
		for i, d := range diags {
			if mask&d[1] != 0 && mask&d[2] != 0 {
				return 9 + i
			}
		}
		for _, i := range [4]int{0, 2, 3, 1} {
			if mask&cards[i] != 0 {
				return 1 + i
			}
		}
		for i, d := range diags {
			if mask&d[0] != 0 {
				return 5 + i
			}
		}
		return 0
	}
	var full []int
	for i, s := range sides {
		if mask&s == s {
			full = append(full, i)
		}
	}
	if len(full) >= 2 {
		for i, d := range diags {
			a, b := -1, -1
			for k, c := range cards {
				if c == d[1] {
					a = k
				}
				if c == d[2] {
					b = k
				}
			}
			if a == full[0] && b == full[1] || b == full[0] && a == full[1] {
				return 5 + i
			}
		}
	}
	if len(full) > 0 {
		return 1 + full[0]
	}
	best, bestScore := -1, 1
	for i, d := range diags {
		if score(d) > bestScore {
			best, bestScore = i, score(d)
		}
	}
	if best >= 0 {
		return 5 + best
	}
	for _, i := range [4]int{0, 2, 3, 1} {
		if mask&cards[i] != 0 {
			return 1 + i
		}
	}
	return 0
}

// TestFitTransitionTable fits the table to the installed maps of the root in
// AGAINROM_ASSETS and, when AGAINROM_FIT_TABLE names an output path, writes
// pkg/mapedit/terraintable.go there.
func TestFitTransitionTable(t *testing.T) {
	x := installedInspector(t)
	c := fitFrom(t, x, installedMapKeys(x))
	codes := c.codes()
	out := os.Getenv("AGAINROM_FIT_TABLE")
	if out == "" {
		return
	}
	var b strings.Builder
	b.WriteString("package mapedit\n\n// shapeTable names, for a cell on the primary or the secondary side of a strip\n// and for water, the shape it shows given the mask of neighbours of the other\n// class (bits NW, N, NE, W, E, SW, S, SE = 1, 2, 4, 8, 16, 32, 64, 128): one\n// digit per mask, the Shape.Code of the commonest installed shape, or the fallback\n// rule's where fewer than eight installed cells show the mask (DIV-3199).\nvar shapeTable = struct{ primary, secondary, water string }{\n")
	for _, table := range codes {
		for row := 0; row < 16; row++ {
			b.WriteString("\t\"")
			for col := 0; col < 16; col++ {
				b.WriteByte("0123456789abcdefg"[table[row*16+col]])
			}
			b.WriteString("\"")
			if row < 15 {
				b.WriteString(" +")
			} else {
				b.WriteString(",")
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("}\n")
	if err := os.WriteFile(out, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The table, fitted on the even-numbered installed maps, predicts the shape of
// the cells of the odd-numbered ones: the held-out agreement of the fitting.
func TestReleaseTerrainTableHeldOut(t *testing.T) {
	x := installedInspector(t)
	keys := installedMapKeys(x)
	var even, odd []string
	for i, k := range keys {
		if i%2 == 0 {
			even = append(even, k)
		} else {
			odd = append(odd, k)
		}
	}
	codes := func() [3][256]int { c := fitFrom(t, x, even); return c.codes() }()
	var agree, total, pure, pureAgree [3]int
	terrainObservations(t, x, odd, func(o observation) {
		total[o.table]++
		if codes[o.table][o.mask] == o.code {
			agree[o.table]++
		}
		if o.code == 0 {
			pure[o.table]++
			if codes[o.table][o.mask] == 0 {
				pureAgree[o.table]++
			}
		}
	})
	for i, name := range []string{"primary", "secondary", "water"} {
		t.Logf("%s side: fitted on %d maps, %d cells of %d other maps, shape agreement %d (%.1f%%), of which pure cells %d of %d",
			name, len(even), total[i], len(odd), agree[i], 100*float64(agree[i])/float64(max(1, total[i])), pureAgree[i], pure[i])
		if agree[i]*100 < total[i]*60 {
			t.Errorf("%s side: held-out agreement below 60%%", name)
		}
	}
}
