package game

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

// townDialogueLayout is the first game's open talk layout and the payload it
// pages, read through the town talk window.
func (t *townScreen) townDialogueLayout() (ui.NoticeLayout, []byte, bool) {
	v := t.talkView()
	if !v.Open {
		return ui.NoticeLayout{}, nil, false
	}
	return t.talkWindow.Layout(v), t.talkPage.payload, true
}

// talkPages is a conversation of fixed pages; at is the shown page, 0 closed.
type talkPages struct {
	pages []string
	at    int
	font  bool
}

func (p *talkPages) view(profile ui.TownTalkProfile) ui.TownTalkView {
	if p.at == 0 {
		return ui.TownTalkView{Profile: profile}
	}
	v := ui.TownTalkView{Profile: profile, Open: true, Page: ui.TownTalkPage{Text: p.pages[p.at-1]}}
	if p.font {
		v.Art.Font = missionFont()
	}
	return v
}

func (p *talkPages) turn() bool {
	if p.at == 0 {
		return false
	}
	if p.at++; p.at > len(p.pages) {
		p.at = 0
	}
	return true
}

func townTalkProfiles() map[string]ui.TownTalkProfile {
	return map[string]ui.TownTalkProfile{"first": firstTownTalkProfile, "second": secondTownTalkProfile}
}

// The one window pages, presses and releases OK, counts revisions and places
// its button the same way under each game's profile.
func TestTownTalkPagesThroughOK(t *testing.T) {
	authored := ui.AuthoredDialogueLayout().Button
	for name, profile := range townTalkProfiles() {
		t.Run(name, func(t *testing.T) {
			var w ui.TownTalk
			p := &talkPages{pages: []string{"One", "Two"}, font: true}
			if _, ok := w.Button(p.view(profile)); ok {
				t.Fatal("closed window offered its button")
			}
			if _, ok := w.Picture(p.view(profile)); ok || w.Shows(p.view(profile)) != 0 {
				t.Fatal("closed window drew or counted a show")
			}
			w.Open(false)
			p.at = 1
			if w.Revision() != 1 || w.Shows(p.view(profile)) != 1 {
				t.Fatalf("open: revision %d shows %d, want 1 and 1", w.Revision(), w.Shows(p.view(profile)))
			}
			button, ok := w.Button(p.view(profile))
			if !ok || button != authored {
				t.Fatalf("button %v %v, want the authored %v", button, ok, authored)
			}
			for page := 1; page <= 2; page++ {
				up, _ := w.Picture(p.view(profile))
				w.SetVisual(ui.DialogueButtonState{Hover: true, Pressed: true, Inside: true}, ui.DialogueBackdrop{})
				down, ok := w.Picture(p.view(profile))
				if !ok || !w.ButtonState().Pressed || imagesEqual(up, down) || w.Revision() != uint64(page) || p.at != page {
					t.Fatalf("page %d: the press did not draw a pressed button or it turned the page", page)
				}
				want := ui.RenderNotice(ui.AuthoredDialogueLayout().WithPortrait(false).WithDialogueButtonState(w.ButtonState()), missionFont(), p.pages[page-1], nil)
				if !imagesEqual(down, want) {
					t.Fatalf("page %d: the window is not the authored notice of its page", page)
				}
				w.SetVisual(ui.DialogueButtonState{Hover: true, Inside: true}, ui.DialogueBackdrop{})
				w.Advance(p.turn)
				if w.Revision() != uint64(page+1) {
					t.Fatalf("page %d: release left revision %d, want %d", page, w.Revision(), page+1)
				}
			}
			if p.at != 0 || w.Shows(p.view(profile)) != 0 {
				t.Fatal("the last OK did not close the window")
			}
			w.Advance(p.turn)
			if w.Revision() != 3 {
				t.Fatal("a turn that did not act counted a revision")
			}
		})
	}
}

// Each game's profile keeps that game's own answers.
func TestTownTalkProfilesKeepEachGame(t *testing.T) {
	frame := &ui.DialogFrame{}
	for name, profile := range townTalkProfiles() {
		var w ui.TownTalk
		p := &talkPages{pages: []string{"One"}, at: 1}
		w.Open(false)
		w.Open(true)
		v := p.view(profile)
		v.Art.Frame = frame
		shows, wantShows := w.Shows(v), uint64(1)
		_, drawn := w.Picture(v)
		bare, _ := w.Frame(v)
		v.Art.Font = missionFont()
		art, _ := w.Frame(v)
		if bare != nil {
			t.Errorf("%s: a window without a font answered a frame", name)
		}
		wantFrame, wantDrawn := (*ui.DialogFrame)(nil), true
		if name == "first" {
			wantShows, wantFrame, wantDrawn = 2, frame, false
		}
		if shows != wantShows || art != wantFrame || drawn != wantDrawn {
			t.Errorf("%s: shows %d frame %p drawn without font %v, want %d %p %v", name, shows, art, drawn, wantShows, wantFrame, wantDrawn)
		}
	}
}

// Neither town screen keeps its own copy of the talk window: each holds one
// ui.TownTalk and no button or backdrop state. The TownDialogue* methods are
// declared once, on the seat both screens embed, and each reaches the window
// rather than composing a notice itself; neither screen declares one.
func TestTownScreensHoldOneTownTalk(t *testing.T) {
	for _, screen := range []any{townScreen{}, secondCampaignScreen{}} {
		typ := reflect.TypeOf(screen)
		windows := 0
		var walk func(reflect.Type)
		walk = func(rt reflect.Type) {
			for i := 0; i < rt.NumField(); i++ {
				f := rt.Field(i)
				switch f.Type {
				case reflect.TypeOf(ui.TownTalk{}):
					windows++
				case reflect.TypeOf(ui.DialogueButtonState{}), reflect.TypeOf(ui.DialogueBackdrop{}):
					t.Errorf("%s.%s is the window's own state", typ.Name(), f.Name)
				}
				if f.Anonymous && f.Type.Kind() == reflect.Struct {
					walk(f.Type)
				}
			}
		}
		walk(typ)
		if windows != 1 {
			t.Errorf("%s holds %d town talk windows, want 1", typ.Name(), windows)
		}
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	methods := 0
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Body == nil {
				continue
			}
			recv := receiverName(fn.Recv.List[0].Type)
			if recv != "townScreen" && recv != "secondCampaignScreen" && recv != "townTalkSeat" {
				continue
			}
			window := townTalkMethods[fn.Name.Name]
			if window && recv != "townTalkSeat" {
				t.Errorf("%s.%s forwards to the talk window itself", recv, fn.Name.Name)
			}
			if window {
				methods++
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.SelectorExpr:
					switch n.Sel.Name {
					case "RenderNotice", "AuthoredDialogueLayout", "WithDialogueButtonState", "WithDialogueBackdrop":
						t.Errorf("%s.%s composes the talk window itself (%s)", recv, fn.Name.Name, n.Sel.Name)
					case "talkWindow":
						window = false
					}
				}
				return true
			})
			if window {
				t.Errorf("%s.%s does not reach the town talk window", recv, fn.Name.Name)
			}
		}
	}
	if methods != 8 {
		t.Errorf("found %d TownDialogue methods, want the seat's 8", methods)
	}
}

// Each game keeps one page cursor: the first game's screen and the second
// game's campaign hold one talkCursor, and no other part or payload field.
func TestTownTalkHasOneCursor(t *testing.T) {
	for _, holder := range []any{townScreen{}, secondCampaignScreen{}, secondCampaign{}, townDialogueState{}} {
		typ := reflect.TypeOf(holder)
		cursors := 0
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			switch {
			case f.Type == reflect.TypeOf(talkCursor{}):
				cursors++
			case f.Name == "said" || f.Name == "part" || f.Name == "payload":
				t.Errorf("%s.%s is a second copy of the cursor", typ.Name(), f.Name)
			}
		}
		want := 0
		if typ.Name() == "townScreen" || typ.Name() == "secondCampaign" {
			want = 1
		}
		if cursors != want {
			t.Errorf("%s holds %d cursors, want %d", typ.Name(), cursors, want)
		}
	}
}

// townTalkMethods is the town talk window's contract on a town screen.
var townTalkMethods = map[string]bool{
	"TownDialogue": true, "TownDialogueButton": true, "AdvanceTownDialogue": true, "TownDialogueRevision": true,
	"TownDialogueButtonState": true, "TownDialogueVisual": true, "TownDialogueShows": true, "TownDialogueFrame": true,
}

func receiverName(e ast.Expr) string {
	if star, ok := e.(*ast.StarExpr); ok {
		e = star.X
	}
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}
