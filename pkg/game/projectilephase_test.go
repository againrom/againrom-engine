package game

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func phaseSoundWorld(t *testing.T) *mapWorld {
	t.Helper()
	caster := sim.Entity{ID: 1, X: 8, Y: 8, Class: 1, HP: 100, MaxHP: 100,
		Owner: 7, Mind: 40, Mana: 100, MaxMana: 100, KnownSpells: 1 << 21,
		AttackCharge: 2, AttackRelax: 60, ScanRange: 30}
	rule := sim.SpellRule{ID: 21, ManaCost: 3, School: 1, MaxRange: 8,
		Area: true, Distribution: 5, Radius: 1, AreaDuration: 1}
	mw := spellSoundWorld(t, caster, rule)
	mw.projectiles = aoSheets()
	return mw
}

func TestProjectilePhaseSoundFollowsTheFreshRecord(t *testing.T) {
	mw := phaseSoundWorld(t)
	silent := phaseSoundWorld(t)
	plays := recordSpellSounds(mw)
	mw.attackOrCast(1, 0, 21, 10, 8, true)
	silent.attackOrCast(1, 0, 21, 10, 8, true)
	born, sounded := map[uint16]bool{}, map[uint16]bool{}
	casts := 0
	for tick := 0; tick < 128; tick++ {
		before := len(*plays)
		var report sim.Report
		mw.tickStep(nil, false, func(r sim.Report) { report = r })
		silent.tickStep(nil, false)
		casts += len(report.Casts)
		if len(report.Casts) != 0 {
			mw.enqueue(1, 8, 8)
			silent.enqueue(1, 8, 8)
		}
		if mw.world.Hash() != silent.world.Hash() || !reflect.DeepEqual(mw.world.SavedProjectiles(), silent.world.SavedProjectiles()) {
			t.Fatalf("tick %d: audio observation changed canonical state", tick)
		}
		for _, p := range mw.world.SavedProjectiles().Items {
			if p.Picture == 51 {
				born[p.ID] = true
			}
		}
		for _, play := range (*plays)[before:] {
			if play.slot != 551 {
				continue
			}
			found := false
			var cellRecords []sim.SavedProjectile
			for _, p := range mw.world.SavedProjectiles().Items {
				if p.Picture == 51 && image.Pt(int(p.X/256), int(p.Y/256)) == play.cell {
					cellRecords = append(cellRecords, p)
					if p.ActionPhase != 8 || p.Phase != 8 || sounded[p.ID] {
						continue
					}
					sounded[p.ID] = true
					found = true
					break
				}
			}
			if !found || play.owner != 7 {
				t.Fatalf("sound %+v at tick %d has no distinct live phase-8 record: %+v", play, tick, cellRecords)
			}
		}
	}
	if casts != 1 || len(born) == 0 || len(sounded) != len(born) {
		t.Fatalf("%d ordinary casts built records %v and sounded records %v; live %+v", casts, born, sounded, mw.world.SavedProjectiles())
	}
	t.Logf("one ordinary cast: %d records each sounded once at actionphase/frame 8; silent control kept equal records and hash", len(born))
}

func phaseSoundFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := restoredFacingFront(t)
	f.Projectiles = aoSheets()
	return f
}

func phaseSoundSource(t *testing.T, f *FrontEnd) []byte {
	t.Helper()
	source, err := sav.Open(restoredFacingSource(t, f))
	if err != nil {
		t.Fatal(err)
	}
	if err := source.SetProjectiles(sav.ProjectileStore{}); err != nil {
		t.Fatal(err)
	}
	return source.Marshal()
}

func phaseSoundSave(t *testing.T, f *FrontEnd) []byte {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil {
		t.Fatal("ordinary SAVE", err)
	}
	raw, err := os.ReadFile(filepath.Join(store.Dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestProjectilePhaseSoundResumesFromOrdinarySAV(t *testing.T) {
	for _, phase := range []int32{7, 8, 9} {
		t.Run(fmt.Sprint(phase), func(t *testing.T) {
			f := phaseSoundFront(t)
			app, _ := openOriginalSAVApp(t, f, phaseSoundSource(t, f), "phase-source.sav")
			t.Cleanup(app.StopAudio)
			id, ok := f.live.world.ReleaseAreaBurst(sim.AreaBurst{CellX: 8, CellY: 7, Picture: 51, Segments: 16, Phases: 16})
			if !ok {
				t.Fatal("record producer refused the cell")
			}
			for range phase {
				f.live.tickStep(nil, false)
			}
			p, ok := savedProjectileByID(f.live.world, id)
			if !ok || p.ActionPhase != phase {
				t.Fatalf("ordinary producer/driver left %+v at phase %d", p, phase)
			}
			raw := phaseSoundSave(t, f)
			cold := phaseSoundFront(t)
			loaded, _ := openOriginalSAVApp(t, cold, raw, "phase-current.sav")
			t.Cleanup(loaded.StopAudio)
			cold.live.semanticSound = nil
			plays := recordSpellSounds(cold.live)
			if !reflect.DeepEqual(cold.live.world.SavedProjectiles(), f.live.world.SavedProjectiles()) {
				t.Fatal("ordinary SAV changed current record leaves")
			}
			for range 20 {
				cold.live.tickStep(nil, false)
			}
			want := 0
			if phase < 8 {
				want = 1
			}
			if len(*plays) != want {
				t.Fatalf("ordinary SAV LOAD at actionphase %d produced %+v; want %d cue(s)", phase, *plays, want)
			}
			if want == 1 && (*plays)[0] != (spellSoundPlay{slot: 551, cell: image.Pt(8, 7)}) {
				t.Fatalf("restored cue %+v", (*plays)[0])
			}
		})
	}
}

func TestProjectilePhaseSoundDropsTheRetiredIdentity(t *testing.T) {
	mw := phaseSoundWorld(t)
	plays := recordSpellSounds(mw)
	cell := image.Pt(8, 7)
	mw.observeAreaPaints([]sim.AreaPaint{{Spell: 21, Owner: 7, Cells: []sim.CellPoint{{X: 8, Y: 7}}}})
	for range 6 {
		mw.tickStep(nil, false)
	}
	store := mw.world.SavedProjectiles()
	if len(store.Items) != 1 || store.Items[0].ActionPhase != 6 {
		t.Fatalf("retirement control lacks its original record: %+v", store)
	}
	old := store.Items[0].ID
	store.Items[0].ActionSegments = 0
	mw.world.SetSavedProjectiles(store)
	mw.tickStep(nil, false)
	id, ok := mw.world.ReleaseAreaBurst(sim.AreaBurst{CellX: 8, CellY: 7, Picture: 51, Segments: 16, Phases: 16, Owner: 7})
	if !ok || id == old {
		t.Fatal("replacement lacks a distinct record identity")
	}
	for phase := int32(1); phase <= 17; phase++ {
		mw.tickStep(nil, false)
		want := 0
		if phase >= 8 {
			want = 1
		}
		if len(*plays) != want {
			t.Fatalf("replacement actionphase %d: %+v, want %d cue(s)", phase, *plays, want)
		}
	}
	if (*plays)[0] != (spellSoundPlay{slot: 551, owner: 7, cell: cell}) {
		t.Fatalf("replacement cue %+v", (*plays)[0])
	}
}

func TestProjectilePhaseSoundDoesNotAdvanceOnPausedFramesOrRedraws(t *testing.T) {
	mw := phaseSoundWorld(t)
	plays := recordSpellSounds(mw)
	mw.world.ReleaseAreaBurst(sim.AreaBurst{CellX: 8, CellY: 7, Picture: 51, Segments: 16, Phases: 16})
	for range 7 {
		mw.tickStep(nil, false)
	}
	mw.stopped = true
	before, hash := mw.world.SavedProjectiles(), mw.world.Hash()
	for range 20 {
		mw.deterministicFrame()
		_ = mw.savedProjectileScene(false)
	}
	if len(*plays) != 0 || !reflect.DeepEqual(before, mw.world.SavedProjectiles()) || hash != mw.world.Hash() {
		t.Fatal("paused frames or redraws changed phase-7 state or emitted a cue")
	}
	mw.stopped = false
	mw.deterministicFrame()
	if len(*plays) != 1 {
		t.Fatalf("resumed phase-8 frame produced %+v", *plays)
	}
	mw.stopped = true
	before, hash = mw.world.SavedProjectiles(), mw.world.Hash()
	for range 20 {
		mw.deterministicFrame()
		_ = mw.savedProjectileScene(false)
	}
	if len(*plays) != 1 || !reflect.DeepEqual(before, mw.world.SavedProjectiles()) || hash != mw.world.Hash() {
		t.Fatal("paused phase-8 presentation replayed or changed the record")
	}
}
