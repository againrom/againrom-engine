package game

import (
	"io/fs"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type flightCatalogSource map[string][]byte

func (s flightCatalogSource) ReadFile(name string) ([]byte, error) {
	if b, ok := s[name]; ok {
		return b, nil
	}
	return nil, fs.ErrNotExist
}

func TestFlightInputsUseRawRegistryWithoutArtOrPictureCeiling(t *testing.T) {
	i := func(name string, n int32) synth.RegNode { return synth.RegNode{Name: name, Kind: 2, Int: n} }
	src := flightCatalogSource{
		ProjectileRegistry: synth.ProjectilesReg(2,
			[]synth.RegNode{i("ID", 1007), i("Phases", 37)},
			[]synth.RegNode{i("ID", 10), i("Phases", 11), i("Homing", 1)}),
		UnitRegistry: synth.UnitsReg(nil, []synth.RegNode{i("ID", 9), i("Projectile", 1007), i("ShootDelay", 3),
			i("TileSize", 2), i("CenterX", 8), i("CenterY", 10), i("SelectionX1", 2), i("SelectionX2", 22),
			i("SelectionY1", 3), i("SelectionY2", 27)}),
	}
	in, err := loadFlightInputs(src, data.FirstSpellPresentation(), firstGameFiles.flight)
	if err != nil || in == nil {
		t.Fatalf("raw inputs: %v, %v", in, err)
	}
	if p := in.Pictures[1007]; !p.Present || p.Phases != 37 || p.Homes {
		t.Fatalf("arbitrary raw picture lost: %+v", p)
	}
	if p := in.Pictures[10]; !p.Present || !p.Homes || p.Phases != 11 || p.LifeDivisor != 200 {
		t.Fatalf("raw picture and life rule disagree: %+v", p)
	}
	if c := in.Classes[9]; c.Projectile != 1007 || c.ShootDelay != 3 || c.TileSize != 2 || c.CenterX != 8 ||
		c.CenterY != 10 || c.SelectionWidth != 20 || c.SelectionHeight != 24 {
		t.Fatalf("class release geometry lost: %+v", c)
	}
	art, err := LoadProjectiles(src)
	if err != nil || !art.HasPicture(1007) || art.Sheet(1007) != nil {
		t.Fatalf("artless arbitrary registry row: %v, %v", art, err)
	}
	delete(src, ProjectileRegistry)
	delete(src, UnitRegistry)
	if empty, err := loadFlightInputs(src, data.FirstSpellPresentation(), firstGameFiles.flight); err != nil || empty == nil ||
		empty.Pictures[10].LifeDivisor != 200 || empty.Pictures[10].Present || len(empty.Classes) != 0 {
		t.Fatalf("absent raw registries lost logical flight rules: %v, %v", empty, err)
	}
}

// A hero's flight class is the class the World stores for him, whatever his
// slots and his source's arm say; another Human uses his Humans-row type and a
// Unit his own class.
func TestFlightClassReadsTheStoredClass(t *testing.T) {
	e := sim.Entity{Class: 14, TypeID: 33, Humanoid: true, ActorLoad: sim.ActorLoad{Source: sim.SourceActor{Class: 2, Fighter: false}}}
	if got := flightClassRule(e); got != 14 {
		t.Fatalf("hero class %d, want the stored 14", got)
	}
	e.Class = 1
	if got := flightClassRule(e); got != 1 {
		t.Fatalf("hero class %d, want the stored 1 over a mage source arm", got)
	}
	e.TypeID, e.Class = 19, 4
	if got := flightClassRule(e); got != 19 {
		t.Fatalf("nonhero Human class %d, want authored 19", got)
	}
	e.TypeID, e.Class, e.ActorLoad.Source.Class = 33, 9, 1
	if got := flightClassRule(e); got != 9 {
		t.Fatalf("Unit class %d, want explicit 9", got)
	}
	e.ActorLoad.Source.Class, e.TypeID, e.Class, e.ClientClassFixed = 2, 19, 33, true
	if got := flightClassRule(e); got != 33 {
		t.Fatalf("fixed class %d, want 33", got)
	}
}

func TestFlightInputsBindWithoutPresentationBundles(t *testing.T) {
	w, err := sim.NewWorld(7, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	mw := &mapWorld{world: w}
	mw.bindFlightInputs()
	if !w.HasFlightInputs() {
		t.Fatal("first-game driver without art kept presentation-owned release")
	}
}

func TestFlightInputsBindRawMapBeforeFirstTick(t *testing.T) {
	in, err := loadFlightInputs(flightCatalogSource{}, data.FirstSpellPresentation(), firstGameFiles.flight)
	if err != nil {
		t.Fatal(err)
	}
	m := worldFixtureMap()
	mw, err := openMapWorld(m, &mapload.Table{FlightInputs: in}, nil, worldFixtureViewer(t, m))
	if err != nil {
		t.Fatal(err)
	}
	if mw.world.Tick() != 0 || !mw.world.HasFlightInputs() {
		t.Fatal("raw map constructor deferred its installed inputs until a window tick")
	}
}

func TestFlightHighBandHireCurrentSAVKeepsClassPolicy(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	f.Bodies = data.NewBodyList(data.BodyUnarmed, data.BodyArcher)
	rows := append(dbCollection(nil), f.Table.Humans.(dbCollection)...)
	params := humansParams(50, 0, 5)
	params[16], params[17] = 33, 1
	rows = append(rows, dbEntry{name: "NPC03_1", params: params})
	f.Table.Humans = rows
	party := MissionParty(nil, f.Bodies, f.Table)
	hire := party[0]
	hire.ID, hire.Name, hire.Class, hire.MercenaryType, hire.DefinitionRow = "high-hire", "NPC03_1", 33, 3, uint16(len(rows)-1)
	hire.StartingHero, hire.PlayerCharacter, hire.Temporary = false, false, true
	party = append(party, hire)
	if err := f.App("high-band hire").OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	id := f.live.mission.state.Start.IDs[len(party)-1]
	before, ok := f.live.world.Entity(id)
	if !ok || before.TypeID != 33 || before.Class != 33 || !before.ClientClassFixed {
		t.Fatalf("constructor lost high-band hire class: %+v", before)
	}
	raw, _, _ := saveCurrentEffect(t, f)
	cold := openCurrentEffectSave(t, f, raw)
	after, ok := cold.live.world.Entity(id)
	if !ok {
		t.Fatal("cold LOAD lost high-band hire")
	}
	t.Logf("high-band source binding before=%+v after=%+v; sourceClass=%d/%d; NativeClass=%+v/%+v", before.SourceBinding, after.SourceBinding, before.SourceNow().Class, after.SourceNow().Class, before.NativeClass, after.NativeClass)
	if before.SourceBinding.Class != after.SourceBinding.Class || before.SourceBinding.DisplayBacking != after.SourceBinding.DisplayBacking {
		t.Fatal("cold LOAD changed generated hire marker provenance")
	}
	if !after.ClientClassFixed {
		t.Fatal("cold LOAD lost fixed hired class policy")
	}
	for _, candidate := range []sim.Entity{before, after} {
		if got := flightClassRule(candidate); got != 33 {
			t.Fatalf("high-band hired class %d, want fixed 33", got)
		}
	}
}

type flightEntryInstall struct {
	installMission
	world *sim.World
}

func (s flightEntryInstall) openMissionDriver(ms *Mission, units *terrain.UnitSet, v *ui.Viewer) *mapWorld {
	ms.World = s.world
	mapload.BindFlightInputs(ms.World, s.in.Table)
	return s.installMission.openMissionDriver(ms, units, v)
}

func TestFlightMissionEntryPreservesRawBurstPhasesWithoutArt(t *testing.T) {
	for _, art := range []*terrain.EffectSet{nil, {Sheets: map[int]*terrain.EffectSheet{13: {Phases: 1}}}} {
		e := newEntryFixture(t, &recordingAudio{})
		install := e.ports.install.(installMission)
		install.in.Table = &mapload.Table{FlightInputs: &sim.FlightInputs{Pictures: map[int32]sim.FlightPicture{13: {Present: true, Phases: 11}}}}
		install.in.Projectiles = art
		w := spellSoundFireBallWorld(t, 100).world
		e.ports.install = flightEntryInstall{installMission: install, world: w}
		if _, err := enterMission(e.request(), e.ports); err != nil {
			t.Fatal(err)
		}
		if !w.HasFlightInputs() || w.Tick() != 0 {
			t.Fatal("ordinary mission entry did not bind installed inputs before its first tick")
		}
		seen := false
		for tick := 0; tick < 64 && !seen; tick++ {
			var commands []sim.Command
			if tick == 0 {
				commands = []sim.Command{sim.CastAt(1, 2, sim.CellPoint{X: 10, Y: 8})}
			}
			sim.Step(w, commands)
			if drivers := w.SavedWorldEffectDrivers(); drivers != nil {
				for _, d := range drivers.Projectiles {
					for _, p := range w.SavedProjectiles().Items {
						if p.ID == d.ID && p.Picture == 13 {
							seen = true
							if d.Phases != 11 {
								t.Fatalf("ordinary setup replaced raw burst phases with art: %+v", d)
							}
						}
					}
				}
			}
		}
		if !seen {
			t.Fatal("ordinary Fire Ball never reached its burst phase witness")
		}
	}
}
