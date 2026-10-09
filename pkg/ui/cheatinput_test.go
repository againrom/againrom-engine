package ui

import (
	"bytes"
	"image"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestCheatChatRoutesEveryCommandThroughAppInput(t *testing.T) {
	commands := []string{"#create goblin", "#modify hero", "#summon goblin", "#killall", "#kill all", "#kill cheaters",
		"#kill hero", "#pickup all", "#show map", "#hide map", "#victory", "#event 3", "#Chicken"}
	for _, command := range commands {
		t.Run(command, func(t *testing.T) {
			a, v := mkOnMap(t)
			v.SetFont(messageFont())
			var got []string
			a.SetCheatCommands(func(line string) { got = append(got, line) }, nil)
			if err := a.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessType(command, false); err != nil {
				t.Fatal(err)
			}
			if line, open := a.HeadlessChatState(); !open || line != command || len(got) != 0 {
				t.Fatalf("draft=(%q,%v), submissions=%v", line, open, got)
			}
			if pic, err := a.HeadlessChatFrame(); err != nil || pic == nil {
				t.Fatalf("chat picture=%v error=%v", pic, err)
			}
			if err := a.HeadlessKey("numpad-enter"); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, []string{command}) {
				t.Fatalf("submissions=%v, want %q once", got, command)
			}
			if _, open := a.HeadlessChatState(); open {
				t.Fatal("submitted chat remains open")
			}
		})
	}
}

func TestCheatAltRoutesEveryLetterThroughAppInput(t *testing.T) {
	for letter := byte('A'); letter <= 'Z'; letter++ {
		t.Run(string(letter), func(t *testing.T) {
			a, v := mkOnMap(t)
			var got []byte
			a.SetCheatCommands(nil, func(letter byte) { got = append(got, letter) })
			before := v.hudHidden
			if err := a.HeadlessKey("alt-" + strings.ToLower(string(letter))); err != nil {
				t.Fatal(err)
			}
			var want []byte
			if letter >= 'B' && letter <= 'Y' && letter != 'S' {
				want = []byte{letter}
			}
			if !slices.Equal(got, want) {
				t.Fatalf("Alt+%c destination=%v, want %v", letter, got, want)
			}
			if v.hudHidden != before || v.spellArmed || v.missionMode() != modeNone {
				t.Fatal("console chord reached ordinary gameplay")
			}
			key := altLetterKeys[letter-'A']
			if got := readAltLetter(true, onlyKey(key)); got != letter {
				t.Fatalf("physical binding=%c, want %c", got, letter)
			}
			if got := readAltLetter(false, onlyKey(key)); got != 0 {
				t.Fatal("plain letter became a console chord")
			}
		})
	}
	if got := readAltLetter(true, onlyKey(ebiten.KeyF1)); got != 0 {
		t.Fatal("function key became a console chord")
	}
}

func TestCheatChatConsumesGameplayAndEditsBoundedText(t *testing.T) {
	a, v := mkOnMap(t)
	v.sel = selection{4}
	v.PostMessage("keep", MessageWhite, time.Hour)
	var chats []string
	var alts []byte
	a.SetCheatCommands(func(line string) { chats = append(chats, line) }, func(letter byte) { alts = append(alts, letter) })
	if err := a.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	before := v.hudHidden
	in := a.headlessIdleInput()
	in.Attack, in.Book, in.SelectAll, in.Kill = true, true, true, true
	in.Typed = "#ChickenЖ"
	a.step(in, a.headlessAt())
	if v.hudHidden != before || v.missionMode() != modeNone || !slices.Equal(v.sel, selection{4}) || len(alts) != 0 {
		t.Fatal("chat draft leaked gameplay input")
	}
	if err := a.HeadlessKey("backspace"); err != nil {
		t.Fatal(err)
	}
	if line, _ := a.HeadlessChatState(); line != "#Chicken" {
		t.Fatalf("rune backspace left %q", line)
	}
	if len(v.MessageLines()) != 1 {
		t.Fatal("chat backspace cleared mission messages")
	}
	if err := a.HeadlessType(strings.Repeat("Ж", cheatChatLimit), false); err != nil {
		t.Fatal(err)
	}
	if line, _ := a.HeadlessChatState(); len(line) > cheatChatLimit || len(line) < cheatChatLimit-1 {
		t.Fatalf("bounded UTF-8 draft has %d bytes", len(line))
	}
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ScreenMap || len(chats) != 0 {
		t.Fatal("Escape submitted chat or opened another screen")
	}
	if line, open := a.HeadlessChatState(); open || line != "" {
		t.Fatal("Escape retained the draft")
	}
}

func TestCheatInputRespectsFocusPopupAndMissionEntry(t *testing.T) {
	a, v := mkOnMap(t)
	var chats []string
	var alts []byte
	a.SetCheatCommands(func(line string) { chats = append(chats, line) }, func(letter byte) { alts = append(alts, letter) })
	if err := a.HeadlessFocus(false); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("alt-d"); err != nil {
		t.Fatal(err)
	}
	if _, open := a.HeadlessChatState(); open || len(alts) != 0 {
		t.Fatal("unfocused input opened chat or ran debug command")
	}
	if err := a.HeadlessFocus(true); err != nil {
		t.Fatal(err)
	}
	v.menuUp = true
	if err := a.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("alt-d"); err != nil {
		t.Fatal(err)
	}
	if _, open := a.HeadlessChatState(); open || !slices.Equal(alts, []byte{'D'}) {
		t.Fatal("popup opened chat or swallowed the mission console command")
	}
	v.menuUp = false
	if err := a.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessType("#Chicken", false); err != nil {
		t.Fatal(err)
	}
	a.flow.viewer = newViewer(t, grid(8, 8))
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if line, open := a.HeadlessChatState(); open || line != "" || len(chats) != 0 {
		t.Fatal("mission entry retained or submitted another mission's chat")
	}
}

func TestCheatChatUsesMissionFontAndInstallEncoder(t *testing.T) {
	for _, name := range []string{"en", "ru"} {
		t.Run(name, func(t *testing.T) {
			a, v := mkOnMap(t)
			v.SetFont(messageFont())
			var runes []rune
			a.flow.encodeMenuKey = func(r rune) (byte, bool) {
				runes = append(runes, r)
				if r == 'Ж' {
					return 0x86, true
				}
				return byte(r), r < 128
			}
			a.SetCheatCommands(func(string) {}, nil)
			if err := a.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
			line := "#Chicken"
			if name == "ru" {
				line += " Ж"
			}
			if err := a.HeadlessType(line, false); err != nil {
				t.Fatal(err)
			}
			pic, err := a.HeadlessChatFrame()
			if err != nil || pic.Bounds().Empty() || !slices.Equal(runes, []rune(line)) {
				t.Fatalf("chat picture=%v encoder runes=%v error=%v", pic, runes, err)
			}
			_, at := a.cheatChatPicture()
			if !pic.Bounds().Add(at).In(image.Rect(0, 0, v.cam.ViewW, v.cam.ViewH)) {
				t.Fatal("chat picture extends outside the map viewport")
			}
		})
	}
}

func TestCheatMapRevealPreservesClientExplorationAcrossFogPushes(t *testing.T) {
	v := newViewer(t, grid(2, 2))
	plane := []byte{FogUnseen, FogExplored, FogVisible, FogUnseen}
	before := bytes.Clone(plane)
	v.SetFog(plane, 2, 2)
	v.SetCheatMapReveal(true)
	for row := 0; row < 2; row++ {
		for col := 0; col < 2; col++ {
			if v.fogAt(col, row) != FogVisible {
				t.Fatal("show map did not reveal every client cell")
			}
		}
	}
	if !bytes.Equal(plane, before) {
		t.Fatal("show map wrote the ordinary world plane")
	}
	v.SetCheatMapReveal(false)
	if want := []byte{FogVisible, FogVisible, FogVisible, FogVisible}; !bytes.Equal(v.fogPlane, want) {
		t.Fatalf("hide map client plane=%v, want %v", v.fogPlane, want)
	}
	if v.FogRevealed() || v.fogAt(0, 0) != FogVisible || !v.fogGateEntity(2, 0, 0) {
		t.Fatal("hide map cleared client visibility before the ordinary fog boundary")
	}
	next := []byte{FogVisible, FogUnseen, FogUnseen, FogUnseen}
	v.SetFog(next, 2, 2)
	if v.fogAt(1, 0) != FogVisible {
		t.Fatal("a state push cleared transient visibility without a fog refresh")
	}
	v.RefreshCheatFog()
	if want := []byte{FogVisible, FogExplored, FogExplored, FogExplored}; !bytes.Equal(v.fogPlane, want) {
		t.Fatalf("next client plane=%v, want %v", v.fogPlane, want)
	}
	if v.fogGateEntity(2, 1, 0) || !v.fogGateGround(1, 0) {
		t.Fatal("ordinary fog refresh lost exploration or kept an unseen enemy visible")
	}
	if !bytes.Equal(next, []byte{FogVisible, FogUnseen, FogUnseen, FogUnseen}) {
		t.Fatal("later fog push mutated ordinary fog")
	}
	cold := newViewer(t, grid(2, 2))
	cold.SetFog(next, 2, 2)
	if cold.FogRevealed() || cold.fogAt(1, 0) != FogUnseen {
		t.Fatal("fresh viewer retained another client's cheat revelation")
	}
}

func TestCheatMapRevealFreezesVisibilityUntilHiddenFogRefresh(t *testing.T) {
	v := newViewer(t, grid(2, 2))
	plane := []byte{FogUnseen, FogUnseen, FogUnseen, FogUnseen}
	v.SetFog(plane, 2, 2)
	v.SetCheatMapReveal(true)
	v.RefreshCheatFog()
	v.SetFog(plane, 2, 2)
	if v.fogAt(0, 0) != FogVisible {
		t.Fatal("ordinary fog refresh cleared the active reveal")
	}
	v.SetCheatMapReveal(false)
	v.SetFog(plane, 2, 2)
	if v.fogAt(0, 0) != FogVisible {
		t.Fatal("hide did not preserve the visible client word")
	}
	v.RefreshCheatFog()
	if v.fogAt(0, 0) != FogExplored || plane[0] != FogUnseen {
		t.Fatal("an unchanged ordinary sample did not clear only client visibility")
	}
}

func TestCheatMapRevealDoesNotChangeTemporaryRevealPolicy(t *testing.T) {
	v := newViewer(t, grid(2, 2))
	plane := []byte{FogUnseen, FogExplored, FogVisible, FogUnseen}
	v.SetFog(plane, 2, 2)
	v.SetFogReveal(true)
	v.SetFogReveal(false)
	if !bytes.Equal(v.fogPlane, plane) || len(v.cheatExplored) != 0 || v.fogAt(0, 0) != FogUnseen {
		t.Fatal("temporary diagnostic reveal became permanent exploration")
	}
}

func TestCheatChatAdmitsHelpAndRetainsDraft(t *testing.T) {
	a, v := mkOnMap(t)
	v.SetFont(messageFont())
	a.flow.words.HelpText = "help"
	v.SetWords(a.flow.words)
	a.SetCheatCommands(func(string) {}, nil)
	if err := a.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessType("#Chicken", false); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("f1"); err != nil {
		t.Fatal(err)
	}
	if !v.HelpOpen() {
		t.Fatal("F1 over chat did not open help")
	}
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if v.HelpOpen() {
		t.Fatal("Escape did not close chat's help panel")
	}
	if line, open := a.HeadlessChatState(); !open || line != "#Chicken" {
		t.Fatalf("help close lost chat draft=(%q,%v)", line, open)
	}
}

func TestCheatChatAllowsAltConsoleWithoutClosingDraft(t *testing.T) {
	a, v := mkOnMap(t)
	v.SetFont(messageFont())
	var letters []byte
	a.SetCheatCommands(func(string) {}, func(letter byte) { letters = append(letters, letter) })
	if err := a.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessType("#Chicken", false); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("alt-h"); err != nil {
		t.Fatal(err)
	}
	if line, open := a.HeadlessChatState(); !open || line != "#Chicken" || string(letters) != "H" {
		t.Fatalf("Alt console while chatting: draft=%q open=%t letters=%q", line, open, letters)
	}
	before := a.cheatInput.line
	in := a.headlessIdleInput()
	in.AltLetter, in.Viewer.Alt, in.Typed, in.Attack = 'H', true, "h", true
	if a.step(in, a.headlessAt()) || a.cheatInput.line != before || string(letters) != "HH" || v.missionMode() != modeNone {
		t.Fatal("physical Alt letter changed the chat draft or missed the console")
	}
}

func TestCheatAltScreenshotBypassesChatAndMissionPopups(t *testing.T) {
	for _, surface := range []string{"map", "chat", "help", "notice", "gold", "gameMenu"} {
		t.Run(surface, func(t *testing.T) {
			a, v := mkOnMap(t)
			if surface == "gold" {
				a, v = purseApp(t)
				v.openGoldModal()
				if !v.goldModalOpen() {
					t.Fatal("setup did not open Drop Gold")
				}
			}
			v.SetFont(messageFont())
			a.SetScreenshotSink(func(image.Image) error { return nil })
			var console []byte
			a.SetCheatCommands(func(string) {}, func(letter byte) { console = append(console, letter) })
			switch surface {
			case "chat":
				if err := a.HeadlessKey("enter"); err != nil {
					t.Fatal(err)
				}
				if err := a.HeadlessType("draft", false); err != nil {
					t.Fatal(err)
				}
			case "help":
				a.flow.words.HelpText = "help"
				v.SetWords(a.flow.words)
				if err := a.HeadlessKey("f1"); err != nil {
					t.Fatal(err)
				}
			case "notice":
				v.SetNotice("event", NoticeDialogue)
			case "gameMenu":
				if err := a.HeadlessKey("escape"); err != nil {
					t.Fatal(err)
				}
			}
			if err := a.HeadlessKey("alt-s"); err != nil {
				t.Fatal(err)
			}
			if !a.screenshot.requested || len(console) != 0 {
				t.Fatalf("Alt+S on %s: request=%t console=%v", surface, a.screenshot.requested, console)
			}
			if surface == "chat" {
				if line, open := a.HeadlessChatState(); !open || line != "draft" {
					t.Fatal("screenshot changed the chat draft")
				}
			}
		})
	}
	a, _ := mkOnMap(t)
	a.SetScreenshotSink(func(image.Image) error { return nil })
	if err := a.HeadlessFocus(false); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("alt-s"); err != nil {
		t.Fatal(err)
	}
	if a.screenshot.requested {
		t.Fatal("unfocused Alt+S requested a screenshot")
	}
}

func TestCheatAltConsoleAndScreenshotUseMissionContext(t *testing.T) {
	for _, surface := range []string{"map", "help", "notice", "gold", "gameMenu", "saveMission", "loadMission", "loadMissionMenu", "town", "townMenu", "saveTown", "loadTownMenu", "loadMenu", "picker"} {
		t.Run(surface, func(t *testing.T) {
			a, v := mkOnMap(t)
			if surface == "gold" {
				a, v = purseApp(t)
				v.openGoldModal()
				if !v.goldModalOpen() {
					t.Fatal("setup did not open Drop Gold")
				}
			}
			v.SetFont(messageFont())
			switch surface {
			case "help":
				a.flow.words.HelpText = "help"
				v.SetWords(a.flow.words)
				if err := a.HeadlessKey("f1"); err != nil {
					t.Fatal(err)
				}
				if !v.HelpOpen() {
					t.Fatal("setup did not open Help")
				}
			case "notice":
				v.SetNotice("event", NoticeDialogue)
			case "gameMenu", "townMenu":
				back := ScreenMap
				if surface == "townMenu" {
					back = ScreenTown
				}
				a.flow.openGameMenu(back)
			case "saveMission", "saveTown":
				spy := &saveDialogSpy{}
				spy.install(a)
				back := ScreenMap
				if surface == "saveTown" {
					back = ScreenTown
				}
				a.flow.openGameMenu(back)
				if err := a.HeadlessGameMenuAction("save"); err != nil {
					t.Fatal(err)
				}
				if a.Screen() != ScreenSave {
					t.Fatal("setup did not open Save")
				}
			case "loadMission":
				a.flow.openLoad(ScreenMap)
			case "loadMissionMenu", "loadTownMenu":
				back := ScreenMap
				if surface == "loadTownMenu" {
					back = ScreenTown
				}
				a.flow.openGameMenu(back)
				a.flow.openLoad(ScreenGameMenu)
			case "loadMenu":
				a.flow.openLoad(ScreenMenu)
			case "town":
				a.flow.setScreen(ScreenTown)
			case "picker":
				a.flow.setScreen(ScreenPicker)
			}
			want := !slices.Contains([]string{"town", "townMenu", "saveTown", "loadTownMenu", "loadMenu", "picker"}, surface)
			var console []byte
			a.SetCheatCommands(func(string) {}, func(letter byte) { console = append(console, letter) })
			a.SetScreenshotSink(func(image.Image) error { return nil })
			if err := a.HeadlessKey("alt-d"); err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessKey("alt-s"); err != nil {
				t.Fatal(err)
			}
			var wantConsole []byte
			if want {
				wantConsole = []byte{'D'}
			}
			if !slices.Equal(console, wantConsole) || a.screenshot.requested != want {
				t.Fatalf("Alt route on %s: console=%v screenshot=%t, want admission=%t", surface, console, a.screenshot.requested, want)
			}
			if surface == "saveMission" {
				before := a.flow.saveDialog.request.Name
				a.step(appInput{AltLetter: 'H', Typed: "h", Viewer: Input{Alt: true}}, commandFrozen)
				if a.flow.saveDialog.request.Name != before || !slices.Equal(console, []byte{'D', 'H'}) {
					t.Fatal("physical Alt console changed the save label or missed its destination")
				}
			}
		})
	}
}
