package game

import (
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// layerCodes returns the example mod's three layer codes: gambeson, cloak,
// boots.
func layerCodes(t *testing.T, f *FrontEnd) (gambeson, cloak, boots uint16) {
	t.Helper()
	items := f.Table.Mods.Items
	if len(items) != 3 || items[0].Key != "gambeson" || items[1].Key != "cloak" || items[2].Key != "boots" {
		t.Fatalf("the mod items are %+v", items)
	}
	for _, it := range items {
		if it.Layer == 0 {
			t.Fatalf("%s is no layer", it.Key)
		}
	}
	return items[0].Code, items[1].Code, items[2].Code
}

// layerWitness checks what a town member who wears the three layers must show:
// the layers are on the member, the slots hold no mod code, the units are still
// in the pack, and the loadout sums the layers' defence (6+2+1) and absorption
// (1) on top of the same member without them.
func layerWitness(f *FrontEnd, gambeson, cloak, boots uint16) error {
	var member *mapload.PartyMember
	for i := range f.Carried {
		if f.Carried[i].ID == "hero" {
			member = &f.Carried[i]
		}
	}
	if member == nil {
		return errors.New("no hero")
	}
	want := []uint16{gambeson, cloak, boots}
	got := slices.Clone(member.Layers)
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		return fmt.Errorf("layers %v, want %v", member.Layers, want)
	}
	worn, pack := memberItemCodes(*member)
	for _, code := range worn {
		if code == gambeson || code == cloak || code == boots {
			return fmt.Errorf("a slot holds a layer item %#x", code)
		}
	}
	for _, code := range want {
		if !slices.Contains(pack, code) {
			return fmt.Errorf("the pack lost %#x, so its weight left the load", code)
		}
	}
	with := mapload.PartyLoadout(*member, f.Table)
	bare := *member
	bare.Layers = nil
	without := mapload.PartyLoadout(bare, f.Table)
	if d := with.Mod.Defence - without.Mod.Defence; d != 9 {
		return fmt.Errorf("the layers add defence %d, want 9", d)
	}
	if a := with.Mod.Absorption - without.Mod.Absorption; a != 1 {
		return fmt.Errorf("the layers add absorption %d, want 1", a)
	}
	return nil
}

func TestReleaseModLayersWearSaveAndLoad(t *testing.T) {
	f := modItemTown(t)
	gambeson, cloak, boots := layerCodes(t, f)
	s := currentTownShop(f, "hero")
	for _, code := range []uint16{gambeson, cloak, boots} {
		currentTownBuy(t, s, roomArmour, code)
	}
	wornBefore, _ := currentTownMember(t, f, "hero")
	if len(f.Carried[0].Layers) != 0 {
		t.Fatalf("buying put on layers %v", f.Carried[0].Layers)
	}
	for _, code := range []uint16{gambeson, cloak, boots} {
		s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: currentTownPackCell(t, s, code)}, ui.ShopControl{Kind: ui.ShopControlDoll})
	}
	if worn, _ := currentTownMember(t, f, "hero"); worn != wornBefore {
		t.Fatalf("wearing layers changed the slots: %v -> %v", wornBefore, worn)
	}
	if err := layerWitness(f, gambeson, cloak, boots); err != nil {
		t.Fatal(err)
	}

	raw := currentTownSave(t, f)
	if holding, _ := modItemObjects(t, raw); holding != 0 {
		t.Fatalf("%d item objects hold a mod code", holding)
	}
	g, err := modItemReload(t, raw, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := layerWitness(g, gambeson, cloak, boots); err != nil {
		t.Fatalf("after the cold LOAD: %v", err)
	}
	if worn, _ := currentTownMember(t, g, "hero"); worn != wornBefore {
		t.Fatalf("the LOAD changed the slots: %v -> %v", wornBefore, worn)
	}
	if _, err := modItemReload(t, raw, true); err == nil || !errors.Is(err, ErrModMark) {
		t.Fatalf("a LOAD without the mod: %v", err)
	}

	// Loss control: the same SAVE and LOAD with the layers taken off must fail
	// the witness.
	g.Carried[0].Layers = nil
	if err := layerWitness(g, gambeson, cloak, boots); err == nil {
		t.Fatal("the witness passes without layers")
	}
	h, err := modItemReload(t, currentTownSave(t, g), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Carried[0].Layers) != 0 || layerWitness(h, gambeson, cloak, boots) == nil {
		t.Fatalf("a SAVE without layers loaded with %v", h.Carried[0].Layers)
	}

	// A second wear of a worn layer takes it off.
	s2 := currentTownShop(f, "hero")
	s2.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: currentTownPackCell(t, s2, cloak)}, ui.ShopControl{Kind: ui.ShopControlDoll})
	if slices.Contains(f.Carried[0].Layers, cloak) || len(f.Carried[0].Layers) != 2 {
		t.Fatalf("layers after taking the cloak off: %v", f.Carried[0].Layers)
	}
}

// A layer sold and bought back is in the pack and not on the doll.
func TestReleaseModLayerBoughtBackIsNotWornAgain(t *testing.T) {
	f := modItemTown(t)
	_, cloak, _ := layerCodes(t, f)
	s := currentTownShop(f, "hero")
	currentTownBuy(t, s, roomArmour, cloak)
	s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: currentTownPackCell(t, s, cloak)}, ui.ShopControl{Kind: ui.ShopControlDoll})
	if !slices.Equal(f.Carried[0].Layers, []uint16{cloak}) {
		t.Fatalf("the cloak is not worn before the sale: %v", f.Carried[0].Layers)
	}
	stacks := s.shopPackStacks()
	i := slices.IndexFunc(stacks, func(st sim.ItemStack) bool { return st.Code == cloak })
	if i < 0 {
		t.Fatal("the pack lost the cloak")
	}
	s.shopFromPack(i, false)
	if act := s.shopSell(); !strings.HasPrefix(act.Msg, "he pays") {
		t.Fatalf("sale: %q", act.Msg)
	}
	if len(f.Carried[0].Layers) != 0 {
		t.Fatalf("the sold cloak is still worn: %v", f.Carried[0].Layers)
	}
	currentTownBuy(t, s, roomArmour, cloak)
	if !modItemPackHolds(f, cloak) {
		t.Fatal("the pack lacks the bought-back cloak")
	}
	if len(f.Carried[0].Layers) != 0 {
		t.Fatalf("the bought-back cloak is worn without a gesture: %v", f.Carried[0].Layers)
	}
	s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: currentTownPackCell(t, s, cloak)}, ui.ShopControl{Kind: ui.ShopControlDoll})
	if !slices.Equal(f.Carried[0].Layers, []uint16{cloak}) {
		t.Fatalf("the gesture does not wear the bought-back cloak: %v", f.Carried[0].Layers)
	}
}

// figureCompose composes the inventory figure of a fighter wearing eq and the
// layers.
func figureCompose(f *FrontEnd, eq data.Equipment, layers []uint16) ui.InventorySubject {
	subject, _ := composeInventorySubjectLayered(f.Archives.Containers, 1, eq, figureLayersFor(layers, f.Table),
		data.FigureDirManFighter, 1)
	return subject
}

func pixelAt(img *image.RGBA, x, y int) [4]uint8 {
	o := img.PixOffset(x, y)
	return [4]uint8{img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3]}
}

// layerOrderWitness composes a doll in which only the anchor item is worn, the
// layer alone, and both, and checks every pixel of the third against the
// painter's rule: an under layer shows only where the anchor item is
// transparent, an over layer covers it. It needs pixels where both are
// opaque and differ (for an occupied anchor) and pixels only the layer paints.
func layerOrderWitness(t *testing.T, f *FrontEnd, name string, layer uint16, anchorCode data.ItemCode) {
	t.Helper()
	item, _ := mapload.LayerItem(f.Table, layer)
	var empty, worn data.Equipment
	occupied := anchorCode != 0
	if occupied {
		worn.SetCode(int(item.Anchor), anchorCode)
	}
	base := figureCompose(f, empty, nil).Figure
	anchorOnly := figureCompose(f, worn, nil).Figure
	layerOnly := figureCompose(f, empty, []uint16{layer})
	both := figureCompose(f, worn, []uint16{layer})
	if base == nil || anchorOnly == nil || layerOnly.Figure == nil || both.Figure == nil {
		t.Fatalf("%s: a figure did not compose", name)
	}
	b := base.Bounds()
	overlap, visible, hidden := 0, 0, 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			bp, ap, lp, xp := pixelAt(base, x, y), pixelAt(anchorOnly, x, y), pixelAt(layerOnly.Figure, x, y), pixelAt(both.Figure, x, y)
			aPainted, lPainted := ap != bp, lp != bp
			want := bp
			switch {
			case item.Layer == mapload.LayerUnder && aPainted:
				want = ap
			case item.Layer == mapload.LayerUnder && lPainted:
				want = lp
			case item.Layer == mapload.LayerOver && lPainted:
				want = lp
			case item.Layer == mapload.LayerOver && aPainted:
				want = ap
			}
			if xp != want {
				t.Fatalf("%s: pixel (%d,%d) is %v, want %v (base %v anchor %v layer %v)", name, x, y, xp, want, bp, ap, lp)
			}
			if lPainted && aPainted && ap != lp {
				overlap++
			}
			if lPainted && !aPainted && xp == lp {
				visible++
			}
			if lPainted && xp != lp {
				hidden++
			}
			if lPainted && xp == lp {
				if slot, owned := both.SlotMask.At(x, y); owned && slot != 0 {
					// the layer owns no slot; a slot here is another item's pixel
					if !(aPainted && item.Layer == mapload.LayerUnder) {
						t.Fatalf("%s: layer pixel (%d,%d) is owned by slot %d", name, x, y, slot)
					}
				}
			}
		}
	}
	if visible == 0 {
		t.Fatalf("%s: the layer shows nowhere", name)
	}
	if occupied && overlap == 0 {
		t.Fatalf("%s: the layer and the anchor item never overlap with distinct colours", name)
	}
	if occupied && item.Layer == mapload.LayerUnder && hidden == 0 {
		t.Fatalf("%s: the anchor item hides none of the under layer", name)
	}
	// Loss control: composing without the layer is not the picture checked above.
	if slices.Equal(figureCompose(f, worn, nil).Figure.Pix, both.Figure.Pix) {
		t.Fatalf("%s: the picture does not change with the layer", name)
	}
}

func TestReleaseModLayersPaintInOrder(t *testing.T) {
	f := modItemFront(t)
	gambeson, cloak, boots := layerCodes(t, f)
	chain := data.ComposeItemCode(2, 7, 1, 16)
	layerOrderWitness(t, f, "gambeson under chain mail", gambeson, chain)
	layerOrderWitness(t, f, "cloak over chain mail", cloak, chain)
	layerOrderWitness(t, f, "boots over bare legs", boots, 0)
}

// TestReleaseModLayersPaperdollPicture writes the paperdoll with and without
// the layers: the figure bare, the figure in chain mail, the figure in chain
// mail under the three layers, and the twelve-slot panel.
func TestReleaseModLayersPaperdollPicture(t *testing.T) {
	f := modItemFront(t)
	gambeson, cloak, boots := layerCodes(t, f)
	var eq data.Equipment
	eq.SetCode(7, data.ComposeItemCode(2, 7, 1, 16))
	bare := figureCompose(f, data.Equipment{}, nil)
	mail := figureCompose(f, eq, nil)
	layered := figureCompose(f, eq, []uint16{gambeson, cloak, boots})
	if slices.Equal(mail.Figure.Pix, layered.Figure.Pix) {
		t.Fatal("the layers change nothing on the paperdoll")
	}
	panel := ui.RenderWorn(layered)
	parts := []*image.RGBA{bare.Figure, mail.Figure, layered.Figure, panel}
	w, h := 0, 0
	for _, p := range parts {
		w += p.Bounds().Dx() + 8
		h = max(h, p.Bounds().Dy())
	}
	sheet := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(sheet, sheet.Bounds(), image.NewUniform(sheet.ColorModel().Convert(image.Black)), image.Point{}, draw.Src)
	x := 0
	for _, p := range parts {
		draw.Draw(sheet, image.Rect(x, 0, x+p.Bounds().Dx(), p.Bounds().Dy()), p, p.Bounds().Min, draw.Over)
		x += p.Bounds().Dx() + 8
	}
	dir := os.Getenv("AGAINROM_SHOT_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	locale := filepath.Base(os.Getenv("AGAINROM_ASSETS"))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	out, err := os.Create(filepath.Join(dir, "paperdoll-"+strings.ReplaceAll(locale, string(filepath.Separator), "_")+".png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(out, sheet); err != nil {
		out.Close()
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}

// layerMission opens mission 30 from a reloaded town and returns the hero.
func layerMission(t *testing.T, f *FrontEnd) (*mapWorld, sim.EntityID) {
	t.Helper()
	app := f.App("mod layers")
	if err := app.OpenMission(f.MissionOpenerWith(30, f.NextParty())); err != nil {
		t.Fatalf("open mission 30: %v", err)
	}
	return f.live, equipmentReturnHero(t, f)
}

func TestReleaseModLayersInTheMissionInventory(t *testing.T) {
	f := modItemTown(t)
	gambeson, cloak, boots := layerCodes(t, f)
	s := currentTownShop(f, "hero")
	for _, code := range []uint16{gambeson, cloak, boots} {
		currentTownBuy(t, s, roomArmour, code)
		s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: currentTownPackCell(t, s, code)}, ui.ShopControl{Kind: ui.ShopControlDoll})
	}
	raw := currentTownSave(t, f)
	bare, bareErr := modItemReload(t, raw, false)
	g, err := modItemReload(t, raw, false)
	if err != nil || bareErr != nil {
		t.Fatal(err, bareErr)
	}
	bare.Carried[0].Layers = nil
	bmw, bid := layerMission(t, bare)
	mw, id := layerMission(t, g)
	baseline := releaseEntity(t, bmw, bid)
	e := releaseEntity(t, mw, id)
	if e.Defence-baseline.Defence != 9 {
		t.Fatalf("the layers add defence %d in the mission, want 9", e.Defence-baseline.Defence)
	}
	if e.Load != baseline.Load {
		t.Fatalf("wearing layers changed the carried load: %d vs %d", e.Load, baseline.Load)
	}
	if got := mw.activeLayers(id); len(got) != 3 {
		t.Fatalf("the mission hero wears layers %v", got)
	}

	// The inventory window takes a worn layer off, and puts it on again, through
	// the equip request of the pack cell.
	if !mw.invSubjectSet || mw.invSubject.ID != uint32(id) {
		t.Fatalf("the inventory window shows %d (set %v), want the hero %d", mw.invSubject.ID, mw.invSubjectSet, id)
	}
	cell := func(code uint16) int {
		stacks, _ := mw.world.CarriedStacks(id)
		i := slices.IndexFunc(stacks, func(st sim.ItemStack) bool { return st.Code == code })
		if i < 0 {
			t.Fatalf("the pack holds no %#x", code)
		}
		return i
	}
	mw.enqueueEquip(cell(cloak))
	mw.tick()
	if got := releaseEntity(t, mw, id).Defence - baseline.Defence; got != 7 || len(mw.activeLayers(id)) != 2 {
		t.Fatalf("after taking the cloak off: defence +%d layers %v", got, mw.activeLayers(id))
	}
	mw.enqueueEquip(cell(cloak))
	mw.tick()
	if got := releaseEntity(t, mw, id).Defence - baseline.Defence; got != 9 {
		t.Fatalf("after wearing the cloak again: defence +%d", got)
	}

	// A layer the member's class cannot wear is refused with the town's wording
	// on the message line, not silently: the cloak is for fighters.
	mw.enqueueEquip(cell(cloak))
	mw.tick()
	if got := releaseEntity(t, mw, id).Defence - baseline.Defence; got != 7 {
		t.Fatalf("after taking the cloak off a second time: defence +%d", got)
	}
	messages := func() []string {
		var out []string
		for _, l := range mw.view.MessageLines() {
			out = append(out, l.Text)
		}
		return out
	}
	posted := len(messages())
	mw.invParty.mage = true
	mw.enqueueEquip(cell(cloak))
	mw.tick()
	mw.invParty.mage = false
	if got := releaseEntity(t, mw, id).Defence - baseline.Defence; got != 7 || len(mw.activeLayers(id)) != 2 {
		t.Fatalf("a refused cloak changed defence to +%d, layers %v", got, mw.activeLayers(id))
	}
	if now := messages(); len(now) != posted+1 || now[len(now)-1] != "he cannot wear that" {
		t.Fatalf("the refusal posted %v after %d lines", now, posted)
	}
	mw.enqueueEquip(cell(cloak))
	mw.tick()
	if got := releaseEntity(t, mw, id).Defence - baseline.Defence; got != 9 {
		t.Fatalf("after wearing the cloak once more: defence +%d", got)
	}

	// A hero whose statistics come from an original save refuses a layer with the
	// same line: his derived values are not recomputed from items.
	{
		src, ok := bmw.world.Entity(bid)
		if !ok {
			t.Fatal("no bare hero")
		}
		source := sim.SourceActor{Class: 2}
		source.Stats[4], source.Stats[6], source.Stats[7] = uint16(src.Speed), uint16(src.Load), uint16(src.Capacity)
		source.Stats[8], source.Stats[9] = uint16(src.HP), uint16(src.MaxHP)
		snap := sim.ActorLoadSnapshot{Inventory: sim.ActorLoad{Present: true, ContainerPresent: true, Source: source},
			Load: src.Load, Capacity: src.Capacity, Speed: src.Speed}
		if err := bmw.world.RestoreActorLoad(bid, snap); err != nil {
			t.Fatalf("a stand-in original-save hero: %v", err)
		}
		bmw.invSubject.ID = uint32(bid)
		stacks, _ := bmw.world.CarriedStacks(bid)
		k := slices.IndexFunc(stacks, func(st sim.ItemStack) bool { return st.Code == boots })
		if k < 0 {
			t.Fatal("the bare hero's pack holds no boots")
		}
		before := len(bmw.view.MessageLines())
		bmw.enqueueEquip(k)
		bmw.tick()
		lines := bmw.view.MessageLines()
		if len(bmw.activeLayers(bid)) != 0 || len(lines) != before+1 || lines[len(lines)-1].Text != "he cannot wear that" {
			t.Fatalf("original-save hero: layers %v, messages %v", bmw.activeLayers(bid), lines)
		}
	}

	// A mission SAVE writes the three layers in the mod leaf, and a cold LOAD
	// under the same mods wears them again; without the mod it refuses.
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := agsSaveSeams(g, store, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil || !IsOriginal(name) {
		t.Fatalf("mission SAVE wrote %q: %v", name, err)
	}
	saved, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(saved)
	if err != nil {
		t.Fatal(err)
	}
	mark, present, err := readModMark(doc)
	if err != nil || !present || len(mark.Layers) != 3 {
		t.Fatalf("the mission SAVE's layers: %+v %v %v", mark.Layers, present, err)
	}
	cold := modItemFront(t)
	_, _, load := agsSaveSeams(cold, store, OriginalStore{}, nil)
	open, town, err := load(localOriginalSaveToken(name))
	if err != nil || town {
		t.Fatalf("LOAD of the mission SAVE: town=%v err=%v", town, err)
	}
	if err := cold.App("mod layers loaded").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	cid := sim.EntityID(0)
	for i, p := range cold.live.mission.party {
		if p.ID == "hero" {
			cid = cold.live.mission.ids[i]
		}
	}
	if got := releaseEntity(t, cold.live, cid).Defence - baseline.Defence; got != 9 || len(cold.live.activeLayers(cid)) != 3 {
		t.Fatalf("after the mission LOAD: defence +%d layers %v", got, cold.live.activeLayers(cid))
	}
	plain := releaseFront(t)
	_, _, plainLoad := agsSaveSeams(plain, store, OriginalStore{}, nil)
	if _, _, err := plainLoad(localOriginalSaveToken(name)); err == nil || !errors.Is(err, ErrModMark) {
		t.Fatalf("a mission LOAD without the mod: %v", err)
	}
}
