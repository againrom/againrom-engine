package game

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// All source scalars here come from the pinned lawful city input, not the
// implementation's load formula or item lookup. The original executable is
// never launched by this witness.
func TestReleaseCitySalesUseSAVAndFreshProcesses(t *testing.T) {
	f := releaseFront(t)
	sourcePath, source := groundCorpusFile(t, "2026-08-15/game0010.sav", "89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4")
	store := SaveStore{Dir: t.TempDir()}
	app := f.App("1102-city-sales")
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: filepath.Dir(sourcePath)}, nil)
	app.SetSaveSeams(save, list, load)
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	label := ""
	for _, e := range list() {
		if e.Name == filepath.Base(sourcePath) {
			label = e.Label
		}
	}
	if label == "" {
		t.Fatal("original missing from picker")
	}
	if err := app.HeadlessActivate(label); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatal(err, app.Screen())
	}
	before := trainingPartyMember(t, f, "hero")
	h, ok := before.OriginalHumanState()
	if !ok {
		t.Fatal("no source Human")
	}
	if h.Weight != 178 || h.InventoryWeight != 6 || h.Load != 181 || h.Capacity != 411 {
		t.Fatalf("source scalars %+v", h)
	}
	party := mapload.CloneParty(f.Carried)
	gold := f.Town.Gold()
	var sourceItem sav.CityInventoryItem
	for _, v := range f.originalCity.saleBinding("hero").inventory {
		if v.Piece.Code == 0x0e06 {
			sourceItem = v
		}
	}
	if sourceItem.Piece.Stack != 2 || sourceItem.Weight != 1 || sourceItem.Piece.Price != 50 || !sourceItem.Exclusive {
		t.Fatal("source stack changed", sourceItem)
	}
	if _, err := citySaleRemainders(*f.originalCity.saleBinding("hero"), nil); err != nil {
		t.Fatalf("source/native container preparation: %v", err)
	}
	sell := func(whole bool) {
		t.Helper()
		s := f.TownScreen().(*townScreen)
		s.room = roomShop
		for i, m := range f.Carried {
			if m.ID == "hero" {
				s.shopMember = i
			}
		}
		index := -1
		for i, v := range s.shopPackStacks() {
			if v.Instance().Code == 0x0e06 {
				index = i
			}
		}
		if index < 0 {
			t.Fatal("source item missing")
		}
		s.ShopClick(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: index + 1, Shift: whole})
		if f.originalCity.trade == nil && len(f.Shop.Table()) == 0 {
			t.Fatal("real pack click did not prepare source sale")
		}
		s.ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 2})
	}
	beforeSale := saleDocument(t, f, "sale")
	sell(false)
	after := trainingPartyMember(t, f, "hero")
	want := h
	want.InventoryWeight, want.Load = 5, 180
	afterHuman, _, ok := currentCityHuman(after)
	if !ok || afterHuman != want || f.Town.Gold() != gold+25 {
		t.Fatal("partial sale did not preserve equal-quotient state", afterHuman, f.Town.Gold())
	}
	// The SAV exported after the sale differs from the one exported before it
	// only in the sold stack's count, the hero's container and Human loads, the
	// purse and the engine's own action supplement. Every identity and every
	// other field stays as it was.
	afterSale := saleDocument(t, f, "sale")
	changed := documentFieldChanges(beforeSale, afterSale)
	if len(changed) == 0 {
		t.Fatal("partial sale changed nothing in the exported SAV")
	}
	allowed := []string{"state:/CurrentState/AgainromActions"}
	for i := range beforeSale.Objects {
		r := &beforeSale.Objects[i]
		code, _ := savedStructureValue(r, "F40")
		name := ""
		for _, text := range r.Texts {
			if text.Name == "Name" {
				name = text.Value
			}
		}
		switch {
		case r.Class == "Player":
			allowed = append(allowed, fmt.Sprintf("%d:Player:value:Money", i+1))
		case r.Class == "Human" && name == before.Name:
			allowed = append(allowed, fmt.Sprintf("%d:Human:value:Inventory20", i+1), fmt.Sprintf("%d:Human:value:U90", i+1))
		case r.Class == "Item" && code == 0x0e06:
			allowed = append(allowed, fmt.Sprintf("%d:Item:value:F42", i+1))
		}
	}
	if len(allowed) != 5 {
		t.Fatalf("sale oracle found %d of its five named fields: %v", len(allowed), allowed)
	}
	for _, change := range changed {
		if !slices.Contains(allowed, change) {
			t.Fatalf("partial sale changed %s outside the sold stack, the hero's loads and the purse (all changes %v)", change, changed)
		}
	}
	for i, m := range f.Carried {
		if m.ID != "hero" && !reflect.DeepEqual(m, party[i]) {
			t.Fatal("other member changed")
		}
	}
	slot := 0
	for i := 1; i <= 5; i++ {
		if h.Base.Skill[i] == 0 {
			slot = i
			break
		}
	}
	if slot == 0 {
		t.Fatal("no untrained slot")
	}
	s := f.TownScreen().(*townScreen)
	s.room, s.schoolCell = roomSchool, slot-1
	if err := app.HeadlessActivate(f.Words.SchoolTrain); err != nil {
		t.Fatal(err)
	}
	if f.Town.Gold() != gold+25-200 {
		t.Fatal("training between sales")
	}
	sell(true)
	if f.Town.Gold() != gold+50-200 {
		t.Fatal("whole sale payment")
	}
	s = f.TownScreen().(*townScreen)
	s.room, s.schoolCell = roomSchool, slot-1
	if err := app.HeadlessActivate(f.Words.SchoolTrain); err != nil {
		t.Fatal(err)
	}
	if f.Town.Gold() != gold-370 {
		t.Fatal("second training")
	}
	final := trainingPartyMember(t, f, "hero")
	if final.OriginalHuman.State.InventoryWeight != 4 || final.OriginalHuman.State.Load != 180 || final.OriginalHuman.State.Base.Skill[slot] != 2 {
		t.Fatal("final continuation", final.OriginalHuman)
	}
	for _, v := range mapload.MemberCarriedItems(final, f.Table) {
		if v.Code == 0x0e06 {
			t.Fatal("whole sold item retained")
		}
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".sav" {
		t.Fatal("ordinary SAVE not SAV", entries, err, app.HeadlessMessage())
	}
	saved := filepath.Join(store.Dir, entries[0].Name)
	wire, err := ReadSaveFile(saved)
	if err != nil || !bytes.HasPrefix(wire, []byte(sav.Magic)) {
		t.Fatal(err)
	}
	child := func(mode, input, output string) {
		t.Helper()
		cmd := exec.Command(os.Args[0], "-test.run=^TestCitySalesProcess$", "-test.v")
		cmd.Env = append(os.Environ(), "AGAINROM_1102_PROCESS="+mode, "AGAINROM_1102_INPUT="+input, "AGAINROM_1102_OUTPUT="+output)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("fresh %s: %v\n%s", mode, err, out)
		}
	}
	loaded := releaseFront(t)
	openLocalTownSAV(t, loaded, store.Dir, entries[0].Name)
	if loaded.Town.Gold() != f.Town.Gold() {
		t.Fatal("LOAD changed the purse", loaded.Town.Gold(), f.Town.Gold())
	}
	resaved := SaveStore{Dir: t.TempDir()}
	resave, _, _ := loaded.SaveSeams(resaved, OriginalStore{}, nil)
	backName, err := resave(false)
	if err != nil {
		t.Fatal(err)
	}
	back, err := resaved.Read(backName)
	if err != nil {
		t.Fatal(err)
	}
	requireSameDocumentButKeys(t, wire, back)
	altered := filepath.Join(t.TempDir(), "altered.sav")
	if err := os.WriteFile(altered, alterSAV(t, wire, func(doc *sav.DocumentData) bool {
		purse := -1
		for i := range doc.Objects {
			if money, err := savedStructureValue(&doc.Objects[i], "Money"); err == nil && doc.Objects[i].Class == "Player" && money == uint32(f.Town.Gold()) {
				if purse >= 0 {
					return false
				}
				purse = i
			}
		}
		return purse >= 0 && savedStructureSetValue(&doc.Objects[purse], "Money", uint32(f.Town.Gold()+1)) == nil
	}), 0o600); err != nil {
		t.Fatal(err)
	}
	loss := releaseFront(t)
	openLocalTownSAV(t, loss, filepath.Dir(altered), filepath.Base(altered))
	if loss.Town.Gold() != f.Town.Gold()+1 {
		t.Fatal("altered purse did not load", loss.Town.Gold())
	}
	backPath := filepath.Join(store.Dir, "roundtrip.sav")
	if err := WriteConvertedSave(backPath, back, os.Getenv("AGAINROM_ASSETS")); err != nil {
		t.Fatal(err)
	}
	child("continue", saved, filepath.Join(store.Dir, "continue-from-sav.sav"))
	child("continue", backPath, filepath.Join(store.Dir, "continue-from-roundtrip.sav"))
	fromSAV, _ := ReadSaveFile(filepath.Join(store.Dir, "continue-from-sav.sav"))
	fromRoundTrip, _ := ReadSaveFile(filepath.Join(store.Dir, "continue-from-roundtrip.sav"))
	requireSameDocumentButKeys(t, fromSAV, fromRoundTrip)
	if got, err := os.ReadFile(sourcePath); err != nil || !bytes.Equal(got, source) {
		t.Fatal("lawful source changed")
	}
	if dir := os.Getenv("AGAINROM_1102_ARTIFACTS"); dir != "" {
		dir = filepath.Join(dir, filepath.Base(os.Getenv("AGAINROM_ASSETS")))
		for name, payload := range map[string][]byte{"sales.sav": wire, "roundtrip.sav": back, "continued.sav": fromSAV} {
			if err := WriteConvertedSave(filepath.Join(dir, name), payload, os.Getenv("AGAINROM_ASSETS")); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Logf("App partial/whole Sell + interleaved school + menu SAVE; source load181/container6 -> load180/container4; purse %d -> %d; SAV LOAD/SAVE equal, fresh continuation equal", gold, f.Town.Gold())
}

func TestCitySalesProcess(t *testing.T) {
	f := releaseFront(t)
	mode := os.Getenv("AGAINROM_1102_PROCESS")
	if mode == "" {
		return
	}
	input, err := ReadSaveFile(os.Getenv("AGAINROM_1102_INPUT"))
	if err != nil {
		t.Fatal(err)
	}
	// A fresh process LOADs through the App picker from its own store.
	store := SaveStore{Dir: t.TempDir()}
	if err := WriteConvertedSave(filepath.Join(store.Dir, "input.sav"), input, os.Getenv("AGAINROM_ASSETS")); err != nil {
		t.Fatal(err)
	}
	app := f.App("1102-fresh-city")
	save, list, load := f.SaveSeams(store, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	rows := list()
	if len(rows) != 1 {
		t.Fatal("fresh input not unique", rows)
	}
	if err := app.HeadlessActivate(rows[0].Label); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatal(err, app.Screen())
	}
	member := trainingPartyMember(t, f, "hero")
	h, ok := member.OriginalHumanState()
	if !ok || h.InventoryWeight != 4 || h.Load != 180 {
		t.Fatal("fresh retained source", h, ok)
	}
	slot := 0
	for i := 1; i <= 5; i++ {
		if h.Base.Skill[i] == 2 {
			slot = i
			break
		}
	}
	if slot == 0 {
		t.Fatal("trained slot absent")
	}
	before := f.Town.Gold()
	price := memberSchoolPrice(member, slot)
	if msg := train1099(t, f, "hero", slot-1); !strings.HasPrefix(msg, "trained ") {
		t.Fatal(msg)
	}
	if f.Town.Gold() != before-price {
		t.Fatal("fresh action debit")
	}
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	out, err := f.ExportOriginalSave(s, "fresh continuation")
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteConvertedSave(os.Getenv("AGAINROM_1102_OUTPUT"), out, os.Getenv("AGAINROM_ASSETS")); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("fresh source-backed training: load180/container4, price%d\n", price)
}

func saleDocument(t *testing.T, f *FrontEnd, label string) sav.DocumentData {
	t.Helper()
	snapshot, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportOriginalSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	d, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// A town SAV written after a LOAD renames the Player and Human constructor
// keys; b's keys map to a's by object before the fields are compared.
func requireSameDocumentButKeys(t *testing.T, a, b []byte) {
	t.Helper()
	first, errFirst := sav.DecodeDocumentData(a)
	second, errSecond := sav.DecodeDocumentData(b)
	if errFirst != nil || errSecond != nil || len(first.Objects) != len(second.Objects) {
		t.Fatal("town documents", errFirst, errSecond)
	}
	renamed := make(map[uint32]uint32)
	for i := range second.Objects {
		for _, name := range []string{"This", "Identity"} {
			old, errOld := savedStructureValue(&first.Objects[i], name)
			key, errKey := savedStructureValue(&second.Objects[i], name)
			if errOld == nil && errKey == nil && key != old {
				renamed[key] = old
			}
		}
	}
	for i := range second.Objects {
		walkDocumentValues(&second.Objects[i], func(v *sav.DocumentValueData) {
			if old, ok := renamed[v.Value]; ok {
				v.Value = old
			}
		})
	}
	if changed := documentFieldChanges(first, second); len(changed) != 0 {
		t.Fatalf("town SAVs differ in %v beyond %d renamed keys", changed, len(renamed))
	}
}

func walkDocumentValues(r *sav.DocumentRecordData, fn func(*sav.DocumentValueData)) {
	for i := range r.Values {
		fn(&r.Values[i])
	}
	for i := range r.Inline {
		walkDocumentValues(&r.Inline[i].Record, fn)
	}
	for i := range r.Groups {
		walkDocumentValues(&r.Groups[i], fn)
	}
}

// documentFieldChanges names every field that differs between two documents
// of the same object layout as "object:Class:kind:Name". A changed layout is a
// change of its own.
func documentFieldChanges(a, b sav.DocumentData) []string {
	var out []string
	for name, same := range map[string]bool{
		"head":     reflect.DeepEqual(a.Head, b.Head),
		"players":  slices.Equal(a.Players, b.Players) && slices.Equal(a.DeadActors, b.DeadActors),
		"label":    bytes.Equal(a.Label, b.Label),
		"trailer":  a.Marker == b.Marker && a.GlobalDWord == b.GlobalDWord && a.Trailer == b.Trailer,
		"state":    a.State.RootKind == b.State.RootKind && reflect.DeepEqual(a.State.DirectoryRecords, b.State.DirectoryRecords) && len(a.State.ValueRecords) == len(b.State.ValueRecords),
		"campaign": reflect.DeepEqual(a.Campaign, b.Campaign),
		"world":    reflect.DeepEqual(a.World, b.World),
	} {
		if !same {
			out = append(out, "document:"+name)
		}
	}
	for i := range min(len(a.State.ValueRecords), len(b.State.ValueRecords)) {
		if !reflect.DeepEqual(a.State.ValueRecords[i], b.State.ValueRecords[i]) {
			out = append(out, "state:"+a.State.ValueRecords[i].Path)
		}
	}
	slices.Sort(out)
	if len(a.Objects) != len(b.Objects) {
		return append(out, "objects")
	}
	var record func(prefix string, x, y sav.DocumentRecordData)
	record = func(prefix string, x, y sav.DocumentRecordData) {
		if x.Class != y.Class || len(x.Values) != len(y.Values) || len(x.Texts) != len(y.Texts) || len(x.Raw) != len(y.Raw) ||
			len(x.Counts) != len(y.Counts) || len(x.RefSlots) != len(y.RefSlots) || len(x.Inline) != len(y.Inline) || len(x.Groups) != len(y.Groups) {
			out = append(out, prefix+"layout")
			return
		}
		for i := range x.Values {
			if x.Values[i] != y.Values[i] {
				out = append(out, prefix+"value:"+x.Values[i].Name)
			}
		}
		for i := range x.Texts {
			if x.Texts[i] != y.Texts[i] {
				out = append(out, prefix+"text:"+x.Texts[i].Name)
			}
		}
		for i := range x.Raw {
			if x.Raw[i].Name != y.Raw[i].Name || !bytes.Equal(x.Raw[i].Bytes, y.Raw[i].Bytes) {
				out = append(out, prefix+"raw:"+x.Raw[i].Name)
			}
		}
		for i := range x.Counts {
			if x.Counts[i] != y.Counts[i] {
				out = append(out, prefix+"count:"+x.Counts[i].Name)
			}
		}
		for i := range x.RefSlots {
			if x.RefSlots[i].Name != y.RefSlots[i].Name || !slices.Equal(x.RefSlots[i].Objects, y.RefSlots[i].Objects) {
				out = append(out, prefix+"refs:"+x.RefSlots[i].Name)
			}
		}
		for i := range x.Inline {
			record(prefix+"inline:"+x.Inline[i].Name+":", x.Inline[i].Record, y.Inline[i].Record)
		}
		for i := range x.Groups {
			record(fmt.Sprintf("%sgroup%d:", prefix, i+1), x.Groups[i], y.Groups[i])
		}
	}
	for i := range a.Objects {
		record(fmt.Sprintf("%d:%s:", i+1, a.Objects[i].Class), a.Objects[i], b.Objects[i])
	}
	return out
}
