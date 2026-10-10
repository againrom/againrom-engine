package game

import (
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/ui"
)

// The generator's own tip text and toggle wiring (DIV-160..164): the
// description's class texts, ChargenSetup's own gate on TipsOff, and the
// toggle callback ChargenSetup hands the screen.
//
// chargen_test.go's own TestChargenSetupBuildsTheDecodedRows already proves
// the row shapes this file's fixtures reuse.

// The first game's description names the class texts by class, not by sex:
// chrgen1f.txt is the fighter's text and chrgen1m.txt the mage's (TOWN-187).
func TestGeneratorTipTextsAreByClass(t *testing.T) {
	tips := generatorDescriptions[base.GameROM1.Edition().Generator].Tips
	if tips.Fighter.File != mainPrefix+"text/tips/chrgen1f.txt" || tips.Mage.File != mainPrefix+"text/tips/chrgen1m.txt" {
		t.Fatalf("class tip texts = %q, %q", tips.Fighter.File, tips.Mage.File)
	}
}

// ChargenSetup resolves TipText to the fighter node's own shipped content —
// the class row's Start is always 0 today — and
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

// The texts are read whatever TipsMode says; each page's enter tests
// TipsMode itself (TOWN-518), so TipsOn carries the gate.
func TestChargenSetupWithTipsOffCarriesTheGate(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Containers: townTextFS(t, []synth.File{
		{Path: "text/tips/chrgen1f.txt", Data: []byte("fighter text")},
		{Path: "text/tips/chrsel2.txt", Data: []byte("second step")}})}}}
	f.tipsOff = true
	s := f.ChargenSetup()
	if s.TipText != "fighter text" || s.TipSelect[1] != "second step" {
		t.Fatalf("texts with TipsOff = %q, %q", s.TipText, s.TipSelect[1])
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
