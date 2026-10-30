package game

import (
	"fmt"
	"hash/fnv"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// flightWitnessDigests: per game and launch family, the digest of every
// tick's records, hash, drivers, SAV rows, trails and looks, as the window
// producers built them.
var flightWitnessDigests = map[string]string{
	"first/shot":              "b79c35c40ed6dae5",
	"first/book":              "8a52fef3def509b6",
	"first/staff":             "6ba99a3286877116",
	"first/staged":            "a7954bd78ee980ff",
	"first/fire-ball":         "5cc85563cf229b15",
	"first/client-lightning":  "d5fc23b3b3c0a1cf",
	"first/client-chain":      "0c9de92bd9ba1ec4",
	"first/teleport-moved":    "2bb7fd4ab8154a58",
	"second/shot":             "3de55e43db311502",
	"second/book":             "6480ba6c5787d85a",
	"second/staff":            "8a2811e78ab8888b",
	"second/staged":           "241d6b45ff2f41be",
	"second/fire-ball":        "27e9aa52c74b4dec",
	"second/client-lightning": "5cffbd541aa96389",
	"second/client-chain":     "0a1f49507aa68f3b",
	"second/teleport-moved":   "70dc54d8662df39a",
}

// The second game's release clock is profile data: the client's swing clock
// releases its shot, and not the first game's.
func TestFlightProfilesAreGameData(t *testing.T) {
	if got, want := secondGameFiles.flight, (flightProfile{FlightProfile: sim.FlightProfile{Release: sim.ReleaseAtClient,
		StartFromMovedCaster: true, ShotsNeedArt: true}, drawnClass: true}); got != want {
		t.Fatalf("second game's flight profile %+v, want %+v", got, want)
	}
	if got, want := firstGameFiles.flight, (flightProfile{registryRows: true}); got != want {
		t.Fatalf("first game's flight profile %+v, want %+v", got, want)
	}
	for _, second := range []bool{false, true} {
		units, pictures := shotArchive(t)
		var script *sim.Script
		if second {
			var err error
			if script, err = sim.NewROM2Script(missionChecks(), nil, nil); err != nil {
				t.Fatal(err)
			}
		}
		archer := shotShooter(1, shotClassArcher, 1, 4, 1000)
		archer.HasAttackTarget, archer.AttackTarget, archer.AttackTargetKind = true, 2, sim.AttackTargetUnit
		w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: swingW, Height: swingH}, sim.ModeCanonical, nil,
			[]sim.Entity{archer, shotVictim(2, 5, 4)}, script, nil)
		if err != nil {
			t.Fatal(err)
		}
		mw := shotMapWorld(t, w, units, pictures)
		if second {
			edition := base.GameROM2.Edition()
			mw.mission = &missionNotices{table: &mapload.Table{Edition: &edition, SpellArms: mapload.SecondGameSpellArms}}
		}
		mw.bindFlightInputs()
		mw.shots.begin(1, sim.AttackCharging)
		mw.swing[1] = shotArcherDelay - 1
		mw.advanceUnitShots()
		if got, want := len(w.SavedProjectiles().Items), map[bool]int{false: 0, true: 1}[second]; got != want {
			t.Errorf("second game %v: the client's swing clock released %d shots, want %d", second, got, want)
		}
	}
}

var flightWitnessFamilies = []string{"shot", "book", "staff", "staged", "fire-ball", "client-lightning", "client-chain", "teleport-moved"}

func TestFlightRecordsMatchTheirPinnedRuns(t *testing.T) {
	for _, game := range []string{"first", "second"} {
		for _, family := range flightWitnessFamilies {
			key := game + "/" + family
			got, records := flightWitnessRun(t, game == "second", family)
			if records == 0 {
				t.Errorf("%s: the run built no record", key)
			}
			if want := flightWitnessDigests[key]; got != want {
				t.Errorf("%q: %q, // want %q", key, got, want)
			}
		}
	}
}

// flightWitnessRun answers a 40-tick run's digest and its record ids.
func flightWitnessRun(t *testing.T, second bool, family string) (string, int) {
	t.Helper()
	units, pictures := shotArchive(t)
	art := spWorld(t).projectiles
	for id, sheet := range art.Sheets {
		pictures.Sheets[id] = sheet
	}
	pictures.Sheets[17] = &terrain.EffectSheet{Phases: 8, RotationPhases: 1, Frames: spFrames(8, 8)}
	if pictures.Pictures == nil {
		pictures.Pictures = map[int]bool{}
	}
	pictures.Pictures[17] = true
	pictures.Sheets[teleportPicture] = &terrain.EffectSheet{Frames: spFrames(4, 8), Phases: 4, RotationPhases: 1, CenterX: 4, CenterY: 4}
	spell := uint16(1)
	rule := sim.SpellRule{ID: 1, School: 1, ManaCost: 3, MaxRange: 8, TargetsUnit: true, Damaging: true, DamageMin: 4, DamageMax: 4}
	switch family {
	case "staged":
		spell, rule.ID, rule.Area, rule.TargetsUnit, rule.Distribution, rule.Radius = 4, 4, true, false, 5, 2
		rule.SelfCast, rule.MaxRange, rule.AreaDuration = true, 0, 1
	case "fire-ball":
		spell, rule.ID, rule.Area, rule.TargetsUnit, rule.Distribution, rule.Radius = 2, 2, true, false, 3, 1
		rule.Delivery, rule.EffectSpeed = 2, 128
	case "client-lightning":
		spell, rule.ID = 13, 13
	case "client-chain":
		spell, rule.ID = 14, 14
	case "teleport-moved":
		spell, rule.ID = 26, 26
	}
	caster := shotShooter(1, shotClassArcher, 14, 14, 4)
	caster.AttackRelax, caster.ScanRange, caster.Mind = 60, 12, 30
	caster.Mana, caster.MaxMana, caster.KnownSpells = 100, 100, 1<<spell
	caster.AlwaysHits, caster.DamageBase = true, 5
	if family == "staff" {
		caster.WeaponSpell, caster.WeaponSpellLevel = spell, 30
	}
	var script *sim.Script
	if second {
		var err error
		if script, err = sim.NewROM2Script(missionChecks(), nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical, nil,
		[]sim.Entity{caster, shotVictim(2, 18, 14)}, script, []sim.SpellRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	viewer, err := ui.NewViewer("flight witness", terrain.Grid{Width: 32, Height: 32, Tiles: make([]uint16, 32*32)}, &terrain.Tileset{})
	if err != nil {
		t.Fatal(err)
	}
	mw := newMapWorld(w, nil, units, viewer)
	mw.projectiles = pictures
	w.SetBurstPhases(uint16(pictures.Sheets[fireBallBurstPicture].Phases))
	if second {
		edition := base.GameROM2.Edition()
		mw.mission = &missionNotices{table: &mapload.Table{Edition: &edition, SpellArms: mapload.SecondGameSpellArms}}
	}
	if family == "client-lightning" || family == "client-chain" {
		if err := w.DeclareCellTails([]sim.CellTail{{X: 15, Y: 14, Bytes: [6]byte{byte(spell), 30, 11, 14, 0, 0}}}); err != nil {
			t.Fatal(err)
		}
	}
	switch family {
	case "shot", "staff":
		mw.strike(1, 2)
	case "staged":
		mw.castAt(1, 1, uint32(spell))
	case "client-lightning", "client-chain":
		mw.enqueue(1, 15, 14)
	case "teleport-moved":
	default:
		mw.castAt(1, 2, uint32(spell))
	}
	h := fnv.New64a()
	most := 0
	for tick := range 40 {
		mw.tickStep(nil, true)
		if family == "teleport-moved" && tick == 0 {
			mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: spell, AtCell: true, Owner: 1,
				FromX: 11, FromY: 12, ToX: 20, ToY: 16, Facing: 64}})
		}
		store := w.SavedProjectiles()
		most = max(most, int(store.FreeIndex))
		fmt.Fprintf(h, "%d|%+v|%016x|", tick, store, w.Hash())
		if d := w.SavedWorldEffectDrivers(); d != nil {
			fmt.Fprintf(h, "%+v|", *d)
		}
		for _, p := range store.Items {
			fmt.Fprintf(h, "%+v|", savedProjectileToSav(p))
		}
		fmt.Fprintf(h, "%v|%v\n", mw.shots.trail, mw.flights)
	}
	return fmt.Sprintf("%016x", h.Sum64()), most
}
