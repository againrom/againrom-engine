package game

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

func gateRetentionInstalledFreshTown(t *testing.T) (*FrontEnd, *ui.App, *townScreen) {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SoundPlayer, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer, f.SpeechPlayer = nil, nil, nil, nil, nil
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Retained child", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	for _, n := range []int{10, 20} {
		if _, ok := f.Town.Won(n); !ok {
			t.Fatalf("controlled prerequisite Won(%d) refused", n)
		}
	}
	f.arriveInTown()
	if f.Town.Chapter() != 30 || f.Town.progress != nil {
		t.Fatal("fresh chapter30 prerequisite is not the fresh model")
	}
	a, s := exteriorApp(t, f)
	if err := a.HeadlessActivate("SHOP"); err != nil {
		t.Fatal("App shop entry", err)
	}
	if !slices.Contains(f.Town.Available(), 31) {
		t.Fatalf("installed shop entry did not accept child31: %v", f.Town.Available())
	}
	for n := 0; s.room == roomTalk && n < 64; n++ {
		if err := a.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
	}
	if s.room != roomShop {
		t.Fatal("shop dialogue did not page back to shop", s.room)
	}
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if !s.AtTownSquare() || a.Screen() != ui.ScreenTown {
		t.Fatal("App shop exit did not reach square")
	}
	t.Cleanup(a.StopAudio)
	r := f.Town.currentRecords().record(31)
	if r == nil || !r.announced || r.age != 0 {
		t.Fatal("fresh accepted child record", r)
	}
	t.Logf("fixture construction: installed root=%s; chargen; controlled Won10/Won20; synthetic initial App town entry; actual App SHOP accepted31; live available=%v age=%d latch=%v", f.Archives.Root, f.Town.Available(), r.age, r.announced)
	return f, a, s
}

func gateRetentionGateResult(t *testing.T, f *FrontEnd, a *ui.App, s *townScreen) (bool, bool) {
	t.Helper()
	s.CloseTip()
	if !s.AtTownSquare() || a.Screen() != ui.ScreenTown {
		t.Fatal("gate prerequisite not at square", a.Screen(), s.room)
	}
	pressGate(t, a, squareGatePoint(t, f))
	return s.AtWorldMap(), gateDialogueOpen(s)
}

func gateRetentionBackToSquare(t *testing.T, a *ui.App, s *townScreen) {
	t.Helper()
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if !s.AtTownSquare() || gateDialogueOpen(s) {
		t.Fatal("gate back prerequisite", s.room)
	}
}

func gateRetentionSave(t *testing.T, f *FrontEnd, a *ui.App, label string) (SaveStore, sav.CampaignProjection) {
	t.Helper()
	base := os.Getenv("AGAINROM_GATE_ARTIFACTS")
	var dir string
	if base == "" {
		dir = t.TempDir()
	} else {
		if err := os.MkdirAll(base, 0700); err != nil {
			t.Fatal(err)
		}
		var err error
		dir, err = os.MkdirTemp(base, filepath.Base(filepath.Clean(f.Archives.Root))+"-"+label+"-")
		if err != nil {
			t.Fatal(err)
		}
	}
	store := SaveStore{Dir: dir}
	f.ConfigureSaveSeams(a, store, OriginalStore{}, nil)
	raw := cityRosterF2Save(t, a, store, label)
	p := autoGetCampaign(t, raw)
	t.Logf("F2 SAV %s: main=%d latch=%v children=%+v", filepath.Join(dir, label+".sav"), p.Main.Mission, p.Main.Announced, p.Children)
	return store, p
}

func gateRetentionColdTown(t *testing.T, store SaveStore) (*FrontEnd, *ui.App, *townScreen) {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SoundPlayer, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer, f.SpeechPlayer = nil, nil, nil, nil, nil
	a := f.App("Retained child cold SAV")
	a.Layout(640, 480)
	f.ConfigureSaveSeams(a, store, OriginalStore{}, nil)
	if err := a.HeadlessActivate("load game"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessActivate("@first"); err != nil || a.Screen() != ui.ScreenTown {
		t.Fatal("actual cold LOAD SAV", err, a.Screen(), a.HeadlessMessage())
	}
	s := f.TownScreen().(*townScreen)
	s.CloseTip()
	t.Cleanup(a.StopAudio)
	t.Logf("actual cold LOAD SAV: main=%d available=%v records=%+v", f.Town.Chapter(), f.Town.Available(), f.Town.progress.children)
	return f, a, s
}

func TestReleaseGateCampaignRetention(t *testing.T) {

	t.Run("fresh-retained-child-cold-SAV", func(t *testing.T) {
		f, a, s := gateRetentionInstalledFreshTown(t)
		if _, ok := f.Town.Won(30); !ok {
			t.Fatal("controlled main Won30 refused")
		}
		f.arriveInTown()
		if f.Town.Chapter() != 40 || !slices.Contains(f.Town.Available(), 31) {
			t.Fatal("retained accepted child prerequisite", f.Town.Chapter(), f.Town.Available())
		}
		r := f.Town.currentRecords().record(31)
		if r == nil || r.age != 1 || !r.announced {
			t.Fatal("fresh retained age/latch", r)
		}
		t.Logf("before SAVE: controlled Won30; main40 unaccepted; available=%v; child31 latch=%v age=%d", f.Town.Available(), r.announced, r.age)
		onMap, line := gateRetentionGateResult(t, f, a, s)
		if !onMap || line {
			t.Fatal("pre-SAVE actual App gate did not expose accepted31", onMap, line)
		}
		gateRetentionBackToSquare(t, a, s)
		store, p := gateRetentionSave(t, f, a, "fresh-after-main30")
		present := false
		for _, r := range p.Children {
			if r.Mission == 31 {
				present = true
				if !r.Announced || r.Age != 1 {
					t.Errorf("retained SAV child31 latch/age=%v/%d", r.Announced, r.Age)
				}
			}
		}
		g, b, v := gateRetentionColdTown(t, store)
		coldMap, coldLine := gateRetentionGateResult(t, g, b, v)
		t.Logf("next actual App gate: before SAVE map=%v npc35=%v; after cold LOAD map=%v npc35=%v; emitted child31=%v", onMap, line, coldMap, coldLine, present)
		if !present || !slices.Contains(g.Town.Available(), 31) || !coldMap || coldLine {
			t.Errorf("current-state SAV loses the accepted retained child: emitted31=%v available=%v next gate map=%v npc35=%v", present, g.Town.Available(), coldMap, coldLine)
		}
		gateRetentionBackToSquare(t, b, v)
		if _, ok := g.Town.Won(40); !ok {
			t.Fatal("post-LOAD controlled main40 completion")
		}
		g.arriveInTown()
		if g.Town.currentRecords().record(31) != nil || slices.Contains(g.Town.Available(), 31) {
			t.Fatal("fresh SAV post-LOAD child expiry")
		}
		expired, ep := gateRetentionSave(t, g, b, "fresh-cold-post-load-expired")
		for _, child := range ep.Children {
			if child.Mission == 31 {
				t.Fatal("post-LOAD expired child emitted")
			}
		}
		h, c, w := gateRetentionColdTown(t, expired)
		if m, line := gateRetentionGateResult(t, h, c, w); m || !line {
			t.Fatal("post-LOAD expired next gate", m, line)
		}
	})
	t.Run("restored-retained-age-and-latch-controls", func(t *testing.T) {
		f, a, _ := gateRetentionInstalledFreshTown(t)
		store, p := gateRetentionSave(t, f, a, "accepted31-before-main30")
		if len(p.Children) != 1 || p.Children[0].Mission != 31 || !p.Children[0].Announced || p.Children[0].Age != 0 {
			t.Fatal("accepted age0 source SAV prerequisite", p.Children)
		}
		g, b, v := gateRetentionColdTown(t, store)
		if _, ok := g.Town.Won(30); !ok {
			t.Fatal("restored main30 completion")
		}
		g.arriveInTown()
		child := g.Town.progress.record(31)
		if child == nil || !child.announced || child.age != 1 {
			t.Fatal("restored retained age1 prerequisite", child)
		}
		onMap, line := gateRetentionGateResult(t, g, b, v)
		if !onMap || line {
			t.Fatal("restored accepted retained child should open gate", onMap, line)
		}
		gateRetentionBackToSquare(t, b, v)
		retained, _ := gateRetentionSave(t, g, b, "restored-retained-age1")
		h, c, w := gateRetentionColdTown(t, retained)
		if r := h.Town.progress.record(31); r == nil || r.age != 1 || !r.announced {
			t.Fatal("retained age1 latch lost in SAV", r)
		}
		open, fallback := gateRetentionGateResult(t, h, c, w)
		if !open || fallback {
			t.Fatal("cold retained-age1 gate", open, fallback)
		}
		gateRetentionBackToSquare(t, c, w)
		if _, ok := h.Town.Won(40); !ok {
			t.Fatal("restored next main40 completion")
		}
		h.arriveInTown()
		if h.Town.progress.record(31) != nil || slices.Contains(h.Town.Available(), 31) {
			t.Fatal("child not removed at age2")
		}
		expired, ep := gateRetentionSave(t, h, c, "expired-after-main40")
		for _, r := range ep.Children {
			if r.Mission == 31 {
				t.Fatal("expired child serialized")
			}
		}
		k, d, x := gateRetentionColdTown(t, expired)
		expiredMap, expiredLine := gateRetentionGateResult(t, k, d, x)
		if expiredMap || !expiredLine {
			t.Fatal("expired child gate did not open npc35", expiredMap, expiredLine)
		}
		t.Log("restored child age0=>1 retains latch through actual SAV/cold LOAD; second main load deletes age2 and next App gate opens npc35")
	})
	t.Run("unaccepted-child-control", func(t *testing.T) {
		f := releaseFront(t)
		f.Options = OptionsStore{}
		f.SoundPlayer, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer, f.SpeechPlayer = nil, nil, nil, nil, nil
		f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Unaccepted", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
		for _, n := range []int{10, 20} {
			if _, ok := f.Town.Won(n); !ok {
				t.Fatal(n)
			}
		}
		f.arriveInTown()
		a, _ := exteriorApp(t, f)
		t.Cleanup(a.StopAudio)
		store, p := gateRetentionSave(t, f, a, "unaccepted31")
		if len(p.Children) != 1 || p.Children[0].Mission != 31 || p.Children[0].Announced {
			t.Fatal("unaccepted source SAV", fmt.Sprint(p.Children))
		}
		g, b, s := gateRetentionColdTown(t, store)
		onMap, line := gateRetentionGateResult(t, g, b, s)
		if onMap || !line {
			t.Fatal("unaccepted child supplies no gate latch", onMap, line)
		}
		t.Log("actual cold LOAD of unaccepted child SAV supplies no gate latch; next App gate opens npc35")
	})
}
