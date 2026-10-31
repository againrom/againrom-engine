package game

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// secondHeldMarkRule is a second-game row of the six held-mark spells
// (R2-ASSET-177), shaped as this package's casts need it.
func secondHeldMarkRule(row uint16, kind sim.EffectKind) sim.SpellRule {
	rules := []sim.SpellRule{{ID: row, ManaCost: 10, School: 1, MaxRange: 8, TargetsUnit: true,
		Defensive: true, EffectKind: kind, EffectMode: sim.EffectDuration, EffectMagnitude: 20,
		SpellDuration: 60}}
	sim.AssignSecondGameArms(rules)
	return rules[0]
}

// secondHeldMarkWorld casts row on the victim of a second-game mission and
// opens its mark. Every picture either game would select has a sheet, so a
// wrong selection draws the wrong sheet instead of nothing.
func secondHeldMarkWorld(t *testing.T, row uint16, kind sim.EffectKind) *mapWorld {
	t.Helper()
	w := emWorld(t, row, secondHeldMarkRule(row, kind))
	emCast(t, w, row)
	second := base.GameROM2.Edition()
	return &mapWorld{world: w, mission: &missionNotices{table: &mapload.Table{Edition: &second}},
		projectiles: emSheets(16, 24, 34, 46, 48, 62, 18, 28, 40, 52, 54, 44)}
}

// secondBlessClaim is R2-ENGINE-693's Bless entries at count c, scale s.
func secondBlessClaim(c uint16, s int) []terrain.EffectMark {
	var out []terrain.EffectMark
	for _, a := range []int{89, 71, 53, 35, 17} {
		rad := float64(a+18*int(c%5)) * math.Pi / 180
		u, v := int(20*math.Cos(rad)), int(20*math.Sin(rad))
		for _, p := range [4][2]int{{u, v}, {v, -u}, {-u, -v}, {-v, u}} {
			out = append(out, terrain.EffectMark{DX: p[0], DY: -32 * s, Depth: p[1], Record: 48, Phase: 4 - a/18})
		}
	}
	return out
}

// TestSecondGameHeldMarksDrawTheirOwnPictures: each of the six second-game
// held marks draws its own projectiles.reg picture (R2-ENGINE-445) with the
// records R2-ENGINE-693 states, at two update counts and two scales, and
// the mark ends with its effect.
func TestSecondGameHeldMarksDrawTheirOwnPictures(t *testing.T) {
	protections := []struct {
		row     uint16
		kind    sim.EffectKind
		picture int
		dx, dy  int
	}{
		{4, sim.EffectProtectionFire, 16, -6, 0},
		{8, sim.EffectProtectionWater, 24, 6, 0},
		{13, sim.EffectProtectionAir, 34, 0, -6},
		{19, sim.EffectProtectionEarth, 46, 0, 6},
	}
	for _, tc := range protections {
		mw := secondHeldMarkWorld(t, tc.row, tc.kind)
		for _, c := range []uint16{0xffff, 0xfffe} {
			mw.advanceEffectMarks()
			for _, s := range []int{1, 2} {
				got := mw.markDraws(2, s)
				want := terrain.EffectMark{DX: tc.dx, DY: tc.dy - 32*s, Record: tc.picture, Phase: int(c % 6)}
				if len(got) != 1 || got[0].Mark != want || got[0].Sheet != mw.projectiles.Sheet(tc.picture) {
					t.Errorf("row %d count %#x scale %d: %d marks %+v, want one %+v on its own sheet",
						tc.row, c, s, len(got), marksOf(got), want)
				}
			}
		}
		for i := 0; i < 20000 && len(mw.world.ActiveEffects()) > 0; i++ {
			sim.Step(mw.world, nil)
		}
		mw.advanceEffectMarks()
		if len(mw.world.ActiveEffects()) != 0 || len(mw.markDraws(2, 1)) != 0 {
			t.Errorf("row %d: the mark outlived its effect", tc.row)
		}
	}

	bless := secondHeldMarkWorld(t, 20, sim.EffectBless)
	for _, c := range []uint16{0xffff, 0xfffe} {
		bless.advanceEffectMarks()
		for _, s := range []int{1, 2} {
			got, want := marksOf(bless.markDraws(2, s)), secondBlessClaim(c, s)
			if len(got) != len(want) {
				t.Fatalf("Bless count %#x scale %d: %d marks, want %d", c, s, len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("Bless count %#x scale %d mark %d = %+v, want %+v", c, s, i, got[i], want[i])
				}
			}
		}
	}

	// Shield at count 0xffff (phase 15) and 0xfffe (phase 14), scale 1: the
	// meridian records at q = +-16 lie at y = -+28 on frame |4 - trunc(80/28)|
	// = 2, those at q = 0 on frame 4. The rings stand at y = -(11 + h) and
	// -(11 - h), h = trunc(28(p/45 - 1)), on frame 4 - |p - 45| div 9 = 1,
	// each pair's -(11 + h) first.
	shield := secondHeldMarkWorld(t, 27, sim.EffectAbsorption)
	for _, tc := range []struct {
		h, upper int
	}{{-18, 7}, {-19, 8}} {
		shield.advanceEffectMarks()
		draws := shield.markDraws(2, 1)
		if len(draws) == 0 {
			t.Fatalf("Shield h %d drew nothing", tc.h)
		}
		rings := 0
		for i, d := range draws {
			m := d.Mark
			if m.Record != 62 || d.Sheet != shield.projectiles.Sheet(62) {
				t.Fatalf("Shield mark %d = %+v, want picture 62's sheet", i, m)
			}
			switch {
			case m.DY == -(11+tc.h) && i+1 < len(draws) && draws[i+1].Mark.DY == -(11-tc.h):
				rings++
				if m.Phase != 1 || draws[i+1].Mark.Phase != 1 {
					t.Errorf("Shield h %d ring pair at %d on frames %d, %d, want 1", tc.h, i, m.Phase, draws[i+1].Mark.Phase)
				}
			case m.DY == 28 || m.DY == -28:
				if m.Phase != 2 {
					t.Errorf("Shield h %d meridian mark %d at y %d on frame %d, want 2", tc.h, i, m.DY, m.Phase)
				}
			case m.DY == 0:
				if m.Phase != 4 {
					t.Errorf("Shield h %d meridian mark %d at y 0 on frame %d, want 4", tc.h, i, m.Phase)
				}
			}
		}
		if rings == 0 {
			t.Errorf("Shield h %d drew no ring pair with y %d first", tc.h, tc.upper)
		}
	}
}

// TestFirstGameHeldMarksDrawAsOnTheBase: the first game selects every arm's
// own kind as program and picture, and markDraws gives every kind, the heal
// and drain-life bursts included, the sheet, records and order the base
// 46c4f6fe gave.
func TestFirstGameHeldMarksDrawAsOnTheBase(t *testing.T) {
	const want = "0f409384ce7b7787052f030c95dd2bfe5458b78547de4fbd5bf03b70643828a1"
	for arm := uint16(0); arm <= 64; arm++ {
		kind := terrain.MarkKind(int(arm))
		if got := firstGameFiles.heldMarkOf(arm); got != (heldMark{kind, kind}) {
			t.Errorf("first-game arm %d selects %+v, want kind %#x as both", arm, got, kind)
		}
	}
	w := emWorld(t, 5, protectionFireRule())
	pictures := make([]int, 0x51)
	for i := range pictures {
		pictures[i] = i
	}
	mw := &mapWorld{world: w, projectiles: emSheets(pictures...)}
	victim, _ := mw.entity(2)
	h := sha256.New()
	write := func() {
		for tile := 1; tile <= 3; tile++ {
			for _, d := range mw.markDraws(2, tile) {
				picture := -1
				for p, sh := range mw.projectiles.Sheets {
					if sh == d.Sheet {
						picture = p
					}
				}
				m := d.Mark
				fmt.Fprintf(h, "%d:%d,%d,%d,%d,%d;", picture, m.DX, m.DY, m.Depth, m.Record, m.Phase)
			}
		}
	}
	for kind := 0; kind <= 0x50; kind++ {
		for _, c := range []uint16{0, 1, 2, 3, 4, 5, 44, 45, 89, 90, 0xfffa, 0xfffb, 0xfffc, 0xfffd, 0xfffe, 0xffff} {
			mw.markElements = []markElement{{Entity: 2, Kind: kind, Picture: kind, Count: c}}
			write()
		}
	}
	for _, kind := range []int{terrain.MarkKind(6), terrain.MarkKind(11)} {
		for c := 1; c <= healBurstLife; c++ {
			for _, seed := range []uint32{0, 0x9e3779b9} {
				mw.markElements = []markElement{{Entity: 2, Kind: kind, Picture: kind, Count: uint16(c),
					Transient: true, Seed: seed, Identity: targetMarkIdentity(victim)}}
				write()
			}
		}
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		t.Fatalf("first-game draws digest %s, want the base's %s", got, want)
	}
}

func marksOf(draws []ui.UnitMark) []terrain.EffectMark {
	out := make([]terrain.EffectMark, len(draws))
	for i, d := range draws {
		out[i] = d.Mark
	}
	return out
}
