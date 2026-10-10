package ui

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
	"time"
)

func TestChargenDifficultyDraftInputAndState(t *testing.T) {
	setup := chargenLegalSetup()
	art := &ChargenPresentation{Layout: testGenerator()}
	for i := range art.Levels {
		for j := range art.Levels[i] {
			pic := image.NewRGBA(image.Rect(0, 0, 8, 8))
			draw.Draw(pic, pic.Bounds(), &image.Uniform{C: color.RGBA{R: uint8(40 + i*30 + j), A: 255}}, image.Point{}, draw.Src)
			art.Levels[i][j] = pic
		}
	}
	setup.PreCreate = &ChargenPreCreate{Art: art}
	c := NewChargen(setup)
	a := newTestApp(t, appRows(1), okLoader(t))
	if err := a.OpenChargen(c, func(ChargenResult) (MapOpener, error) { return okOpener(t), nil }); err != nil {
		t.Fatal(err)
	}
	if c.Difficulty() != 2 {
		t.Fatal("new generator must default to Normal")
	}
	// Keyboard: cycle through all three controls using the public control list.
	for _, level := range []int{1, 3, 2} {
		state, _ := a.HeadlessChargenState()
		control, ok := state.Control(ChargenControlDifficulty, []string{"easy", "normal", "hard"}[level-1])
		if !ok {
			t.Fatal("missing difficulty control")
		}
		for c.Focus() != control.Focus {
			if err := a.HeadlessKey("down"); err != nil {
				t.Fatal(err)
			}
		}
		if err := a.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
		if c.Difficulty() != level || c.Stage() != PreCreateStage {
			t.Fatalf("keyboard level=%d stage=%v", c.Difficulty(), c.Stage())
		}
	}
	// Pointer: the press selects at once (VIDEO-SFX-058); the release changes
	// nothing.
	point := preLevelOrigin[2].Add(image.Pt(2, 2))
	now := time.Unix(10, 0)
	a.step(appInput{CursorX: point.X, CursorY: point.Y, PrimaryPressed: true}, now)
	if c.Difficulty() != 3 {
		t.Fatal("press did not select Hard")
	}
	a.step(appInput{CursorX: point.X, CursorY: point.Y, PrimaryReleased: true}, now)
	if c.Difficulty() != 3 {
		t.Fatal("release changed the draft")
	}
	c.Forward()
	c.Reset()
	if c.Difficulty() != 3 {
		t.Fatal("Forward/Reset lost difficulty")
	}
	c.SelectDifficulty(0)
	if c.Difficulty() != 3 {
		t.Fatal("detailed page changed difficulty")
	}
	if !c.Back() || c.Difficulty() != 3 {
		t.Fatal("Back lost difficulty")
	}
	c.SelectDifficulty(-1)
	c.SelectDifficulty(3)
	if c.Difficulty() != 3 {
		t.Fatal("invalid level accepted")
	}
	c.Forward()
	res, ok := c.Result()
	if !ok || res.Difficulty != 3 {
		t.Fatal("result did not carry selected difficulty")
	}
	if NewChargen(setup).Difficulty() != 2 {
		t.Fatal("new generator inherited a previous draft")
	}
}

func TestChargenDifficultyArtStates(t *testing.T) {
	setup := chargenLegalSetup()
	art := &ChargenPresentation{Layout: testGenerator()}
	for i := range art.Levels {
		for j := range art.Levels[i] {
			pic := image.NewRGBA(image.Rect(0, 0, 4, 4))
			draw.Draw(pic, pic.Bounds(), &image.Uniform{C: color.RGBA{R: uint8(30 + i*10 + j), A: 255}}, image.Point{}, draw.Src)
			art.Levels[i][j] = pic
		}
	}
	setup.PreCreate = &ChargenPreCreate{Art: art}
	c := NewChargen(setup)
	for _, tc := range []struct {
		level        int
		hover, press chargenControl
		state        int
	}{
		{1, chargenNone, chargenNone, 0}, {0, chargenLevel0, chargenNone, 1},
		{1, chargenLevel1, chargenNone, 2}, {0, chargenLevel0, chargenLevel0, 1},
	} {
		frame := composeChargenPage(c, tc.hover, tc.press)
		p := preLevelOrigin[tc.level].Add(image.Pt(1, 1))
		if got := frame.RGBAAt(p.X, p.Y); got.R != uint8(30+tc.level*10+tc.state) {
			t.Fatalf("state %+v: pixel %v", tc, got)
		}
	}
}

// TestChargenDifficultyHeroOverlapOwnership: only Mask.bmp decides what a
// press hits (TOWN-520), whatever art lies under the point, and an unchosen
// portrait draws no overlay over a chosen level (TOWN-521).
func TestChargenDifficultyHeroOverlapOwnership(t *testing.T) {
	setup := chargenLegalSetup()
	art := &ChargenPresentation{Layout: testGenerator(), PreMask: chargenMask(640, 480)}
	hero := image.NewRGBA(image.Rect(0, 0, 26, 10))
	level := image.NewRGBA(image.Rect(0, 0, 102, 111))
	hero.SetRGBA(23, 8, color.RGBA{40, 31, 18, 255})
	for x := 147; x <= 149; x++ {
		art.PreMask.SetColorIndex(x, 174, 100)
		level.SetRGBA(x-48, 174-65, color.RGBA{90, 80, 70, 255})
	}
	art.PreMask.SetColorIndex(146, 174, 60)
	level.SetRGBA(146-48, 174-65, color.RGBA{90, 80, 70, 255})
	for state := 0; state < 3; state++ {
		art.Choices[2][state] = hero
		art.Levels[2][state] = level
	}
	setup.PreCreate = &ChargenPreCreate{Art: art}
	c := NewChargen(setup)
	c.SelectDifficulty(2)
	frame := composeChargenPage(c, chargenNone, chargenNone)
	for _, tc := range []struct {
		x    int
		want chargenControl
	}{{147, chargenChoice2}, {148, chargenChoice2}, {149, chargenChoice2}, {146, chargenLevel2}} {
		if got := frame.RGBAAt(tc.x, 174); got != (color.RGBA{90, 80, 70, 255}) {
			t.Fatalf("composed pixel (%d,174)=%v, want the chosen level's art", tc.x, got)
		}
		if got := preControlAt(c, image.Pt(tc.x, 174)); got != tc.want {
			t.Errorf("hit (%d,174)=%v want%v", tc.x, got, tc.want)
		}
	}
	c.SelectDifficulty(1)
	a := newTestApp(t, appRows(1), okLoader(t))
	if err := a.OpenChargen(c, func(ChargenResult) (MapOpener, error) { return okOpener(t), nil }); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"press", "release"} {
		if err := a.HeadlessPointer(action, 148, 174); err != nil {
			t.Fatal(err)
		}
	}
	if c.PreChoice() != 2 || c.Difficulty() != 2 {
		t.Fatalf("hero click: choice=%d difficulty=%d, want choice2 Normal", c.PreChoice(), c.Difficulty())
	}
}
