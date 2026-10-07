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
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func playerWireN4(t *testing.T, raw []byte) (*sav.File, sav.DocumentData) {
	t.Helper()
	f, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	d, _, err := sav.DecodeDocumentDataWithOrigins(raw)
	if err != nil {
		t.Fatal(err)
	}
	return f, d
}

func playerSettingsN4(t *testing.T, d sav.DocumentData) []byte {
	t.Helper()
	for _, object := range d.Objects {
		if object.Class != "Player" {
			continue
		}
		for _, raw := range object.Raw {
			if raw.Name == "PRaw32" {
				return raw.Bytes
			}
		}
	}
	t.Fatal("emitted Player has no settings block")
	return nil
}

func checkPlayerWireN4(t *testing.T, raw []byte, mode uint8, setting int32, current ...map[string]uint32) {
	t.Helper()
	f, d := playerWireN4(t, raw)
	expected := map[string]uint32{"F44": 0, "F50": 0, "F58": 95, "Players": 1}
	if len(current) != 0 {
		for key, value := range current[0] {
			expected[key] = value
		}
	}
	if len(f.Players) != int(expected["Players"]) || f.Players[0].Slot != 1 || f.Players[0].Participant != 0 {
		t.Fatal("native Player role", f.Players)
	}
	p := f.Players[0]
	// Read final Body bytes through the independent Player grammar locators.
	mask := binary.LittleEndian.Uint16(f.Body[p.Fields.Off["F2C"]:])
	wantMask := uint16(1) << uint(p.Slot%16) // SAV-664, not the writer's helper.
	if mask != wantMask {
		t.Errorf("final Player mask=%#x; registered slot%d/participant0 requires %#x", mask, p.Slot, wantMask)
	}
	if uint32(f.Body[p.Fields.Off["F44"]]) != expected["F44"] {
		t.Error("Player colour differs from current owner or fresh constructor")
	}
	for _, name := range []string{"F50", "F58"} {
		want := expected[name]
		if got, ok := p.Field(name); !ok || got != want {
			t.Errorf("known Player constructor field %s=%d/%v want%d", name, got, ok, want)
		}
	}
	wantSettings := make([]byte, 32)
	wantSettings[31] = mode // Fresh default or the preserved current choice.
	gotSettings := playerSettingsN4(t, d)
	if !reflect.DeepEqual(gotSettings, wantSettings) {
		t.Errorf("final Player settings=%x; want complete constructor %x", gotSettings, wantSettings)
	}
	found := false
	for _, r := range d.State.ValueRecords {
		if r.Path == "/GameOptions/Formation" {
			found = true
			if r.Value.Kind != 2 || r.Value.Int32 != setting {
				t.Errorf("application formation=%+v want%d", r.Value, setting)
			}
		}
	}
	if !found {
		t.Error("missing application formation")
	}
	graph, err := f.ActorGraph()
	if err != nil {
		t.Fatal(err)
	}
	hero, _ := p.Field("Hero")
	heroes := 0
	for _, a := range graph.Actors {
		if a.OwnerSlot != a.TokenOwnerSlot || len(current) == 0 && a.OwnerSlot != p.Slot {
			t.Fatal("generated actor ownership changed", a.Identity)
		}
		if a.Identity == hero {
			if a.OwnerSlot != p.Slot {
				t.Fatal("current hero belongs to a different Player")
			}
			heroes++
		}
	}
	if heroes != 1 {
		t.Fatal("hero key has no unique owned actor", heroes)
	}
	// SAV-1093: town and mission producers publish every Human with mask 2.
	wantT18 := uint32(2)
	for _, r := range d.Objects {
		if r.Class != "Human" {
			continue
		}
		for _, v := range r.Values {
			if v.Name == "T18" && v.Value != wantT18 {
				t.Fatal("Human Token publication state mismatch", v.Value, "want", wantT18)
			}
		}
	}
	// Apply only SAV-678's published helper laws to the emitted mask. This is
	// a mathematical admission oracle, not an original message/runtime run.
	var token uint16
	publications := 0
	for i := 0; i < 3; i++ {
		if token&mask == 0 {
			publications++
			token |= mask
		}
	}
	if publications != 1 || token != wantMask {
		t.Errorf("published helper laws never stabilize: mask%#x publications%d token%#x", mask, publications, token)
	}
	t.Logf("wire Player slot%d mask%#x settings%x; helper-law first-publication count%d", p.Slot, mask, gotSettings, publications)
}

func TestReleaseNativeCityPlayerConstruction(t *testing.T) {
	releaseFront(t)
	for sex := 0; sex < 2; sex++ {
		for mage := 0; mage < 2; mage++ {
			t.Run(fmt.Sprintf("sex%d_class%d", sex, mage), func(t *testing.T) {
				f := releaseFront(t)
				f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Player construction", Choices: []int{sex, mage, 0}, Stats: []int{30, 30, 30, 30}})
				f.arriveInTown()
				f.addChapterCompanions(f.Town.Chapter())
				s, _, err := f.Snapshot(false)
				if err != nil {
					t.Fatal(err)
				}
				raw, err := f.ExportNativeCitySave(s, "Player construction")
				if err != nil {
					t.Fatal(err)
				}
				checkPlayerWireN4(t, raw, 2, 1)
			})
		}
	}
	t.Run("explicit_town_application", func(t *testing.T) {
		f := releaseFront(t)
		f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Stored choice", Choices: []int{0, 0, 0}, Stats: []int{30, 30, 30, 30}})
		f.arriveInTown()
		f.addChapterCompanions(f.Town.Chapter())
		s, _, err := f.Snapshot(false)
		if err != nil {
			t.Fatal(err)
		}
		view := ui.SaveApplicationState{Zoom: 1, PeriodUS: 62000}
		for _, tc := range []struct {
			setting int32
			mode    uint8
		}{{0, 0}, {1, 2}, {2, 1}, {9, 2}} {
			s.ApplicationState = &SnapshotApplicationState{Version: 1, View: view, Baseline: view, Original: OriginalStateData{Formation: tc.setting}}
			before := cloneApplicationState(s.ApplicationState)
			raw, err := f.ExportNativeCitySave(s, "Stored choice")
			if err != nil {
				t.Fatal(err)
			}
			checkPlayerWireN4(t, raw, tc.mode, tc.setting)
			if !reflect.DeepEqual(before, s.ApplicationState) {
				t.Fatal("export changed explicit town application")
			}
		}
	})
	t.Run("current_mission_and_pending_choice", func(t *testing.T) {
		f := releaseFront(t)
		f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Mission choice", Choices: []int{0, 0, 0}, Stats: []int{30, 30, 30, 30}})
		f.arriveInTown()
		f.addChapterCompanions(f.Town.Chapter())
		app := f.App("native Player formation")
		if err := app.OpenMission(f.MissionOpenerWith(f.Town.Chapter(), f.NextParty())); err != nil {
			t.Fatal(err)
		}
		view := f.live.view.SaveApplication()
		f.live.applicationState = &SnapshotApplicationState{Version: 1, View: view, Baseline: view, Original: OriginalStateData{Formation: 1}}
		check := func(mode uint8, setting int32) {
			t.Helper()
			s, _, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			before := f.live.world.Hash()
			pending := append(f.live.pending[:0:0], f.live.pending...)
			raw, err := f.ExportCurrentWorldSave(s, "Current formation")
			if err != nil {
				t.Fatal(err)
			}
			percent, _ := f.live.world.AutoHealing(sim.SelfSlot)
			checkPlayerWireN4(t, raw, mode, setting, map[string]uint32{"Players": uint32(len(f.live.mission.state.Map.Groups)), "F44": f.live.mission.state.Map.Groups[sim.SelfSlot-1].Color + 1, "F58": percent})
			if f.live.world.Hash() != before || !reflect.DeepEqual(pending, f.live.pending) {
				t.Fatal("current mission SAVE changed live world or queued command")
			}
		}
		check(2, 1)
		f.live.cycleFormation() // Auto -> On, then apply the real command.
		f.live.tick()
		check(1, 2)
		f.live.cycleFormation() // Queued Off: authored0, canonical still1.
		check(1, 0)
		f.live.tick()
		check(0, 0)
		f.live.cycleFormation()
		f.live.tick()
		check(2, 1)
	})
	if root := os.Getenv("AGAINROM_N4_AUDIT_ROOT"); root != "" {
		t.Run("preserved_N4_source", func(t *testing.T) {
			read := func(path, hash string) []byte {
				t.Helper()
				raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
				if err != nil {
					t.Fatal(err)
				}
				if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != hash {
					t.Fatalf("immutable input SHA changed: %s %s", path, got)
				}
				return raw
			}
			prior := read("owner-N4-05259d2/candidates/game9015.sav", "8ffa9d0da3bde05ca3392e6471f32a1553cb95dee364f1eb506639930b0d6aaa")
			old, oldDoc := playerWireN4(t, prior)
			if m, _ := old.Players[0].Field("F2C"); m != 0 || playerSettingsN4(t, oldDoc)[31] != 0 {
				t.Fatal("preserved old-constructor discriminator changed")
			}
			baseline := read("story1168/f68f3f9/en/TestReleaseTownReturnCurrentCampaign1168-fresh-mission20-baseline.ags", "5b04851951e81595d2a8ec12d4229c98ea490deba32a20235b6b2616ba7054af")
			s, _, err := DecodeSave(baseline)
			if err != nil {
				t.Fatal(err)
			}
			f := releaseFront(t)
			if _, town, err := f.Restore(s); err != nil || !town {
				t.Fatal(town, err)
			}
			raw, err := f.ExportNativeCitySave(s, "Player constructor proof")
			if err != nil {
				t.Fatal(err)
			}
			checkPlayerWireN4(t, raw, 2, 1)
			next, nextDoc := playerWireN4(t, raw)
			checkPreservedPlayerN4(t, f, s, old, oldDoc, next, nextDoc)
			restored := currentTownReload(t, raw)
			if len(restored.Carried) != len(s.Party) || restored.Town.gold != s.Gold || restored.Town.Chapter() != s.Offered {
				t.Fatal("N4 cold LOAD changed party population, gold or current chapter")
			}
			for i, want := range s.Party {
				got := restored.Carried[i]
				wantWorn, wantPack := memberItemCodes(want)
				gotWorn, gotPack := memberItemCodes(got)
				if got.ID != want.ID || got.Name != want.Name || got.Hero != want.Hero || got.Class != want.Class ||
					got.StartingHero != want.StartingHero || got.KnownSpells != want.KnownSpells || !reflect.DeepEqual(got.Weapon, want.Weapon) ||
					gotWorn != wantWorn || !slices.Equal(gotPack, wantPack) {
					t.Fatalf("N4 cold LOAD changed current party member %d (%q)", i, want.ID)
				}
			}
		})
	}
}

// The old witness predates the unified producer. Keep its entire ordinary
// document as an independent oracle, changing only demonstrated constructor
// fields. Allocator keys may change; their bijection and every graph edge may not.
func checkPreservedPlayerN4(t *testing.T, f *FrontEnd, s Snapshot, old *sav.File, want sav.DocumentData, next *sav.File, got sav.DocumentData) {
	t.Helper()
	body := func(d sav.DocumentData) []byte {
		t.Helper()
		b, err := sav.EncodeDocumentData(d)
		if err != nil {
			t.Fatal(err)
		}
		file, err := sav.Open(b)
		if err != nil {
			t.Fatal(err)
		}
		return file.Body
	}
	if !bytes.Equal(body(want), old.Body) {
		t.Fatal("N4 oracle codec changed the preserved ordinary Body")
	}
	if s.OriginalCity != nil || s.CityObjects != nil || len(s.Party) != 2 || s.Party[0].ID != "hero" || s.Party[0].Name != "Town return" || s.Party[1].ID != "npc:22" {
		t.Fatal("N4 legacy input is no longer the two-member source-less city")
	}
	pack := mapload.MemberCarriedItems(s.Party[0], f.Table)
	if len(pack) != 1 || pack[0].Code != 0x0e1c || pack[0].Weight != 0 || pack[0].WeightPresent ||
		s.Party[0].Carry != nil && s.Party[0].Carry.LiveLoad != nil {
		t.Fatal("N4 legacy document no longer has code-only, zero-weight pack state")
	}
	value := func(r *sav.DocumentRecordData, name string) *uint32 {
		t.Helper()
		for i := range r.Values {
			if r.Values[i].Name == name {
				return &r.Values[i].Value
			}
		}
		t.Fatalf("N4 %s has no %s", r.Class, name)
		return nil
	}
	classes := []string{"Player", "Human", "Weapon", "Item", "Armor", "Human", "Weapon", "Effect", "Spell", "Spell", "Spell", "Spell", "Spell", "Armor", "Armor"}
	if len(want.Objects) != len(classes) || len(got.Objects) != len(classes) {
		t.Fatal("N4 object population changed")
	}
	keys, seenKeys, seenRuntime := map[uint32]uint32{}, map[uint32]bool{}, map[uint32]bool{}
	for i, class := range classes {
		a, b := &want.Objects[i], &got.Objects[i]
		if a.Class != class || b.Class != class {
			t.Fatalf("N4 object %d changed class/order", i+1)
		}
		field := "Identity"
		if class == "Player" || class == "Spell" {
			field = "This"
		}
		before, after := value(a, field), *value(b, field)
		if *before == 0 || after == 0 || keys[*before] != 0 || seenKeys[after] {
			t.Fatal("N4 allocator keys are not a nonzero bijection")
		}
		keys[*before], seenKeys[after], *before = after, true, after
		if field == "Identity" {
			id := *value(b, "RuntimeID")
			if id == 0 || seenRuntime[id] {
				t.Fatal("N4 runtime identities are absent or repeated")
			}
			seenRuntime[id], *value(a, "RuntimeID") = true, id
		}
	}
	rebase := func(r *sav.DocumentRecordData, field string) {
		t.Helper()
		v := value(r, field)
		key, ok := keys[*v]
		if !ok {
			t.Fatalf("N4 %s.%s has an unresolved source key", r.Class, field)
		}
		*v = key
	}
	want.Head.PlayerListField = 2 // One registered Player plus the end slot.
	p := &want.Objects[0]
	*value(p, "F2C") = 2
	rebase(p, "Hero")
	for i := range p.Texts {
		if p.Texts[i].Name == "Name" {
			p.Texts[i].Value = s.Party[0].Name
		}
	}
	playerSettingsN4(t, want)[31] = 2
	if len(p.Inline) != 1 || p.Inline[0].Name != "Diary" || len(p.Groups) != 1 {
		t.Fatal("N4 inline Diary/group population changed")
	}
	rebase(&p.Inline[0].Record, "D2C")
	rebase(&p.Groups[0], "G44")
	for i := range want.Objects {
		r := &want.Objects[i]
		switch r.Class {
		case "Human":
			rebase(r, "Reference")
			*value(r, "T18") = 2
		case "Weapon", "Armor", "Item", "Effect":
			// No retained Item/Effect token exists in this legacy snapshot.
			// Native constructor policy (DIV-1214) leaves Reference zero.
			*value(r, "Reference") = 0
		}
	}
	*value(&want.Objects[1], "Inventory20") = 0 // Current pack weight.
	*value(&want.Objects[3], "F4A") = 0         // Code-only document weight.
	*value(&want.Objects[7], "T0E") = 33        // Native Effect constructor.
	if !bytes.Equal(body(want), next.Body) {
		t.Fatal("N4 ordinary Body differs beyond the named fields/key bijection, including the complete Human/Item/Spell graph")
	}
	if string(got.Label) != "Player constructor proof" {
		t.Fatal("N4 current save label changed")
	}
	// A typed continuation is additional current state, never a replacement
	// for the ordinary graph above. Cold LOAD below checks its source values.
	a, err := readCurrentActions(&got)
	if err != nil || a == nil || len(a.Party) != 2 || len(a.Bindings) != 2 || a.Session == nil {
		t.Fatal("N4 current continuation missing or invalid", err)
	}
	for i, object := range []uint16{2, 6} {
		p := a.Party[i]
		if a.Bindings[i] != (currentActionBinding{ID: sim.EntityID(i), Object: object}) || p.Entity != sim.EntityID(i) ||
			string(p.ID) != s.Party[i].ID || p.Member != nil || p.Policy == nil || p.Base == nil || p.City == nil {
			t.Fatal("N4 current continuation changed party identity/order or replayed a whole member")
		}
	}
	if a.Session.Offered != s.Offered || a.Session.DocumentMission == nil || *a.Session.DocumentMission != s.DocumentMission || !slices.Equal(a.Session.Won, s.Won) {
		t.Fatal("N4 current continuation changed session progress")
	}
	removed := 0
	for i := 0; i < len(got.State.ValueRecords); i++ {
		if got.State.ValueRecords[i].Path == "/CurrentState/AgainromActions" {
			got.State.ValueRecords = slices.Delete(got.State.ValueRecords, i, i+1)
			removed++
			i--
		}
	}
	for i := range want.State.ValueRecords {
		if want.State.ValueRecords[i].Path == "/GameOptions/Formation" {
			want.State.ValueRecords[i].Value.Int32 = 1
		}
	}
	if removed != 1 || !reflect.DeepEqual(want.State, got.State) {
		t.Fatal("N4 state changed beyond formation and its validated current continuation")
	}
	// The historical writer omitted these derived campaign fields. Read the
	// installed definition directly; do not call the producer's projection.
	c := f.Campaign.Value()
	_, hasAuto := c.Auto[30]
	if c.MapObjects[30] != 9 || c.MapObjects[31] != 33 || hasAuto ||
		len(c.Main) < 3 || !slices.Equal(c.Main[:3], []int{10, 20, 30}) || len(c.Offered) == 0 || c.Offered[0] != 30 {
		t.Fatal("N4 installed campaign discriminator changed")
	}
	unlocks := map[int]bool{}
	for _, mission := range []int{10, 20} {
		for _, typ := range c.Chapters[mission].EnableMercenary {
			unlocks[typ] = true
		}
	}
	wantUnlocks := []uint16{2, 3, 4, 6, 7, 9, 12, 13, 14}
	if len(unlocks) != len(wantUnlocks) {
		t.Fatal("N4 installed prologue unlock count changed")
	}
	for _, typ := range wantUnlocks {
		if !unlocks[int(typ)] {
			t.Fatal("N4 installed prologue unlock changed", typ)
		}
	}
	want.Campaign.Base.DWords[1] = 9
	want.Campaign.Children[0].Base.DWords[1] = 33
	want.Campaign.Arrays[1] = wantUnlocks
	want.Campaign.Scalars[2] = ^uint32(0)
	if !reflect.DeepEqual(want.Campaign, got.Campaign) {
		t.Fatal("N4 campaign changed beyond installed map objects, prologue unlocks and absent automatic successor")
	}
}
