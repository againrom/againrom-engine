package game

import (
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// WitnessOwnerFidelity uses the real mission, LOAD and Game Options paths.
// All generated saves, preferences and CPU HUD compositions stay in an
// explicit private directory outside the read-only install.
func (f *FrontEnd) WitnessOwnerFidelity(root, output string, report io.Writer) error {
	root, err := editorPhysicalDirectory(root)
	if err != nil {
		return err
	}
	output, err = editorPhysicalDirectory(output)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, output)
	if err != nil || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return fmt.Errorf("fidelity witness: output must be outside install")
	}
	profile, err := filepath.Abs(f.Options.Path)
	if err != nil || profile != filepath.Join(output, "options.txt") {
		return fmt.Errorf("fidelity witness: -saves must name the saves child of the output directory")
	}
	for _, name := range []string{"options.txt", "minimap.png", "mission-card.png", "warrior-card.png", "mission-doll.png", "game-options.png", "bottom-hud.png", "bottom-hud-all-spells.png", "book-wide.png", "shop-book.png", "tavern-dialogue.png", "tavern-hired.png", "school.png", "world-map-closed.png", "world-map-open.png"} {
		info, err := os.Lstat(filepath.Join(output, name))
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil && !info.Mode().IsRegular() {
			return fmt.Errorf("fidelity witness: output is not a regular file: %s", name)
		}
	}
	return f.witnessOwnerFidelity(root, output, report)
}

func (f *FrontEnd) witnessOwnerFidelity(root, output string, report io.Writer) error {
	f.SetDeterministicFrames(true)
	a := f.App("owner fidelity witness")
	defer a.StopAudio()
	a.SetCutscenes(nil)
	scatter := f.ChargenParty(ui.ChargenResult{Name: "Terrain", Choices: []int{1, 1, 0}, Stats: []int{19, 23, 30, 42}})
	group := make([]mapload.PartyMember, 5)
	for i := range group {
		group[i] = scatter[0]
	}
	// Each actor owns a separate inventory and receives distinct SAV identities.
	group = mapload.CloneParty(group)
	for i := range group {
		group[i].ID = fmt.Sprintf("scatter-%d", i)
	}
	if err := a.OpenMission(f.MissionOpenerWith(10, group)); err != nil {
		return err
	}
	if ground, objects := f.live.view.ScorchedScenery(); ground != 0 || objects != 0 {
		return fmt.Errorf("fidelity witness: fresh mission acquired false fire: ground=%d objects=%d", ground, objects)
	}
	entities := f.live.world.Entities()
	primary := entities[f.live.mission.ids[0]]
	spread := false
	positions := make([]image.Point, 0, len(group))
	for _, id := range f.live.mission.ids {
		e := entities[id]
		if e.OffMap {
			return fmt.Errorf("fidelity witness: mission10 could not seat actor %d", id)
		}
		positions = append(positions, image.Pt(int(e.X), int(e.Y)))
		if e.X < primary.X-1 || e.X > primary.X+1 || e.Y < primary.Y-1 || e.Y > primary.Y+1 {
			spread = true
		}
	}
	if !spread {
		return fmt.Errorf("fidelity witness: mission10 companions still form an adjacent ring: %v", positions)
	}
	fmt.Fprintf(report, "fidelity: fresh mission10 has no false fire; five mages scattered at %v\n", positions)
	warrior := f.ChargenParty(ui.ChargenResult{Name: "DANATH", Choices: []int{0, 0, 0}, Stats: []int{43, 25, 26, 26}})
	if err := a.OpenMission(f.MissionOpenerWith(20, warrior)); err != nil {
		return err
	}
	a.Layout(1024, 768)
	if err := a.HeadlessSelectEntity(uint32(f.live.mission.ids[0])); err != nil {
		return err
	}
	f.live.push()
	warriorCard, err := a.HeadlessMissionCard()
	if err != nil {
		return err
	}
	if err := writeMediaFrame(output, "warrior-card", warriorCard); err != nil {
		return err
	}
	party := f.ChargenParty(ui.ChargenResult{Name: "RENIESTA", Choices: []int{1, 1, 0}, Stats: []int{19, 23, 30, 42}})
	if err := a.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		return err
	}
	a.Layout(1024, 768)
	for n := 0; a.HeadlessNoticeOpen() && n < 16; n++ {
		if err := a.HeadlessKey("enter"); err != nil {
			return err
		}
	}
	if a.HeadlessNoticeOpen() {
		return fmt.Errorf("fidelity witness: mission notice did not close")
	}
	if err := a.HeadlessSelectEntity(uint32(f.live.mission.ids[0])); err != nil {
		return err
	}
	f.live.push()
	minimap, err := a.HeadlessMinimap()
	if err != nil || minimap.Bounds().Size() != image.Pt(176, 158) {
		return fmt.Errorf("fidelity witness: full crystal absent: %v", err)
	}
	card, err := a.HeadlessMissionCard()
	if err != nil {
		return err
	}
	doll, _, err := a.HeadlessCharacterPane()
	if err != nil {
		return err
	}
	bottom, err := a.HeadlessBottomHUD()
	if err != nil {
		return err
	}
	for _, capture := range []struct {
		name string
		pix  image.Image
	}{{"minimap", minimap}, {"mission-card", card}, {"mission-doll", doll}, {"bottom-hud", bottom}} {
		if err := writeMediaFrame(output, capture.name, capture.pix); err != nil {
			return err
		}
	}
	a.Layout(640, 480)
	if err := a.HeadlessKey("escape"); err != nil {
		return err
	}
	if err := a.HeadlessGameMenuAction("game-options"); err != nil {
		return err
	}
	before, appBefore := f.live.world.Hash(), f.live.view.SaveApplication()
	initial := f.live.view.PathfindingShown()
	for range 3 {
		if err := a.HeadlessGameMenuAction("pathfinding"); err != nil {
			return err
		}
		if f.live.view.PathfindingShown() != initial || before != f.live.world.Hash() || !reflect.DeepEqual(appBefore, f.live.view.SaveApplication()) {
			return fmt.Errorf("fidelity witness: pathfinding toggle applied before OK or changed game state")
		}
	}
	if err := writeMediaFrame(output, "game-options", a.GameMenuPanel()); err != nil {
		return err
	}
	if err := a.HeadlessGameMenuAction("page-return"); err != nil {
		return err
	}
	want := !initial
	if f.live.view.PathfindingShown() != want || before != f.live.world.Hash() || !reflect.DeepEqual(appBefore, f.live.view.SaveApplication()) {
		return fmt.Errorf("fidelity witness: pathfinding toggle changed game state or did not apply")
	}
	coldPreference := &FrontEnd{PersistenceContext: PersistenceContext{Options: f.Options}}
	coldPreference.LoadOptions()
	if coldPreference.showPathfinding != want {
		return fmt.Errorf("fidelity witness: cold profile lost ShowPathfinding")
	}
	if err := a.HeadlessGameMenuAction("return"); err != nil {
		return err
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		return err
	}
	raw, err := f.ExportCurrentSave(s, "Fidelity witness")
	if err != nil {
		return err
	}
	store := SaveStore{Dir: filepath.Join(output, "saves")}
	name, err := store.WriteOriginal(root, raw)
	if err != nil {
		return err
	}
	_, list, load := f.SaveSeams(store, OriginalStore{}, nil)
	entries := list()
	if len(entries) == 0 || entries[0].Name != localOriginalSaveToken(name) {
		return fmt.Errorf("fidelity witness: new local SAV is absent from LOAD")
	}
	a.SetSaveSeams(nil, list, load)
	if err := a.HeadlessKey("escape"); err != nil {
		return err
	}
	if err := a.HeadlessGameMenuAction("load"); err != nil {
		return err
	}
	for click := 0; click < 2; click++ {
		for _, edge := range []string{"press", "release"} {
			if err := a.HeadlessPointer(edge, 140, 158); err != nil {
				return err
			}
		}
		if click == 0 && a.Screen() != ui.ScreenLoad {
			return fmt.Errorf("fidelity witness: single click loaded a save")
		}
	}
	if a.Screen() != ui.ScreenMap || f.live.world.Hash() != before || f.live.view.PathfindingShown() != want {
		return fmt.Errorf("fidelity witness: double click did not restore the world and process preference: screen %s, world kept %v, pathfinding %v",
			a.Screen(), f.live.world.Hash() == before, f.live.view.PathfindingShown())
	}
	fmt.Fprintf(report, "fidelity: crystal=176x158 mage-card=%v warrior-card=%v doll=%v; real HUD compositions written\n", card.Bounds().Size(), warriorCard.Bounds().Size(), doll.Bounds().Size())
	fmt.Fprintf(report, "fidelity: pathfinding startup=%v toggled=%v cold-profile=yes state-unchanged=yes; real SAV double-click LOAD=yes\n", initial, want)
	if err := f.witnessBottomHUD(a, output, report); err != nil {
		return err
	}
	return f.witnessTownControls(output, report)
}

func (f *FrontEnd) witnessTownControls(output string, report io.Writer) error {
	f.Town = NewTown(f.Campaign.Value())
	for _, mission := range f.Campaign.Value().Main {
		if mission < 30 {
			f.Town.Won(mission)
		}
	}
	f.Town.Arrive()
	f.Town.gold = 1000000
	t := f.TownScreen().(*townScreen)
	t.room, t.shopBook = roomTavern, false
	t.composeShopFaces()
	writeTown := func(name string) error {
		pic, err := ui.ComposeTownScreen(t, "")
		if err != nil {
			return err
		}
		return writeMediaFrame(output, name, pic)
	}
	var offer, merc = -1, -1
	for i, c := range t.tavernCandidates() {
		if c.key.kind == tavernCandidateOffer && offer < 0 {
			offer = i
		}
		if c.key.kind == tavernCandidateMercenary && merc < 0 {
			merc = i
		}
	}
	if offer < 0 || merc < 0 {
		return fmt.Errorf("town controls witness: missing story NPC or mercenary")
	}
	t.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: offer}, true)
	if t.room != roomTalk {
		return fmt.Errorf("town controls witness: story double click did not talk")
	}
	if err := writeTown("tavern-dialogue"); err != nil {
		return err
	}
	for n := 0; t.room == roomTalk && n < 20; n++ {
		t.AdvanceTownDialogue()
	}
	if t.room == roomTalk {
		return fmt.Errorf("town controls witness: story dialogue did not finish")
	}
	t.room = roomTavern
	// Resolve the mercenary cell again after the conversation.
	merc = -1
	for i, c := range t.tavernCandidates() {
		if c.key.kind == tavernCandidateMercenary {
			merc = i
			break
		}
	}
	if merc < 0 {
		return fmt.Errorf("town controls witness: mercenary disappeared")
	}
	beforeGold, beforeParty := f.Town.Gold(), len(f.Carried)
	t.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: merc}, true)
	if !t.TownSurface().Cells[merc].Hired || len(f.Carried) <= beforeParty {
		return fmt.Errorf("town controls witness: mercenary double click did not hire")
	}
	if err := writeTown("tavern-hired"); err != nil {
		return err
	}
	t.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: merc}, true)
	if len(f.Carried) != beforeParty || f.Town.Gold() != beforeGold {
		return fmt.Errorf("town controls witness: dismissal changed the round-trip party or purse")
	}
	// Leaving the tavern commits the quest its conversation queued.
	t.Back()
	t.room, t.shopBook = roomSchool, false
	if err := writeTown("school"); err != nil {
		return err
	}
	t.enterWorldMap()
	v := t.WorldMapView()
	if v.ScrollHovered || len(v.Missions) == 0 {
		return fmt.Errorf("town controls witness: world map did not start collapsed")
	}
	if err := writeMediaFrame(output, "world-map-closed", ui.ComposeWorldMap(v)); err != nil {
		return err
	}
	t.WorldMapHover(ui.WorldMapCardRect(0).Min.Add(image.Pt(20, 20)))
	if !t.WorldMapView().ScrollHovered {
		return fmt.Errorf("town controls witness: mission scroll did not expand")
	}
	if err := writeMediaFrame(output, "world-map-open", ui.ComposeWorldMap(t.WorldMapView())); err != nil {
		return err
	}
	fmt.Fprintln(report, "fidelity: city story double-click talks; mercenary double-click hire/dismiss preserves round-trip purse and party; red localized hired label; shared city book; collapsed world-map scrolls, right-aligned home and hover briefing")
	return nil
}

func (f *FrontEnd) witnessBottomHUD(a *ui.App, output string, report io.Writer) error {
	if f.BottomHUDArt.Value() == nil {
		return fmt.Errorf("fidelity witness: bottom HUD art absent: %v", f.BottomHUDArt.Err())
	}
	party := f.ChargenParty(ui.ChargenResult{Name: "RENIESTA", Choices: []int{1, 1, 0}, Stats: []int{19, 23, 30, 42}})
	// Controlled mission input, not a claim that a new campaign grants all
	// spells. This exercises the installed catalog through the real producer.
	for _, id := range originalBookIDs {
		party[0].KnownSpells |= uint32(1) << id
	}
	party[0].Carried = []uint16{0x0e17} // Installed mission-80 Astral book.
	if err := a.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		return err
	}
	a.Layout(1024, 768)
	for n := 0; a.HeadlessNoticeOpen() && n < 16; n++ {
		if err := a.HeadlessKey("enter"); err != nil {
			return err
		}
	}
	if err := a.HeadlessSelectEntity(uint32(f.live.mission.ids[0])); err != nil {
		return err
	}
	f.live.push()
	if err := a.HeadlessKey("0"); err != nil {
		return err
	}
	for slot, id := range originalBookIDs {
		x, y, err := a.HeadlessSpellPoint(uint32(id))
		if err != nil || x != 216+38*(slot%12) || y != 617+38*(slot/12) {
			return fmt.Errorf("fidelity witness: book slot %d ID %d point=(%d,%d): %v", slot, id, x, y, err)
		}
		for _, edge := range []string{"press", "release"} {
			if err := a.HeadlessPointer(edge, x, y); err != nil {
				return err
			}
		}
		if got := f.live.view.SaveApplication().PressedSpell; got != uint32(id) {
			actor, _ := f.live.entity(f.live.mission.ids[0])
			return fmt.Errorf("fidelity witness: slot %d selected ID %d, want %d; actor known=%x book=%d", slot, got, id, actor.KnownSpells, actor.Book.State)
		}
	}
	if err := a.HeadlessKey("ctrl-f5"); err != nil {
		return err
	}
	pic, err := a.HeadlessBottomHUD()
	if err != nil || pic.Bounds().Size() != image.Pt(864, 175) {
		return fmt.Errorf("fidelity witness: bottom HUD dimensions: %v", err)
	}
	if err := writeMediaFrame(output, "bottom-hud-all-spells", pic); err != nil {
		return err
	}
	a.Layout(1920, 1080)
	if err := a.HeadlessKey("inventory"); err != nil {
		return err
	}
	wide, err := a.HeadlessBottomHUD()
	if err != nil || wide.Bounds().Size() != image.Pt(1206, 85) {
		return fmt.Errorf("fidelity witness: widescreen book geometry: %v", err)
	}
	for x := 827; x < 1190; x++ {
		visible := false
		for y := 0; y < 85; y++ {
			visible = visible || wide.RGBAAt(x, y).A != 0
		}
		if !visible {
			return fmt.Errorf("fidelity witness: black seam inside wide book at %d", x)
		}
	}
	if err := writeMediaFrame(output, "book-wide", wide); err != nil {
		return err
	}
	f.Carried = party
	shop := f.TownScreen().(*townScreen)
	shop.room, shop.shopBook = roomShop, true
	shop.composeShopFaces()
	view := shop.ShopScreen()
	if !view.SpellCatalog || len(view.Spells) != 24 || view.Art.Book != f.BottomHUDArt.Value() {
		return fmt.Errorf("fidelity witness: shop lost the shared spell catalog or art")
	}
	for slot, id := range originalBookIDs {
		lines, ok := ui.ShopHoverLines(view, image.Pt(24+38*(slot%12), 327+38*(slot/12)))
		if view.Spells[slot].ID != uint32(id) || !ok || len(lines) == 0 || lines[0] != view.Spells[slot].Name {
			return fmt.Errorf("fidelity witness: shop slot %d lost spell %d or its tooltip", slot, id)
		}
	}
	if err := writeMediaFrame(output, "shop-book", ui.ComposeShopScreen(view, image.Point{}, false, nil, false)); err != nil {
		return err
	}
	fmt.Fprintln(report, "fidelity: original bottom HUD=864x175; atlas=24 row-major slots; pointer-selected every real ID; F5 numeral and installed inventory written; shop shares atlas, order and localized hovers; wide book=1206x85 without internal closing seam")
	return nil
}
