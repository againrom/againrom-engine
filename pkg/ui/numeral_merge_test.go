package ui

import (
	"againrom/pkg/render/terrain"
	"bytes"
	"fmt"
	"image"
	"testing"
	"time"
)

func TestDamageNumeralMergeKeyAndLossControls(t *testing.T) {
	for _, tc := range []struct {
		name    string
		ticks   [2]uint64
		victims [2]uint32
		want    []int
	}{
		{"same victim and tick", [2]uint64{17, 17}, [2]uint32{1, 1}, []int{20}},
		{"next raw tick", [2]uint64{17, 18}, [2]uint32{1, 1}, []int{7, 13}},
		{"another victim", [2]uint64{17, 17}, [2]uint32{1, 2}, []int{7, 13}},
		{"same full tick", [2]uint64{16, 17}, [2]uint32{1, 1}, []int{7, 13}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := litViewer(t)
			u := numeralUnit(1, 1, 100)
			other := numeralUnit(2, 1, 100)
			push(v, at0, u, other)
			for i, amount := range []int{7, 13} {
				e := u
				if tc.victims[i] == 2 {
					e = other
				}
				v.AppendDamageMessages([]DamageMessage{{Tick: tc.ticks[i], Entity: e, AfterHP: e.HP - amount}})
				if e.ID == 1 {
					u.HP -= amount
				} else {
					other.HP -= amount
				}
			}
			push(v, at0, u, other)
			if len(v.numerals) != len(tc.want) {
				t.Fatalf("figures=%d want%d: %+v", len(v.numerals), len(tc.want), v.DamageNumeralStates())
			}
			for i, want := range tc.want {
				if v.numerals[i].damage != want {
					t.Fatalf("figure%d=%d want%d", i, v.numerals[i].damage, want)
				}
			}
			// A snapshot decrease has no causal message and must not add a figure.
			u.HP -= 5
			push(v, at0, u, other)
			for i, want := range tc.want {
				if v.numerals[i].damage != want {
					t.Fatal("snapshot fabricated damage")
				}
			}
		})
	}
}

func TestDamageNumeralMergeKeepsBirthFlightAndColour(t *testing.T) {
	v := litViewer(t)
	v.SetPeriod(100000)
	v.SetLocalOwner(1)
	u := numeralUnit(1, 1, 100)
	push(v, at0, u)
	v.AppendDamageMessages([]DamageMessage{{Tick: 4, Entity: u, AfterHP: 93}})
	u.HP = 93
	push(v, at0, u)
	first := v.numerals[0]
	glyph := bytes.Clone(first.pic.Pix)
	u.Owner = 2
	v.SetLocalOwner(2)
	v.AppendDamageMessages([]DamageMessage{{Tick: 4, Entity: u, AfterHP: 80}})
	u.HP = 80
	push(v, at0.Add(300*time.Millisecond), u)
	n := v.numerals[0]
	if len(v.numerals) != 1 || n.damage != 20 || n.born != first.born || n.colour != first.colour || n.drift != first.drift || n.off != first.off.Add(image.Pt(3, -6)) {
		t.Fatalf("merge refreshed record: %+v", v.DamageNumeralStates())
	}
	if bytes.Equal(n.pic.Pix, glyph) {
		t.Fatal("merge did not reformat installed amount")
	}
	push(v, at0.Add(time.Second+time.Nanosecond), u)
	if len(v.numerals) != 0 {
		t.Fatal("merge refreshed expiry")
	}
}

func TestDamageNumeralLifeAtEveryShippedSpeed(t *testing.T) {
	for i, ms := range []int{125, 100, 83, 71, 62, 50, 41, 35, 31} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			v := numeralViewer(t)
			if got := terrain.SpeedIndexPeriod(i); got != ms*1000 {
				t.Fatalf("speed%d period%d want%d", i, got, ms*1000)
			}
			v.SetPeriod(terrain.SpeedIndexPeriod(i))
			v.SetLocalOwner(1)
			u := numeralUnit(1, 1, 100)
			push(v, at0, u)
			v.AppendDamageMessages([]DamageMessage{{Tick: 1, Entity: u, AfterHP: 93}})
			u.HP = 93
			push(v, at0, u)
			for tick := 1; tick <= 1000/ms; tick++ {
				push(v, at0.Add(time.Duration(tick*ms)*time.Millisecond), u)
			}
			n := v.numerals[0]
			want := image.Pt(16+1000/ms, -48-2*(1000/ms))
			if n.off != want {
				t.Fatalf("speed%d %dms drift=%v want%v", i, ms, n.off, want)
			}
			push(v, at0.Add(time.Second), u)
			if len(v.numerals) != 1 {
				t.Fatal("inclusive 1000ms boundary lost")
			}
			push(v, at0.Add(time.Second+time.Nanosecond), u)
			if len(v.numerals) != 0 {
				t.Fatal("life became a speed-dependent tick budget")
			}
			t.Logf("speed%d dt%dms nominal ticks%d..%d offset%v", i, ms, 1000/ms, 1000/ms+1, want)
		})
	}
}

func TestDamageNumeralCatchUpOffsetsFollowRawTicks(t *testing.T) {
	v := litViewer(t)
	u := numeralUnit(1, 1, 100)
	v.SetLocalOwner(1)
	push(v, at0, u)
	v.AppendDamageMessages([]DamageMessage{{Tick: 17, Entity: u, AfterHP: 93}, {Tick: 18, Entity: numeralUnit(1, 1, 93), AfterHP: 80}})
	v.SetLightClock(19)
	u.HP = 80
	push(v, at0, u)
	if len(v.numerals) != 2 || v.numerals[0].off != image.Pt(18, -52) || v.numerals[1].off != image.Pt(17, -50) {
		t.Fatal("catch-up figures lost their tick offsets", v.DamageNumeralStates())
	}
}

func TestDamageNumeralKeepsWholeApplicationAmount(t *testing.T) {
	v := numeralViewer(t)
	u := numeralUnit(1, 1, 100000)
	push(v, at0, u)
	v.AppendDamageMessages([]DamageMessage{{Tick: 1, Entity: u, AfterHP: 30000}})
	u.HP = 30000
	push(v, at0, u)
	if len(v.numerals) != 1 || v.numerals[0].damage != 70000 {
		t.Fatal("damage amount narrowed", v.DamageNumeralStates())
	}
}
