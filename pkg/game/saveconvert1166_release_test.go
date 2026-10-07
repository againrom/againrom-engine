package game

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
)

func city1166Check(t *testing.T, f *FrontEnd, want Snapshot) {
	t.Helper()
	if f.Town.Gold() != want.Gold || len(f.Carried) != len(want.Party) {
		t.Fatalf("current city: gold%d/%d party%d/%d", f.Town.Gold(), want.Gold, len(f.Carried), len(want.Party))
	}
	for i, before := range want.Party {
		after := f.Carried[i]
		if after.Name != before.Name || after.StartingHero != before.StartingHero || after.CompanionNPC != before.CompanionNPC || after.MercenaryType != before.MercenaryType || after.Hero != before.Hero || after.Worn != before.Worn || !reflect.DeepEqual(after.Carried, before.Carried) || after.KnownSpells != before.KnownSpells {
			t.Fatalf("current party member%d changed: %s/%s", i, before.Name, after.Name)
		}
	}
	for typ := 1; typ < 16; typ++ {
		if f.Town.MercenaryPool(typ) != want.MercenaryPool[typ] || f.Town.MercenaryEnabled(typ) != want.MercenaryEnabled[typ] || f.Town.MercenaryHired(typ) != want.MercenaryHired[typ] {
			t.Fatalf("current mercenary state%d", typ)
		}
	}
	for _, mission := range want.Won {
		if !f.Town.Done(mission) {
			t.Fatalf("lost completed mission%d", mission)
		}
	}
}

func TestReleaseGeneratedCityConversion1166(t *testing.T) {
	f := releaseFront(t)
	root := os.Getenv("AGAINROM_ASSETS")
	hero, screen := reachabilityWalkArrive(t, f, "Converted Hero")
	f.addChapterCompanions(f.Town.Chapter())
	f.Town.gold = 5_000_000 // Controlled purchase budget, not a campaign reward.
	if _, ok := screen.toggleMercenary(14); !ok {
		t.Fatal("ordinary tavern hire refused")
	}
	beforeTraining := trainingPartyMember(t, f, hero.ID)
	price := memberSchoolPrice(beforeTraining, 1)
	beforeGold := f.Town.Gold()
	if msg := train1099(t, f, hero.ID, 0); !strings.HasPrefix(msg, "trained ") {
		t.Fatal("ordinary school action", msg)
	}
	if f.Town.Gold() != beforeGold-price {
		t.Fatal("school action did not debit its current price")
	}
	want, _, err := f.Snapshot(false)
	if err != nil || want.OriginalCity != nil || want.CampaignState || len(want.World) != 0 {
		t.Fatal("generated city input", err)
	}
	if want.Gold != 4999280 || len(want.Party) != 5 {
		t.Fatalf("installed hire/training result: gold%d members%d", want.Gold, len(want.Party))
	}
	label := "generated\xffcity"
	input, err := EncodeSave(want, label)
	if err != nil {
		t.Fatal(err)
	}
	inputCopy := bytes.Clone(input)
	converter := releaseFront(t)
	output, keptLabel, err := converter.ConvertCitySave(input, "sav", nil)
	if err != nil || keptLabel != label {
		t.Fatal("generated AGS to SAV", keptLabel, err)
	}
	if !bytes.Equal(input, inputCopy) {
		t.Fatal("conversion changed source bytes")
	}
	// converter's own live Town is untouched by the write: the repair below
	// only ever changes the bytes ExportNativeCitySave produces, never the
	// live session that produced them (nativecity.go copies t.mercEnabled
	// rather than mutating it in place), so this first check still wants the
	// state the live session actually reached on its own.
	city1166Check(t, converter, want)
	store := SaveStore{Dir: t.TempDir()}
	if err := WriteConvertedSave(filepath.Join(store.Dir, "converted.sav"), output, root); err != nil {
		t.Fatal(err)
	}
	// A native SAV write now repairs the auto-prologue's own permanent
	// mercenary unlock (nativecity.go's impliedMercenaryUnlocks) rather than
	// leaving it blank, since this fixture's live Town arrived straight at
	// the town boundary and never called Town.Won for the missions before
	// it. The LOAD below installs that repaired PermanentMercenaries
	// unconditionally, so from here on "want" must name the state a native
	// SAV of this city carries, not the narrower state the live session
	// happened to reach on its own.
	for _, implied := range impliedMercenaryUnlocks(f.Campaign.Value(), f.Town.Chapter()) {
		for _, typ := range f.Campaign.Value().Chapters[implied].EnableMercenary {
			if typ > 0 && typ < len(want.MercenaryEnabled) {
				want.MercenaryEnabled[typ] = true
			}
		}
	}
	fresh := releaseFront(t)
	save, list, load := agsSaveSeams(fresh, store, OriginalStore{}, nil)
	rows := list()
	if len(rows) != 1 {
		t.Fatal(rows)
	}
	if open, town, err := load(rows[0].Name); err != nil || !town || open != nil {
		t.Fatal("source-free production LOAD", town, err)
	}
	city1166Check(t, fresh, want)
	// Actual school input after the new SAV LOAD must remain usable, including
	// the imported writer that the loaded city now owns.
	nextHero := trainingPartyMember(t, fresh, hero.ID)
	nextPrice := memberSchoolPrice(nextHero, 1)
	if msg := train1099(t, fresh, hero.ID, 0); !strings.HasPrefix(msg, "trained ") {
		t.Fatal("next school action", msg)
	}
	if fresh.Town.Gold() != want.Gold-nextPrice {
		t.Fatal("next school debit")
	}
	if name, err := save(false); err != nil || !IsOriginal(name) {
		t.Fatal("next ordinary SAVE", name, err)
	}
	for _, test := range []struct {
		name, carrier string
		change        func(*FrontEnd)
		held          func(*FrontEnd) bool
		written       func(*currentActionData) bool
		strip         func(*currentActionData)
	}{
		{"custom quick spell", "AgainromActions session (DIV-1369)",
			func(g *FrontEnd) { g.quickSpells[0] = 65535 },
			func(g *FrontEnd) bool { return g.quickSpells[0] == 65535 },
			func(a *currentActionData) bool {
				return a.Session != nil && slices.Contains(a.Session.QuickSpells, currentQuickSpell{Slot: 0, ID: 65535})
			},
			func(a *currentActionData) { a.Session.QuickSpells = nil }},
		{"noncurrent advisory", "AgainromActions session (DIV-1369)",
			func(g *FrontEnd) { g.Offered = 31 },
			func(g *FrontEnd) bool { return g.Offered == 31 },
			func(a *currentActionData) bool { return a.Session != nil && a.Session.Offered == 31 },
			func(a *currentActionData) { a.Session.Offered = 0 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := releaseFront(t)
			openLocalTownSAV(t, g, store.Dir, "converted.sav")
			if test.held(g) {
				t.Fatal("control not armed")
			}
			test.change(g)
			if !test.held(g) {
				t.Fatal("live change not held")
			}
			raw, _, loaded := townSAVRoundTrip(t, g)
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			if a, err := readCurrentActions(&doc); err != nil || a == nil || !test.written(a) {
				t.Fatal("SAV does not carry the current value", err)
			}
			if !test.held(loaded) {
				t.Fatal("LOAD through the load window lost the current value")
			}
			stripped := alterSAV(t, raw, func(doc *sav.DocumentData) bool {
				a, err := readCurrentActions(doc)
				if err != nil || a == nil {
					return false
				}
				test.strip(a)
				b, err := json.Marshal(a)
				return err == nil && sav.SetNativeActions(&doc.State, b) == nil
			})
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "altered.sav"), stripped, 0o600); err != nil {
				t.Fatal(err)
			}
			lossy := releaseFront(t)
			openLocalTownSAV(t, lossy, dir, "altered.sav")
			if test.held(lossy) {
				t.Fatal("the value survived its removal from the document")
			}
			for _, front := range []*FrontEnd{g, loaded} {
				if msg := schoolTrain(front, hero.ID); !strings.HasPrefix(msg, "trained ") {
					t.Fatal("next school action", msg)
				}
			}
			live, cold := trainingPartyMember(t, g, hero.ID), trainingPartyMember(t, loaded, hero.ID)
			if g.Town.Gold() != loaded.Town.Gold() || live.Carry.SkillXP != cold.Carry.SkillXP || g.Offered != loaded.Offered || g.quickSpells != loaded.quickSpells {
				t.Fatalf("next school action diverged: gold %d/%d", g.Town.Gold(), loaded.Town.Gold())
			}
			t.Logf("%s written from current state into %s; load window LOAD keeps it; stripped copy loads without it", test.name, test.carrier)
		})
	}
	if dir := os.Getenv("AGAINROM_1166_ARTIFACTS"); dir != "" {
		for name, b := range map[string][]byte{"generated.ags": input, "converted.sav": output} {
			if err := WriteConvertedSave(filepath.Join(dir, name), b, root); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Logf("generated city with ordinary hire and school action: %d members, gold%d; source-free LOAD, next school action and ordinary SAV; source label bytes retained", len(want.Party), want.Gold)
}
