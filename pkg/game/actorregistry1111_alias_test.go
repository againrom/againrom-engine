package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/internal/cityfixture"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func aliasFront1111(t *testing.T) *FrontEnd {
	f := actorRegistryFront1111(t)
	rows := f.Table.Humans.(dbCollection)
	rows[1].name, rows[5].name = "PC_alias", "PC_basis"
	return f
}

// Three emitted Human objects, with one repeated reference to the middle
// object's identity. Every runtime/map ID is deliberately equal, so neither
// namespace can supply the expected uniqueness or order. The named hero is
// not first; its second occurrence must not defeat the single-leader rule.
func aliasBody1111(twoGroups bool) ([]byte, []*poolFixtureActor) {
	runtime := uint32(77)
	actors := []*poolFixtureActor{
		{human: true, name: "First companion", cell: 0x100f, hp: 11, maxHP: 31, runtime: &runtime},
		{human: true, name: "Repeated leader", cell: 0x120f, hp: 17, maxHP: 41, runtime: &runtime},
		{human: true, name: "Last companion", cell: 0x140f, hp: 23, maxHP: 51, runtime: &runtime},
	}
	groups := [][]*poolFixtureActor{{actors[0], actors[1], actors[2], actors[1]}}
	if twoGroups {
		groups = [][]*poolFixtureActor{actors, {actors[1]}}
	}
	body := poolFixtureBody([]*poolFixturePlayer{{groups: groups, hero: actors[1], reserve: 37}}, nil)
	for _, a := range actors {
		binary.LittleEndian.PutUint16(body[a.off+17:], 33)
		binary.LittleEndian.PutUint32(body[a.off+33:], 0xdeadbeef) // unresolved Token, resolved Player suffix
		body[a.off+189] = 100
	}
	return body, actors
}

func assertAliasWorld1111(t *testing.T, w *sim.World, ids []sim.EntityID, heroGroup uint32) {
	t.Helper()
	if !reflect.DeepEqual(ids, []sim.EntityID{1, 2, 3}) {
		t.Fatalf("unique leader-first party IDs: %v", ids)
	}
	seen := make(map[sim.EntityID]bool)
	sourceCount := 0
	for _, e := range w.Entities() {
		if seen[e.ID] {
			t.Fatal("duplicate native entity", e.ID)
		}
		seen[e.ID] = true
		if e.SourceBinding.Class == 0 {
			if e.ID != 0 || e.MapUnitID != 91 {
				t.Fatal("authored placement changed", e)
			}
			continue
		}
		sourceCount++
		want, ok := map[sim.EntityID]struct {
			index uint16
			hp    int32
			group uint32
		}{1: {5, 17, heroGroup}, 2: {4, 11, 1}, 3: {6, 23, 1}}[e.ID]
		if !ok || e.SourceBinding.ArchiveIndex != want.index || e.SourceBinding.Identity != 1000+uint32(want.index) ||
			e.SourceBinding.RuntimeID != 77 || e.MapUnitID != 0 || e.SourceBinding.Class != 2 ||
			e.SourceBinding.TokenRow != 1 || e.SourceBinding.TypeID != 33 || e.SourceBinding.GroupIndex != want.group ||
			e.SourceBinding.GroupOwnerResolved || e.SourceBinding.GroupOwnerKey != 0 || e.Owner != 1 ||
			e.HP != want.hp || !e.ActorLoad.Source.HasOwner || e.ActorLoad.Source.ManaReservePercent != 37 {
			t.Fatalf("one source object -> one complete binding: %+v", e)
		}
	}
	if sourceCount != 3 || len(seen) != 4 {
		t.Fatalf("literal population: %d source, %d total", sourceCount, len(seen))
	}
}

func assertAliasPresentation1111(t *testing.T, f *FrontEnd) {
	t.Helper()
	names := map[uint32]string{1: "Repeated leader", 2: "First companion", 3: "Last companion"}
	for _, draw := range f.live.entityDraws() {
		if name, ok := names[draw.ID]; ok {
			if draw.Name != name || draw.Art == nil || draw.Owner != 1 {
				t.Fatalf("alias render/person binding: %+v", draw)
			}
			delete(names, draw.ID)
		}
	}
	if len(names) != 0 || len(f.live.mission.party) != 3 || !f.live.mission.party[0].StartingHero ||
		f.live.mission.party[1].StartingHero || f.live.mission.party[2].StartingHero {
		t.Fatal("missing or duplicated named leader/presentation", names, f.live.mission.party)
	}
}

func TestActorRegistry1111PersistentAliasesBothDoorsNativeNextAction(t *testing.T) {
	if path := os.Getenv("AGAINROM_ALIAS1111_NATIVE"); path != "" {
		f := aliasFront1111(t)
		app := f.App("fresh alias process")
		save, list, load := f.SaveSeams(SaveStore{Dir: filepath.Dir(path)}, OriginalStore{}, nil)
		app.SetSaveSeams(save, list, load)
		groundAppLoad(t, app, list, filepath.Base(path))
		group := uint32(1)
		if os.Getenv("AGAINROM_ALIAS1111_TWO_GROUPS") == "true" {
			group = 2
		}
		assertAliasWorld1111(t, f.live.world, f.live.mission.ids, group)
		assertAliasPresentation1111(t, f)
		if fmt.Sprintf("%x", f.live.world.Hash()) != os.Getenv("AGAINROM_ALIAS1111_HASH") {
			t.Fatal("fresh native World differs")
		}
		if e, _ := f.live.entity(1); e.X != 16 || e.Y != 18 {
			t.Fatal("pre-SAVE action lost", e)
		}
		f.live.enqueue(1, 17, 18)
		registryReach1111(t, f.live, 1, 17, 18)
		t.Log("unique alias party fresh-process next action PASS")
		return
	}
	for _, twoGroups := range []bool{false, true} {
		t.Run(fmt.Sprintf("twoGroups=%t", twoGroups), func(t *testing.T) {
			body, actors := aliasBody1111(twoGroups)
			sf, err := sav.Open(savedContainer(body))
			if err != nil {
				t.Fatal(err)
			}
			raw, _, err := sf.PartyWalk()
			if err != nil || len(raw) != 4 || !raw[1].Hero || !raw[3].Hero {
				t.Fatal("raw archive occurrences changed", raw, err)
			}
			unique, err := sf.Party()
			if err != nil || len(unique) != 3 || unique[0].Off != actors[0].off || unique[1].Off != actors[1].off || unique[2].Off != actors[2].off {
				t.Fatal("first-reference projection lost distinct equal runtime/map IDs", unique, err)
			}
			group := uint32(1)
			if twoGroups {
				group = 2
			}
			f := aliasFront1111(t)
			ms, report, err := loadOriginalMission(f, savedContainer(body))
			if err != nil {
				t.Fatal(err)
			}
			if report.Party.Characters != 3 || report.Party.LeadRule != LeadNamed || report.Party.LeadFrom != 1 ||
				!reflect.DeepEqual(report.Party.SourceOffsets, []int{actors[1].off, actors[0].off, actors[2].off}) {
				t.Fatal("party/source vector or single leader differs", report.Party)
			}
			assertAliasWorld1111(t, ms.World, ms.Start.IDs, group)
			for _, fromMap := range []bool{false, true} {
				t.Run(fmt.Sprintf("fromMap=%t", fromMap), func(t *testing.T) {
					f := aliasFront1111(t)
					app := f.App("alias original")
					if fromMap {
						if err := app.OpenMission(f.MissionOpener(10)); err != nil {
							t.Fatal(err)
						}
					}
					dir := t.TempDir()
					if err := os.WriteFile(filepath.Join(dir, "game1111.sav"), savedContainer(body), 0600); err != nil {
						t.Fatal(err)
					}
					store := SaveStore{Dir: t.TempDir()}
					save, list, load := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
					app.SetSaveSeams(save, list, load)
					groundAppLoad(t, app, list, "game1111.sav")
					assertAliasWorld1111(t, f.live.world, f.live.mission.ids, group)
					assertAliasPresentation1111(t, f)
					if err := app.HeadlessKey("e"); err != nil || !slices.Contains(app.HeadlessSelection(), uint32(1)) {
						t.Fatal("unique leader is not selectable", app.HeadlessSelection(), err)
					}
					f.live.enqueue(1, 16, 18)
					registryReach1111(t, f.live, 1, 16, 18)
					if err := app.HeadlessKey("escape"); err != nil {
						t.Fatal(err)
					}
					if err := app.HeadlessGameMenuAction("save"); err != nil {
						t.Fatal(err)
					}
					entries, err := store.List()
					if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".sav" {
						t.Fatalf("menu SAVE: %+v %v; %s", entries, err, app.HeadlessMessage())
					}
					exe, err := os.Executable()
					if err != nil {
						t.Fatal(err)
					}
					cmd := exec.Command(exe, "-test.run=^TestActorRegistry1111PersistentAliasesBothDoorsNativeNextAction$", "-test.v")
					cmd.Env = append(os.Environ(), "AGAINROM_ALIAS1111_NATIVE="+filepath.Join(store.Dir, entries[0].Name),
						fmt.Sprintf("AGAINROM_ALIAS1111_TWO_GROUPS=%t", twoGroups), fmt.Sprintf("AGAINROM_ALIAS1111_HASH=%x", f.live.world.Hash()))
					output, err := cmd.CombinedOutput()
					if err != nil || !bytes.Contains(output, []byte("next action PASS")) {
						t.Fatalf("fresh native process: %v\n%s", err, output)
					}
					t.Log(string(output))
				})
			}
		})
	}
}

func TestActorRegistry1111AliasFinalPlayerOwnerAndCityProjection(t *testing.T) {
	a := &poolFixtureActor{human: true, name: "Rebound hero", cell: 0x100f, hp: 17, maxHP: 31}
	body := poolFixtureBody([]*poolFixturePlayer{
		{groups: [][]*poolFixtureActor{{a, a}}, hero: a, reserve: 17},
		{groups: [][]*poolFixtureActor{{a}}, reserve: 63},
	}, nil)
	binary.LittleEndian.PutUint16(body[a.off+17:], 33)
	binary.LittleEndian.PutUint32(body[a.off+33:], 900) // Token originally names Player1
	sf, err := sav.Open(savedContainer(body))
	if err != nil {
		t.Fatal(err)
	}
	f := aliasFront1111(t)
	source, err := originalCityCharacters(sf, f.Table)
	if err != nil || len(source) != 1 || !source[0].Hero || !source[0].Basis.Human.HasOwner || source[0].Basis.Human.ManaReservePercent != 63 {
		t.Fatal("shared city projection lost unique hero or final Player inputs", source, err)
	}
	party, report := RestoreParty(sf, nil, f.Bodies, f.Table)
	if len(party) != 1 || !party[0].StartingHero || report.LeadRule != LeadNamed || !reflect.DeepEqual(report.SourceOffsets, []int{a.off}) {
		t.Fatal("shared restore disagrees with source projection", party, report)
	}
	ms, _, err := loadOriginalMission(f, savedContainer(body))
	if err != nil {
		t.Fatal(err)
	}
	e := ms.World.Entities()[1]
	if e.SourceBinding.ArchiveIndex != 4 || e.Owner != 2 || e.SourceBinding.GroupIndex != 2 || e.SourceBinding.GroupOwnerResolved || e.ActorLoad.Source.ManaReservePercent != 63 {
		t.Fatal("dedup restored earlier Token/Player owner", e)
	}
}

func TestActorRegistry1111AliasLateFailuresKeepBindings(t *testing.T) {
	for _, twoGroups := range []bool{false, true} {
		body, actors := aliasBody1111(twoGroups)
		f := aliasFront1111(t)
		app := f.App("alias atomic current")
		open, _, err := f.RestoreOriginal(savedContainer(body))
		if err != nil {
			t.Fatal(err)
		}
		if err := app.OpenMission(open); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessKey("e"); err != nil {
			t.Fatal(err)
		}
		prior, town, art, hash, selection := f.live, f.Town, f.Units, f.live.world.Hash(), app.HeadlessSelection()
		unchanged := func(err error) {
			t.Helper()
			if err == nil || f.live != prior || f.Town != town || f.Units != art || f.live.world.Hash() != hash || !reflect.DeepEqual(selection, app.HeadlessSelection()) {
				t.Fatal("late alias candidate published", err)
			}
		}
		for _, collision := range []bool{false, true} {
			bad := append([]byte(nil), body...)
			if collision {
				// A different archive record with the hero's key is not an alias.
				binary.LittleEndian.PutUint32(bad[actors[2].off+29:], 1005)
			} else {
				bad[actors[2].off+510] = 0
			}
			if _, _, err := loadOriginalMission(f, savedContainer(bad)); err == nil {
				t.Fatal("diagnostic door admitted late invalid distinct actor")
			}
			open, _, err := f.RestoreOriginal(savedContainer(bad))
			if err == nil {
				err = app.OpenMission(open)
			}
			unchanged(err)
		}
		saved, _, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		saved.ActorManifest = cloneActorManifest(saved.ActorManifest)
		saved.ActorManifest.Actors[len(saved.ActorManifest.Actors)-1].ID = saved.ActorManifest.Actors[0].ID
		_, _, err = f.Restore(saved)
		unchanged(err)
	}
}

func TestActorRegistry1111AliasesCityNativeRoster(t *testing.T) {
	for _, twoGroups := range []bool{false, true} {
		t.Run(fmt.Sprintf("twoGroups=%t", twoGroups), func(t *testing.T) {
			source, err := cityfixture.Original(false)
			if err != nil {
				t.Fatal(err)
			}
			sf, err := sav.Open(source)
			if err != nil {
				t.Fatal(err)
			}
			_, record, err := sf.PartyWalk()
			if err != nil || len(record.Groups) != 1 {
				t.Fatal("synthetic city extent", err)
			}
			group := record.Groups[0]
			body := append([]byte(nil), sf.Body...)
			// Only the insertion site comes from the walker. The independent
			// city fixture emits exactly two Human objects, indices4/5; append
			// a reference to5, never a third body or a reminted identity.
			at, extra := group.End-12, []byte{5, 0}
			binary.LittleEndian.PutUint32(body[group.Off+84:], 3)
			if twoGroups {
				binary.LittleEndian.PutUint32(body[group.Off+84:], 2)
				binary.LittleEndian.PutUint32(body[group.Off-4:], 2)
				at, extra = group.End, make([]byte, 102) // 88 prefix + ref2 + tail12
				binary.LittleEndian.PutUint32(extra[84:], 1)
				binary.LittleEndian.PutUint16(extra[88:], 5)
			}
			body = append(append(append([]byte(nil), body[:at]...), extra...), body[at:]...)
			sf.Body = body // retain the fixture's separate framed state and campaign
			f, _ := city1095Front(t)
			if _, town, err := f.RestoreOriginal(sf.Marshal()); err != nil || !town {
				t.Fatal("city alias LOAD", town, err)
			}
			if len(f.Carried) != 2 || !f.Carried[0].StartingHero || f.Carried[1].StartingHero ||
				f.Carried[0].Name != "Leader" || f.Carried[1].Name != "Companion" || f.Carried[0].ID == f.Carried[1].ID {
				t.Fatal("city source/party projection disagrees", f.Carried)
			}
			if f.originalCity == nil {
				t.Fatal("missing city export boundary")
			}
			if f.originalCity.unavailable == nil || !strings.Contains(f.originalCity.unavailable.Error(), "repeats archive object") {
				t.Fatalf("alias original-writer boundary changed: %+v", f.originalCity.unavailable)
			}
			want := mapload.CloneParty(f.Carried)
			store := SaveStore{Dir: t.TempDir()}
			save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
			name, err := save(false)
			if err != nil || filepath.Ext(name) != ".sav" {
				t.Fatal("alias city must use SAV", name, err)
			}
			raw, err := ReadSaveFile(filepath.Join(store.Dir, name))
			if err != nil {
				t.Fatal(err)
			}
			fresh := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil), Table: f.Table, Bodies: f.Bodies}}
			for cycle := 0; cycle < 2; cycle++ {
				if _, town, err := fresh.RestoreOriginal(raw); err != nil || !town || !reflect.DeepEqual(want, fresh.Carried) {
					if len(want) == len(fresh.Carried) {
						for i := range want {
							x, y := reflect.ValueOf(want[i]), reflect.ValueOf(fresh.Carried[i])
							for field := 0; field < x.NumField(); field++ {
								if !reflect.DeepEqual(x.Field(field).Interface(), y.Field(field).Interface()) {
									t.Log("different member field", i, x.Type().Field(field).Name)
									if x.Type().Field(field).Name == "Carry" {
										t.Logf("actor loads: before %+v after %+v", want[i].Carry.LiveLoad, fresh.Carried[i].Carry.LiveLoad)
									}
								}
							}
						}
					}
					t.Fatal("fresh SAV city lost unique roster", cycle, town, err)
				}
				snapshot, label, err := fresh.Snapshot(false)
				if err != nil {
					t.Fatal(err)
				}
				raw, err = fresh.ExportCurrentSave(snapshot, label)
				if err != nil {
					t.Fatal("current alias roster did not save", err)
				}
			}
		})
	}
}
