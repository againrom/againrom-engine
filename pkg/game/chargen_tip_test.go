package game

import (
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/ui"
)

// The generator's own tip text and toggle wiring (1018 spec behaviours 1, 2,
// 3, 4; DIV-160..164): chargenTipPath's own class branch, ChargenSetup's own
// gate on TipsOff, and the toggle callback ChargenSetup hands the screen.
//
// chargen_test.go's own TestChargenSetupBuildsTheDecodedRows already proves
// the row shapes this file's fixtures reuse.

// chargenTipPath: the class row's own un-cycled opening index (0, Fighter)
// resolves to the fighter address; any other value resolves to the mage
// address (TOWN-187's own class-conditional branch). The class row's Start
// is 0 in every setup ChargenSetup currently builds (chargenTipPath's own
// doc), so this is the only place the mage branch is exercised at all today.
func TestChargenTipPathSelectsByClassStart(t *testing.T) {
	if got := chargenTipPath(0); got != ChargenFighterTipPath {
		t.Errorf("chargenTipPath(0) = %q, want ChargenFighterTipPath", got)
	}
	if got := chargenTipPath(1); got != ChargenMageTipPath {
		t.Errorf("chargenTipPath(1) = %q, want ChargenMageTipPath", got)
	}
}

// ChargenSetup resolves TipText to the fighter node's own shipped content —
// the class row's Start is always 0 today (chargenTipPath's own doc) — and
// leaves TipsOn, TipArt and SetTipsOn wired for the screen (spec behaviours
// 1, 3, 4).
func TestChargenSetupResolvesTheFighterTip(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Containers: townTextFS(t, []synth.File{{Path: "text/tips/chrgen1f.txt", Data: []byte("fighter text")}, {Path: "text/tips/chrgen1m.txt", Data: []byte("mage text")}})}}}
	f.Words = ui.AuthoredWords()
	f.Words.TipClose = "installed close"
	f.Words.TipShowNext = "installed toggle"
	s := f.ChargenSetup()
	if s.TipText != "fighter text" {
		t.Fatalf("TipText = %q, want the fighter node's own text", s.TipText)
	}
	if !s.TipsOn {
		t.Fatal("TipsOn = false, want the shipped default (true)")
	}
	if s.SetTipsOn == nil {
		t.Fatal("SetTipsOn is nil")
	}
	if s.TipClose != "installed close" || s.TipToggle != "installed toggle" {
		t.Fatalf("tip captions = %q / %q, want the front end's install words", s.TipClose, s.TipToggle)
	}
}

// TipsOff gates the read itself (TOWN-186's own construction-time gate,
// tips.go's own loadTip precedent): a suppressed front end's ChargenSetup
// never opens the node, so TipText is empty even though the install ships
// one.
func TestChargenSetupWithTipsOffLeavesTipTextEmpty(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Containers: townTextFS(t, []synth.File{{Path: "text/tips/chrgen1f.txt", Data: []byte("fighter text")}})}}}
	f.tipsOff = true
	s := f.ChargenSetup()
	if s.TipText != "" {
		t.Fatalf("TipText with TipsOff = %q, want empty", s.TipText)
	}
	if s.TipsOn {
		t.Fatal("TipsOn with TipsOff = true, want false")
	}
}

// A room whose node the install does not ship resolves an empty TipText, the
// same "no shipped node, no panel" shape tips.go's own loadTip already gives
// the four other screens (spec behaviour 2).
func TestChargenSetupWithNoShippedNodeLeavesTipTextEmpty(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Containers: townTextFS(t, nil)}}}
	s := f.ChargenSetup()
	if s.TipText != "" {
		t.Fatalf("TipText with no shipped chrgen1f.txt = %q, want empty", s.TipText)
	}
}

// SetTipsOn round-trips through FrontEnd.SetTipsOff, on ToggleTips' own
// contract (pkg/ui's chargen_tip_test.go proves the model calls it with the
// negated ToggleOn; this proves the callback itself lands on the front end's
// own store gate).
func TestChargenSetupSetTipsOnCallsSetTipsOff(t *testing.T) {
	f := &FrontEnd{}
	s := f.ChargenSetup()
	s.SetTipsOn(false)
	if !f.TipsOff() {
		t.Fatal("SetTipsOn(false) did not set the front end's own TipsOff")
	}
	s.SetTipsOn(true)
	if f.TipsOff() {
		t.Fatal("SetTipsOn(true) did not clear the front end's own TipsOff")
	}
}
