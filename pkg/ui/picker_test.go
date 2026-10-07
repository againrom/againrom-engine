package ui

import (
	"fmt"
	"image"
	"strings"
	"testing"
	"unicode/utf8"
)

const pickerTestRows = 38

// pickerFixture builds n synthetic rows, all choosable. No game data.
func pickerFixture(n int) []PickerRow {
	rows := make([]PickerRow, n)
	for i := range rows {
		rows[i] = PickerRow{Text: fmt.Sprintf("map%02d.alm", i), Choosable: true}
	}
	return rows
}

// pickerWantTop is this test's own transcription of DD13's "scrolls top
// minimally to keep sel visible": the window moves only when the selection
// would otherwise fall outside it, and then by the least amount that brings it
// back in.
func pickerWantTop(prevTop, sel, n int) int {
	top := prevTop
	if sel < top {
		top = sel
	}
	if sel >= top+pickerVisible {
		top = sel - pickerVisible + 1
	}
	if hi := n - pickerVisible; top > hi {
		top = hi
	}
	if top < 0 {
		top = 0
	}
	return top
}

// pickerWantWindow is the window DD13 says the draw path iterates: pickerVisible
// rows from top, or fewer when the list runs out first.
func pickerWantWindow(top, n int) (int, int) {
	count := pickerVisible
	if n-top < count {
		count = n - top
	}
	if count < 0 {
		count = 0
	}
	return top, count
}

// pickerWantRowAt is DD13's hit-test, recomputed: row = top + (y-pickerTop)/pickerLine,
// rejected above the list, past the visible window, or past the last row. p.X is
// not consulted.
func pickerWantRowAt(y, top, count int) (int, bool) {
	if y < pickerTop {
		return 0, false
	}
	i := (y - pickerTop) / pickerLine
	if i >= count {
		return 0, false
	}
	return top + i, true
}

// pickerCheckWindow asserts the invariants DD13 states for Visible(): it never
// reports a row that does not exist, never more than the window holds, and
// always contains the selection.
func pickerCheckWindow(t *testing.T, p *Picker, what string) {
	t.Helper()
	top, n := p.Visible()
	if top < 0 || n < 0 {
		t.Fatalf("%s: Visible() = (%d, %d), want non-negative", what, top, n)
	}
	if n > pickerVisible {
		t.Fatalf("%s: Visible() count %d exceeds pickerVisible %d", what, n, pickerVisible)
	}
	if top+n > p.Len() {
		t.Fatalf("%s: Visible() = (%d, %d) reports rows past Len() = %d", what, top, n, p.Len())
	}
	if p.Len() > 0 {
		if sel := p.Selection(); sel < top || sel >= top+n {
			t.Fatalf("%s: selection %d outside window (%d, %d)", what, sel, top, n)
		}
	}
}

func TestPicker(t *testing.T) {
	// The layout constants the rest of this test computes from. Stated in DD13;
	// pinned here so a silent change to one of them fails loudly rather than
	// quietly rewriting every expectation below.
	t.Run("layout constants", func(t *testing.T) {
		for _, c := range []struct {
			name string
			got  int
			want int
		}{
			{"pickerLeft", pickerLeft, 8},
			{"pickerTop", pickerTop, 40},
			{"pickerLine", pickerLine, 16},
			{"pickerVisible", pickerVisible, 25},
			{"pickerCols", pickerCols, 104},
		} {
			if c.got != c.want {
				t.Errorf("%s = %d, want %d (DD13)", c.name, c.got, c.want)
			}
		}
		// DD13 derives the window from the list's pixel extent: y in [40, 440).
		if pickerTop+pickerVisible*pickerLine != 440 {
			t.Errorf("list extent = [%d, %d), want [40, 440) (DD13)",
				pickerTop, pickerTop+pickerVisible*pickerLine)
		}
	})

	t.Run("empty list is safe", func(t *testing.T) {
		p := NewPicker(nil)
		if got := p.Len(); got != 0 {
			t.Fatalf("Len() = %d, want 0", got)
		}
		if got := p.Selection(); got != -1 {
			t.Fatalf("Selection() on an empty list = %d, want -1", got)
		}
		// Move must be a no-op, not a panic and not an index out of range.
		p.Move(1)
		p.Move(-1)
		p.Move(1000)
		p.Move(-1000)
		if got := p.Selection(); got != -1 {
			t.Fatalf("Selection() after Move on an empty list = %d, want -1", got)
		}
		if i, ok := p.Choose(); ok {
			t.Fatalf("Choose() on an empty list = (%d, true), want ok=false", i)
		}
		if top, n := p.Visible(); top != 0 || n != 0 {
			t.Fatalf("Visible() on an empty list = (%d, %d), want (0, 0)", top, n)
		}
		if ok := p.Select(0); ok {
			t.Fatalf("Select(0) on an empty list = true, want false")
		}
		if i, ok := p.RowAt(image.Pt(pickerLeft, pickerTop)); ok {
			t.Fatalf("RowAt on an empty list = (%d, true), want ok=false", i)
		}
		// SetUnusable must not panic on an empty list either.
		p.SetUnusable(0)
	})

	t.Run("initial state", func(t *testing.T) {
		p := NewPicker(pickerFixture(pickerTestRows))
		if got := p.Len(); got != pickerTestRows {
			t.Fatalf("Len() = %d, want %d", got, pickerTestRows)
		}
		if got := p.Selection(); got != 0 {
			t.Fatalf("Selection() on a fresh picker = %d, want 0 (the first row)", got)
		}
		wantTop, wantN := pickerWantWindow(0, pickerTestRows)
		if top, n := p.Visible(); top != wantTop || n != wantN {
			t.Fatalf("Visible() = (%d, %d), want (%d, %d)", top, n, wantTop, wantN)
		}
		pickerCheckWindow(t, p, "fresh picker")
		// Every fixture row is listed, in order, with its text intact.
		rows := p.Rows()
		if len(rows) != pickerTestRows {
			t.Fatalf("len(Rows()) = %d, want %d", len(rows), pickerTestRows)
		}
		for i, r := range rows {
			if want := fmt.Sprintf("map%02d.alm", i); r.Text != want {
				t.Fatalf("Rows()[%d].Text = %q, want %q", i, r.Text, want)
			}
			if !r.Choosable {
				t.Fatalf("Rows()[%d] is not choosable, want choosable", i)
			}
		}
	})

	t.Run("Move clamps at both ends", func(t *testing.T) {
		p := NewPicker(pickerFixture(pickerTestRows))
		p.Move(-1)
		if got := p.Selection(); got != 0 {
			t.Fatalf("Move(-1) from row 0: Selection() = %d, want 0", got)
		}
		p.Move(-1000)
		if got := p.Selection(); got != 0 {
			t.Fatalf("Move(-1000) from row 0: Selection() = %d, want 0 (clamp, not wrap)", got)
		}
		p.Move(1000)
		last := pickerTestRows - 1
		if got := p.Selection(); got != last {
			t.Fatalf("Move(1000): Selection() = %d, want %d (clamp, not wrap)", got, last)
		}
		p.Move(1)
		if got := p.Selection(); got != last {
			t.Fatalf("Move(1) from the last row: Selection() = %d, want %d", got, last)
		}
		p.Move(1000)
		if got := p.Selection(); got != last {
			t.Fatalf("Move(1000) from the last row: Selection() = %d, want %d (clamp, not wrap)", got, last)
		}
		pickerCheckWindow(t, p, "after clamping at the end")
		// A large jump back clamps to the first row and scrolls the window home.
		p.Move(-1000)
		if got := p.Selection(); got != 0 {
			t.Fatalf("Move(-1000) from the last row: Selection() = %d, want 0", got)
		}
		if top, n := p.Visible(); top != 0 || n != pickerVisible {
			t.Fatalf("Visible() back at row 0 = (%d, %d), want (0, %d)", top, n, pickerVisible)
		}
	})

	t.Run("scroll window follows the selection minimally", func(t *testing.T) {
		p := NewPicker(pickerFixture(pickerTestRows))
		top := 0
		for sel := 1; sel < pickerTestRows; sel++ {
			p.Move(1)
			top = pickerWantTop(top, sel, pickerTestRows)
			if got := p.Selection(); got != sel {
				t.Fatalf("walking down, step %d: Selection() = %d, want %d", sel, got, sel)
			}
			wantTop, wantN := pickerWantWindow(top, pickerTestRows)
			gotTop, gotN := p.Visible()
			if gotTop != wantTop || gotN != wantN {
				t.Fatalf("walking down, sel %d: Visible() = (%d, %d), want (%d, %d)",
					sel, gotTop, gotN, wantTop, wantN)
			}
			pickerCheckWindow(t, p, fmt.Sprintf("walking down, sel %d", sel))
		}
		// The window must have moved by exactly the overflow, and no further.
		if want := pickerTestRows - pickerVisible; top != want {
			t.Fatalf("top at the last row = %d, want %d (minimal scroll)", top, want)
		}
		for sel := pickerTestRows - 2; sel >= 0; sel-- {
			p.Move(-1)
			top = pickerWantTop(top, sel, pickerTestRows)
			if got := p.Selection(); got != sel {
				t.Fatalf("walking up, step %d: Selection() = %d, want %d", sel, got, sel)
			}
			wantTop, wantN := pickerWantWindow(top, pickerTestRows)
			gotTop, gotN := p.Visible()
			if gotTop != wantTop || gotN != wantN {
				t.Fatalf("walking up, sel %d: Visible() = (%d, %d), want (%d, %d)",
					sel, gotTop, gotN, wantTop, wantN)
			}
			pickerCheckWindow(t, p, fmt.Sprintf("walking up, sel %d", sel))
		}
		if top != 0 {
			t.Fatalf("top back at row 0 = %d, want 0", top)
		}
	})

	t.Run("a list shorter than the window reports all of it", func(t *testing.T) {
		for _, n := range []int{1, 5, pickerVisible - 1, pickerVisible} {
			p := NewPicker(pickerFixture(n))
			top, count := p.Visible()
			if top != 0 || count != n {
				t.Fatalf("%d rows: Visible() = (%d, %d), want (0, %d)", n, top, count, n)
			}
			p.Move(1000)
			top, count = p.Visible()
			if top != 0 || count != n {
				t.Fatalf("%d rows after Move(1000): Visible() = (%d, %d), want (0, %d)", n, top, count, n)
			}
			pickerCheckWindow(t, p, fmt.Sprintf("%d rows", n))
		}
	})

	t.Run("Select is absolute and scrolls into view", func(t *testing.T) {
		p := NewPicker(pickerFixture(pickerTestRows))
		for _, sel := range []int{30, 37, 0, 12, 24, 25} {
			before, _ := p.Visible()
			if ok := p.Select(sel); !ok {
				t.Fatalf("Select(%d) = false, want true", sel)
			}
			if got := p.Selection(); got != sel {
				t.Fatalf("Select(%d): Selection() = %d, want %d", sel, got, sel)
			}
			wantTop, wantN := pickerWantWindow(pickerWantTop(before, sel, pickerTestRows), pickerTestRows)
			gotTop, gotN := p.Visible()
			if gotTop != wantTop || gotN != wantN {
				t.Fatalf("Select(%d): Visible() = (%d, %d), want (%d, %d)",
					sel, gotTop, gotN, wantTop, wantN)
			}
			pickerCheckWindow(t, p, fmt.Sprintf("Select(%d)", sel))
		}
		// Out of range refuses and changes nothing.
		wantSel := p.Selection()
		wantTop, wantN := p.Visible()
		for _, bad := range []int{-1, -1000, pickerTestRows, pickerTestRows + 1, 1 << 20} {
			if ok := p.Select(bad); ok {
				t.Fatalf("Select(%d) = true, want false (out of range)", bad)
			}
			if got := p.Selection(); got != wantSel {
				t.Fatalf("Select(%d) moved the selection to %d, want %d unchanged", bad, got, wantSel)
			}
			if gotTop, gotN := p.Visible(); gotTop != wantTop || gotN != wantN {
				t.Fatalf("Select(%d) moved the window to (%d, %d), want (%d, %d) unchanged",
					bad, gotTop, gotN, wantTop, wantN)
			}
		}
	})

	t.Run("RowAt maps y to the row under it", func(t *testing.T) {
		p := NewPicker(pickerFixture(pickerTestRows))

		check := func(what string, y int) {
			t.Helper()
			top, count := p.Visible()
			wantRow, wantOK := pickerWantRowAt(y, top, count)
			gotRow, gotOK := p.RowAt(image.Pt(pickerLeft, y))
			if gotOK != wantOK {
				t.Fatalf("%s: RowAt(y=%d) ok = %v, want %v", what, y, gotOK, wantOK)
			}
			if wantOK && gotRow != wantRow {
				t.Fatalf("%s: RowAt(y=%d) = %d, want %d (top=%d)", what, y, gotRow, wantRow, top)
			}
		}

		// Unscrolled: top == 0.
		top, count := p.Visible()
		if top != 0 || count != pickerVisible {
			t.Fatalf("precondition: Visible() = (%d, %d), want (0, %d)", top, count, pickerVisible)
		}
		check("first visible row, first pixel line", pickerTop)
		check("first visible row, last pixel line", pickerTop+pickerLine-1)
		check("last visible row, first pixel line", pickerTop+(count-1)*pickerLine)
		check("last visible row, last pixel line", pickerTop+count*pickerLine-1)
		check("one line above the list", pickerTop-1)
		check("well above the list", 0)
		check("one line past the last visible row", pickerTop+count*pickerLine)
		check("far past the list", pickerTop+count*pickerLine+pickerLine*4)

		// Spelled out, so the expected values are visible and not only computed.
		if row, ok := p.RowAt(image.Pt(pickerLeft, pickerTop)); !ok || row != 0 {
			t.Fatalf("RowAt(y=%d) = (%d, %v), want (0, true)", pickerTop, row, ok)
		}
		if row, ok := p.RowAt(image.Pt(pickerLeft, 439)); !ok || row != pickerVisible-1 {
			t.Fatalf("RowAt(y=439) = (%d, %v), want (%d, true)", row, ok, pickerVisible-1)
		}
		if _, ok := p.RowAt(image.Pt(pickerLeft, 39)); ok {
			t.Fatalf("RowAt(y=39) reported a row, want none (above the list)")
		}
		if _, ok := p.RowAt(image.Pt(pickerLeft, 440)); ok {
			t.Fatalf("RowAt(y=440) reported a row, want none (past the window)")
		}

		// After scrolling, the mapping must track top rather than assume 0.
		if ok := p.Select(pickerTestRows - 1); !ok {
			t.Fatalf("Select(%d) = false", pickerTestRows-1)
		}
		top, count = p.Visible()
		wantTop := pickerTestRows - pickerVisible
		if top != wantTop || count != pickerVisible {
			t.Fatalf("after scrolling: Visible() = (%d, %d), want (%d, %d)",
				top, count, wantTop, pickerVisible)
		}
		check("scrolled, first visible row", pickerTop)
		check("scrolled, second visible row", pickerTop+pickerLine)
		check("scrolled, last visible row", pickerTop+(count-1)*pickerLine)
		check("scrolled, one line above the list", pickerTop-1)
		check("scrolled, one line past the last visible row", pickerTop+count*pickerLine)
		if row, ok := p.RowAt(image.Pt(pickerLeft, pickerTop)); !ok || row != wantTop {
			t.Fatalf("scrolled RowAt(y=%d) = (%d, %v), want (%d, true)", pickerTop, row, ok, wantTop)
		}
		if row, ok := p.RowAt(image.Pt(pickerLeft, pickerTop+(pickerVisible-1)*pickerLine)); !ok ||
			row != pickerTestRows-1 {
			t.Fatalf("scrolled RowAt on the last visible line = (%d, %v), want (%d, true)",
				row, ok, pickerTestRows-1)
		}

		// x is irrelevant: DD13 hit-tests on y alone.
		for _, y := range []int{pickerTop, pickerTop + 3*pickerLine, pickerTop + 24*pickerLine} {
			wantRow, wantOK := p.RowAt(image.Pt(pickerLeft, y))
			for _, x := range []int{0, 1, pickerLeft, 100, 320, 639} {
				gotRow, gotOK := p.RowAt(image.Pt(x, y))
				if gotRow != wantRow || gotOK != wantOK {
					t.Fatalf("RowAt(x=%d, y=%d) = (%d, %v), want (%d, %v) — x must not matter",
						x, y, gotRow, gotOK, wantRow, wantOK)
				}
			}
		}

		// A short list: a y inside the 25-line window but past the final row.
		short := NewPicker(pickerFixture(5))
		if row, ok := short.RowAt(image.Pt(pickerLeft, pickerTop+4*pickerLine)); !ok || row != 4 {
			t.Fatalf("short list RowAt on the last row = (%d, %v), want (4, true)", row, ok)
		}
		for _, i := range []int{5, 6, pickerVisible - 1} {
			y := pickerTop + i*pickerLine
			if y >= pickerTop+pickerVisible*pickerLine {
				t.Fatalf("test bug: y=%d is outside the window", y)
			}
			if row, ok := short.RowAt(image.Pt(pickerLeft, y)); ok {
				t.Fatalf("short list RowAt(y=%d) = (%d, true), want none (past the last row)", y, row)
			}
		}
	})

	t.Run("click path RowAt to Select to Choose", func(t *testing.T) {
		click := func(t *testing.T, p *Picker, y int) (int, bool) {
			t.Helper()
			row, ok := p.RowAt(image.Pt(pickerLeft+17, y))
			if !ok {
				return 0, false
			}
			if !p.Select(row) {
				t.Fatalf("Select(%d) after RowAt = false", row)
			}
			return p.Choose()
		}

		p := NewPicker(pickerFixture(pickerTestRows))
		for _, i := range []int{0, 1, 7, pickerVisible - 1} {
			got, ok := click(t, p, pickerTop+i*pickerLine)
			if !ok || got != i {
				t.Fatalf("click on window line %d: Choose() = (%d, %v), want (%d, true)", i, got, ok, i)
			}
			if sel := p.Selection(); sel != i {
				t.Fatalf("click on window line %d: Selection() = %d, want %d", i, sel, i)
			}
		}
		// After scrolling, the same window lines must reach different rows.
		if ok := p.Select(pickerTestRows - 1); !ok {
			t.Fatalf("Select(%d) = false", pickerTestRows-1)
		}
		top, _ := p.Visible()
		for _, i := range []int{0, 5, pickerVisible - 1} {
			want := top + i
			got, ok := click(t, p, pickerTop+i*pickerLine)
			if !ok || got != want {
				t.Fatalf("scrolled click on window line %d: Choose() = (%d, %v), want (%d, true)",
					i, got, ok, want)
			}
		}
	})

	t.Run("Choose respects choosability", func(t *testing.T) {
		rows := pickerFixture(pickerTestRows)
		rows[3].Choosable = false
		p := NewPicker(rows)

		if !p.Select(3) {
			t.Fatalf("Select(3) = false")
		}
		if got, ok := p.Choose(); ok {
			t.Fatalf("Choose() on an unchoosable row = (%d, true), want ok=false", got)
		}
		if !p.Select(4) {
			t.Fatalf("Select(4) = false")
		}
		if got, ok := p.Choose(); !ok || got != 4 {
			t.Fatalf("Choose() on a choosable row = (%d, %v), want (4, true)", got, ok)
		}
	})

	t.Run("SetUnusable demotes but keeps the row listed", func(t *testing.T) {
		p := NewPicker(pickerFixture(pickerTestRows))
		before := p.Rows()
		wantText := before[9].Text

		if !p.Select(9) {
			t.Fatalf("Select(9) = false")
		}
		if got, ok := p.Choose(); !ok || got != 9 {
			t.Fatalf("Choose() before SetUnusable = (%d, %v), want (9, true)", got, ok)
		}
		p.SetUnusable(9)
		if got, ok := p.Choose(); ok {
			t.Fatalf("Choose() after SetUnusable(9) = (%d, true), want ok=false", got)
		}
		if got := p.Len(); got != pickerTestRows {
			t.Fatalf("Len() after SetUnusable = %d, want %d — the row must stay listed",
				got, pickerTestRows)
		}
		after := p.Rows()
		if len(after) != pickerTestRows {
			t.Fatalf("len(Rows()) after SetUnusable = %d, want %d", len(after), pickerTestRows)
		}
		if after[9].Text != wantText {
			t.Fatalf("Rows()[9].Text = %q, want %q", after[9].Text, wantText)
		}
		if after[9].Choosable {
			t.Fatalf("Rows()[9].Choosable = true after SetUnusable(9), want false")
		}
		for i, r := range after {
			if i == 9 {
				continue
			}
			if !r.Choosable {
				t.Fatalf("SetUnusable(9) also demoted row %d", i)
			}
		}
		// Out of range is a safe no-op.
		for _, bad := range []int{-1, -1000, pickerTestRows, 1 << 20} {
			p.SetUnusable(bad)
		}
		if got := p.Len(); got != pickerTestRows {
			t.Fatalf("Len() after out-of-range SetUnusable = %d, want %d", got, pickerTestRows)
		}
		for i, r := range p.Rows() {
			if want := i != 9; r.Choosable != want {
				t.Fatalf("out-of-range SetUnusable changed row %d: Choosable = %v, want %v",
					i, r.Choosable, want)
			}
		}
	})

	t.Run("RowText marks exactly the selected row", func(t *testing.T) {
		p := NewPicker(pickerFixture(pickerTestRows))

		markers := func(t *testing.T, sel int) {
			t.Helper()
			for i := 0; i < p.Len(); i++ {
				got := p.RowText(i)
				want := "  " + fmt.Sprintf("map%02d.alm", i)
				if i == sel {
					want = "> " + fmt.Sprintf("map%02d.alm", i)
				}
				if got != want {
					t.Fatalf("selection %d: RowText(%d) = %q, want %q", sel, i, got, want)
				}
			}
		}

		markers(t, 0)
		p.Move(1)
		markers(t, 1)
		if !p.Select(30) {
			t.Fatalf("Select(30) = false")
		}
		markers(t, 30)
		p.Move(-3)
		markers(t, 27)

		// A short text is not padded: exactly the two-rune prefix plus the text.
		if got, want := p.RowText(0), "  map00.alm"; got != want {
			t.Fatalf("RowText(0) = %q, want %q (no padding)", got, want)
		}
		if n := utf8.RuneCountInString(p.RowText(0)); n != 2+len("map00.alm") {
			t.Fatalf("RowText(0) is %d runes, want %d — a short row must not be padded",
				n, 2+len("map00.alm"))
		}

		// A very long text is clipped to at most pickerCols runes in total,
		// prefix included, and the kept part is a prefix of the original.
		long := strings.Repeat("abcdefghij", 30) // 300 ASCII runes
		lp := NewPicker([]PickerRow{
			{Text: long, Choosable: true},
			{Text: long, Choosable: true},
		})
		for _, i := range []int{0, 1} {
			got := lp.RowText(i)
			if n := utf8.RuneCountInString(got); n > pickerCols {
				t.Fatalf("RowText(%d) is %d runes, want at most pickerCols = %d", i, n, pickerCols)
			}
			wantPrefix := "  "
			if i == lp.Selection() {
				wantPrefix = "> "
			}
			if !strings.HasPrefix(got, wantPrefix) {
				t.Fatalf("RowText(%d) = %q, want prefix %q", i, got, wantPrefix)
			}
			// A CLIPPED ROW ENDS IN THE CLIP MARK and the rest of it is a prefix of
			// the original (1032 return 1). The cut was silent before, so a row and a
			// longer row with the same first 104 runes drew identically and neither
			// said it had been cut.
			body := strings.TrimPrefix(got, wantPrefix)
			if !strings.HasSuffix(body, clipMark) {
				t.Fatalf("RowText(%d) body %q does not end in the clip mark %q",
					i, body, clipMark)
			}
			if !strings.HasPrefix(long, strings.TrimSuffix(body, clipMark)) {
				t.Fatalf("RowText(%d) body %q is not a prefix of the row text",
					i, body)
			}
		}
		// The clip must actually bite: 2 + 300 runes cannot fit in 104.
		if n := utf8.RuneCountInString(lp.RowText(0)); n != pickerCols {
			t.Fatalf("RowText on a 300-rune row is %d runes, want exactly pickerCols = %d",
				n, pickerCols)
		}
	})
}

// TestPickerHeaderText is SC-16 / AC-13 / FR-2a: the screen must say how long the
// list is and which part of it is showing.
//
// The numbers are read back OUT of the header and compared against Visible(),
// not against literals. That is the whole point of the criterion: a header that
// stated a range of its own would be free to disagree with the rows the draw
// path renders and RowAt accepts, which is the drift the shared Visible() exists
// to prevent. A literal-based assertion would pass a header computed from
// anything at all.
func TestPickerHeaderText(t *testing.T) {
	// wantExtent is this test's own transcription of the header's contract:
	// 1-based first row, 1-based last row, total.
	wantExtent := func(p *Picker) string {
		top, n := p.Visible()
		if p.Len() == 0 {
			return "no maps"
		}
		return fmt.Sprintf("maps %d-%d of %d", top+1, top+n, p.Len())
	}

	check := func(t *testing.T, p *Picker, what string) {
		t.Helper()
		got := p.HeaderText()
		if want := wantExtent(p); !strings.Contains(got, want) {
			t.Fatalf("%s: header %q does not carry %q", what, got, want)
		}
		if !strings.Contains(got, pickerTitle) {
			t.Errorf("%s: header %q does not name the screen", what, got)
		}
		if n := utf8.RuneCountInString(got); n > pickerCols {
			t.Errorf("%s: header is %d runes, want at most pickerCols = %d", what, n, pickerCols)
		}
	}

	t.Run("a list longer than the window states its whole extent", func(t *testing.T) {
		p := NewPicker(pickerFixture(pickerTestRows))

		// Top of the list: rows 1..25 of 38, and the total is what makes the
		// difference between "a 25-row install" and "a 38-row list showing 25".
		check(t, p, "at the top")
		if got := p.HeaderText(); !strings.Contains(got, "of 38") {
			t.Fatalf("at the top: header %q does not state the total of 38", got)
		}
		if strings.Contains(p.HeaderText(), "1-38") {
			t.Fatalf("at the top: header %q claims to be showing all 38 rows", p.HeaderText())
		}

		// Mid-scroll and at the bottom: the range tracks the window.
		p.Select(20)
		check(t, p, "mid-scroll")
		p.Select(pickerTestRows - 1)
		check(t, p, "at the bottom")
		if top, n := p.Visible(); top+n != pickerTestRows {
			t.Fatalf("fixture: at the bottom Visible() = (%d, %d), want the window to end at %d",
				top, n, pickerTestRows)
		}

		// ...and back up, so the range is not one-way.
		p.Select(0)
		check(t, p, "back at the top")
	})

	t.Run("a list that fits reports its whole self", func(t *testing.T) {
		p := NewPicker(pickerFixture(pickerVisible - 3))
		check(t, p, "short list")
		if got, want := p.HeaderText(), fmt.Sprintf("maps 1-%d of %d", pickerVisible-3, pickerVisible-3); !strings.Contains(got, want) {
			t.Fatalf("short list: header %q does not carry %q", got, want)
		}
	})

	t.Run("an empty list reports no range", func(t *testing.T) {
		p := NewPicker(nil)
		got := p.HeaderText()
		if strings.Contains(got, "-") || strings.Contains(got, " of ") {
			t.Fatalf("empty list: header %q reports a range over no rows", got)
		}
		check(t, p, "empty list")
	})
}
