package ui

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"image"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/words"
)

// placePlayersSizes are the editor window sizes the place palette, the
// Players window, the player window and the delete question hold the
// layout properties at.
var placePlayersSizes = []image.Point{{1920, 1080}}

// ninePlayers are the most players a shipped map has (scenario 90).
var ninePlayers = []string{"Self", "Monsters", "Beists", "Nocturnal", "Villagers", "Friends", "Enemies", "Catapult", "Dragon"}

// placePaletteHost offers the Units tab two blocks of unit classes as large
// as the shipped data's, 72 and 233 cells, and the map's players as the
// Add window's owner step, "slot: name".
type placePaletteHost struct {
	*modesFull
	blocks [2]string // the blocks' names, as the game's words name them
}

func (h *placePaletteHost) NewRecordStep(r InspectionNewRecord, key string) InspectionNewStep {
	switch {
	case key == addOwner.Key:
		var out InspectionNewStep
		for i, name := range h.players {
			out.Choices = append(out.Choices, InspectionChoice{Value: int64(i + 1), Label: fmt.Sprintf("%d: %s", i+1, name)})
		}
		return out
	case r.Kind == KindUnit && key == "identity":
		step := InspectionNewStep{Value: 1}
		v := int64(0)
		block := func(name string, heads []string, n int) {
			for k, head := range heads {
				g := InspectionChoiceGroup{Block: name, Head: head}
				count := n / len(heads)
				if k == len(heads)-1 {
					count = n - count*(len(heads)-1)
				}
				for range count {
					v++
					step.Choices = append(step.Choices, InspectionChoice{Value: v, Label: fmt.Sprintf("%d: Unit class %d", v, v)})
					g.Values = append(g.Values, v)
				}
				step.Groups = append(step.Groups, g)
			}
		}
		block(h.blocks[0], []string{"Humans", "Goblins", "Orcs"}, 72)
		block(h.blocks[1], []string{"Story heroes", "Player characters", "Mercenaries", "Fighters", "Mages", "Riders"}, 233)
		return step
	}
	return h.modesFull.NewRecordStep(r, key)
}

// placePaletteEditor is the modes editor in lang at size with nine players
// and the Units tab open on its largest block.
func placePaletteEditor(t *testing.T, lang string, size image.Point) *MapEditor {
	t.Helper()
	return placePaletteBlocks(t, lang, size, "")
}

// placePaletteBlocks is placePaletteEditor with the second block named
// second, when it is not empty.
func placePaletteBlocks(t *testing.T, lang string, size image.Point, second string) *MapEditor {
	t.Helper()
	e, _ := newModesEditor(t, "full")
	e.SetWords(words.For(lang))
	f := e.doc.Edits.(*modesFull)
	f.players = slices.Clone(ninePlayers)
	host := &placePaletteHost{modesFull: f, blocks: [2]string{e.word("editor.unit.groups.creatures"), e.word("editor.unit.groups.people")}}
	if second != "" {
		host.blocks[1] = second
	}
	host.self, e.doc.Edits = host, host
	e.place = placeGrid{}
	e.Layout(size.X, size.Y)
	e.places[popList].placed = false
	chromeClick(e, barPoint(t, e, "units"))
	s := mustListSpec(t, e)
	if len(s.Tabs) != 2 {
		t.Fatalf("the Units tab shows %d block tabs, want 2", len(s.Tabs))
	}
	chromeClick(e, center(layoutPopup(s, e.font, e.bodyFont()).Tabs[1]))
	return e
}

// playersEditor is the modes editor in lang at size whose map holds names,
// each player's record labelled "slot: name" as the game's host labels it.
func playersEditor(t *testing.T, lang string, size image.Point, names []string) *MapEditor {
	t.Helper()
	e, _ := newModesEditor(t, "full")
	e.SetWords(words.For(lang))
	f := e.doc.Edits.(*modesFull)
	f.players = slices.Clone(names)
	f.relations = map[[2]int]int64{{0, 1}: 2, {1, 0}: 0, {1, 2}: 2}
	d := f.doc(f.self)
	for i := range d.Records {
		if r := &d.Records[i]; r.Kind == KindPlayer {
			r.Label = fmt.Sprintf("%d: %s", r.Index+1, names[r.Index])
		}
	}
	e.doc.Records, e.relations = d.Records, relationMatrix{}
	e.Layout(size.X, size.Y)
	return e
}

// openPlayerWindow selects player index elsewhere than the Players window
// and returns its window.
func openPlayerWindow(t *testing.T, e *MapEditor, index int) popupSpec {
	t.Helper()
	at := slices.IndexFunc(e.doc.Records, func(r InspectionRecord) bool { return r.Kind == KindPlayer && r.Index == index })
	e.Select(at, false)
	s, _, ok := e.objectSpec()
	if !ok || e.barWindow() == "players" {
		t.Fatalf("no player window for player %d", index)
	}
	return s
}

// holdsWindowProperties holds window id's spec s at size to the layout
// properties (checkWindowProperties), and finds window id at the middle of
// each of its rows, buttons, tabs and grid cells through the editor's own
// press routing (chromeHit).
func holdsWindowProperties(t *testing.T, e *MapEditor, id editorPopupID, s popupSpec, size image.Point) {
	t.Helper()
	l := layoutPopup(s, e.font, e.bodyFont())
	if len(s.Rows) > 0 {
		checkWindowProperties(t, e, s, size)
	} else if !l.Rect.In(image.Rectangle{Max: size}) {
		// A question holds no rows: its notes and buttons are held here.
		t.Errorf("%v: the window stands at %v, outside the screen", size, l.Rect)
	}
	for _, r := range slices.Concat(l.Notes, l.Buttons) {
		if !r.In(l.Rect) {
			t.Errorf("%v: a note or button at %v lies outside the window %v", size, r, l.Rect)
		}
	}
	var rects []image.Rectangle
	for _, rr := range l.Rows {
		rects = append(rects, rr.Rect)
	}
	rects = slices.Concat(rects, l.Buttons, l.Tabs, l.GridCells)
	for _, r := range rects {
		if got := e.chromeHit(center(r)); got != id {
			t.Errorf("%v: a press at %v reaches window %d, want %d", size, center(r), got, id)
		}
	}
}

func TestPlacePaletteHoldsTheLayoutProperties(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, size := range placePlayersSizes {
			t.Run(fmt.Sprint(lang, size), func(t *testing.T) {
				e := placePaletteEditor(t, lang, size)
				s := mustListSpec(t, e)
				l := layoutPopup(s, e.font, e.bodyFont())
				if l.GridBar.Empty() {
					t.Errorf("%v: the 233-cell block shows every cell; the test needs a grid that pages", size)
				}
				holdsWindowProperties(t, e, popList, s, size)
			})
		}
	}
}

func TestPlayersWindowHoldsTheLayoutProperties(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, size := range placePlayersSizes {
			t.Run(fmt.Sprint(lang, size), func(t *testing.T) {
				e := playersEditor(t, lang, size, ninePlayers)
				if err := e.Key("players"); err != nil {
					t.Fatal(err)
				}
				s := mustListSpec(t, e)
				if s.Grid == nil || len(s.Grid.ColHeads) != 9 {
					t.Fatalf("%v: no nine-player matrix", size)
				}
				holdsWindowProperties(t, e, popList, s, size)
			})
		}
	}
}

func TestPlayerWindowHoldsTheLayoutProperties(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, size := range placePlayersSizes {
			t.Run(fmt.Sprint(lang, size), func(t *testing.T) {
				e := playersEditor(t, lang, size, ninePlayers)
				s := openPlayerWindow(t, e, 1)
				holdsWindowProperties(t, e, popObject, s, size)
			})
		}
	}
}

func TestPlayerDeleteQuestionHoldsTheLayoutProperties(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, size := range placePlayersSizes {
			t.Run(fmt.Sprint(lang, size), func(t *testing.T) {
				e := playersEditor(t, lang, size, ninePlayers)
				if err := e.Key("players"); err != nil {
					t.Fatal(err)
				}
				e.askPlayerDelete(1)
				s, ok := e.promptSpec()
				if !ok || s.Title != e.word("editor.players.deletetitle") {
					t.Fatalf("%v: no delete question", size)
				}
				holdsWindowProperties(t, e, popPrompt, s, size)
				// The notes wrap to the window's text width: each line is as
				// wide as the line its notes rectangle holds, and the
				// first line takes more than the old wrap left it.
				l := layoutPopup(s, e.font, e.bodyFont())
				widest := 0
				for i, line := range s.Notes {
					w := e.bodyFont().Advance(line)
					widest = max(widest, w)
					if w > l.Notes[i].Dx() {
						t.Errorf("%v: note line %q is %d wide in a %d wide line", size, line, w, l.Notes[i].Dx())
					}
				}
				if len(s.Notes) > 1 && widest <= s.Width-2*popupPad-60 {
					t.Errorf("%v: the notes wrap at %d, short of the window's text width %d", size, widest, s.Width-2*popupPad)
				}
			})
		}
	}
}

// Answer 1: the palette's owner row is one choice showing «1: Self», and
// its list of players opens inside the editor window.
func TestPlacePaletteOwnerRowIsOneChoiceWhoseListOpensInside(t *testing.T) {
	for _, size := range placePlayersSizes {
		e := placePaletteEditor(t, "ru", size)
		s := mustListSpec(t, e)
		at := e.placeOwnerAt(s)
		if at < 0 || s.Rows[at].Kind != popupChoice || s.Rows[at].Value != "1: Self" || len(s.Rows[at].Buttons) != 0 {
			t.Fatalf("%v: the owner row is not one choice reading 1: Self: %+v", size, s.Rows)
		}
		box := ownerBox(t, e)
		chromeClick(e, center(box))
		d, ok := e.placeOwnerDropdown()
		if !ok || !d.Open || d.List.Len() != 9 {
			t.Fatalf("%v: a press on the owner box opened no list of nine players", size)
		}
		if b := d.Bounds(e.bodyFont()); !b.In(image.Rect(0, 0, size.X, size.Y)) {
			t.Errorf("%v: the open list stands at %v, outside the window", size, b)
		}
		if e.chromeHit(center(d.listBox(e.bodyFont()).Row(0))) != popList {
			t.Errorf("%v: a press on the open list does not reach the palette", size)
		}
	}
}

// Answer 2: the palette keeps its 400 width, and a block tab whose name is
// wider than its share of that width takes a line of its own, so the tabs
// stand one under the other, each name whole.
func TestPlacePaletteBlockTabsStackInItsWidth(t *testing.T) {
	long := "Именованные персонажи, их спутники и наёмники"
	for _, tc := range []struct {
		lang, second string
		stacked      bool
	}{{"en", "", false}, {"ru", "", false}, {"en", long, true}, {"ru", long, true}} {
		lang, stacked := tc.lang, tc.stacked
		e := placePaletteBlocks(t, lang, image.Pt(1920, 1080), tc.second)
		s := mustListSpec(t, e)
		l := layoutPopup(s, e.font, e.bodyFont())
		if s.Width != 400 || len(l.Tabs) != 2 || (l.Tabs[0].Min.Y != l.Tabs[1].Min.Y) != stacked {
			t.Errorf("%s: width %d, tabs %v; want 400 wide, tabs stacked %v", lang, s.Width, l.Tabs, stacked)
		}
		for i, r := range l.Tabs {
			if w := e.bodyFont().Advance(s.Tabs[i]); w > r.Dx()-16 {
				t.Errorf("%s: tab %q is %d wide in a %d wide tab", lang, s.Tabs[i], w, r.Dx())
			}
		}
	}
}

// Answer 3: the matrix's cells are the palette's 44, and the window is as
// wide as the matrix at that side.
func TestPlayersMatrixCellsAreThePalettesSide(t *testing.T) {
	for _, names := range [][]string{ninePlayers, ninePlayers[:3]} {
		e := playersEditor(t, "ru", image.Pt(1920, 1080), names)
		if err := e.Key("players"); err != nil {
			t.Fatal(err)
		}
		s := mustListSpec(t, e)
		g, l := s.Grid, layoutPopup(s, e.font, e.bodyFont())
		if g.side() != gridCellSide || l.GridCells[1].Dx() != gridCellSide-2 {
			t.Fatalf("%d players: cells %d wide, want %d", len(names), l.GridCells[1].Dx()+2, gridCellSide)
		}
		if want := 2*popupPad + g.HeadW + len(names)*gridCellSide + widgetSpriteSize; s.Width != want {
			t.Errorf("%d players: the window is %d wide, the matrix asks %d", len(names), s.Width, want)
		}
	}
}

// Answer 4: the legend is a row a relation, each with its name, its
// picture and a (?) that says what it means; no instruction heads it.
func TestPlayersLegendRowsNameIconAndHelp(t *testing.T) {
	e := playersEditor(t, "ru", image.Pt(1920, 1080), ninePlayers[:3])
	if err := e.Key("players"); err != nil {
		t.Fatal(err)
	}
	s := mustListSpec(t, e)
	l := layoutPopup(s, e.font, e.bodyFont())
	var legend []string
	for _, rr := range l.Rows {
		row := s.Rows[rr.Index]
		if row.HeadOf > 0 {
			continue
		}
		if row.Kind == popupHeading {
			t.Errorf("the Players window holds the heading %q", row.Label)
			continue
		}
		v := len(legend)
		if v >= 3 || len(row.Icons) != 1 || row.Icons[0] != relationIcon(int64(v)) || row.Help == "" || rr.Help.Empty() {
			t.Fatalf("legend row %d: %+v", v, row)
		}
		legend = append(legend, row.Label+" ? "+row.Help)
	}
	want := []string{"Нейтралитет ? Враг после нападения.", "Вражда ? Враг с самого начала.", "Союз ? Нападение не меняет."}
	if !slices.Equal(legend, want) {
		t.Errorf("legend %q, want %q", legend, want)
	}
	if s.TitleHelp != e.word("editor.players.help") {
		t.Errorf("the title's (?) says %q", s.TitleHelp)
	}
}

// Answer 5: the delete cross is the word «x» from the words table, and its
// hover line says «Удалить игрока».
func TestPlayersDeleteCrossIsAWordWithItsTip(t *testing.T) {
	e := playersEditor(t, "ru", image.Pt(1920, 1080), ninePlayers[:3])
	if err := e.Key("players"); err != nil {
		t.Fatal(err)
	}
	s := mustListSpec(t, e)
	l := layoutPopup(s, e.font, e.bodyFont())
	crosses := 0
	for _, rr := range l.Rows {
		row := s.Rows[rr.Index]
		if row.HeadOf == 0 || len(row.Buttons) == 0 {
			continue
		}
		crosses++
		if row.Buttons[0] != e.word("editor.players.delete") || row.Buttons[0] != "x" || row.Tip != "Удалить игрока" {
			t.Fatalf("the cross reads %q with the tip %q", row.Buttons, row.Tip)
		}
		tip, ok := popupTipAt(image.Rect(0, 0, 1920, 1080), e.font, e.bodyFont(), s, l, center(rr.Button), true)
		if !ok || !slices.Equal(tip.Lines, []string{"Удалить игрока"}) {
			t.Fatalf("the cross's hover line %v %q", ok, tip.Lines)
		}
	}
	if crosses != 3 {
		t.Errorf("%d crosses for 3 players", crosses)
	}
}

// relationRows are the player window's rows under its Relations heading.
func relationRows(t *testing.T, e *MapEditor, s popupSpec) []popupRow {
	t.Helper()
	at := slices.IndexFunc(s.Rows, func(r popupRow) bool { return r.Label == e.word("editor.player.relations") })
	if at < 0 || s.Rows[at].Kind != popupHeading || s.Rows[at].Help != e.word("editor.player.relations.help") {
		t.Fatalf("the player window has no Relations heading with its (?): %+v", s.Rows)
	}
	var out []popupRow
	for _, r := range s.Rows[at+1:] {
		if r.Kind == popupHeading {
			break
		}
		out = append(out, r)
	}
	return out
}

// Answer 6: the player window lists its relation toward each other player
// and leaves out its relation to itself.
func TestPlayerWindowOmitsItsRelationToItself(t *testing.T) {
	e := playersEditor(t, "en", image.Pt(1920, 1080), ninePlayers)
	s := openPlayerWindow(t, e, 1)
	var labels []string
	for _, r := range relationRows(t, e, s) {
		labels = append(labels, r.Label)
	}
	want := []string{"1: Self", "3: Beists", "4: Nocturnal", "5: Villagers", "6: Friends", "7: Enemies", "8: Catapult", "9: Dragon"}
	if !slices.Equal(labels, want) {
		t.Errorf("relation rows %q, want %q", labels, want)
	}
}

// Answer 7: the control and each relation read as words, a relation with
// its picture, from the named values.
func TestPlayerWindowNamesControlAndRelationValues(t *testing.T) {
	e := playersEditor(t, "ru", image.Pt(1920, 1080), ninePlayers[:3])
	s := openPlayerWindow(t, e, 1)
	control := slices.IndexFunc(s.Rows, func(r popupRow) bool { return r.Label == e.word("editor.add.control") })
	if control < 0 || s.Rows[control].Value != "Компьютер" {
		t.Fatalf("the control row is %+v", s.Rows)
	}
	var got []string
	for _, r := range relationRows(t, e, s) {
		if len(r.Icons) != 1 {
			t.Errorf("relation %q shows %d pictures", r.Label, len(r.Icons))
		}
		got = append(got, r.Label+" = "+r.Value)
	}
	if want := []string{"1: Self = Нейтралитет", "3: Beists = Союз"}; !slices.Equal(got, want) {
		t.Errorf("relations %q, want %q", got, want)
	}
	if r := relationRows(t, e, s); r[1].Icons[0] != relationIcon(2) {
		t.Error("the ally relation shows another picture")
	}
}

// Answer 8: the player window's Data block is folded when the window
// opens, and a press on its heading opens it.
func TestPlayerWindowDataFoldIsClosedAtFirst(t *testing.T) {
	e := playersEditor(t, "en", image.Pt(1920, 1080), ninePlayers[:3])
	s := openPlayerWindow(t, e, 1)
	data := slices.IndexFunc(s.Rows, func(r popupRow) bool { return r.Label == e.word("editor.object.data") })
	if data < 0 || s.Rows[data].Fold != FoldClosed || data != len(s.Rows)-1 {
		t.Fatalf("the Data heading is %+v, rows after it %d", s.Rows[max(0, data)], len(s.Rows)-1-data)
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	for _, rr := range l.Rows {
		if rr.Index == data {
			chromeClick(e, center(rr.Rect))
		}
	}
	s, _, _ = e.objectSpec()
	data = slices.IndexFunc(s.Rows, func(r popupRow) bool { return r.Label == e.word("editor.object.data") })
	if data < 0 || s.Rows[data].Fold != FoldOpen || data == len(s.Rows)-1 {
		t.Errorf("a press on the Data heading left it %+v", s.Rows[max(0, data)])
	}
}

// The player window's title is the player's number and name, in the
// editor's language.
func TestPlayerWindowTitleIsTheNumberAndName(t *testing.T) {
	for lang, want := range map[string]string{"ru": "Игрок 2: Monsters", "en": "Player 2: Monsters"} {
		e := playersEditor(t, lang, image.Pt(1920, 1080), ninePlayers)
		if s := openPlayerWindow(t, e, 1); s.Title != want {
			t.Errorf("%s: title %q, want %q", lang, s.Title, want)
		}
	}
}

// The cut-label defect: at 1920x1080 every label of the player window shows
// whole, in Russian and English.
func TestPlayerWindowLabelsAreWholeAtFullHDSize(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		e := playersEditor(t, lang, image.Pt(1920, 1080), ninePlayers)
		s := openPlayerWindow(t, e, 1)
		body := e.bodyFont()
		l := layoutPopup(s, e.font, body)
		for _, rr := range l.Rows {
			row := s.Rows[rr.Index]
			if row.Label == "" || row.placed() {
				continue
			}
			room := rr.Label
			if row.Kind == popupHeading || room.Empty() {
				room = rr.Rect
				if !rr.Help.Empty() {
					room.Max.X = rr.Help.Min.X
				}
			}
			label := row.Label
			if row.Fold != FoldNone {
				label = rowFoldMarks[row.Fold] + " " + label
			}
			if w := body.Advance(label); w > room.Dx() {
				t.Errorf("%s: label %q is %d wide in a %d wide place", lang, label, w, room.Dx())
			}
		}
	}
}

// placePlayersBuilders are the builders of the four windows, by file.
var placePlayersBuilders = map[string][]string{
	"mapeditorplace.go":   {"placeGridSpec", "placeOwnerRow", "placeNoteRows", "placeOwnerDropdown"},
	"mapeditorplayers.go": {"relationGridSpec", "playerHeadRows", "relationFit", "playerFieldRow", "playerTitle", "controlName", "relationName", "playerDeleteSpec"},
}

// wordID is a words table id.
var wordID = regexp.MustCompile(`^[a-z]+(\.[a-z0-9]+)+$`)

// No builder of these windows holds an English word: every string literal
// is a words table id or holds no letter.
func TestPlacePlayersBuildersHoldNoEnglishLiteral(t *testing.T) {
	for file, names := range placePlayersBuilders {
		f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		found := 0
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !slices.Contains(names, fn.Name.Name) {
				continue
			}
			found++
			// A literal compared with == or != is a key the code tests, not
			// a word it shows.
			keys := map[ast.Expr]bool{}
			ast.Inspect(fn, func(n ast.Node) bool {
				if b, ok := n.(*ast.BinaryExpr); ok && (b.Op == token.EQL || b.Op == token.NEQ) {
					keys[b.X], keys[b.Y] = true, true
				}
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING || keys[lit] {
					return true
				}
				s, err := strconv.Unquote(lit.Value)
				if err != nil || wordID.MatchString(s) || !strings.ContainsFunc(s, func(r rune) bool { return r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' }) {
					return true
				}
				t.Errorf("%s: %s holds the literal %q", file, fn.Name.Name, s)
				return true
			})
		}
		if found != len(names) {
			t.Errorf("%s: %d of the builders %q found", file, found, names)
		}
	}
}

// An open owner list is the topmost element: with the player's window moved
// under it, a press on a drawn list row chooses that player and leaves the
// window as it was.
func TestOwnerListTakesAPressOverAnObjectWindowMovedUnderIt(t *testing.T) {
	e := placePaletteEditor(t, "en", image.Pt(1920, 1080))
	openPlayerWindow(t, e, 1)
	chromeClick(e, center(ownerBox(t, e)))
	d, ok := e.placeOwnerDropdown()
	if !ok || !d.Open {
		t.Fatal("a press on the owner box opened no list")
	}
	row := center(d.listBox(e.bodyFont()).Row(3))
	e.places[e.storeRow(popObject)] = popupStanding{at: row.Sub(image.Pt(20, 20)), placed: true, slack: true}
	object, _, ok := e.objectSpec()
	if !ok || !row.In(layoutPopup(object, e.font, e.bodyFont()).Rect) {
		t.Fatalf("the player's window does not stand under the list row at %v", row)
	}
	if got := e.chromeHit(row); got != popList {
		t.Fatalf("a press on a list row reaches window %d, want %d", got, popList)
	}
	before := object.Title
	chromeClick(e, row)
	if _, open := e.placeOwnerDropdown(); open {
		t.Error("the press left the list open")
	}
	s := mustListSpec(t, e)
	if got := s.Rows[e.placeOwnerAt(s)].Value; got != "4: Nocturnal" {
		t.Errorf("the owner is %q after the press, want 4: Nocturnal", got)
	}
	if after, _, ok := e.objectSpec(); !ok || after.Title != before {
		t.Errorf("the player's window changed: %q, want %q", after.Title, before)
	}
}

// The Players window shows every row of its matrix at 1920x1080 for 2, 9
// and 16 players, and the rows the room holds are fitted by fitGrid alone:
// the window's source decides no row count of its own.
func TestPlayersWindowRowsAreFittedByTheGridFitterOnly(t *testing.T) {
	for _, n := range []int{2, 9, 16} {
		names := make([]string, n)
		for i := range names {
			names[i] = fmt.Sprintf("Player%d", i+1)
		}
		e := playersEditor(t, "en", image.Pt(1920, 1080), names)
		if err := e.Key("players"); err != nil {
			t.Fatal(err)
		}
		s := mustListSpec(t, e)
		if l := layoutPopup(s, e.font, e.bodyFont()); l.GridRows != n || !l.GridBar.Empty() {
			t.Errorf("%d players: the matrix shows %d rows, bar %v; want all %d", n, l.GridRows, l.GridBar, n)
		}
	}
	src, err := os.ReadFile("mapeditorplayers.go")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(src, []byte("Rows--")) || bytes.Contains(src, []byte("Rows -= ")) {
		t.Error("mapeditorplayers.go lowers a grid's row count by its own loop; fitGrid decides the rows")
	}
}

// With 16 players, and with a 32-letter name typed, the Players window at
// 1920x1080 stands inside the screen and its presses reach its cells.
func TestPlayersWindowHoldsSixteenPlayersAndALongName(t *testing.T) {
	long := strings.Repeat("W", 32)
	for _, lang := range []string{"en", "ru"} {
		names := make([]string, 16)
		for i := range names {
			names[i] = fmt.Sprintf("Player%d", i+1)
		}
		names[5] = long
		e := playersEditor(t, lang, image.Pt(1920, 1080), names)
		if err := e.Key("players"); err != nil {
			t.Fatal(err)
		}
		s := mustListSpec(t, e)
		if s.Grid == nil || len(s.Grid.ColHeads) != 16 {
			t.Fatalf("%s: no sixteen-player matrix", lang)
		}
		holdsWindowProperties(t, e, popList, s, image.Pt(1920, 1080))
	}
}
