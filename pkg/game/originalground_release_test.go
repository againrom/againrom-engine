package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func groundCorpusFile(t *testing.T, relative, hash string) (string, []byte) {
	t.Helper()
	corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
	if corpus == "" {
		t.Skip("no AGAINROM_SAVE_CORPUS: original ground witness requires owner saves")
	}
	path := filepath.Join(corpus, filepath.FromSlash(relative))
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(b)); hash != "" && got != hash {
		t.Fatalf("fixture %s changed: %s", relative, got)
	}
	return path, b
}

func groundAppLoad(t *testing.T, app *ui.App, list ui.SaveList, name string) {
	t.Helper()
	label := ""
	rows := list()
	for _, row := range rows {
		if row.Name == name {
			label = row.Label
			break
		}
	}
	if label == "" && IsOriginal(name) {
		for _, row := range rows {
			if row.Name == localOriginalSaveToken(name) {
				label = row.Label
				break
			}
		}
	}
	if label == "" {
		t.Fatalf("save %s is absent from production list", name)
	}
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate(label); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenMap {
		t.Fatalf("load stayed at %s: %s", app.Screen(), app.HeadlessMessage())
	}
}

func groundAt(sacks []sim.Sack, x, y int32) *sim.Sack {
	for i := range sacks {
		if sacks[i].X == x && sacks[i].Y == y {
			return &sacks[i]
		}
	}
	return nil
}

// Join an original archive object to its new native ID through the exact DTO
// binding, never by equal item values. The value oracles below remain literal.
func releaseSourceItemObject1076(t *testing.T, state *SnapshotSAVDocument, w *sim.World, source sav.DocumentData, index uint16, owner sim.SavedObjectOwner) sim.SavedItemObject {
	t.Helper()
	if state == nil || state.Objects == nil || state.Objects.Version != 2 || index == 0 || int(index) > len(source.Objects) {
		t.Fatal("source Item lacks exact current ownership metadata")
	}
	var id sim.SavedObjectID
	for _, b := range state.Objects.Items {
		if b.ObjectIndex == index {
			if id != 0 || b.ID == 0 || b.Unavailable != "" {
				t.Fatal("ambiguous/unavailable original Item binding", b)
			}
			id = b.ID
		}
	}
	row, ok := w.SavedObjects().Item(id)
	if !ok || row.Origin != (sim.SavedObjectOrigin{Kind: sim.SavedObjectOriginal}) || !itemOnlyAt(w, id, owner) || !reflect.DeepEqual(state.Document.Objects[index-1], source.Objects[index-1]) || row.Token != releaseSourceToken1076(t, source.Objects[index-1]) {
		t.Fatal("native Item ID is not bound to its exact source record/owner", index, row)
	}
	for _, child := range []struct {
		name string
		ids  []sim.SavedObjectID
		meta []SnapshotSAVObjectBinding
	}{{"Effects", row.Effects, state.Objects.Effects}, {"WeaponSpell", []sim.SavedObjectID{row.Spell}, state.Objects.Spells}} {
		refs, present := savedObjectRefs(&source.Objects[index-1], child.name)
		if !present {
			continue
		}
		if len(refs) != len(child.ids) {
			t.Fatal("source child order/count differs", child.name)
		}
		for i, ref := range refs {
			if ref == 0 {
				if child.ids[i] != 0 {
					t.Fatal("null source child acquired an identity")
				}
				continue
			}
			b := itemMutationBinding1115(t, child.meta, child.ids[i])
			if b.ObjectIndex != ref || b.Unavailable != "" {
				t.Fatal("source child ID is not its exact archive edge", child.name, i)
			}
		}
	}
	return row
}

func releaseSourceToken1076(t *testing.T, record sav.DocumentRecordData) sim.SavedObjectToken {
	t.Helper()
	v := func(name string) uint32 { return itemObjectValue1115(t, record, name) }
	var position [12]byte
	copy(position[:], crossingRawField(t, &record, "Block12"))
	return sim.SavedObjectToken{Position: position, RuntimeID: v("RuntimeID"), T0C: uint8(v("T0C")), T0E: uint16(v("T0E")), T08: v("T08"), T18: uint16(v("T18")), T1C: v("T1C"), Identity: v("Identity"), Reference: v("Reference")}
}

// A structurally valid save can still fail once the installed map supplies its
// bounds. That refusal must leave the old game's complete SAVE owner intact.
func TestReleaseOriginalGround1076RejectedLoadPreservesLiveGame(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1076-rejected-original-ground")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	seedWorldSession(t, f)
	f.townUI.worldSelectedOnce = map[int]bool{50: true}
	f.Offered = 77
	f.Carried = f.liveParty
	f.Shop = &Shop{}
	f.originalCity = &originalCitySaveState{}
	f.live.world.SetPurse(sim.SelfSlot, 12345)
	oldLive, oldTown, oldShop, oldCity, oldUnits := f.live, f.Town, f.Shop, f.originalCity, f.Units
	oldTownUI := *f.townUI
	oldHash := f.live.world.Hash()
	originals := t.TempDir()
	payload := groundSave(t, []sav.GroundSack{{Identity: 501, Cell: 0xffff, FineX: 128, FineY: 128, Gold: 9}})
	if err := os.WriteFile(filepath.Join(originals, "game9999.sav"), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := agsSaveSeams(f, store, OriginalStore{Dir: originals}, nil)
	app.SetSaveSeams(save, list, load)
	rows := list()
	if len(rows) != 1 {
		t.Fatalf("original LOAD rows = %d, want 1", len(rows))
	}
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	before, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate(rows[0].Label); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenLoad || !strings.Contains(app.HeadlessMessage(), "outside 80x80") {
		t.Fatalf("refused LOAD = %s %q", app.Screen(), app.HeadlessMessage())
	}
	after, _, err := f.Snapshot(true)
	uiNow := *f.townUI
	uiNow.openMission, oldTownUI.openMission = nil, nil
	if err != nil || !reflect.DeepEqual(after, before) || f.live != oldLive || f.Town != oldTown ||
		f.Shop != oldShop || f.originalCity != oldCity || f.Units != oldUnits ||
		!reflect.DeepEqual(uiNow, oldTownUI) || oldLive.world.Hash() != oldHash {
		t.Fatalf("rejected LOAD changed the previous session: liveSame=%t townSame=%t offered=%d snapshotErr=%v",
			f.live == oldLive, f.Town == oldTown, f.Offered, err)
	}
	// SAVE still works through the production seam, and the resulting SAV
	// contains the game that was running before the refused LOAD.
	name, err := save(true)
	if err != nil {
		t.Fatal(err)
	}
	b, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	if !IsOriginal(name) {
		t.Fatalf("post-refusal SAVE published %q, want the sole SAV format", name)
	}
	cold := releaseFront(t)
	open, town, err := cold.RestoreOriginal(b)
	if err != nil || town || open == nil {
		t.Fatalf("post-refusal SAV did not reopen the previous mission: town=%v err=%v", town, err)
	}
	if err := cold.App("1076 post-refusal SAV").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if cold.liveMission != f.liveMission || cold.live.world.Hash() != oldHash {
		t.Fatalf("post-refusal SAV changed mission/hash: %d/%d %016x/%016x", cold.liveMission, f.liveMission, cold.live.world.Hash(), oldHash)
	}
	t.Logf("refusal=%q preservedHash=%016x native=%s", app.HeadlessMessage(), oldHash, name)
}

// The same owner save is materialized through each EN/RU root. These are two
// production integration witnesses, not two independent original observations.
func TestReleaseOriginalGround1076ProductionAppReplacesFreshLootAndSavesPickup(t *testing.T) {
	f := releaseFront(t)
	path, payload := groundCorpusFile(t, "2026-08-12/game0012.sav", "12b05b4eb3ff6921cdddfcf89965f525c40dc169331ee28c5f39b9a63c6c8f44")
	f.SetDeterministicFrames(true)
	app := f.App("1076-original-ground")
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := nativeContinuationSeams1170(t, f, store, OriginalStore{Dir: filepath.Dir(path)}, nil)
	app.SetSaveSeams(save, list, load)
	// Baseline proves the omitted sack exists in fresh installed map 10.
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	if len(f.live.world.Sacks()) != 4 || groundAt(f.live.world.Sacks(), 20, 65) == nil {
		t.Fatal("installed fresh map no longer witnesses the missing-source-sack case")
	}
	groundAppLoad(t, app, list, filepath.Base(path))
	w := f.live.world
	got := w.Sacks()
	if len(got) != 3 || groundAt(got, 20, 65) != nil {
		t.Fatalf("picked-up source Sack respawned: %+v", got)
	}
	for _, want := range []struct {
		x, y int32
		code uint16
	}{{10, 14, 2586}, {12, 50, 4642}, {38, 64, 63013}} {
		s := groundAt(got, want.x, want.y)
		if s == nil || s.Gold != 0 || !reflect.DeepEqual(s.Items, []uint16{want.code}) {
			t.Fatalf("source Sack (%d,%d) = %+v", want.x, want.y, s)
		}
	}
	// The headless diagnostic importer must expose exactly the same population.
	ms, report, err := loadOriginalMission(f, payload)
	if err != nil || !report.GroundApplied || !reflect.DeepEqual(ms.World.Sacks(), got) {
		t.Fatalf("diagnostic importer disagrees: %v report=%+v", err, report)
	}
	// Return to the corresponding pre-pickup owner save: its close starting
	// sack permits the real walk command without running a combat scenario.
	groundAppLoad(t, app, list, "game0011.sav")
	w = f.live.world
	if len(w.Sacks()) != 4 || groundAt(w.Sacks(), 20, 65) == nil {
		t.Fatal("pre-pickup witness changed")
	}
	// Exercise the production walk-then-pickup command, not a replacement of
	// the inventory.
	id := f.live.mission.ids[0]
	f.live.grab(uint32(id), 20, 65, true)
	for i := 0; i < 400 && groundAt(w.Sacks(), 20, 65) != nil; i++ {
		f.live.tick()
	}
	if groundAt(w.Sacks(), 20, 65) != nil {
		t.Fatal("production pickup did not consume the restored Sack")
	}
	before := w.Sacks()
	// A successful pickup can raise its normal item notice. Dismiss that
	// modal through App before asking the same App to save.
	if _, _, open := f.LiveNotice(); open {
		if err := app.HeadlessActivate("notice"); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenGameMenu {
		t.Fatalf("save menu: %s %s", app.Screen(), app.HeadlessMessage())
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := listAGS(store)
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".ags" {
		t.Fatalf("mission SAVE must be explicit lossless .ags, not stale original bytes: %+v %v screen=%s message=%s", entries, err, app.Screen(), app.HeadlessMessage())
	}
	groundAppLoad(t, app, list, entries[0].Name)
	if !reflect.DeepEqual(f.live.world.Sacks(), before) {
		t.Fatal("native App reload respawned or changed imported ground loot")
	}
	carried, ok := f.live.world.Carried(id)
	found := false
	for _, code := range carried {
		found = found || code == 4358
	}
	if !ok || !found {
		t.Fatal("native App reload lost the picked-up weapon")
	}
	t.Logf("source=%x fresh=4 imported=3 after-pickup=%d native=%s", sha256.Sum256(payload), len(before), entries[0].Name)
}

func TestReleaseOriginalGround1076GoldEffectsAndStackSurviveNativeContinuation(t *testing.T) {
	f := releaseFront(t)
	_, payload := groundCorpusFile(t, "2026-08-15/game0018.sav", "1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b")
	sf, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	// Fixed source anchors: Armor archive177's Item members follow its one
	// Effect at body72534; F4A follows F40/F42/F44/F45/F46/F48, nine bytes on.
	// The expected instance weight is a literal source word, not our import.
	if len(sf.Body) < 72546 || binary.LittleEndian.Uint16(sf.Body[72534:]) != 13377 || int16(binary.LittleEndian.Uint16(sf.Body[72543:])) != 3 {
		t.Fatal("enchanted source code or signed weight changed")
	}
	f.SetDeterministicFrames(true)
	open, town, err := f.RestoreOriginal(payload)
	if err != nil || town {
		t.Fatalf("restore: %t %v", town, err)
	}
	app := f.App("1076 ground pickup")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	w := f.live.world
	sacks := w.Sacks()
	if len(sacks) != 6 {
		t.Fatalf("sacks=%d want6", len(sacks))
	}
	stack := groundAt(sacks, 77, 110)
	enchanted := groundAt(sacks, 13, 130)
	doc, origins, err := sav.DecodeDocumentDataWithOrigins(payload)
	if err != nil {
		t.Fatal(err)
	}
	state := f.live.mission.state.savedDocument
	var enchantedID sim.SavedObjectID
	for _, sack := range []*sim.Sack{stack, enchanted} {
		if sack == nil || sack.ObjectID == 0 {
			t.Fatal("source Sack lacks native identity")
		}
		binding := itemMutationBinding1115(t, state.Objects.Sacks, sack.ObjectID)
		if binding.ObjectIndex == 0 || !slices.Contains(doc.World.Sacks, binding.ObjectIndex) || binding.Unavailable != "" {
			t.Fatal("source Sack ID does not name its exact root")
		}
		record := doc.Objects[binding.ObjectIndex-1]
		token := releaseSourceToken1076(t, record)
		if int32(token.Position[2]) != sack.X || int32(token.Position[3]) != sack.Y {
			t.Fatal("source Sack binding crossed cells")
		}
		found := false
		for _, row := range w.SavedObjects().Sacks {
			if row.ID == sack.ObjectID {
				found = row.Origin.Kind == sim.SavedObjectOriginal && row.Token == token && !row.Retired && row.Gold == sack.Gold
			}
		}
		if !found {
			t.Fatal("source Sack identity/Token was reconstructed")
		}
		refs, _ := savedObjectRefs(&record, "Contents")
		var flatIDs []sim.SavedObjectID
		for _, index := range refs {
			row := releaseSourceItemObject1076(t, state, w, doc, index, sim.SavedObjectOwner{Kind: sim.SavedOwnerSack, Object: sack.ObjectID})
			for range itemObjectValue1115(t, doc.Objects[index-1], "F42") {
				flatIDs = append(flatIDs, row.ID)
			}
			if sack == enchanted {
				for _, origin := range origins {
					if origin.ArchiveIndex == 177 && origin.ObjectIndex == index {
						enchantedID = row.ID
					}
				}
			}
		}
		if len(flatIDs) != len(sack.ItemInstances) {
			t.Fatal("source Item count/expansion identity differs")
		}
		for i, id := range flatIDs {
			if sack.ItemInstances[i].ObjectID != id {
				t.Fatal("source Sack item identity/order differs", i)
			}
		}
	}
	if stack == nil || stack.Gold != 78 || !reflect.DeepEqual(stack.Items, []uint16{4356, 4356}) {
		t.Fatalf("stack=%+v", stack)
	}
	// Armor's derived tail starts after the 12-byte Item block. This literal
	// source check is independent of the importer and native item projection.
	if len(sf.Body) < 72569 || !bytes.Equal(sf.Body[72546:72569], append([]byte{1}, append(make([]byte, 21), 4)...)) {
		t.Fatal("source Armor defence/own-kind tail changed")
	}
	if enchantedID == 0 {
		t.Fatal("Armor archive177 did not retain exact native identity")
	}
	// SourceEquipment is also a code-constructed record here (ground stock,
	// never worn or a saved holdings runtime), so it is independently a pure
	// function of Code 13377 = 0x3441 and the table: DefinitionRow =
	// code&0x1f = 1, and class nibble (code>>8)&0xf = 4 selects
	// data.ArmorFromCode's branch, matching SourceArmor. Defence[0]=1 is that
	// row's own table column -- the same values the raw check above pins.
	want := sim.ItemInstance{ObjectID: enchantedID, Code: 13377, Kind: 1, Price: 1102, Weight: 3, WeightPresent: true, Effects: []sim.ItemEffect{{Kind: 11, Operand: 1}},
		SourceEquipment: sim.SourceEquipment{Class: sim.SourceArmor, DefinitionRow: 1, OwnKind: 4, Defence: [22]byte{1}}}
	if enchanted == nil || !reflect.DeepEqual(enchanted.ItemInstances, []sim.ItemInstance{want}) {
		t.Fatalf("enchanted=%+v", enchanted)
	}
	id := f.live.mission.ids[0]
	purse := w.Purse(sim.SelfSlot)
	for _, sack := range []*sim.Sack{stack, enchanted} {
		liveTakeAt(t, f.live, id, sack.X, sack.Y)
	}
	if w.Purse(sim.SelfSlot) != purse+78 {
		t.Fatal("ground purse was not transferred")
	}
	owned, ok := w.SavedObjects().Item(enchantedID)
	if !ok || !itemOnlyAt(w, enchantedID, sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: id}) || !reflect.DeepEqual(owned.Value.Instance(), want) {
		t.Fatal("pickup changed the exact enchanted Item identity/value/owner")
	}
	for _, old := range []*sim.Sack{stack, enchanted} {
		for _, row := range w.SavedObjects().Sacks {
			if row.ID == old.ObjectID && !row.Retired {
				t.Fatal("picked source Sack identity did not retire")
			}
		}
	}
	hash := w.Hash()
	store, name, written := menuSAVE(t, f, app, OriginalStore{})
	cold, _ := loadSAVWindow(t, store, name)
	back := cold.live.world
	requireSameItemObjects(t, w, back)
	if back.Hash() != hash || !reflect.DeepEqual(back.Sacks(), w.Sacks()) || back.Purse(sim.SelfSlot) != purse+78 {
		t.Fatal("SAV LOAD changed the ground, purse or World hash")
	}
	requireAlteredItemWeight(t, w, written, enchantedID)
	for range 20 {
		sim.Step(w, nil)
		sim.Step(back, nil)
		if w.Hash() != back.Hash() {
			t.Fatal("ground continuation differs after the SAV LOAD")
		}
	}
	requireSameItemObjects(t, w, back)
	source, present, err := sf.GroundSacks()
	if err != nil || !present || len(source) != 6 {
		t.Fatal("reader census changed")
	}
	t.Logf("source=%x sacks=6 stack=2 effect=(11,0,1) gold+78 savHash=%016x", sha256.Sum256(payload), hash)
}
