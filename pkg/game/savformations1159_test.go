package game

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func formation1159Front(t *testing.T) *FrontEnd {
	t.Helper()
	// Literal ALM: instant7(+08 identifier2, raw255), then Group77 Move(30,30).
	// Command SelfSlot1 instead matches Player+04=1 on the OTHER Player.
	// This App fixture keeps the current original-reader +04==+08 admission;
	// the independent native controls discriminate unequal identifiers.
	b := make([]byte, 12+3*796+184)
	put := func(at int, n uint32) { binary.LittleEndian.PutUint32(b[at:], n) }
	put(0, 2)
	put(4+0x40, 7)
	put(4+0x44, 10)
	put(4+0x4c, 2)
	put(4+0x74, 3)
	put(4+0x50, 255)
	put(4+0x78, 1)
	a := 4 + 796
	put(a+0x40, 6)
	put(a+0x44, 11)
	put(a+0x4c, 77)
	put(a+0x74, 2)
	for i, v := range []uint32{4, 30, 30} {
		put(a+0x50+4*i, v)
		put(a+0x78+4*i, 1)
	}
	c := 4 + 2*796
	put(c, 1)
	c += 4
	put(c+0x40, 0x10002)
	put(c+0x44, 1)
	put(c+0x50, 1)
	put(c+0x74, 7)
	tr := 8 + 3*796
	put(tr, 1)
	tr += 4
	put(tr+0x80, 1)
	put(tr+0x84, 1)
	put(tr+0x98, 10)
	put(tr+0x9c, 11)
	put(tr+0xb4, 1)
	f := poolFixtureFrontMap(t, synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Type7Payload: b}))
	f.Table, f.Campaign = actorRegistryTable(), resolved(saveCampaign(), nil)
	return f
}

func formation1159Literal(t *testing.T, f *FrontEnd) []byte {
	t.Helper()
	// Both actors start inside the playable rectangle, beyond its eight-cell border.
	a := &poolFixtureActor{cell: 0x0c0c, hp: 100, maxHP: 100, name: "Formation A", loadWords: &[4]int16{20, 0, 0, 300}}
	b := &poolFixtureActor{cell: 0x0c0e, hp: 100, maxHP: 100, name: "Formation B", loadWords: &[4]int16{30, 0, 0, 300}}
	left, right := &poolFixturePlayer{}, &poolFixturePlayer{groups: [][]*poolFixtureActor{{a, b}}}
	body := poolFixtureBody([]*poolFixturePlayer{left, nil, left, right, right}, nil)
	for _, actor := range []*poolFixtureActor{a, b} {
		body[actor.off+16] = 1
		binary.LittleEndian.PutUint16(body[actor.off+17:], 35)
		body[actor.off+509], body[actor.off+510], body[actor.off+511] = 1, 1, 3
		body[actor.off+179], body[actor.off+189] = 64, 100
	}
	doc, err := sav.DecodeDocumentData(completeDocumentTail1115(t, f, savedContainer(body)))
	if err != nil {
		t.Fatal(err)
	}
	for i, index := range []uint16{doc.Players[0], doc.Players[3]} {
		p := &doc.Objects[index-1]
		newGroupSetValue1115(t, p, "Slot", uint32(2-i))
		newGroupSetValue1115(t, p, "SlotAgain", uint32(2-i))
		newGroupSetValue1115(t, p, "This", uint32(0x1111*(i+1)))
		for j := range p.Raw {
			if p.Raw[j].Name == "PRaw32" {
				for k := 0; k < 31; k++ {
					p.Raw[j].Bytes[k] = byte(10*i + k + 1)
				}
				p.Raw[j].Bytes[31] = byte(i)
			}
		}
	}
	g := &doc.Objects[doc.Players[3]-1].Groups[0]
	newGroupSetValue1115(t, g, "G1C", 77)
	newGroupSetValue1115(t, g, "G44", 0x1111)
	for i := range g.Raw {
		if g.Raw[i].Name == "G3C" {
			g.Raw[i].Bytes[0x45] = 1 // the fixture's live Group primary dispatch
		}
	}
	out, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func formation1159Initial(t *testing.T, raw []byte, f *FrontEnd) {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := nativeExpectedPlayers(file)
	if err != nil {
		t.Fatal(err)
	}
	locations, err := file.DocumentObjectLocations()
	if err != nil {
		t.Fatal(err)
	}
	bySource := map[uint16]*player1154Raw{}
	// Object locations give structural identity only. Expected fields and byte31
	// were read independently from original Body, before any Snapshot projection.
	for _, loc := range locations {
		if p := want.players[loc.ArchiveIndex]; p != nil {
			bySource[loc.ArchiveIndex] = p
		}
	}
	origins, err := players1154Origins(raw)
	if err != nil {
		t.Fatal(err)
	}
	state := f.live.mission.state.savedDocument
	got, present := f.live.world.SavedPlayerFormations()
	if !present || len(got) != len(want.players) {
		t.Fatalf("raw Player count=%d native=%d present=%v", len(want.players), len(got), present)
	}
	// Independent source root order and exact object-origin bridge; neither
	// natural identifier agreement nor the projected DTO supplies expected mode.
	seen := map[uint16]bool{}
	at := 0
	for _, index := range want.roots {
		if index == 0 || seen[index] {
			continue
		}
		seen[index] = true
		p := bySource[index]
		current := got[at]
		binding := state.GroupBindings.Players[at]
		if origins[index] == 0 || binding.ObjectIndex != origins[index] || current.PlayerID != binding.ID || current.CommandID != int16(uint16(p.values["Slot"])) || current.TriggerID != p.values["SlotAgain"] || current.Mode != p.tail[31] {
			t.Fatalf("raw original Player at%d disagrees with native %+v", p.off, current)
		}
		at++
	}
}

func formation1159CurrentProjection(t *testing.T, f *FrontEnd, initial Snapshot) {
	t.Helper()
	current, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	players, present := f.live.world.SavedPlayerFormations()
	if !present || len(players) != len(initial.SavedDocument.GroupBindings.Players) {
		t.Fatal("current exact formation population changed")
	}
	for i, p := range players {
		oldBinding := initial.SavedDocument.GroupBindings.Players[i]
		binding := current.SavedDocument.GroupBindings.Players[i]
		command, trigger, old, err := savedPlayerFormationFields(&initial.SavedDocument.Document.Objects[oldBinding.ObjectIndex-1])
		if err != nil {
			t.Fatal(err)
		}
		_, _, now, err := savedPlayerFormationFields(&current.SavedDocument.Document.Objects[binding.ObjectIndex-1])
		if err != nil || p.PlayerID != oldBinding.ID || p.PlayerID != binding.ID || p.CommandID != command || p.TriggerID != trigger || now[31] != p.Mode || !bytes.Equal(now[:31], old[:31]) {
			t.Fatal("current projection changed formation identity or opaque residue", p, err)
		}
	}
}

func formation1159SaveCycle(t *testing.T, raw []byte, front func(*testing.T) *FrontEnd, synthetic bool) {
	t.Helper()
	if _, err := OriginalSaveLabel(raw, 0); err != nil {
		t.Fatalf("fixture cannot enter the ordinary chooser: %v", err)
	}
	for _, door := range []string{"title", "map-menu"} {
		t.Run(door, func(t *testing.T) {
			f := front(t)
			f.SetDeterministicFrames(true)
			app := f.App("saved formation")
			app.Layout(1024, 768)
			path := filepath.Join(t.TempDir(), "game1159.sav")
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			store := SaveStore{Dir: t.TempDir()}
			save, list, load := nativeContinuationSeams1170(t, f, store, OriginalStore{Dir: filepath.Dir(path)}, nil)
			app.SetSaveSeams(save, list, load)
			if door == "map-menu" {
				groundAppLoad(t, app, list, "game1159.sav")
				formation1159Initial(t, raw, f)
			}
			prior := f.live
			groundAppLoad(t, app, list, "game1159.sav")
			if f.live == nil || f.live == prior {
				t.Fatal("original LOAD did not replace driver")
			}
			formation1159Initial(t, raw, f)
			initial, _, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			starts := map[sim.EntityID][2]int32{}
			for _, e := range f.live.world.Entities() {
				if e.SourceBinding.Class != 0 {
					starts[e.ID] = [2]int32{e.X, e.Y}
				}
			}
			if synthetic {
				groups, _, _ := f.live.world.SavedGroups()
				if len(groups) != 1 || groups[0].OwnerID != 1 || groups[0].ContainerID != 2 || groups[0].Owner.Owner != 2 {
					t.Fatal("discordant original owner/container collapsed", groups)
				}
				for _, e := range f.live.world.Entities() {
					if e.Owner != 1 {
						t.Fatal("actor did not retain enclosing owner", e.Owner)
					}
				}
			}
			beforeMode, found := f.live.world.CommandFormationMode(sim.SelfSlot)
			if !found {
				t.Fatal("fixture lacks current command target")
			}
			f.live.cycleFormation()
			beforeTick := f.live.world.Tick()
			f.live.tick()
			if f.live.world.Tick() <= beforeTick {
				t.Fatal("command tick did not advance")
			}
			afterMode, _ := f.live.world.CommandFormationMode(sim.SelfSlot)
			if afterMode == beforeMode {
				t.Fatal("real client formation command changed no byte")
			}
			if synthetic {
				for tick := 0; tick < 16; tick++ {
					p, _ := f.live.world.SavedPlayerFormations()
					if p[0].Mode == 255 {
						break
					}
					f.live.tick()
				}
				p, _ := f.live.world.SavedPlayerFormations()
				if p[0].Mode != 255 || p[1].Mode != 0 {
					t.Fatal("literal ALM instant7/client targeted wrong Players", p, f.live.world.Script().Triggers(), f.live.world.Script().Instants())
				}
				actors := 0
				for _, e := range f.live.world.Entities() {
					if e.SourceBinding.Class == 0 {
						continue
					}
					if !e.HasTarget || e.TargetX != int32(29+2*actors) || e.TargetY != 30 {
						t.Fatalf("script next Move actor%d target=%d,%d; issues=%v", e.ID, e.TargetX, e.TargetY, f.live.world.SavedGroupIssues())
					}
					actors++
				}
				if actors != 2 {
					t.Fatalf("script movement compared %d of two source actors", actors)
				}
			}
			current, _, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("first native SAVE tick=%d", f.live.world.Tick())
			for i, b := range current.SavedDocument.GroupBindings.Players {
				_, _, old, err := savedPlayerFormationFields(&initial.SavedDocument.Document.Objects[initial.SavedDocument.GroupBindings.Players[i].ObjectIndex-1])
				if err != nil {
					t.Fatal(err)
				}
				_, _, now, err := savedPlayerFormationFields(&current.SavedDocument.Document.Objects[b.ObjectIndex-1])
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(old[:31], now[:31]) {
					t.Fatal("formation projection changed opaque residue")
				}
				if synthetic && old[31] == now[31] {
					t.Fatal("current formation was not distinguished from the original byte")
				}
			}
			if err := app.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessGameMenuAction("save"); err != nil {
				t.Fatal(err)
			}
			entries, err := listAGS(store)
			if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".ags" {
				t.Fatal("ordinary menu SAVE", entries, err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			fresh := front(t)
			fresh.SetDeterministicFrames(true)
			app2 := fresh.App("source free formation LOAD")
			app2.Layout(1024, 768)
			save, list, load = nativeContinuationSeams1170(t, fresh, store, OriginalStore{}, nil)
			app2.SetSaveSeams(save, list, load)
			groundAppLoad(t, app2, list, entries[0].Name)
			if fresh.live.world.Hash() != f.live.world.Hash() {
				t.Fatal("source-free LOAD lost current native state")
			}
			for step := 0; step < 20; step++ {
				left, right := f.live.world.Tick(), fresh.live.world.Tick()
				f.live.tick()
				fresh.live.tick()
				if f.live.world.Tick() <= left || fresh.live.world.Tick() <= right || f.live.world.Hash() != fresh.live.world.Hash() {
					t.Fatalf("EACH advancing continuation step%d", step)
				}
			}
			if synthetic {
				for _, driver := range []*mapWorld{f.live, fresh.live} {
					moved := 0
					for _, e := range driver.world.Entities() {
						if start, source := starts[e.ID]; source && start != [2]int32{e.X, e.Y} {
							moved++
						}
					}
					if moved != 2 {
						t.Fatalf("twenty advancing steps physically moved %d of two source actors", moved)
					}
				}
			}
			// A new ordinary movement constructs a native Group. Its owner must
			// keep the current Player carrier through the second SAVE/LOAD cycle.
			for _, driver := range []*mapWorld{f.live, fresh.live} {
				var cmds []sim.Command
				for _, e := range driver.world.Entities() {
					if synthetic && e.SourceBinding.Class == 0 {
						continue
					}
					if e.Owner == sim.SelfSlot && e.Alive() && !e.OffMap {
						cmds = append(cmds, sim.Command{Kind: sim.KindGroupMoveTo, Entity: e.ID, X: 20, Y: 20})
						if len(cmds) == 2 {
							break
						}
					}
				}
				if len(cmds) == 0 {
					t.Fatal("next action lacks live owner subject")
				}
				t.Logf("next native Move at tick%d commands=%+v", driver.world.Tick(), cmds)
				driver.pending = append(driver.pending, cmds...)
				driver.tick()
				if synthetic {
					for _, e := range driver.world.Entities() {
						if e.SourceBinding.Class != 0 && (!e.HasTarget || e.TargetX != 20 || e.TargetY != 20) {
							t.Fatal("new native Group did not read current containing Player's off mode", e.ID, e.TargetX, e.TargetY)
						}
					}
				}
			}
			if f.live.world.Hash() != fresh.live.world.Hash() {
				t.Fatal("next action diverged")
			}
			leftPlayers, _ := f.live.world.SavedPlayerFormations()
			rightPlayers, _ := fresh.live.world.SavedPlayerFormations()
			leftGroups, _, _ := f.live.world.SavedGroups()
			rightGroups, _, _ := fresh.live.world.SavedGroups()
			if !slices.Equal(leftPlayers, rightPlayers) || !reflect.DeepEqual(leftGroups, rightGroups) {
				t.Fatal("cross-session exact formations or Group owners differ")
			}
			formation1159CurrentProjection(t, f, initial)
			formation1159CurrentProjection(t, fresh, initial)
			// The second SAVE belongs to the source-free session. Superseded
			// original-motion residue is retained independently of current World
			// state, so compare this writer with its own complete Snapshot.
			last, _, err := fresh.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			if err := app2.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if err := app2.HeadlessGameMenuAction("save"); err != nil {
				t.Fatal(err)
			}
			nextEntries, err := listAGS(store)
			if err != nil || len(nextEntries) != 2 {
				t.Fatal("second ordinary menu SAVE", nextEntries, err)
			}
			var secondName string
			for _, entry := range nextEntries {
				if entry.Name != entries[0].Name {
					secondName = entry.Name
				}
			}
			round, err := os.ReadFile(filepath.Join(store.Dir, secondName))
			if err != nil || secondName == "" {
				t.Fatal(err)
			}
			back, _, err := DecodeSave(round)
			if err != nil || !slices.Equal(back.World, last.World) || !reflect.DeepEqual(back.SavedDocument, last.SavedDocument) {
				t.Fatal("new native Group save lost formation", err)
			}
			t.Logf("%s: raw formation import, client change, menu SAVE, source removed, fresh LOAD, 20 EACH advancing steps, next Move and second native save; synthetic script/discordance=%v", door, synthetic)
		})
	}
}

func TestSavedFormation1159AppIdentityAndContinuation(t *testing.T) {
	f := formation1159Front(t)
	formation1159SaveCycle(t, formation1159Literal(t, f), formation1159Front, true)
}

func TestReleaseSavedFormation1159AppContinuation(t *testing.T) {
	for _, subject := range []struct{ path, sha string }{
		{"2026-08-24/game0021.sav", "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c"},
		{"2027-09-07/game0036.sav", "dd45e761d806ffe5d0ad2104d3dd21cd30f5985dcd5695ac47bdf2ce5348b530"},
	} {
		t.Run(subject.path, func(t *testing.T) {
			_, raw := groundCorpusFile(t, subject.path, subject.sha)
			formation1159SaveCycle(t, raw, releaseFront, false)
		})
	}
}
