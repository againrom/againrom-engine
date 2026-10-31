package rules

import "testing"

// previousDrive is the former per-game drive table, the oracle.
func previousDrive(game SpellGame, id int32) EffectDrive {
	rows := map[SpellGame][]struct {
		ids   []int32
		drive EffectDrive
	}{
		SpellGameFirst: {
			{[]int32{34, 36}, EffectDrive{EffectPath, FrameRamp}},
			{[]int32{18, 24, 28, 40, 44, 48, 52, 54, 56, 62, 64}, EffectDrive{EffectFollow, FrameHalf}},
			{[]int32{20, 30}, EffectDrive{EffectFollow, FrameResolvedOne}},
			{[]int32{60}, EffectDrive{EffectStill, FrameCounterLess1}},
			{[]int32{51}, EffectDrive{EffectTravel, FrameCounter}},
		},
		SpellGameSecond: {
			{[]int32{28, 30}, EffectDrive{EffectPath, FrameRamp}},
			{[]int32{16, 20, 24, 34, 44, 46, 48, 50, 62, 64, 66}, EffectDrive{EffectFollow, FrameCounter}},
			{[]int32{56, 60}, EffectDrive{EffectFollow, FrameHalf}},
			{[]int32{54}, EffectDrive{EffectStill, FrameCounterLess1}},
			{[]int32{23}, EffectDrive{EffectStill, FrameCounter}},
		},
	}
	for _, row := range rows[game] {
		for _, v := range row.ids {
			if v == id {
				return row.drive
			}
		}
	}
	return EffectDrive{}
}

// previousMark is the former sim and game mark test, the oracle.
func previousMark(game SpellGame, p int32) TargetMark {
	switch {
	case game != SpellGameFirst:
		return TargetMarkNone
	case p == 20:
		return TargetMarkHeal
	case p == 30:
		return TargetMarkDrainLife
	}
	return TargetMarkNone
}

// previousUnbound is the former SAV metadata picture list, the oracle.
func previousUnbound(p int32) bool {
	return p == 13 || p == 51 || p == 36 || p == 18 || p == 24 || p == 28 || p == 40 || p == 44 || p == 48 || p == 52 || p == 54 || p == 56 || p == 62 || p == 64 || p == 20 || p == 30
}

// previousSoundPhase is the former storm phase test, the oracle.
func previousSoundPhase(game SpellGame, p int32) int32 {
	if game == SpellGameFirst && p == 51 {
		return 8
	}
	return 0
}

func TestEachPictureGetsTodaysMarkDriveUnboundAndSoundPhase(t *testing.T) {
	for _, game := range []SpellGame{SpellGameFirst, SpellGameSecond, SpellGameSecond + 1} {
		for p := int32(-2); p <= 300; p++ {
			row := PictureCapabilityOf(game, p)
			if row.Picture != p {
				t.Fatalf("game %d picture %d: row names picture %d", game, p, row.Picture)
			}
			if got, want := row.Drive, previousDrive(game, p); got != want || EffectDriveOf(game, p) != want {
				t.Fatalf("game %d picture %d: drive %+v, want %+v", game, p, got, want)
			}
			if got, want := row.Mark, previousMark(game, p); got != want {
				t.Fatalf("game %d picture %d: mark %d, want %d", game, p, got, want)
			}
			if got, want := row.SoundPhase, previousSoundPhase(game, p); got != want {
				t.Fatalf("game %d picture %d: sound phase %d, want %d", game, p, got, want)
			}
			if got, want := row.Unbound, game == SpellGameFirst && previousUnbound(p); got != want {
				t.Fatalf("game %d picture %d: unbound %t, want %t", game, p, got, want)
			}
		}
	}
}

func TestOnlyPictureSevenIsTheDeformation(t *testing.T) {
	for _, game := range []SpellGame{SpellGameFirst, SpellGameSecond, SpellGameSecond + 1} {
		for p := int32(-2); p <= 300; p++ {
			row := PictureCapabilityOf(game, p)
			want := p == 7 && game != SpellGameSecond+1
			if row.Deformation != want {
				t.Fatalf("game %d picture %d: deformation %t, want %t", game, p, row.Deformation, want)
			}
			if p == 7 {
				if zero := (PictureCapability{Picture: 7, Deformation: want}); row != zero {
					t.Fatalf("game %d picture 7: row %+v, want %+v", game, row, zero)
				}
			}
		}
	}
}

func TestEachGameNamesAPictureInOneRow(t *testing.T) {
	for game, rows := range pictureCapabilities {
		seen := map[int32]bool{}
		for _, row := range rows {
			if seen[row.Picture] {
				t.Fatalf("game %d names picture %d twice", game, row.Picture)
			}
			seen[row.Picture] = true
		}
	}
}
