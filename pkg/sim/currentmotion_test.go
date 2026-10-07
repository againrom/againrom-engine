package sim

import (
	"bytes"
	"testing"
)

func TestIdleNativeMotionExportsCurrentEntryCell(t *testing.T) {
	actor := Entity{ID: 7, Owner: 4, X: 12, Y: 13, PostX: 4, PostY: 5, HP: 100, MaxHP: 100, RotationSpeed: 19}
	motion, err := ProjectActorMotion(actor, SavedActorMotion{Entity: actor.ID}, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := motion.Mover[0x82:0x86], []byte{12, 13, 128, 128}; !bytes.Equal(got, want) {
		t.Fatalf("idle entry cache %x, want current occupied cell %x; guard post %d,%d", got, want, actor.PostX, actor.PostY)
	}
	if motion.Position.Cell != 0x0d0c || motion.Position.FineX != 128 || motion.Position.FineY != 128 {
		t.Fatal("entry cache replaced current Position", motion.Position)
	}
	if actor.PostX != 4 || actor.PostY != 5 {
		t.Fatal("entry cache changed guard post", actor)
	}
}

func TestNativeMotionPreservesCacheOutsideIdleConstruction(t *testing.T) {
	base := Entity{ID: 7, Owner: 4, X: 12, Y: 13, PostX: 4, PostY: 5, HP: 100, MaxHP: 100, RotationSpeed: 19}
	for _, tc := range []struct {
		name  string
		actor Entity
		fresh bool
		cache [4]byte
		want  [4]byte
	}{
		{name: "imported", actor: base, cache: [4]byte{17, 18, 37, 39}, want: [4]byte{17, 18, 37, 39}},
		{name: "existing fresh", actor: base, fresh: true, cache: [4]byte{17, 18, 37, 39}, want: [4]byte{17, 18, 37, 39}},
		{name: "player", actor: func() Entity { e := base; e.Owner = SelfSlot; return e }(), fresh: true},
		{name: "transit", actor: func() Entity {
			e := base
			e.X = 13
			e.Transit = 1
			e.TransitTotal = 2
			e.Stride = NativeStride{Present: true, FromX: 12, FromY: 13, ToX: 13, ToY: 13, Rate: 64, StepX: 64, Direction: 2}
			return e
		}(), fresh: true, want: [4]byte{4, 5, 128, 128}},
		{name: "pending move", actor: func() Entity { e := base; e.HasTarget = true; e.TargetX = 13; e.TargetY = 13; return e }(), fresh: true, want: [4]byte{4, 5, 128, 128}},
		{name: "turning", actor: func() Entity { e := base; e.TurnRemaining = 1; e.TurnTotal = 1; return e }(), fresh: true, want: [4]byte{4, 5, 128, 128}},
		{name: "attacking", actor: func() Entity { e := base; e.HasAttackTarget = true; return e }(), fresh: true, want: [4]byte{4, 5, 128, 128}},
		{name: "off map", actor: func() Entity { e := base; e.OffMap = true; return e }(), fresh: true, want: [4]byte{4, 5, 128, 128}},
		{name: "dead", actor: func() Entity { e := base; e.HP = 0; return e }(), fresh: true, want: [4]byte{4, 5, 128, 128}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			old := SavedActorMotion{Entity: tc.actor.ID}
			copy(old.Mover[0x82:0x86], tc.cache[:])
			got, err := ProjectActorMotion(tc.actor, old, nil, tc.fresh)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got.Mover[0x82:0x86], tc.want[:]) {
				t.Fatalf("cache %x, want preserved spelling %x", got.Mover[0x82:0x86], tc.want)
			}
		})
	}
}
