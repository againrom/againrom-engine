package ui

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"strings"
	"testing"

	"againrom/pkg/render/frame"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
)

// The golden holds rectangles, line counts and pixel hashes of every modal
// dialog the engine shows, for each window size and locale. The fixture font
// and words are authored here; no game byte enters the file.

//go:embed testdata/dialog_witness.golden
var dialogWitnessGolden []byte

var dialogWitnessWindows = []image.Point{{640, 480}, {1024, 768}, {1280, 800}}

type dialogWitnessLocale struct {
	name, body, title, hint string
	font                    *text.Font
}

func dialogWitnessLocales() []dialogWitnessLocale {
	en := "The band reached the gate at dusk. The warden read the seal twice, then asked who would answer for the wagons and their cargo.\r\nNobody spoke."
	ru := "\xc2 \xf1\xf3\xec\xe5\xf0\xea\xe0\xf5 \xee\xf2\xf0\xff\xe4 \xef\xee\xe4\xee\xf8\xb8\xeb \xea \xe2\xee\xf0\xee\xf2\xe0\xec. \xd1\xf2\xf0\xe0\xe6 \xe4\xe2\xe0\xe6\xe4\xfb \xef\xf0\xee\xf7\xb8\xeb \xef\xe5\xf7\xe0\xf2\xfc \xe8 \xf1\xef\xf0\xee\xf1\xe8\xeb \xee \xef\xee\xea\xee\xf7\xe5\xed\xfb\xf5 \xe3\xf0\xf3\xe7\xe0\xf5.\r\n\xd2\xe8\xf8\xe8\xed\xe0."
	return []dialogWitnessLocale{
		{name: "en", body: en, title: "Mods differ", hint: "Load the save again after matching the mods.", font: dialogueClaimFont()},
		{name: "ru", body: ru, title: "\xcc\xee\xe4\xfb \xf0\xe0\xe7\xeb\xe8\xf7\xe0\xfe\xf2\xf1\xff", hint: "\xc7\xe0\xe3\xf0\xf3\xe7\xe8\xf2\xe5 \xf1\xee\xf5\xf0\xe0\xed\xe5\xed\xe8\xe5 \xf1\xed\xee\xe2\xe0.", font: solidFont15()},
	}
}

// noticeDialog names the description type the notices use, so the witness
// keeps compiling while the type moves.
type noticeDialog = Dialog

// dialogWitnessEntry is one instance at one window and locale.
type dialogWitnessEntry struct {
	Name     string
	W, H     int
	Hash     string
	Rects    []string
	Lines    int
	LineHash string
}

func dialogPixHash(pix []byte) string {
	sum := sha256.Sum256(pix)
	return hex.EncodeToString(sum[:8])
}

func dialogRect(name string, r image.Rectangle) string {
	return fmt.Sprintf("%s=%d,%d,%d,%d", name, r.Min.X, r.Min.Y, r.Max.X, r.Max.Y)
}

func dialogWitnessRow(name string, pic *image.RGBA, rects []string, lines []string) dialogWitnessEntry {
	e := dialogWitnessEntry{Name: name, Rects: rects, Lines: len(lines)}
	if pic != nil {
		e.W, e.H, e.Hash = pic.Bounds().Dx(), pic.Bounds().Dy(), dialogPixHash(pic.Pix)
	}
	if len(lines) > 0 {
		e.LineHash = dialogPixHash([]byte(strings.Join(lines, "\n")))
	}
	return e
}

func noticeWitnessRects(l noticeDialog) []string {
	return []string{dialogRect("box", l.Box), dialogRect("text", l.Text), dialogRect("button", l.Button),
		dialogRect("second", l.SecondaryButton), dialogRect("portrait", l.Portrait), dialogRect("beside", l.TextBesidePortrait), dialogRect("bar", l.Scrollbar)}
}

func dialogFrameWitnessArt(t *testing.T) *DialogFrame {
	t.Helper()
	art := dialogueClaimFrame()
	return art
}

func dialogWitnessShape() Dialog {
	return Dialog{ShapeSize: image.Pt(288, 224), TextTop: 57, TextWidth: 208, LinePitch: 17,
		AnswerSize: image.Pt(82, 24), AnswerGap: 46, AnswerBottom: 28}
}

// dialogWitness composes every instance for one window and locale.
func dialogWitness(t *testing.T, loc dialogWitnessLocale, win image.Point) []dialogWitnessEntry {
	t.Helper()
	var out []dialogWitnessEntry
	prefix := fmt.Sprintf("%s/%dx%d/", loc.name, win.X, win.Y)
	f := loc.font
	art := dialogFrameWitnessArt(t)
	face := solidPic(160, 240, color.RGBA{R: 200, G: 40, B: 90, A: 255})

	notice := func(name string, l noticeDialog) {
		pic := RenderNotice(l, f, loc.body, face)
		out = append(out, dialogWitnessRow(prefix+"notice/"+name, pic, noticeWitnessRects(l), NoticeLayoutOf(l, f, loc.body)))
	}
	dialogueArt := AuthoredDialogueLayout()
	dialogueArt.Frame = art
	notice("dialogue-portrait-art", dialogueArt.WithPortrait(true))
	notice("dialogue-noportrait-art", dialogueArt.WithPortrait(false))
	notice("dialogue-portrait-colours", AuthoredDialogueLayout().WithPortrait(true))
	notice("dialogue-noportrait-colours", AuthoredDialogueLayout().WithPortrait(false))
	bodied := AuthoredOutcomeLayout().WithBody(f, loc.body)
	notice("outcome", bodied)
	success := AuthoredSuccessLayout()
	success.Frame = art
	notice("success-art", success.WithBody(f, loc.body))
	notice("failure", AuthoredFailureLayout().WithBody(f, loc.body))
	hover := AuthoredOutcomeLayout()
	hover.ButtonState = DialogueButtonState{Hover: true, Pressed: true, Inside: true}
	notice("outcome-pressed", hover)
	portraitPlain := AuthoredDialogueLayout().WithPortrait(true)
	portraitPlain.Style = NoticeStylePlain
	notice("plain-portrait-colours", portraitPlain)

	// ComposeDialogueNotice over a scene.
	scene := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
	for i := range scene.Pix {
		scene.Pix[i] = byte(i*7 + 31)
	}
	composed := dialogueArt.WithPortrait(false)
	ComposeDialogueNotice(scene, composed, f, loc.body, nil, composed.Box.Min)
	out = append(out, dialogWitnessRow(prefix+"notice/composed-dialogue", scene, noticeWitnessRects(composed), nil))

	// The viewer's placement of notices and the help panel at this window.
	v, err := NewViewer("t", grid(20, 20), &terrain.Tileset{})
	if err != nil {
		t.Fatal(err)
	}
	v.font = f
	layoutViewport(v, win.X-MissionPanelW, win.Y)
	v.SetDialogFrame(art)
	place := func(name string, l noticeDialog) {
		pic, at, s, ok := v.noticePresent()
		if !ok || pic == nil {
			t.Fatalf("%s: no notice", name)
		}
		rects := append(noticeWitnessRects(l), dialogRect("placed", image.Rectangle{Min: at, Max: at.Add(image.Pt(int(float64(pic.Bounds().Dx())*s), int(float64(pic.Bounds().Dy())*s)))}), fmt.Sprintf("scale=%.4f", s))
		out = append(out, dialogWitnessRow(prefix+"placed/"+name, pic, rects, NoticeLayoutOf(l, f, loc.body)))
	}
	v.SetNotice(loc.body, NoticeDialogue)
	place("dialogue", v.noticeLayout())
	v.ClearNotice()
	v.SetNotice(loc.body, NoticeOutcome)
	place("outcome", v.noticeLayout())
	v.ClearNotice()
	body := loc.body
	for i := 0; i < 6; i++ {
		body += "\r\n" + loc.body
	}
	if !v.OpenHelp(body) {
		t.Fatal("help did not open")
	}
	place("help", v.noticeLayout())
	v.ClearNotice()

	// Town talk.
	var talk TownTalk
	talk.Open(false)
	words := AuthoredWords()
	view := TownTalkView{Open: true, Profile: TownTalkProfile{NeedsFont: true, ExposesFrame: true},
		Page: TownTalkPage{Text: loc.body, Face: face, FaceWindow: image.Rect(10, 10, 82, 106), Portrait: true},
		Art:  TownTalkArt{Words: words, Frame: art, Font: f}}
	pic, _ := talk.Picture(view)
	l := talk.Layout(view)
	_, body2 := talk.Frame(view)
	out = append(out, dialogWitnessRow(prefix+"towntalk/portrait", pic, append(noticeWitnessRects(l), dialogRect("frame", body2)), NoticeLayoutOf(l, f, loc.body)))

	// The notice windows: a list window and a text window over a page panel.
	panel := image.Rect(20, 20, 620, 460)
	if win.X < 700 {
		panel = image.Rect(10, 10, 600, 440)
	}
	identity := func(s string) string { return s }
	listRows := []noticeRow{{{Text: "Base game: saved "}, {Text: "rom1-en", Alert: true}, {Text: ", now "}, {Text: "rom1-ru", Alert: true}}}
	for i := 0; i < 12; i++ {
		listRows = append(listRows, noticeRow{{Text: loc.body, Alert: i%3 == 1}})
	}
	list := dialogRun{popupSpec: popupSpec{Title: loc.title, Hint: loc.hint, Answers: []string{"OK"}, Runs: listRows}}
	windowCase := func(name string, w dialogRun, press int) {
		dst := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
		ll := w.layout(f, panel)
		lines := w.noticeLines(f, ll, identity)
		if press >= 0 {
			w.press.Press(press, true)
		}
		w.compose(dst, f, panel, art, loadScrollTestFrames(), identity, image.Pt(panel.Min.X+20, panel.Max.Y-40), true)
		rects := []string{dialogRect("body", ll.Body), dialogRect("title", ll.HeadingBox), dialogRect("hint", ll.Hint), dialogRect("text", ll.Text),
			dialogRect("list", ll.List.Rect), fmt.Sprintf("rows=%d", ll.List.Rows), fmt.Sprintf("pitch=%d", ll.LinePitch)}
		for _, a := range ll.Answers {
			rects = append(rects, dialogRect("answer", a))
		}
		var plain []string
		for _, ln := range lines {
			var b strings.Builder
			fmt.Fprintf(&b, "%d|", ln.indent)
			for _, r := range ln.runs {
				fmt.Fprintf(&b, "%v:%s;", r.Alert, r.Text)
			}
			plain = append(plain, b.String())
		}
		out = append(out, dialogWitnessRow(prefix+"window/"+name, dst, rects, plain))
	}
	windowCase("list", list, -1)
	scrolled := list
	scrolled.top = 3
	windowCase("list-scrolled-pressed", scrolled, 0)
	question := PageConfirm{Before: "Delete the save#", After: "#for good?", Window: dialogWitnessShape(), Yes: "Yes", No: "No"}.question(loc.title)
	windowCase("question-text", question, -1)
	longQuestion := PageConfirm{Before: strings.Repeat(loc.body+" ", 3), Window: dialogWitnessShape(), Yes: "Yes", No: "No"}.question("")
	windowCase("question-clamped", longQuestion, 1)

	// The gold modal at this window.
	goldApp := inventoryTestApp(t)
	gv := goldApp.flow.viewer
	gv.font = f
	layoutViewport(gv, win.X-MissionPanelW, win.Y)
	gv.SetEntities([]MapEntity{{ID: 5, Cell: image.Pt(1, 1)}})
	gv.sel = selection{5}
	gv.SetInventorySubject(InventorySubject{ID: 5, Pack: []*image.RGBA{solidPic(8, 8, color.RGBA{R: 0xff, A: 0xff})}, PackCount: []uint32{2500}, PackPurse: []bool{true}})
	gv.openGoldModal()
	if !gv.goldModalOpen() {
		t.Fatal("gold did not open")
	}
	goldPic, goldAt, ok := gv.goldModalPresent()
	if !ok {
		t.Fatal("gold picture absent")
	}
	edit, action, cancel := gv.goldModalControls()
	out = append(out, dialogWitnessRow(prefix+"gold/open", goldPic, []string{dialogRect("box", gv.goldModalBox()), dialogRect("edit", edit), dialogRect("action", action), dialogRect("cancel", cancel), fmt.Sprintf("at=%d,%d", goldAt.X, goldAt.Y)}, nil))

	// The game menu panel, through the app, at this window.
	menuApp := newTestApp(t, nil, nil)
	menuApp.SetWords(AuthoredWords(), f, nil)
	menuApp.SetGameMenuArt(art)
	menuApp.Layout(win.X, win.Y)
	menuApp.flow.openGameMenu(ScreenTown)
	menuPic := menuApp.GameMenuPanel()
	menuRows := menuApp.flow.menuRows()
	panelRect := gameMenuPanelRectFor(menuApp.flow.menuPanelSurface(), menuRows, menuApp.flow.menuGeometryProfile())
	var rowLabels []string
	for i, row := range menuRows {
		rowLabels = append(rowLabels, fmt.Sprintf("%s|%v|%v|%v|%s", row.Label, row.Enabled, row.Status, row.Literal, dialogRect("row", gameMenuRowRect(panelRect, i))))
	}
	out = append(out, dialogWitnessRow(prefix+"gamemenu/town", menuPic, []string{dialogRect("panel", panelRect)}, rowLabels))

	// The save dialog page and a Load page with a mod-mismatch window.
	saveApp, _, _ := saveChooserStyleApp(t, 3)
	saveApp.SetWords(AuthoredWords(), f, nil)
	saveApp.SetGameMenuArt(art)
	saveApp.Layout(win.X, win.Y)
	savePaint, err := saveApp.savePagePaint()
	if err != nil {
		t.Fatal(err)
	}
	out = append(out, dialogWitnessRow(prefix+"save/page", savePaint.pix, []string{dialogRect("panel", saveApp.flow.saveScreen.page.Panel)}, nil))

	mismatch := mismatchApp(t, fixtureMismatch{everyMismatchKind})
	mismatch.SetGameMenuArt(art)
	mismatch.Layout(win.X, win.Y)
	mismatchKey(t, mismatch, "down")
	mismatchKey(t, mismatch, "enter")
	mm, open := mismatch.HeadlessModMismatch()
	if !open {
		t.Fatal("no mismatch window")
	}
	mmPix, _, err := mismatch.HeadlessPicture(HeadlessRequest{Part: HeadlessPartFrame})
	if err != nil {
		t.Fatal(err)
	}
	out = append(out, dialogWitnessRow(prefix+"window/mismatch", mmPix, []string{dialogRect("list", mm.List), fmt.Sprintf("visible=%d", mm.Visible), fmt.Sprintf("lines=%d", mm.Lines), fmt.Sprintf("top=%d", mm.Top)}, mm.Rows))

	// An editor popup, the row-derived form.
	spec := popupSpec{Title: loc.title, At: image.Pt(40, 30), Width: 320, Bounds: image.Rectangle{Max: win},
		Rows:  []popupRow{{Label: "Name", Value: "value"}, {Kind: popupHeading, Label: loc.title}, {Kind: popupSwitch, Label: "Mode", Value: "On"}},
		Notes: []string{loc.hint}, NoteInk: color.RGBA{200, 180, 90, 255}, Buttons: []string{"OK", "Cancel"}}
	pop := layoutPopup(spec, f, f)
	popPic := image.NewRGBA(image.Rect(0, 0, win.X, win.Y))
	drawPopup(popPic, f, f, loadScrollTestFrames(), spec, pop, editorPopupInk, image.Pt(60, 60), true)
	popRects := []string{dialogRect("rect", pop.Rect), dialogRect("title", pop.Title), dialogRect("close", pop.Close)}
	for _, r := range pop.Buttons {
		popRects = append(popRects, dialogRect("button", r))
	}
	for _, r := range pop.Rows {
		popRects = append(popRects, dialogRect("row", r.Rect))
	}
	out = append(out, dialogWitnessRow(prefix+"popup/editor", popPic, popRects, nil))
	return out
}

func dialogWitnessAll(t *testing.T) []dialogWitnessEntry {
	var all []dialogWitnessEntry
	for _, loc := range dialogWitnessLocales() {
		for _, win := range dialogWitnessWindows {
			all = append(all, dialogWitness(t, loc, win)...)
		}
	}
	return all
}

// dialogWitnessDiff names every entry that differs from the golden.
func dialogWitnessDiff(want, got []dialogWitnessEntry) []string {
	var out []string
	if len(want) != len(got) {
		out = append(out, fmt.Sprintf("entries: want %d, got %d", len(want), len(got)))
	}
	byName := map[string]dialogWitnessEntry{}
	for _, e := range want {
		byName[e.Name] = e
	}
	for _, e := range got {
		old, ok := byName[e.Name]
		if !ok {
			out = append(out, e.Name+": new entry")
			continue
		}
		if fmt.Sprint(old) != fmt.Sprint(e) {
			out = append(out, fmt.Sprintf("%s: golden %v\ngot %v", e.Name, old, e))
		}
	}
	return out
}

// TestDialogWitnessEqualsGolden: every dialog's rectangles, lines and pixels
// at three window sizes and two locales equal the checked-in golden, written
// on the commit before the dialogs became one description.
// AGAINROM_DIALOG_WITNESS_WRITE=<path> rewrites the golden at that path.
func TestDialogWitnessEqualsGolden(t *testing.T) {
	got := dialogWitnessAll(t)
	if path := os.Getenv("AGAINROM_DIALOG_WITNESS_WRITE"); path != "" {
		b, err := json.MarshalIndent(got, "", " ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, b, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	var want []dialogWitnessEntry
	if err := json.Unmarshal(dialogWitnessGolden, &want); err != nil {
		t.Fatalf("golden: %v", err)
	}
	if len(want) == 0 {
		t.Fatal("empty golden")
	}
	if diff := dialogWitnessDiff(want, got); len(diff) != 0 {
		t.Fatalf("dialog witness differs in %d entries:\n%s", len(diff), strings.Join(diff, "\n"))
	}
}

// TestDialogWitnessFailsAMovedPixelAndAMovedRectangle is the planted fault: a
// one-pixel change of a composed picture and a one-pixel move of a rectangle
// each fail the comparison.
func TestDialogWitnessFailsAMovedPixelAndAMovedRectangle(t *testing.T) {
	var want []dialogWitnessEntry
	if err := json.Unmarshal(dialogWitnessGolden, &want); err != nil {
		t.Fatalf("golden: %v", err)
	}
	if len(want) == 0 {
		t.Fatal("empty golden")
	}
	loc, win := dialogWitnessLocales()[0], dialogWitnessWindows[0]
	got := dialogWitness(t, loc, win)
	full := append([]dialogWitnessEntry(nil), want...)
	if diff := dialogWitnessDiff(want, full); len(diff) != 0 {
		t.Fatalf("a golden compared with itself differs: %v", diff)
	}
	moved := dialogWitnessRow(got[0].Name, RenderNotice(AuthoredDialogueLayout(), loc.font, loc.body+"!", nil), got[0].Rects, nil)
	full[0] = moved
	if diff := dialogWitnessDiff(want, full); len(diff) == 0 {
		t.Fatal("a changed picture passed the witness")
	}
	rect := AuthoredDialogueLayout()
	rect.Box = rect.Box.Add(image.Pt(1, 0))
	full[0] = want[0]
	full[0].Rects = noticeWitnessRects(rect)
	if diff := dialogWitnessDiff(want, full); len(diff) == 0 {
		t.Fatal("a moved rectangle passed the witness")
	}
}
