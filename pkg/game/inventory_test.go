package game

import (
	"encoding/binary"
	"image"
	"image/color"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const (
	partyFigureDir  = data.FigureDirManFighter
	partyFigureFace = 1
)

// invBaseSheet is a 2x2 figure base, every pixel opaque at palette index 1 —
// RED — so a test can tell "the base painted" from "nothing painted" without
// decoding anything.
func invBaseSheet() []byte {
	pal := make([]color.RGBA, 2)
	pal[1] = color.RGBA{R: 0xff}
	return synth.Sheet256(synth.Sheet256Options{
		Palette: pal,
		Frames: []synth.Frame256{{
			Width: 2, Height: 2,
			Pixels: []synth.Pixel256{
				{Index: 1, Opaque: true}, {Index: 1, Opaque: true},
				{Index: 1, Opaque: true}, {Index: 1, Opaque: true},
			},
		}},
	})
}

func invLayerSheet() []byte {
	pal := make([]color.RGBA, 2)
	pal[1] = color.RGBA{G: 0xff}
	return synth.Sheet256(synth.Sheet256Options{
		Palette: pal,
		Frames: []synth.Frame256{{
			Width: 2, Height: 2,
			Pixels: []synth.Pixel256{
				{Index: 1, Opaque: true}, {},
				{}, {},
			},
		}},
	})
}

// invIconStream is a 1x1 .16a icon: one literal pixel at palette index 5,
// level 15 — full coverage, cursor_test.go's own "the palette entry survives
// premultiplication untouched" case — so the decoded pixel is the palette
// entry exactly and a test needs no coverage arithmetic of its own.
func invIconStream() []byte {
	const (
		opLiteral = 0 << 14
		index     = 5
		level     = 15
	)
	pal := make([]byte, 1024)
	pal[index*4+0] = 0x30 // on-disk B
	pal[index*4+1] = 0x20 // on-disk G
	pal[index*4+2] = 0x10 // on-disk R

	block := binary.LittleEndian.AppendUint16(nil, opLiteral|1)
	block = binary.LittleEndian.AppendUint16(block, uint16(index<<1)|uint16(level<<9))

	out := append([]byte(nil), pal...)
	out = binary.LittleEndian.AppendUint32(out, 1) // width
	out = binary.LittleEndian.AppendUint32(out, 1) // height
	out = binary.LittleEndian.AppendUint32(out, uint32(len(block)))
	out = append(out, block...)
	return binary.LittleEndian.AppendUint32(out, 1|0x80000000)
}

// packIconStream is invIconStream's own grammar with the palette entry
// supplied by the caller, so two codes' icons can be told apart by the
// colour they decode to without touching the sprite reader's own internals.
func packIconStream(r, g, b byte) []byte {
	const (
		opLiteral = 0 << 14
		index     = 5
		level     = 15
	)
	pal := make([]byte, 1024)
	pal[index*4+0] = b // on-disk B
	pal[index*4+1] = g // on-disk G
	pal[index*4+2] = r // on-disk R

	block := binary.LittleEndian.AppendUint16(nil, opLiteral|1)
	block = binary.LittleEndian.AppendUint16(block, uint16(index<<1)|uint16(level<<9))

	out := append([]byte(nil), pal...)
	out = binary.LittleEndian.AppendUint32(out, 1) // width
	out = binary.LittleEndian.AppendUint32(out, 1) // height
	out = binary.LittleEndian.AppendUint32(out, uint32(len(block)))
	out = append(out, block...)
	return binary.LittleEndian.AppendUint32(out, 1|0x80000000)
}

func TestInventoryGoldIconUsesTheDecodedMoneySprite(t *testing.T) {
	src := missionSource{inventoryGoldIconAddr: packIconStream(0xd0, 0xa0, 0x20)}
	got := loadInventoryGoldIcon(src)
	if got == nil || got.Bounds().Dx() != 1 || got.Bounds().Dy() != 1 {
		t.Fatalf("loadInventoryGoldIcon = %v, want the decoded graphics.res money frame", got)
	}
}

type countingSource struct {
	missionSource
	reads *int
}

func (s countingSource) ReadFile(name string) ([]byte, error) {
	*s.reads++
	return s.missionSource.ReadFile(name)
}

// invWeapon is one weapon fixture whose code addresses deterministic,
// distinct strings — the VALUES of its four fields are arbitrary; nothing
// here reads weaponItemClass back out of them, so the fixture does not need
// to agree with it.
func invWeapon() *data.Weapon {
	return &data.Weapon{Name: "Fixture Blade",
		Code: data.ItemCode(uint16(3)<<12 | uint16(1)<<8 | uint16(2)<<5 | uint16(9))}
}

// invMission builds a *Mission with exactly one party member holding w (nil
// for bare) at entity id 42 — buildInventorySubject's own pairing of
// ms.Party[0] with ms.Start.IDs[0], exercised over the shortest fixture that
// carries one of each.
func invMission(w *data.Weapon) *Mission {
	return &Mission{
		Party: []mapload.PartyMember{{Weapon: w}},
		Start: mapload.Start{IDs: []sim.EntityID{42}},
	}
}

// invArchive lays base, layer and icon down at the addresses
// buildInventorySubject itself composes for invWeapon's code — the same
// functions the production code calls, so a fixture that does not match a
// real change to the address composition fails at the READ, not silently.
// A nil argument omits that entry, which is how the "unread" cases below are
// built.
func invArchive(base, layer, icon []byte) missionSource {
	code := invWeapon().Code
	src := missionSource{}
	if base != nil {
		src[graphicsPrefix+data.ItemFigureBasePath(partyFigureDir, partyFigureFace)] = base
	}
	if layer != nil {
		src[graphicsPrefix+data.ItemFigureLayerPath(partyFigureDir, code)] = layer
	}
	if icon != nil {
		src[graphicsPrefix+data.ItemIconPath(code)] = icon
	}
	return src
}

func TestBuildInventorySubjectComposesTheFigureAndTheFirstSlot(t *testing.T) {
	src := invArchive(invBaseSheet(), invLayerSheet(), invIconStream())
	subject, unread := buildInventorySubject(src, invMission(invWeapon()))

	if len(unread) != 0 {
		t.Fatalf("unread = %v, want none — every address is in the archive", unread)
	}
	if subject.ID != 42 {
		t.Errorf("ID = %d, want 42 (Start.IDs[0])", subject.ID)
	}
	if subject.Figure == nil {
		t.Fatal("Figure is nil with a readable base")
	}
	if b := subject.Figure.Bounds(); b.Dx() != 2 || b.Dy() != 2 {
		t.Fatalf("Figure is %v, want the base's own 2x2", b)
	}
	// (0,0) is the layer's own green, painted over the base's red; every other
	// pixel is the base's red, untouched.
	if got := subject.Figure.RGBAAt(0, 0); got.G != 0xff || got.R != 0 {
		t.Errorf("(0,0) = %+v, want the layer's green painted over the base", got)
	}
	for _, p := range [][2]int{{1, 0}, {0, 1}, {1, 1}} {
		if got := subject.Figure.RGBAAt(p[0], p[1]); got.R != 0xff || got.G != 0 {
			t.Errorf("(%d,%d) = %+v, want the base's own red untouched", p[0], p[1], got)
		}
	}
	if subject.Slots[0] == nil {
		t.Fatal("Slots[0] is nil with a readable icon")
	}
	if got := subject.Slots[0].RGBAAt(0, 0); got.R != 0x10 || got.G != 0x20 || got.B != 0x30 || got.A != 0xff {
		t.Errorf("Slots[0] pixel = %+v, want the icon's palette entry at full coverage", got)
	}
	for i := 1; i < len(subject.Slots); i++ {
		if subject.Slots[i] != nil {
			t.Errorf("Slots[%d] is non-nil; this fixture's own equipment occupies slot 1 alone", i)
		}
	}
}

func TestBuildInventorySubjectIsBareWithNoWeapon(t *testing.T) {
	src := invArchive(invBaseSheet(), invLayerSheet(), invIconStream())
	subject, unread := buildInventorySubject(src, invMission(nil))

	if len(unread) != 0 {
		t.Fatalf("unread = %v, want none — a bare hero attempts only the base, which reads fine", unread)
	}
	if subject.Figure == nil {
		t.Fatal("Figure is nil for a bare hero; the base sheet alone is still the figure")
	}
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			if got := subject.Figure.RGBAAt(x, y); got.R != 0xff || got.G != 0 {
				t.Errorf("(%d,%d) = %+v, want the base's own red with no layer painted", x, y, got)
			}
		}
	}
	for i, s := range subject.Slots {
		if s != nil {
			t.Errorf("Slots[%d] is non-nil for a bare hero", i)
		}
	}
}

func TestBuildInventorySubjectIsTotalOverNoFirstMember(t *testing.T) {
	src := invArchive(invBaseSheet(), invLayerSheet(), invIconStream())
	for _, tc := range []struct {
		name string
		ms   *Mission
	}{
		{"no party at all", &Mission{}},
		{"a party but no ids", &Mission{Party: []mapload.PartyMember{{Weapon: invWeapon()}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			subject, unread := buildInventorySubject(src, tc.ms)
			if unread != nil {
				t.Errorf("unread = %v, want nil — nothing was attempted", unread)
			}
			// FIELD BY FIELD AND NOT BY EQUALITY (0140): the pack became a
			// slice when the bar started scrolling, so ui.InventorySubject is
			// no longer comparable and `!= (ui.InventorySubject{})` no longer
			// compiles. The claim is unchanged — nothing was attempted, so
			// every field is still its own zero.
			if subject.ID != 0 || subject.Figure != nil || subject.Pack != nil || subject.PackCount != nil {
				t.Errorf("subject = %+v, want the zero value", subject)
			}
			for i, s := range subject.Slots {
				if s != nil {
					t.Errorf("subject.Slots[%d] is non-nil, want the zero value", i)
				}
			}
		})
	}
}

// D-11: each address that will not read is reported, in address order, and
// leaves the rest of the picture whole — the base, the layer and the icon in
// turn, plus a payload the sprite reader refuses outright.
func TestBuildInventorySubjectReportsEachUnreadAddressAndLeavesTheRestWhole(t *testing.T) {
	w := invWeapon()
	baseAddr := graphicsPrefix + data.ItemFigureBasePath(partyFigureDir, partyFigureFace)
	layerAddr := graphicsPrefix + data.ItemFigureLayerPath(partyFigureDir, w.Code)
	iconAddr := graphicsPrefix + data.ItemIconPath(w.Code)

	t.Run("the base is not in the archive", func(t *testing.T) {
		src := invArchive(nil, invLayerSheet(), invIconStream())
		subject, unread := buildInventorySubject(src, invMission(w))
		if len(unread) != 1 || unread[0] != baseAddr {
			t.Fatalf("unread = %v, want exactly [%q]", unread, baseAddr)
		}
		if subject.Figure != nil {
			t.Error("Figure is non-nil with no base to build it from")
		}
		if subject.Slots[0] == nil {
			t.Error("Slots[0] is nil though the icon read fine — the base's own failure reached it")
		}
	})

	t.Run("the layer is not in the archive", func(t *testing.T) {
		src := invArchive(invBaseSheet(), nil, invIconStream())
		subject, unread := buildInventorySubject(src, invMission(w))
		if len(unread) != 1 || unread[0] != layerAddr {
			t.Fatalf("unread = %v, want exactly [%q]", unread, layerAddr)
		}
		if subject.Figure == nil {
			t.Fatal("Figure is nil though the base read fine")
		}
		if got := subject.Figure.RGBAAt(0, 0); got.R != 0xff || got.G != 0 {
			t.Errorf("(0,0) = %+v, want the base's own red with the layer left unpainted", got)
		}
		if subject.Slots[0] == nil {
			t.Error("Slots[0] is nil though the icon read fine")
		}
	})

	t.Run("the icon is not in the archive", func(t *testing.T) {
		src := invArchive(invBaseSheet(), invLayerSheet(), nil)
		subject, unread := buildInventorySubject(src, invMission(w))
		if len(unread) != 1 || unread[0] != iconAddr {
			t.Fatalf("unread = %v, want exactly [%q]", unread, iconAddr)
		}
		if subject.Figure == nil {
			t.Fatal("Figure is nil though the base and the layer both read fine")
		}
		if subject.Slots[0] != nil {
			t.Error("Slots[0] is non-nil with no icon to build it from")
		}
	})

	t.Run("a payload the sprite reader refuses, named by its address", func(t *testing.T) {
		src := invArchive(invBaseSheet(), invLayerSheet(), []byte("not a sprite"))
		subject, unread := buildInventorySubject(src, invMission(w))
		if len(unread) != 1 || unread[0] != iconAddr {
			t.Fatalf("unread = %v, want exactly [%q]", unread, iconAddr)
		}
		if subject.Figure == nil {
			t.Error("Figure is nil though the base and the layer both read fine")
		}
	})

	t.Run("none of the three read: all three in address order", func(t *testing.T) {
		src := invArchive(nil, nil, nil)
		_, unread := buildInventorySubject(src, invMission(w))
		want := []string{baseAddr, layerAddr, iconAddr}
		if len(unread) != len(want) {
			t.Fatalf("unread = %v, want %v", unread, want)
		}
		for i := range want {
			if unread[i] != want[i] {
				t.Errorf("unread[%d] = %q, want %q", i, unread[i], want[i])
			}
		}
	})
}

// A nil source reads nothing and reports every address it would have
// attempted rather than panicking on the sheet cache it never gets to build.
func TestBuildInventorySubjectRefusesANilSourceWithoutPanicking(t *testing.T) {
	w := invWeapon()
	want := []string{
		graphicsPrefix + data.ItemFigureBasePath(partyFigureDir, partyFigureFace),
		graphicsPrefix + data.ItemFigureLayerPath(partyFigureDir, w.Code),
		graphicsPrefix + data.ItemIconPath(w.Code),
	}

	subject, unread := buildInventorySubject(nil, invMission(w))
	if subject.ID != 42 {
		t.Errorf("ID = %d, want 42 — the id is read before the source is", subject.ID)
	}
	if subject.Figure != nil || subject.Slots[0] != nil {
		t.Errorf("subject = %+v, want no picture with no source to read", subject)
	}
	if len(unread) != len(want) {
		t.Fatalf("unread = %v, want %v", unread, want)
	}
	for i := range want {
		if unread[i] != want[i] {
			t.Errorf("unread[%d] = %q, want %q", i, unread[i], want[i])
		}
	}
}

func TestOpenMissionBuildsTheInventorySubjectWithoutMovingTheWorldsDigest(t *testing.T) {
	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)
	w, err := sim.NewWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, nil, nil)
	if err != nil {
		t.Fatalf("sim.NewWorld: %v", err)
	}
	before := w.Hash()

	ms := &Mission{Number: 1, Map: m, World: w,
		Party: []mapload.PartyMember{{Weapon: invWeapon()}},
		Start: mapload.Start{IDs: []sim.EntityID{7}}}
	src := invArchive(invBaseSheet(), invLayerSheet(), invIconStream())

	mw := openMission(ms, nil, nil, v, src, nil, nil)
	if mw == nil {
		t.Fatal("openMission returned nil")
	}
	if got := w.Hash(); got != before {
		t.Errorf("World.Hash() changed from %d to %d after openMission built the inventory subject", before, got)
	}
}

// D-11: the front end's own reporting surface. openMission discards the
// unread list (it writes to no stream); MissionLine — the headless report
// cmd/againrom already prints at "-check -mission N" — recomputes it and
// names what it found. missionFrontEnd's archive (frontend_test.go) carries
// no graphics.res at all, so the figure base is always unread here, which is
// exactly the criterion this test can fail: silence this reporting and the
// substring below stops appearing.
func TestMissionLineReportsUnreadInventoryArt(t *testing.T) {
	f := missionFrontEnd(t)
	line, err := f.MissionLine(10)
	if err != nil {
		t.Fatalf("MissionLine: %v", err)
	}
	baseAddr := graphicsPrefix + data.ItemFigureBasePath(partyFigureDir, partyFigureFace)
	if !strings.Contains(line, "inventory art unread") || !strings.Contains(line, baseAddr) {
		t.Errorf("line %q does not report the unread figure base %q", line, baseAddr)
	}
}

// invWornCode is a plain item code fixture, distinct from invWeapon's own
// code and from each other by field D alone — nothing below reads any of
// their other three fields, so the values need not resemble a real armour
// code (0136-armour-counts T5: spec FR-10b, FR-10c, FR-10d).
func invWornCode(d int) data.ItemCode {
	return data.ItemCode(uint16(4)<<12 | uint16(3)<<8 | uint16(2)<<5 | uint16(d))
}

// invLayerSheetColored is invLayerSheet's own grammar with the opaque
// pixel's colour supplied by the caller: every fixture built with it below
// paints the SAME (0,0) pixel, so which colour survives at the end is a
// direct read of the order composeInventorySubject painted them in.
func invLayerSheetColored(r, g, b byte) []byte {
	pal := make([]color.RGBA, 2)
	pal[1] = color.RGBA{R: r, G: g, B: b}
	return synth.Sheet256(synth.Sheet256Options{
		Palette: pal,
		Frames: []synth.Frame256{{
			Width: 2, Height: 2,
			Pixels: []synth.Pixel256{
				{Index: 1, Opaque: true}, {},
				{}, {},
			},
		}},
	})
}

// invSparseSheet builds one 2x2 layer with a single colour at the caller's
// opaque pixel indices. It lets the mage-cloak regression distinguish three
// composition relations at once: body over primary, primary visible through
// body transparency, and secondary over both.
func invSparseSheet(col color.RGBA, opaque ...int) []byte {
	pal := make([]color.RGBA, 2)
	pal[1] = col
	pixels := make([]synth.Pixel256, 4)
	for _, i := range opaque {
		pixels[i] = synth.Pixel256{Index: 1, Opaque: true}
	}
	return synth.Sheet256(synth.Sheet256Options{
		Palette: pal,
		Frames:  []synth.Frame256{{Width: 2, Height: 2, Pixels: pixels}},
	})
}

// HERO-FIGURE-059's mage sequence brackets the body with slot 8: its primary
// sheet is behind the body and its secondary sheet is the final foreground.
// Both shipped composers use that sequence, so this one fixture checks the
// inventory doll and the on-map selected-unit doll against the same pixels.
func TestMageCloakPrimaryIsBehindTheBodyAndSecondaryIsInFront(t *testing.T) {
	code := invWornCode(8)
	var eq data.Equipment
	eq.SetCode(8, code)
	dir := data.FigureDirWomanMage
	src := missionSource{
		graphicsPrefix + data.ItemFigureBasePath(dir, 1):              invSparseSheet(color.RGBA{R: 0xff}, 0),
		graphicsPrefix + data.ItemFigureLayerPath(dir, code):          invSparseSheet(color.RGBA{G: 0xff}, 0, 1),
		graphicsPrefix + data.ItemFigureSecondaryLayerPath(dir, code): invSparseSheet(color.RGBA{B: 0xff}, 2),
	}
	inventory, _ := composeInventorySubject(src, 7, eq, dir, 1)
	unit, unitMask := composeUnitFigure(src, eq, figureID{Dir: dir, Face: 1})

	for name, got := range map[string]*image.RGBA{"inventory": inventory.Figure, "unit": unit} {
		if got == nil {
			t.Fatalf("%s mage figure is nil", name)
		}
		if px := got.RGBAAt(0, 0); px.R != 0xff || px.G != 0 || px.B != 0 {
			t.Errorf("%s (0,0) = %+v, want body red over cloak primary", name, px)
		}
		if px := got.RGBAAt(1, 0); px.R != 0 || px.G != 0xff || px.B != 0 {
			t.Errorf("%s (1,0) = %+v, want cloak-primary green through transparent body", name, px)
		}
		if px := got.RGBAAt(0, 1); px.R != 0 || px.G != 0 || px.B != 0xff {
			t.Errorf("%s (0,1) = %+v, want cloak-secondary blue in final foreground", name, px)
		}
	}

	// THE MASK AGREES WITH THE PICTURE AT EVERY ONE OF THE SAME THREE PIXELS
	// (1005, item 1), for both composers: (0,0) is the BODY's own pixel, not
	// the cloak's, even though the cloak (slot 8) painted there first — the
	// mask names no slot, the same "the base claims nothing" rule the
	// draw-order test below states. (1,0) and (0,1) are the cloak's own
	// primary and secondary sheets respectively, and both are slot 8: a
	// paired slot's two sheets are one equipment slot, not two.
	for name, mask := range map[string]*ui.SlotMask{"inventory": inventory.SlotMask, "unit": unitMask} {
		if _, ok := mask.At(0, 0); ok {
			t.Errorf("%s mask.At(0,0) named a slot, want none — the body painted there, not the cloak", name)
		}
		if n, ok := mask.At(1, 0); !ok || n != 8 {
			t.Errorf("%s mask.At(1,0) = (%d,%v), want (8,true) — the cloak's own primary sheet", name, n, ok)
		}
		if n, ok := mask.At(0, 1); !ok || n != 8 {
			t.Errorf("%s mask.At(0,1) = (%d,%v), want (8,true) — the cloak's own secondary sheet", name, n, ok)
		}
	}
}

// TestClearingTheTopSlotExposesTheLowerSlotInBothPictureAndMask is the
// compositor half of 1005's suppressed-mask witness, added at the round-1
// landing.
//
// pkg/ui's two suppression tests witness how dollFigureSlotAt READS the mask
// pushed with a suppressed figure. This one witnesses what refreshDollDrag
// hands it: the suppressed picture and the suppressed mask come from one
// composeInventorySubject call over equipment with the dragged slot's code
// cleared, so clearing the top layer over a stack must expose the layer below
// in BOTH — DIV-085's stated property, that a hit test and a drawn pixel can
// never disagree.
//
// Its lever on the compositor is not a new one: making the mask keep the
// FIRST layer that paints a pixel instead of the last reddens three
// pre-existing tests as well as this one. What it adds is the lifted
// composition itself. No other test composes an equipment set with an
// occupied slot's code cleared, or asserts that the picture and the mask
// agree on which layer that exposes.
//
// Slots 1, 7 and 12 are stacked on pixel (0,0) in data.FigureDrawOrder — 12,
// then 7, then 1 — and slot 1 is cleared the way refreshDollDrag clears the
// dragged slot's own code. The picture must then show slot 7's own colour and
// the mask must name slot 7: not slot 1, which is lifted, and not nothing.
func TestClearingTheTopSlotExposesTheLowerSlotInBothPictureAndMask(t *testing.T) {
	codeA, codeB, codeC := invWornCode(1), invWornCode(2), invWornCode(3)
	var eq data.Equipment
	eq.SetCode(1, codeA)
	eq.SetCode(7, codeB)
	eq.SetCode(12, codeC)

	src := missionSource{
		graphicsPrefix + data.ItemFigureBasePath(partyFigureDir, partyFigureFace): invBaseSheet(),
		graphicsPrefix + data.ItemFigureLayerPath(partyFigureDir, codeA):          invLayerSheetColored(0, 0, 0xff),    // blue, slot 1, painted last
		graphicsPrefix + data.ItemFigureLayerPath(partyFigureDir, codeB):          invLayerSheetColored(0, 0xff, 0xff), // cyan, slot 7, painted second
		graphicsPrefix + data.ItemFigureLayerPath(partyFigureDir, codeC):          invLayerSheetColored(0xff, 0xff, 0), // yellow, slot 12, painted first
		graphicsPrefix + data.ItemIconPath(codeA):                                 packIconStream(0xaa, 0, 0),
		graphicsPrefix + data.ItemIconPath(codeB):                                 packIconStream(0, 0xaa, 0),
		graphicsPrefix + data.ItemIconPath(codeC):                                 packIconStream(0, 0, 0xaa),
	}

	// refreshDollDrag's own two lines: clone the equipment, clear the
	// dragged slot's code, recompose.
	lifted := eq
	lifted.SetCode(1, 0)
	composed, unread := composeInventorySubject(src, 42, lifted, partyFigureDir, partyFigureFace)

	if len(unread) != 0 {
		t.Fatalf("unread = %v, want none — the lifted slot's own sheet is simply not asked for", unread)
	}
	if composed.Figure == nil {
		t.Fatal("Figure is nil with a readable base")
	}
	if got := composed.Figure.RGBAAt(0, 0); got.R != 0 || got.G != 0xff || got.B != 0xff {
		t.Errorf("(0,0) = %+v, want slot 7's own cyan — slot 1 is lifted, so the layer under it shows", got)
	}
	if n, ok := composed.SlotMask.At(0, 0); !ok || n != 7 {
		t.Errorf("SlotMask.At(0,0) = (%d,%v), want (7,true) — the exposed layer, not the lifted one and not nothing", n, ok)
	}
}

func TestComposeInventorySubjectPaintsEveryOccupiedSlotInFigureDrawOrder(t *testing.T) {
	codeA, codeB, codeC := invWornCode(1), invWornCode(2), invWornCode(3)
	var eq data.Equipment
	eq.SetCode(1, codeA)
	eq.SetCode(7, codeB)
	eq.SetCode(12, codeC)

	src := missionSource{
		graphicsPrefix + data.ItemFigureBasePath(partyFigureDir, partyFigureFace): invBaseSheet(),
		graphicsPrefix + data.ItemFigureLayerPath(partyFigureDir, codeA):          invLayerSheetColored(0, 0, 0xff),    // blue, slot 1, painted last
		graphicsPrefix + data.ItemFigureLayerPath(partyFigureDir, codeB):          invLayerSheetColored(0, 0xff, 0xff), // cyan, slot 7, painted second
		graphicsPrefix + data.ItemFigureLayerPath(partyFigureDir, codeC):          invLayerSheetColored(0xff, 0xff, 0), // yellow, slot 12, painted first
		graphicsPrefix + data.ItemIconPath(codeA):                                 packIconStream(0xaa, 0, 0),
		graphicsPrefix + data.ItemIconPath(codeB):                                 packIconStream(0, 0xaa, 0),
		graphicsPrefix + data.ItemIconPath(codeC):                                 packIconStream(0, 0, 0xaa),
	}

	subject, unread := composeInventorySubject(src, 42, eq, partyFigureDir, partyFigureFace)

	if len(unread) != 0 {
		t.Fatalf("unread = %v, want none — every address is in the archive", unread)
	}
	if subject.Figure == nil {
		t.Fatal("Figure is nil with a readable base")
	}
	if got := subject.Figure.RGBAAt(0, 0); got.R != 0 || got.G != 0 || got.B != 0xff {
		t.Errorf("(0,0) = %+v, want slot 1's own blue — the weapon slot, painted last", got)
	}
	for _, p := range [][2]int{{1, 0}, {0, 1}, {1, 1}} {
		if got := subject.Figure.RGBAAt(p[0], p[1]); got.R != 0xff || got.G != 0 || got.B != 0 {
			t.Errorf("(%d,%d) = %+v, want the base's own red — no layer touches this pixel", p[0], p[1], got)
		}
	}

	// THE MASK NAMES SLOT 1 AT (0,0), THE SAME PIXEL THE COLOUR CHECK ABOVE
	// ALREADY PROVES SLOT 1 PAINTED LAST (1005, item 1) — mask.At and the
	// picture's own RGBAAt read the same paint order from two different
	// buffers built by the same calls. THE OTHER THREE PIXELS NAME NO SLOT
	// AT ALL: only the base ever touches them, and the base is not itself a
	// slot.
	if n, ok := subject.SlotMask.At(0, 0); !ok || n != 1 {
		t.Errorf("SlotMask.At(0,0) = (%d,%v), want (1,true) — the weapon slot, painted last", n, ok)
	}
	for _, p := range [][2]int{{1, 0}, {0, 1}, {1, 1}} {
		if _, ok := subject.SlotMask.At(p[0], p[1]); ok {
			t.Errorf("SlotMask.At(%d,%d) named a slot, want none — only the base touches this pixel", p[0], p[1])
		}
	}

	wantSlots := map[int]color.RGBA{
		0:  {R: 0xaa, A: 0xff},
		6:  {G: 0xaa, A: 0xff},
		11: {B: 0xaa, A: 0xff},
	}
	for i := range subject.Slots {
		want, occupied := wantSlots[i]
		if !occupied {
			if subject.Slots[i] != nil {
				t.Errorf("Slots[%d] is non-nil; only slots 1, 7 and 12 (indices 0, 6, 11) are occupied", i)
			}
			continue
		}
		if subject.Slots[i] == nil {
			t.Fatalf("Slots[%d] is nil though its own icon is in the archive", i)
		}
		if got := subject.Slots[i].RGBAAt(0, 0); got != want {
			t.Errorf("Slots[%d] pixel = %+v, want %+v — its own slot's icon, not another's", i, got, want)
		}
	}
}

func TestBuildInventorySubjectComposesFromTheMembersWornSet(t *testing.T) {
	wornCode := invWornCode(5)
	weapon := invWeapon()

	t.Run("a worn slot 1 is composed with no weapon at all", func(t *testing.T) {
		var worn [sim.EquipSlots]uint16
		worn[0] = uint16(wornCode)
		ms := &Mission{
			Party: []mapload.PartyMember{{Worn: worn}},
			Start: mapload.Start{IDs: []sim.EntityID{42}},
		}
		src := missionSource{
			graphicsPrefix + data.ItemFigureBasePath(partyFigureDir, partyFigureFace): invBaseSheet(),
			graphicsPrefix + data.ItemFigureLayerPath(partyFigureDir, wornCode):       invLayerSheet(),
			graphicsPrefix + data.ItemIconPath(wornCode):                              invIconStream(),
		}

		subject, unread := buildInventorySubject(src, ms)
		if len(unread) != 0 {
			t.Fatalf("unread = %v, want none — every address is in the archive", unread)
		}
		if subject.Slots[0] == nil {
			t.Fatal("Slots[0] is nil though the worn set's own slot 1 icon is in the archive")
		}
	})

	t.Run("a weapon fills slot 1 only when the worn set leaves it empty", func(t *testing.T) {
		ms := &Mission{
			Party: []mapload.PartyMember{{Weapon: weapon}}, // Worn is the zero value
			Start: mapload.Start{IDs: []sim.EntityID{42}},
		}
		src := invArchive(invBaseSheet(), invLayerSheet(), invIconStream())

		subject, unread := buildInventorySubject(src, ms)
		if len(unread) != 0 {
			t.Fatalf("unread = %v, want none — every address is in the archive", unread)
		}
		if subject.Slots[0] == nil {
			t.Fatal("Slots[0] is nil though the weapon's own fallback address is in the archive")
		}
	})

	t.Run("a worn slot 1 wins over a weapon when both are present", func(t *testing.T) {
		var worn [sim.EquipSlots]uint16
		worn[0] = uint16(wornCode)
		ms := &Mission{
			Party: []mapload.PartyMember{{Weapon: weapon, Worn: worn}},
			Start: mapload.Start{IDs: []sim.EntityID{42}},
		}
		src := missionSource{
			graphicsPrefix + data.ItemFigureBasePath(partyFigureDir, partyFigureFace): invBaseSheet(),
			graphicsPrefix + data.ItemFigureLayerPath(partyFigureDir, wornCode):       invLayerSheet(),
			graphicsPrefix + data.ItemIconPath(wornCode):                              invIconStream(),
			// The weapon's own layer and icon addresses are deliberately
			// absent: if the weapon were consulted at all despite slot 1
			// already being occupied, this compose would report them
			// unread, and it must not.
		}

		subject, unread := buildInventorySubject(src, ms)
		if len(unread) != 0 {
			t.Fatalf("unread = %v, want none — the worn set's own addresses are all this compose should attempt", unread)
		}
		if subject.Slots[0] == nil {
			t.Fatal("Slots[0] is nil though the worn set's own slot 1 icon is in the archive")
		}
	})
}

// TestBuildInventorySubjectPrefersCarryEquippedOverStaleWorn is C1's own
// witness (round-2 adversarial review, twelfth pass): a member who has
// already finished one mission carries the CURRENT truth in
// member.Carry.Equipped (mapload/carry.go's CarryParty), while member.Worn
// is frozen at whatever it read at the LAST assembly and is never written
// again — the same pair mapload.EquipmentFromParty and the sim world's own
// seed (mapload/start.go) already prefer Carry over Worn for. Before this
// pass buildInventorySubject read member.Worn directly, so the doll and its
// SlotMask disagreed with the entity's own live equipment for exactly such a
// member.
func TestBuildInventorySubjectPrefersCarryEquippedOverStaleWorn(t *testing.T) {
	staleCode := invWornCode(5) // what slot 1 held at the last assembly
	liveCode := invWornCode(9)  // what member.Carry.Equipped says now

	t.Run("the carried set names a DIFFERENT slot 1 item", func(t *testing.T) {
		var worn [sim.EquipSlots]uint16
		worn[0] = uint16(staleCode)
		var carried [sim.EquipSlots]uint16
		carried[0] = uint16(liveCode)
		ms := &Mission{
			Party: []mapload.PartyMember{{Worn: worn, Carry: &mapload.Carry{Equipped: carried}}},
			Start: mapload.Start{IDs: []sim.EntityID{42}},
		}
		// The stale code's own layer and icon addresses are deliberately
		// absent: if the frozen Worn array were consulted at all, this
		// compose would report them unread.
		src := missionSource{
			graphicsPrefix + data.ItemFigureBasePath(partyFigureDir, partyFigureFace): invBaseSheet(),
			graphicsPrefix + data.ItemFigureLayerPath(partyFigureDir, liveCode):       invLayerSheet(),
			graphicsPrefix + data.ItemIconPath(liveCode):                              invIconStream(),
		}

		subject, unread := buildInventorySubject(src, ms)
		if len(unread) != 0 {
			t.Fatalf("unread = %v, want none — the carried code's own addresses are all this compose should attempt", unread)
		}
		if subject.Slots[0] == nil {
			t.Fatal("Slots[0] is nil though the carried code's own icon is in the archive")
		}
	})

	t.Run("the carried set left slot 1 truly empty", func(t *testing.T) {
		var worn [sim.EquipSlots]uint16
		worn[0] = uint16(staleCode)
		ms := &Mission{
			// Carry is present and empty: the member sold or dropped the
			// item after finishing his last mission. Weapon is nil so the
			// slot-1 fallback cannot mask the outcome.
			Party: []mapload.PartyMember{{Worn: worn, Carry: &mapload.Carry{}}},
			Start: mapload.Start{IDs: []sim.EntityID{42}},
		}
		// The stale code's own addresses are absent from the archive, on
		// the same reasoning as the case above.
		src := invArchive(invBaseSheet(), nil, nil)

		subject, unread := buildInventorySubject(src, ms)
		if len(unread) != 0 {
			t.Fatalf("unread = %v, want none — slot 1 is empty, so no address is even attempted for it", unread)
		}
		if subject.Slots[0] != nil {
			t.Error("Slots[0] is non-nil though the carried set leaves slot 1 empty")
		}
	})
}

// TestBuildInventorySubjectWeaponFallbackStaysOffOnceMaterialized is
// counterexample A/B's own witness for buildInventorySubject (round-2
// adversarial review, fifth pass): a member whose worn set leaves slot 1
// empty and who still carries Weapon (the starting-weapon record is never
// cleared) draws no fallback at all once WeaponMaterialized is true — the
// weapon's own layer and icon addresses are never even composed, so an
// archive that does not carry them still reports no unread entries.
func TestBuildInventorySubjectWeaponFallbackStaysOffOnceMaterialized(t *testing.T) {
	weapon := invWeapon()
	src := invArchive(invBaseSheet(), nil, nil)
	ms := &Mission{
		Party: []mapload.PartyMember{{Weapon: weapon, WeaponMaterialized: true}},
		Start: mapload.Start{IDs: []sim.EntityID{42}},
	}

	subject, unread := buildInventorySubject(src, ms)
	if len(unread) != 0 {
		t.Fatalf("unread = %v, want none: a materialized weapon's own layer and icon addresses must not even be attempted", unread)
	}
	if subject.Slots[0] != nil {
		t.Error("Slots[0] is non-nil though WeaponMaterialized is true and the worn set leaves slot 1 empty")
	}
	if subject.WeaponFallback {
		t.Error("WeaponFallback is true though the weapon has already materialized")
	}
}

// packCode1, packCode2 and packCode3 are three item codes whose addresses
// (through data.ItemIconPath) are distinct, arbitrary sixteen-bit values —
// nothing below reads a field of any of them beyond the path it composes.
const (
	packCode1 = uint16(0x1101)
	packCode2 = uint16(0x1202)
	packCode3 = uint16(0x1303)
)

func TestBuildInventoryPackComposesOneIconPerCodeInCarriedOrder(t *testing.T) {
	addr1 := graphicsPrefix + data.ItemIconPath(data.ItemCode(packCode1))
	addr2 := graphicsPrefix + data.ItemIconPath(data.ItemCode(packCode2))
	src := missionSource{
		addr1: packIconStream(0xff, 0, 0),
		addr2: packIconStream(0, 0xff, 0),
	}

	pack, _, unread := buildInventoryPack(src,
		[]sim.ItemStack{{Code: packCode1, Count: 1}, {Code: packCode2, Count: 1}}, map[uint16]*image.RGBA{})

	if len(unread) != 0 {
		t.Fatalf("unread = %v, want none — both codes are in the archive", unread)
	}
	if len(pack) != 2 {
		t.Fatalf("len(pack) = %d, want 2 — one entry per carried element and nothing beyond", len(pack))
	}
	if pack[0] == nil || pack[1] == nil {
		t.Fatalf("pack = %+v, want both of the first two cells filled", pack)
	}
	if got := pack[0].RGBAAt(0, 0); got.R != 0xff || got.G != 0 {
		t.Errorf("pack[0] = %+v, want the first code's own red — carried order preserved", got)
	}
	if got := pack[1].RGBAAt(0, 0); got.G != 0xff || got.R != 0 {
		t.Errorf("pack[1] = %+v, want the second code's own green — carried order preserved", got)
	}
}

// 0140: the cap is gone. 0112 D-3 disclosed a WINDOW limit here — a carrier
// holding more elements than there were cells had the remainder undrawn — and
// the owner's scrolling bar removed the reason for one, so this builder now
// composes every element the container holds. Ten is well past the eight that
// used to be the ceiling, which is what makes this test the negation of the
// one it replaced rather than a restatement of it.
func TestBuildInventoryPackComposesEveryCarriedElementWithNoCap(t *testing.T) {
	const carried = 10
	stacks := make([]sim.ItemStack, carried)
	src := missionSource{}
	for i := range stacks {
		stacks[i] = sim.ItemStack{Code: uint16(0x1100 | (i + 1)), Count: 1}
		addr := graphicsPrefix + data.ItemIconPath(data.ItemCode(stacks[i].Code))
		src[addr] = packIconStream(byte(i), 0, 0)
	}

	pack, counts, unread := buildInventoryPack(src, stacks, map[uint16]*image.RGBA{})

	if len(unread) != 0 {
		t.Fatalf("unread = %v, want none — every one of the ten elements is in the archive", unread)
	}
	if len(pack) != carried || len(counts) != carried {
		t.Fatalf("len(pack), len(counts) = %d, %d, want %d each", len(pack), len(counts), carried)
	}
	for i := range pack {
		if pack[i] == nil {
			t.Errorf("pack[%d] is nil though code %d is in the archive", i, stacks[i].Code)
		}
	}
}

// A carrier of nothing answers nil rather than an empty slice, so a subject
// built for an empty container presents exactly as one never built at all.
func TestBuildInventoryPackAnswersNilForAnEmptyCarrier(t *testing.T) {
	pack, counts, unread := buildInventoryPack(missionSource{}, nil, map[uint16]*image.RGBA{})
	if pack != nil || counts != nil || unread != nil {
		t.Errorf("buildInventoryPack over no elements = (%v, %v, %v), want three nils", pack, counts, unread)
	}
}

// AC-10, D-11's own rule restated for the pack: each address that will not
// read is reported, in carried order, and every code that DOES read still
// composes.
func TestBuildInventoryPackReportsUnreadAddressesInCarriedOrder(t *testing.T) {
	addr1 := graphicsPrefix + data.ItemIconPath(data.ItemCode(packCode1))
	addr2 := graphicsPrefix + data.ItemIconPath(data.ItemCode(packCode2))
	addr3 := graphicsPrefix + data.ItemIconPath(data.ItemCode(packCode3))
	src := missionSource{addr2: packIconStream(0, 0xff, 0)} // only the middle code reads

	pack, _, unread := buildInventoryPack(src, []sim.ItemStack{
		{Code: packCode1, Count: 1}, {Code: packCode2, Count: 1}, {Code: packCode3, Count: 1},
	}, map[uint16]*image.RGBA{})

	want := []string{addr1, addr3}
	if len(unread) != len(want) {
		t.Fatalf("unread = %v, want %v", unread, want)
	}
	for i := range want {
		if unread[i] != want[i] {
			t.Errorf("unread[%d] = %q, want %q", i, unread[i], want[i])
		}
	}
	if pack[0] != nil {
		t.Error("pack[0] is non-nil with no icon to build it from")
	}
	if pack[1] == nil {
		t.Error("pack[1] is nil though its icon read fine")
	}
	if pack[2] != nil {
		t.Error("pack[2] is non-nil with no icon to build it from")
	}
}

// A nil src reads nothing and reports every address it would have attempted
// rather than panicking on the cache it never gets to consult —
// buildInventorySubject's own rule (LoadAttackPointer's, ultimately).
func TestBuildInventoryPackRefusesANilSourceWithoutPanicking(t *testing.T) {
	want := []string{
		graphicsPrefix + data.ItemIconPath(data.ItemCode(packCode1)),
		graphicsPrefix + data.ItemIconPath(data.ItemCode(packCode2)),
	}

	pack, _, unread := buildInventoryPack(nil,
		[]sim.ItemStack{{Code: packCode1, Count: 1}, {Code: packCode2, Count: 1}}, map[uint16]*image.RGBA{})

	for i, p := range pack {
		if p != nil {
			t.Errorf("pack[%d] is non-nil with no source to read", i)
		}
	}
	if len(unread) != len(want) {
		t.Fatalf("unread = %v, want %v", unread, want)
	}
	for i := range want {
		if unread[i] != want[i] {
			t.Errorf("unread[%d] = %q, want %q", i, unread[i], want[i])
		}
	}
}

func TestBuildInventoryPackCacheAvoidsRereadingAResolvedCode(t *testing.T) {
	addr := graphicsPrefix + data.ItemIconPath(data.ItemCode(packCode1))
	reads := 0
	src := countingSource{missionSource{addr: packIconStream(0xff, 0, 0)}, &reads}
	cache := map[uint16]*image.RGBA{}

	if _, _, unread := buildInventoryPack(src, []sim.ItemStack{{Code: packCode1, Count: 1}}, cache); len(unread) != 0 {
		t.Fatalf("unread = %v, want none", unread)
	}
	if reads != 1 {
		t.Fatalf("reads after the first compose = %d, want 1", reads)
	}

	if _, _, unread := buildInventoryPack(src, []sim.ItemStack{{Code: packCode1, Count: 1}}, cache); len(unread) != 0 {
		t.Fatalf("unread = %v, want none", unread)
	}
	if reads != 1 {
		t.Errorf("reads after a second compose of the same code = %d, want still 1 — the cache answered", reads)
	}
}

// 0138 D-10: the count beside the picture is the ELEMENT's own count, set
// whether or not its icon reads, and independent of any other cell's.
func TestBuildInventoryPackAnswersEachElementsOwnCount(t *testing.T) {
	addr1 := graphicsPrefix + data.ItemIconPath(data.ItemCode(packCode1))
	src := missionSource{addr1: packIconStream(0xff, 0, 0)} // packCode2 is left unread on purpose

	_, counts, unread := buildInventoryPack(src,
		[]sim.ItemStack{{Code: packCode1, Count: 3}, {Code: packCode2, Count: 70_000}}, map[uint16]*image.RGBA{})

	if len(unread) != 1 {
		t.Fatalf("unread = %v, want exactly one address — packCode2's own", unread)
	}
	if counts[0] != 3 {
		t.Errorf("counts[0] = %d, want 3", counts[0])
	}
	if counts[1] != 70_000 {
		t.Errorf("counts[1] = %d, want 70000 — an unread icon does not erase its element's own count "+
			"(0138 D-1's own 32-bit width)", counts[1])
	}
	if len(counts) != 2 {
		t.Errorf("len(counts) = %d, want 2 — only two elements were carried", len(counts))
	}
}

func TestHoverSlotMaskRetainsAnAuthoredPixelForACompletelyCoveredWornLayer(t *testing.T) {
	top := &ui.SlotMask{W: 3, H: 1, Slot: []uint8{2, 2, 0}}
	coverage := newFigureSlotCoverage(image.Rect(0, 0, 3, 1))
	// Slot 1 paints first, then slot 2 covers its whole two-pixel layer.
	for x := 0; x < 2; x++ {
		coverage.mark(x, 0, 1)
		coverage.mark(x, 0, 2)
	}
	hover := coverage.hoverMask(top)
	seen := [3]bool{}
	for _, slot := range hover.Slot {
		if slot < uint8(len(seen)) {
			seen[slot] = true
		}
	}
	if !seen[1] || !seen[2] {
		t.Fatalf("hover ownership = %v, want distinct authored pixels for covered slot 1 and top slot 2", hover.Slot)
	}
	if got := top.Slot; got[0] != 2 || got[1] != 2 || got[2] != 0 {
		t.Fatalf("interactive topmost mask mutated to %v", got)
	}
}
