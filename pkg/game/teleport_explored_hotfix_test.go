package game

import (
	"testing"

	"againrom/pkg/sim"
)

const (
	teleportFogWidth  = 48
	teleportFogHeight = 8
)

func teleportFogWorld(t *testing.T, blockedX int) (*mapWorld, *sim.World) {
	t.Helper()
	caster := sim.Entity{ID: 1, X: 2, Y: 3, HP: 100, MaxHP: 100, Owner: sim.SelfSlot, TokenSize: 1,
		Mind: 100, Mana: 200, MaxMana: 200, KnownSpells: 1 << 26, ScanRange: 4,
		AttackCharge: 1, AttackRelax: 1}
	caster.Skill[5] = 30
	block := make([]byte, teleportFogWidth*teleportFogHeight)
	if blockedX >= 0 {
		block[3*teleportFogWidth+blockedX] = 1
	}
	w, err := sim.NewStockedSpelledWorld(0x7e20,
		sim.Bounds{Width: teleportFogWidth, Height: teleportFogHeight}, sim.ModeCanonical,
		sim.Terrain{Block: block}, []sim.Entity{caster}, nil, sim.Relations{}, nil, nil,
		[]sim.SpellRule{{ID: 26, ManaCost: 60, School: 5, MaxRange: 8, Defensive: true}})
	if err != nil {
		t.Fatalf("NewStockedSpelledWorld: %v", err)
	}
	mw := &mapWorld{world: w, fog: newFogPlane(teleportFogWidth, teleportFogHeight),
		commanded: map[sim.EntityID]bool{}}
	return mw, w
}

func exploredIndex(x, y int) int { return y*teleportFogWidth + x }

func applyQueuedCast(t *testing.T, mw *mapWorld) {
	t.Helper()
	before := teleportCaster(t, mw.world)
	cmds := mw.commands()
	mw.pending = mw.pending[:0]
	sim.Step(mw.world, cmds)
	for i := 0; i < 32; i++ {
		after := teleportCaster(t, mw.world)
		if after.Mana != before.Mana || after.X != before.X || after.Y != before.Y {
			return
		}
		sim.Step(mw.world, nil)
	}
}

func teleportCaster(t *testing.T, w *sim.World) sim.Entity {
	t.Helper()
	for _, e := range w.Entities() {
		if e.ID == 1 {
			return e
		}
	}
	t.Fatal("world holds no Teleport caster")
	return sim.Entity{}
}

func TestExploredHiddenCellQueuesAndCompletesHighRangeTeleport(t *testing.T) {
	mw, w := teleportFogWorld(t, -1)
	const farX, farY = 35, 3 // distance 33 exceeds sight 4 and fits Teleport range 41.
	from := &mapWorld{fog: newFogPlane(teleportFogWidth, teleportFogHeight)}
	from.fog.explored[exploredIndex(farX, farY)] = 1
	payload, err := EncodeSave(Snapshot{Mission: 1, World: []byte{0}, Residue: from.residue()},
		"explored Teleport")
	if err != nil {
		t.Fatalf("EncodeSave: %v", err)
	}
	back, _, err := DecodeSave(payload)
	if err != nil {
		t.Fatalf("DecodeSave: %v", err)
	}
	mw.applyResidue(back.Residue)
	if !mw.fog.exploredAt(farX, farY) {
		t.Fatal("explored destination did not survive the save envelope")
	}
	if mw.fog.visible[exploredIndex(farX, farY)] != 0 {
		t.Fatal("far-cell fixture is currently visible; it must exercise explored-only admission")
	}

	mw.attackOrCast(1, 0, 26, farX, farY, true)
	if len(mw.pending) != 1 || mw.pending[0].Kind != sim.KindCastAt {
		t.Fatalf("explored hidden Teleport queued %+v, want one KindCastAt", mw.pending)
	}
	applyQueuedCast(t, mw)
	got := teleportCaster(t, w)
	if got.X != farX || got.Y != farY || got.Mana != 140 {
		t.Fatalf("completed Teleport caster = cell (%d,%d), mana %d; want (%d,%d), 140",
			got.X, got.Y, got.Mana, farX, farY)
	}
}

func TestUnexploredTeleportQueuesAndCompletes(t *testing.T) {
	mw, w := teleportFogWorld(t, -1)
	mw.attackOrCast(1, 0, 26, 35, 3, true)
	if len(mw.pending) != 1 || mw.pending[0].Kind != sim.KindCastAt {
		t.Fatalf("unexplored Teleport queued %+v, want one KindCastAt", mw.pending)
	}
	if !mw.commanded[1] {
		t.Fatal("unexplored Teleport did not mark the caster commanded")
	}
	applyQueuedCast(t, mw)
	got := teleportCaster(t, w)
	if got.X != 35 || got.Y != 3 || got.Mana != 140 {
		t.Fatalf("unexplored Teleport completed at cell (%d,%d), mana %d; want (35,3), 140",
			got.X, got.Y, got.Mana)
	}
}

func TestExploredTeleportKeepsRangeRefusalAndPaysBlockedPlacement(t *testing.T) {
	for _, tc := range []struct {
		name     string
		x        int
		blockedX int
	}{
		{name: "out of range", x: 44, blockedX: -1}, // distance 42, range 41: walks toward the cell.
		{name: "blocked destination", x: 35, blockedX: 35},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mw, w := teleportFogWorld(t, tc.blockedX)
			mw.fog.explored[exploredIndex(tc.x, 3)] = 1
			mw.attackOrCast(1, 0, 26, tc.x, 3, true)
			if len(mw.pending) != 1 {
				t.Fatalf("explored target queued %d commands, want one for simulation admission", len(mw.pending))
			}
			applyQueuedCast(t, mw)
			got := teleportCaster(t, w)
			wantMana := int32(200)
			if tc.blockedX >= 0 {
				wantMana = 140
			}
			// An out of range order walks the caster toward the cell and pays
			// nothing until it casts in range; a blocked placement in range is
			// paid and leaves the caster where he stood.
			moved := got.X > 2
			if got.Y != 3 || got.Mana != wantMana || (tc.blockedX < 0 && (!moved || got.CastWait != 0)) ||
				(tc.blockedX >= 0 && got.X != 2) {
				t.Fatalf("Teleport left caster at (%d,%d), mana %d, recovery %d",
					got.X, got.Y, got.Mana, got.CastWait)
			}
		})
	}
}

func TestCurrentlyVisibleTeleportStillWorks(t *testing.T) {
	mw, w := teleportFogWorld(t, -1)
	const nearX, nearY = 5, 3
	i := exploredIndex(nearX, nearY)
	mw.fog.visible[i], mw.fog.explored[i] = 1, 1
	mw.attackOrCast(1, 0, 26, nearX, nearY, true)
	applyQueuedCast(t, mw)
	got := teleportCaster(t, w)
	if got.X != nearX || got.Y != nearY || got.Mana != 140 {
		t.Fatalf("visible Teleport caster = cell (%d,%d), mana %d; want (%d,%d), 140",
			got.X, got.Y, got.Mana, nearX, nearY)
	}
}

func TestTeleportAdmissionKeepsNoFogSemantics(t *testing.T) {
	var absent *fogPlane
	if !absent.exploredAt(35, 3) || !(&fogPlane{}).exploredAt(35, 3) ||
		!absent.contains(35, 3) || !(&fogPlane{}).contains(35, 3) {
		t.Fatal("nil or empty fog plane stopped behaving as no fog")
	}
	p := newFogPlane(4, 2)
	if p.exploredAt(4, 1) || p.exploredAt(-1, 0) || p.contains(4, 1) || p.contains(-1, 0) {
		t.Fatal("populated fog plane admitted a cell outside its measured extent")
	}

	mw, _ := teleportFogWorld(t, -1)
	mw.attackOrCast(1, 0, 26, teleportFogWidth, 3, true)
	if len(mw.pending) != 0 || mw.commanded[1] {
		t.Fatalf("out-of-plane Teleport reached the queue: pending=%+v commanded=%v",
			mw.pending, mw.commanded[1])
	}
}
