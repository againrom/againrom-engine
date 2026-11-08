package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestCheckInnBuildersNamesEachShape(t *testing.T) {
	for _, c := range []struct{ name, path, code, kind string }{
		{"roster filter", "pkg/game/copy.go", "func f(o innOffer) { _ = o.speaker() }", "talker roster"},
		{"roster owner", "pkg/game/copy.go", "func innRoster() {}", "talker roster"},
		{"seat concatenation", "pkg/game/copy.go", "func f(c *secondCampaign) { x = append(x, c.speakers()...) }", "seat list"},
		{"seat literal", "pkg/game/copy.go", "func f() { x = innSeat{index: 2} }", "seat list"},
		{"actor literal", "pkg/game/copy.go", "func f() { x = tavernTalkObject{live: -1} }", "selected NPC"},
		{"actor owner", "pkg/game/copy.go", "func innActor() {}", "selected NPC"},
		{"hire dispatch", "pkg/game/copy.go", "func f(t *townScreen) { t.pressMercenary(3) }", "seat dispatch"},
		{"bank dispatch", "pkg/game/copy.go", "func f(c *secondCampaign) { c.seatHire(3) }", "seat dispatch"},
		{"talk open", "pkg/game/copy.go", "func f(t *townScreen) { t.openTalk(t,p,false) }", "talk start"},
		{"talk effect", "pkg/game/copy.go", "func f(c *secondCampaign) { c.talkTo(o) }", "talk start"},
		{"join timing", "pkg/game/copy.go", "func f(s *CampaignSession) { s.joinAtTalk(in,9) }", "talk start"},
		{"cursor start", "pkg/game/copy.go", "func f(c *talkCursor) { c.openSection(k,p) }", "talk start"},
		{"seat talk open", "pkg/game/townhost.go", "func (t *townScreen) Choose(i int) { t.openOfferDialogue(TownTavern, o, o.NPC) }", "seat dispatch"},
		{"adapter branch", "pkg/game/secondinn.go", "func (c *secondCampaign) speakers() { if o.npc == 1 {} }", "adapter decision"},
	} {
		t.Run(c.name, func(t *testing.T) {
			vs := CheckInnBuilders(map[string]string{c.path: "package game\n" + c.code})
			if len(vs) == 0 || !strings.Contains(vs[0].Reason, c.kind) || !strings.Contains(vs[0].From, c.path) {
				t.Fatalf("planted %s: %v", c.name, vs)
			}
			t.Log(vs[0])
		})
	}
	for _, file := range []string{innBuilderFile, "pkg/game/copy_test.go"} {
		if vs := CheckInnBuilders(map[string]string{file: "package game\nfunc f() { x = innSeat{index: 2}; t.openTalk(t,p,false) }"}); len(vs) != 0 {
			t.Fatal(vs)
		}
	}
	for entry, calls := range innEffectEntries {
		parts := strings.Split(entry, ":")
		for call := range calls {
			if vs := CheckInnBuilders(map[string]string{parts[0]: "package game\nfunc " + parts[1] + "() { c." + call + "() }"}); len(vs) != 0 {
				t.Fatal(vs)
			}
		}
	}
}

func TestInnBuildersHaveOneOwner(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for name := range innBuilders {
		if !strings.Contains(files[innBuilderFile], " "+name+"(") {
			t.Errorf("builder %s missing", name)
		}
	}
	for _, v := range CheckInnBuilders(files) {
		t.Error(v)
	}
}
