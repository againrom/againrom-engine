package ui

import (
	"bytes"
	"image"
	"image/color"
	"slices"
	"testing"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/terrain"
)

func TestMissionScreenColumnWidthIsGameData(t *testing.T) {
	for _, tc := range []struct {
		screen MissionScreen
		want   image.Point
	}{
		{MissionScreen{}, image.Pt(864, 768)},
		{MissionScreen{ColumnW: MissionPanelW}, image.Pt(864, 768)},
		{MissionScreen{ColumnW: 176}, image.Pt(848, 768)},
	} {
		v, err := NewViewer("column", bigGrid(), &terrain.Tileset{})
		if err != nil {
			t.Fatal(err)
		}
		v.SetMissionScreen(tc.screen)
		v.SetPackShown(false)
		v.SetSpellBookShown(false)
		v.Layout(MissionFrameW, MissionFrameH)
		if got := v.ViewportSize(); got != tc.want {
			t.Errorf("column %d: map view %v, want %v", tc.screen.ColumnW, got, tc.want)
		}
	}
}

// R2-ENGINE-588: the data says whether the opening dialogue stands on the
// screen or on the map view.
func TestMissionNoticeAreaIsGameData(t *testing.T) {
	for _, tc := range []struct {
		screen MissionScreen
		want   image.Point
	}{
		{MissionScreen{}, image.Pt(864, 768)},
		{MissionScreen{ColumnW: 176, NoticeOnScreen: true}, image.Pt(1024, 768)},
	} {
		v, err := NewViewer("notice", bigGrid(), &terrain.Tileset{})
		if err != nil {
			t.Fatal(err)
		}
		v.SetMissionScreen(tc.screen)
		v.Layout(MissionFrameW, MissionFrameH)
		if got := v.noticeArea(); got != tc.want {
			t.Errorf("%+v: notice area %v, want %v", tc.screen, got, tc.want)
		}
	}
}

// MISSION-VIEW-019: a new mission's view origin is the hero's cell less half
// the view's span in whole cells, whatever the cell's lift.
func TestSAVStartViewIsTheHeroCellLessHalfTheView(t *testing.T) {
	for _, g := range []terrain.Grid{bigGrid(), bigAltGrid()} {
		v, err := NewViewer("start", g, &terrain.Tileset{})
		if err != nil {
			t.Fatal(err)
		}
		v.SetSAVStartView(raisedCell)
		v.Layout(MissionFrameW, MissionFrameH)
		cols, rows := v.cam.ViewW/camera.CellSize, v.cam.ViewH/camera.CellSize
		wantX := float64((raisedCell.X - cols/2) * camera.CellSize)
		wantY := float64((raisedCell.Y-rows/2)*camera.CellSize) - v.projectionMinY()
		if v.cam.X != wantX || v.cam.Y != wantY || v.cam.Zoom != 1 {
			t.Errorf("altitudes %v: camera (%v,%v) zoom %v, want (%v,%v) zoom 1",
				g.Altitudes != nil, v.cam.X, v.cam.Y, v.cam.Zoom, wantX, wantY)
		}
	}
}

// MISSION-ROUTE-093, DIV-336: Cast is drawn from the empty-cell art and
// answers no hover for a selection that cannot cast, and is live for one that
// can.
func TestCastCellIsEmptyForASelectionThatCannotCast(t *testing.T) {
	active := color.RGBA{G: 0x80, A: 0xff}
	empty := color.RGBA{R: 0x80, A: 0xff}
	for _, caster := range []bool{false, true} {
		v := columnViewer(t)
		v.SetCommandPanelArt(&CommandPanelArt{
			Heads:    solidPic(160, 80, color.RGBA{B: 0x80, A: 0xff}),
			Active:   solidPic(160, 80, active),
			Disabled: solidPic(160, 80, empty),
		})
		v.words.Command[commandCellCast] = "CAST"
		ents := []MapEntity{panelEntity(1, "Warrior", 63, 100, 4, 4)}
		ents[0].CastCapable = caster
		v.SetEntities(ents)
		bar, ok := v.commandPanelBar()
		if !ok {
			t.Fatal("no command panel")
		}
		cell := commandCellRects(bar)[commandCellCast]
		mid := cell.Min.Add(cell.Size().Div(2))
		got := composeCommandPanel(v, bar).RGBAAt(mid.X-bar.Min.X, mid.Y-bar.Min.Y)
		want := empty
		if caster {
			want = active
		}
		if got != want {
			t.Errorf("caster %v: Cast cell drawn %v, want %v", caster, got, want)
		}
		v.cursorX, v.cursorY, v.hasCursor = mid.X, mid.Y, true
		if hover := v.tooltipTarget().kind == tooltipCommand; hover != caster {
			t.Errorf("caster %v: Cast cell answers a hover %v", caster, hover)
		}
	}
}

// MISSION-ROUTE-093, R2-SESSION-221: the install's `~~` draws as one tilde in
// both games' hover texts.
func TestHoverTextDrawsTheTildeEscapeAsOneTilde(t *testing.T) {
	var w Words
	w.Hover[0] = "Open Inventory <~~>,<I>"
	got := mainTooltip(w, 0, "corner", nil).lines
	if want := []string{"Open Inventory <~>,<I>"}; !slices.Equal(got, want) {
		t.Fatalf("hover lines %q, want %q", got, want)
	}
}

// A new mission's minimap shows its background and view outline alone while
// the opening notice is up, in both games' shots (the first game's B01, the
// second game's k2.01), and its explored cells once the notice closes.
func TestMinimapHoldsItsContentUnderTheOpeningNotice(t *testing.T) {
	newView := func(fog uint8) *Viewer {
		v, err := NewViewer("hold", grid(8, 8, 1, 2, 3, 4, 5, 6, 7, 8), &terrain.Tileset{})
		if err != nil {
			t.Fatal(err)
		}
		plane := make([]byte, 64)
		for i := range plane {
			plane[i] = fog
		}
		v.SetFog(plane, 8, 8)
		return v
	}
	picture := func(v *Viewer) []byte {
		pic, _, ok := v.minimapPresent()
		if !ok {
			t.Fatal("no minimap")
		}
		return pic.Pix
	}
	blank := picture(newView(FogUnseen))
	v := newView(FogVisible)
	open := picture(v)
	if bytes.Equal(open, blank) {
		t.Fatal("fixture: an explored minimap draws as a blank one")
	}
	v.SetNotice("opening", NoticeDialogue)
	v.HoldMinimapContent()
	if !bytes.Equal(picture(v), blank) {
		t.Error("held minimap shows explored cells")
	}
	v.ReleaseMinimapContent()
	if !bytes.Equal(picture(v), blank) {
		t.Error("the hold ended with the notice still open")
	}
	v.ClearNotice()
	if !bytes.Equal(picture(v), open) {
		t.Error("the minimap stays blank after the notice closed")
	}
}

// R2-ASSET-148, R2-ENGINE-588: the second game opens with its hero selected;
// SelectUnit takes only a selectable unit the local participant owns.
func TestSelectUnitTakesOneOwnedUnit(t *testing.T) {
	v := panelViewer(t)
	v.SetLocalOwner(1)
	mine, theirs := panelEntity(1, "Hero", 50, 50, 1, 1), panelEntity(2, "Orc", 50, 50, 2, 2)
	mine.Owner, theirs.Owner = 1, 2
	v.SetEntities([]MapEntity{mine, theirs})
	if v.SelectUnit(2) || len(v.sel) != 0 {
		t.Fatalf("a foreign unit was selected: %v", v.sel)
	}
	if !v.SelectUnit(1) || !slices.Equal(v.sel, selection{1}) {
		t.Fatalf("the hero was not selected: %v", v.sel)
	}
}
