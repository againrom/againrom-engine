package game

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// installFixture is a synthetic container source: the two text addresses
// this story reads, and nothing else. No game bytes enter the repository —
// a fixture line that stands for a Russian one is written as bytes.
type installFixture map[string][]byte

func (s installFixture) ReadFile(name string) ([]byte, error) {
	b, ok := s[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return b, nil
}

// textFile builds a payload of n CRLF-terminated lines with the given lines
// filled in, which is the shipped form the decoded loader walks.
func textFile(n int, lines map[int]string) []byte {
	out := []byte{}
	for i := 0; i < n; i++ {
		out = append(out, lines[i]...)
		out = append(out, '\r', '\n')
	}
	return out
}

// AC-1, AC-2: the split is the decoded walk and not a CRLF split, and a line's
// bytes come back exactly as shipped.
func TestSplitTextTableIsTheDecodedWalk(t *testing.T) {
	t.Run("CRLF is two lines", func(t *testing.T) {
		got := SplitTextTable([]byte("A\r\nB\r\n"))
		if got.Lines() != 2 {
			t.Fatalf("lines = %d, want 2", got.Lines())
		}
		if s, ok := got.At(0); !ok || s != "A" {
			t.Fatalf("line 0 = %q, %v", s, ok)
		}
		if s, ok := got.At(1); !ok || s != "B" {
			t.Fatalf("line 1 = %q, %v", s, ok)
		}
	})

	// THE DISCRIMINATOR. bytes.Split on "\r\n" answers ONE line here, `A\rXB`,
	// because there is no CRLF to split on. The decoded scan ends the line at
	// the CR and advances the cursor by 2, so the X is consumed and B is the
	// next line. A reader that agreed with the split would be reading a format
	// nothing decodes.
	t.Run("the byte after a CR is skipped whatever it is", func(t *testing.T) {
		got := SplitTextTable([]byte("A\rXB\r\n"))
		if got.Lines() != 2 {
			t.Fatalf("lines = %d, want 2", got.Lines())
		}
		if s, _ := got.At(0); s != "A" {
			t.Fatalf("line 0 = %q, want A", s)
		}
		if s, _ := got.At(1); s != "B" {
			t.Fatalf("line 1 = %q, want B", s)
		}
	})

	t.Run("a trailing run with no CR is not a line", func(t *testing.T) {
		if got := SplitTextTable([]byte("A\r\nB")); got.Lines() != 1 {
			t.Fatalf("lines = %d, want 1", got.Lines())
		}
	})

	// AC-2. A Russian install's bytes are CP866 and no code-page pass runs at
	// load; conversion happens at draw, through the font's selector.
	t.Run("high bytes are returned unchanged", func(t *testing.T) {
		raw := []byte{0x80, 0x9f, 0xa0, 0xff}
		payload := append(append([]byte{}, raw...), '\r', '\n')
		s, ok := SplitTextTable(payload).At(0)
		if !ok || s != string(raw) {
			t.Fatalf("line 0 = % x, %v, want % x", s, ok, raw)
		}
	})
}

// AC-3: the three shapes of absent.
func TestTextTableAbsent(t *testing.T) {
	tab := SplitTextTable([]byte("A\r\n\r\nC\r\n"))
	for _, tc := range []struct {
		name string
		i    int
	}{
		{"below zero", -1},
		{"at the line count", 3},
		{"past the line count", 400},
		{"an empty line", 1},
	} {
		if s, ok := tab.At(tc.i); ok {
			t.Errorf("%s: At(%d) = %q, want absent", tc.name, tc.i, s)
		}
	}
	if s, ok := tab.At(2); !ok || s != "C" {
		t.Fatalf("At(2) = %q, %v", s, ok)
	}
	// A nil table is the missing-file case and reports absent rather than
	// panicking.
	var nilTable *TextTable
	if _, ok := nilTable.At(0); ok || nilTable.Lines() != 0 {
		t.Fatal("a nil table stated a line")
	}
}

// AC-4: a source stating neither text file resolves to the authored word set.
func TestInstallWordsWithoutTablesAreAuthored(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  installFixture
	}{
		{"an empty source", installFixture{}},
		{"a source with other files", installFixture{LanguagePath: []byte("russian 1")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := ui.AuthoredWords()
			if LanguageSelector(tc.src) == 1 {
				want.ItemMagic, want.ItemSpellOf = "\x8c\x80\x83\x88\x9f:", "\xe1 \xa7\xa0\xaa\xab\xa8\xad\xa0\xad\xa8\xa5\xac"
				want.DurationUnit = "\xe1\xa5\xaa"
				want.ItemRays, want.TavernSleep = "\x8b\xe3\xe7\xa8", "\x91\xaf\xa0\xe2\xec"
			}
			if got := LoadInstallWords(tc.src).Words(); got != want {
				t.Fatal("missing tables did not retain authored labels for the chosen language")
			}
		})
	}
	// A nil source and a nil word set take the same arm.
	if got := LoadInstallWords(nil).Words(); got != ui.AuthoredWords() {
		t.Fatal("a nil source resolved something")
	}
	var nilWords *InstallWords
	if got := nilWords.Words(); got != ui.AuthoredWords() {
		t.Fatal("a nil word set resolved something")
	}
}

// AC-5, AC-6, AC-7: every decoded index this story reads, resolved from a
// fixture that states it.
func TestInstallWordsResolveEveryDecodedIndex(t *testing.T) {
	src := installFixture{
		MainTextPath: textFile(274, map[int]string{
			0:                    "Bey",
			1:                    "Idi",
			2:                    "Ohranyai",
			3:                    "Zaschischai",
			4:                    "Koldui",
			5:                    "V ataku",
			6:                    "Stoi na meste",
			7:                    "Otstupai",
			mainSlotNoticeButton: "Ok",
			mainSlotMissionWon:   "Mission Completed",
			mainSlotMissionLost:  "Mission Failed",
			mainSlotPauseNotice:  "Game halted. Press OK.",
			mainSlotTipClose:     "Shut",
			mainSlotTipShowNext:  "Keep showing tips",
			mainSlotShopUndo:     "Undo Table",
			mainSlotShopBuy:      "Buy Item",
			mainSlotShopSell:     "Sell Item",
			mainSlotShopExit:     "Leave Shop",
		}),
		DialogsTextPath: textFile(166, map[int]string{
			dialogsSlotSave:            "~Save Game",
			dialogsSlotLoad:            "~Load Game",
			dialogsSlotDiplomacy:       "Diplomacy",
			dialogsSlotGameOptions:     "Game ~Options",
			dialogsSlotSoundOptions:    "Sou~nd Options",
			dialogsSlotQuestObjectives: "~Quest Objectives",
			dialogsSlotEndQuest:        "~End Quest",
			dialogsSlotReturn:          "~Return to Game",
			dialogsSlotAbort:           "Abort Game",
			dialogsSlotChangeMap:       "Change Map",
			dialogsSlotVictory:         "~Victory!",
			dialogsSlotContinue:        "Continue",
			dialogsSlotExitMain:        "~Exit to Main Menu",
			dialogsSlotExitWindows:     "Exit to ~Windows",
		}),
	}
	got := LoadInstallWords(src).Words()
	want := ui.AuthoredWords()
	want.NoticeButton = "Ok"
	want.MissionWon = "Mission Completed"
	want.MissionLost = "Mission Failed"
	want.PauseNotice = "Game halted. Press OK."
	want.TipClose = "Shut"
	want.TipShowNext = "Keep showing tips"
	want.MenuSave = "~Save Game"
	want.MenuLoad = "~Load Game"
	want.MenuDiplomacy = "Diplomacy"
	want.MenuGameOptions = "Game ~Options"
	want.MenuSoundOptions = "Sou~nd Options"
	want.MenuQuestObjectives = "~Quest Objectives"
	want.MenuEndQuest = "~End Quest"
	want.MenuReturn = "~Return to Game"
	want.MenuAbort = "Abort Game"
	want.MenuChangeMap = "Change Map"
	want.MenuVictory = "~Victory!"
	want.OutcomeContinue = "Continue"
	want.MenuExitMain = "~Exit to Main Menu"
	want.MenuExitWindows = "Exit to ~Windows"
	want.ShopUndo = "Undo Table"
	want.ShopBuy = "Buy Item"
	want.ShopSell = "Sell Item"
	want.ShopExit = "Leave Shop"
	want.Command = [8]string{
		"Bey", "Idi", "Ohranyai", "Zaschischai",
		"Koldui", "V ataku", "Stoi na meste", "Otstupai",
	}
	for i, s := range want.Command {
		want.Hover[i] = s
	}
	for i, s := range map[int]string{77: "Ok", 140: "Mission Completed", 141: "Mission Failed", 119: "Game halted. Press OK.", 127: "Shut", 128: "Keep showing tips", 72: "Undo Table", 70: "Buy Item", 71: "Sell Item", 73: "Leave Shop"} {
		want.Hover[i] = s
	}
	if got != want {
		t.Fatalf("words = %#v, want %#v", got, want)
	}
}

// TOWN-206 names the tip panel captions as exact global main.txt slots 127
// and 128. Distinct neighbours make a swap or one-off index fail here rather
// than merely draw another plausible line from the same table.
func TestTipPanelWordsUseMainSlots127And128(t *testing.T) {
	src := installFixture{MainTextPath: textFile(130, map[int]string{
		126: "before", 127: "Close", 128: "Show tips next time", 129: "after",
	})}
	got := LoadInstallWords(src).Words()
	if got.TipClose != "Close" || got.TipShowNext != "Show tips next time" {
		t.Fatalf("tip captions = %q / %q, want main.txt[127] / [128]", got.TipClose, got.TipShowNext)
	}
}

func TestMercenaryRoleUsesMainSlot84(t *testing.T) {
	src := installFixture{MainTextPath: textFile(86, map[int]string{
		83: "before", 84: "Mercenary", 85: "after",
	})}
	if got := LoadInstallWords(src).Words().PanelCaptions[mainSlotMercenary]; got != "Mercenary" {
		t.Fatalf("mercenary role = %q, want main.txt[84]", got)
	}
}

func TestInstallWordsRetainsTheThreeNewLocalTables(t *testing.T) {
	src := installFixture{
		UnitNameTextPath: textFile(81, map[int]string{23: "installed unit"}),
		StatsTextPath:    textFile(50, map[int]string{43: "installed damage"}),
		SitesTextPath:    textFile(19, map[int]string{0: "installed home"}),
		MainTextPath: textFile(263, map[int]string{
			15: "installed body", 47: "none-a", 48: "none-b", 49: "many-a", 50: "many-b",
			mainSlotSaveAcknowledgement: "saved", mainSlotWorldReturn: "return", mainSlotWorldPayment: "payment",
		}),
	}
	w := LoadInstallWords(src)
	got := w.Words()
	if got.UnitNames[23] != "installed unit" || got.ItemStats[43] != "installed damage" ||
		got.WorldHomeTitle != "installed home" || got.PanelCaptions[15] != "installed body" {
		t.Fatalf("new local tables did not reach Words: %#v", got)
	}
	if got.SaveAcknowledgement != "saved" || got.WorldHomeDetail != "return" || got.WorldPayment != "payment" ||
		got.SelectionStatus != ([4]string{"none-a", "none-b", "many-a", "many-b"}) {
		t.Fatalf("new main.txt consumers did not resolve: %#v", got)
	}
}

func TestInstallWordsCarryBuildingNamesAndTheSelectedLanguageItemWords(t *testing.T) {
	src := installFixture{
		LanguagePath:     []byte("1"),
		BuildingTextPath: textFile(66, map[int]string{20: "installed ogre house"}),
		StatsTextPath:    textFile(50, map[int]string{38: "range word", 42: "casts word", 43: "damage word"}),
	}
	got := LoadInstallWords(src).Words()
	if got.BuildingNames[21] != "installed ogre house" || got.BuildingNames[20] != "" {
		t.Fatalf("building names are not indexed by structure class ID: %q %q", got.BuildingNames[20], got.BuildingNames[21])
	}
	if got.ItemCasts != "casts word" || got.ItemDamage != "damage word" || got.ItemRange != "range word" {
		t.Fatalf("spell lines did not take the install's stats words: %q %q %q", got.ItemCasts, got.ItemDamage, got.ItemRange)
	}
	authored := ui.AuthoredWords()
	if got.ItemRays == authored.ItemRays || got.TavernSleep == authored.TavernSleep {
		t.Fatalf("selector 1 left the authored English Rays or Sleep: %q %q", got.ItemRays, got.TavernSleep)
	}
	src[LanguagePath] = []byte("0")
	got = LoadInstallWords(src).Words()
	if got.ItemCasts != authored.ItemCasts || got.ItemRays != authored.ItemRays || got.TavernSleep != authored.TavernSleep {
		t.Fatalf("selector 0 changed the authored words: %q %q %q", got.ItemCasts, got.ItemRays, got.TavernSleep)
	}
}

// TestDiplomacyUsesDialogsLocalIndex4C writes distinct neighbours so changing
// the production constant to 0x4b or 0x4d cannot keep this witness green.
func TestDiplomacyUsesDialogsLocalIndex4C(t *testing.T) {
	src := installFixture{DialogsTextPath: textFile(0x4e, map[int]string{
		0x4b: "before", 0x4c: "Diplomacy", 0x4d: "Abort Game",
	})}
	if got := LoadInstallWords(src).Words().MenuDiplomacy; got != "Diplomacy" {
		t.Fatalf("MenuDiplomacy = %q, want dialogs.txt local index 0x4c", got)
	}
}

// TestConfirmationWordsUseDialogsLocalIndices2ATo2D uses literal indices and
// distinct neighbours so moving, swapping, or aliasing any production slot
// cannot keep the confirmation-label witness green.
func TestConfirmationWordsUseDialogsLocalIndices2ATo2D(t *testing.T) {
	src := installFixture{DialogsTextPath: textFile(0x2f, map[int]string{
		0x29: "before",
		0x2a: "Change Map",
		0x2b: "~Victory!",
		0x2c: "~Exit to Main Menu",
		0x2d: "Exit to ~Windows",
		0x2e: "after",
	})}
	w := LoadInstallWords(src).Words()
	got := []string{w.MenuChangeMap, w.MenuVictory, w.MenuExitMain, w.MenuExitWindows}
	want := []string{"Change Map", "~Victory!", "~Exit to Main Menu", "Exit to ~Windows"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("confirmation label %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSuccessContinueUsesDialogsLocalIndex154(t *testing.T) {
	src := installFixture{DialogsTextPath: textFile(156, map[int]string{
		153: "before", 154: "Continue", 155: "after",
	})}
	if got := LoadInstallWords(src).Words().OutcomeContinue; got != "Continue" {
		t.Fatalf("OutcomeContinue = %q, want dialogs.txt local index 154", got)
	}
}

// AC-8: resolution is per index. A table that states some of the lines resolves
// those and leaves the rest authored.
func TestInstallWordsResolvePerIndex(t *testing.T) {
	src := installFixture{
		MainTextPath:    textFile(274, map[int]string{mainSlotMissionWon: "done"}),
		DialogsTextPath: textFile(166, map[int]string{dialogsSlotAbort: "stop"}),
	}
	got := LoadInstallWords(src).Words()
	authored := ui.AuthoredWords()
	if got.MissionWon != "done" || got.MenuAbort != "stop" {
		t.Fatalf("stated lines did not resolve: %#v", got)
	}
	if got.MissionLost != authored.MissionLost || got.NoticeButton != authored.NoticeButton {
		t.Fatalf("unstated main lines were not authored: %#v", got)
	}
	if got.MenuSave != authored.MenuSave || got.MenuReturn != authored.MenuReturn {
		t.Fatalf("unstated dialogs lines were not authored: %#v", got)
	}
}

// The two indexing styles are separate methods and read separate tables.
// Handing a table-local index to Global, or the reverse, must not silently
// find the other table's line.
func TestGlobalAndDialogsAreDifferentIndexSpaces(t *testing.T) {
	src := installFixture{
		MainTextPath:    textFile(274, map[int]string{34: "main line 34"}),
		DialogsTextPath: textFile(166, map[int]string{34: "dialogs line 34"}),
	}
	w := LoadInstallWords(src)
	if s, ok := w.Global(34); !ok || s != "main line 34" {
		t.Fatalf("Global(34) = %q, %v", s, ok)
	}
	if s, ok := w.Dialogs(34); !ok || s != "dialogs line 34" {
		t.Fatalf("Dialogs(34) = %q, %v", s, ok)
	}
	// main.txt is the loader's FIRST call, so its 274 lines are global 0..273
	// and a global subscript past them belongs to another table, which this one
	// does not hold (TEXT-STRTAB-023).
	if _, ok := w.Global(274); ok {
		t.Fatal("Global reached past main.txt's own lines")
	}
}

// AC-12: how many of the seventeen seam-carried words resolve from a real install,
// per root. It reads the install and asserts nothing about its bytes beyond
// their presence and their difference between roots.
//
// AGAINROM_ASSETS names the first root and AGAINROM_ASSETS_RU the second;
// each half skips when its variable is unset, rather than assuming a sibling
// directory.
func TestInstallWordsOverALawfulInstall(t *testing.T) {
	count := func(t *testing.T, root string) ui.Words {
		t.Helper()
		archives, err := OpenArchives(root)
		if err != nil {
			t.Fatalf("OpenArchives(%s): %v", root, err)
		}
		w := LoadInstallWords(archives.Containers)
		got := w.Words()
		authored := ui.AuthoredWords()
		resolved := 0
		for _, p := range []struct {
			name    string
			got     string
			dialogs bool
			index   int
		}{
			{"NoticeButton", got.NoticeButton, false, mainSlotNoticeButton},
			{"MissionWon", got.MissionWon, false, mainSlotMissionWon},
			{"MissionLost", got.MissionLost, false, mainSlotMissionLost},
			{"MenuSave", got.MenuSave, true, dialogsSlotSave},
			{"MenuLoad", got.MenuLoad, true, dialogsSlotLoad},
			{"MenuDiplomacy", got.MenuDiplomacy, true, dialogsSlotDiplomacy},
			{"MenuGameOptions", got.MenuGameOptions, true, dialogsSlotGameOptions},
			{"MenuSoundOptions", got.MenuSoundOptions, true, dialogsSlotSoundOptions},
			{"MenuQuestObjectives", got.MenuQuestObjectives, true, dialogsSlotQuestObjectives},
			{"MenuEndQuest", got.MenuEndQuest, true, dialogsSlotEndQuest},
			{"MenuReturn", got.MenuReturn, true, dialogsSlotReturn},
			{"MenuAbort", got.MenuAbort, true, dialogsSlotAbort},
			{"MenuChangeMap", got.MenuChangeMap, true, dialogsSlotChangeMap},
			{"MenuVictory", got.MenuVictory, true, dialogsSlotVictory},
			{"OutcomeContinue", got.OutcomeContinue, true, dialogsSlotContinue},
			{"MenuExitMain", got.MenuExitMain, true, dialogsSlotExitMain},
			{"MenuExitWindows", got.MenuExitWindows, true, dialogsSlotExitWindows},
		} {
			var raw string
			var ok bool
			if p.dialogs {
				raw, ok = w.Dialogs(p.index)
			} else {
				raw, ok = w.Global(p.index)
			}
			if !ok {
				t.Errorf("%s: install lookup at index %d is absent", p.name, p.index)
				continue
			}
			if p.got != raw {
				t.Errorf("%s = %q, raw install lookup = %q", p.name, p.got, raw)
				continue
			}
			resolved++
		}
		t.Logf("%s: selector %d, main.txt %d lines, dialogs.txt %d lines, %d/17 install lookups resolved",
			filepath.Base(root), w.Selector, w.main.Lines(), w.dialogs.Lines(), resolved)
		if resolved != 17 {
			t.Errorf("%s: %d of 17 install lookups resolved", root, resolved)
		}
		if got == authored {
			t.Errorf("%s: every word equals this build's authored English", root)
		}
		return got
	}

	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the install word count needs a lawful install")
	}
	first := count(t, root)

	other := os.Getenv("AGAINROM_ASSETS_RU")
	if other == "" {
		t.Log("no AGAINROM_ASSETS_RU: the second root's half is not run")
		return
	}
	second := count(t, other)
	// EVERY ONE OF THE SEVENTEEN DIFFERS. TEXT-NAMETAB-026 measures 17 of
	// main.txt's 274 lines byte-identical across roots — seven structural lines
	// and the ten hall-of-fame names — and none of this set is in that group.
	same := 0
	for _, p := range [][2]string{
		{first.NoticeButton, second.NoticeButton}, {first.MissionWon, second.MissionWon},
		{first.MissionLost, second.MissionLost}, {first.MenuSave, second.MenuSave},
		{first.MenuLoad, second.MenuLoad}, {first.MenuDiplomacy, second.MenuDiplomacy},
		{first.MenuGameOptions, second.MenuGameOptions},
		{first.MenuSoundOptions, second.MenuSoundOptions},
		{first.MenuQuestObjectives, second.MenuQuestObjectives},
		{first.MenuEndQuest, second.MenuEndQuest}, {first.MenuReturn, second.MenuReturn},
		{first.MenuAbort, second.MenuAbort},
		{first.MenuChangeMap, second.MenuChangeMap}, {first.MenuVictory, second.MenuVictory},
		{first.OutcomeContinue, second.OutcomeContinue},
		{first.MenuExitMain, second.MenuExitMain}, {first.MenuExitWindows, second.MenuExitWindows},
	} {
		if p[0] == p[1] {
			same++
		}
	}
	t.Logf("words identical across the two roots: %d of 17", same)
	if same != 0 {
		t.Errorf("%d of 17 words are byte-identical across the two roots", same)
	}
}

func TestOriginalSixteenTextTableCountsOverBothLawfulInstalls(t *testing.T) {
	paths := []string{
		"main/text/main.txt", "main/text/heropicture.txt", "main/text/stats.txt",
		"main/text/spells.txt", "main/text/spell.txt", "main/text/dialogs.txt",
		"main/text/unitname.txt", "main/text/building.txt", "main/text/itemname.txt",
		"main/text/sites.txt", "main/text/npcnames.txt", "main/text/cutscene.txt",
		"main/text/cutpaths.txt", "main/text/tunes.txt", "patch/patch.txt",
		"main/text/credits.txt",
	}
	common := []int{274, 26, 50, 24, 28, 166, 81, 66, 416, 19, 95, 14, 14, 21, 67}
	paired := os.Getenv("AGAINROM_ASSETS_RU") != ""
	for _, tc := range []struct {
		env, name      string
		credits, total int
	}{
		{"AGAINROM_ASSETS", "EN", 207, 1568}, {"AGAINROM_ASSETS_RU", "RU", 166, 1527},
	} {
		root := os.Getenv(tc.env)
		if root == "" {
			t.Logf("%s skipped: %s is unset", tc.name, tc.env)
			continue
		}
		fsys, err := vfs.Open([]string{filepath.Join(root, MainArchive), filepath.Join(root, "patch.res")}, nil)
		if err != nil {
			t.Fatalf("%s vfs: %v", tc.name, err)
		}
		credits, wantTotal := tc.credits, tc.total
		name := tc.name
		if tc.env == "AGAINROM_ASSETS" && !paired {
			// The release gate deliberately runs the same primary variable once
			// per root. In that form identify the root by credits.txt, the only
			// table whose count differs, rather than assuming primary means EN.
			credits, wantTotal = 0, 0
			name = filepath.Base(root)
		}
		total := 0
		for i, path := range paths {
			table := LoadTextTable(fsys, path)
			want := credits
			if i < len(common) {
				want = common[i]
			} else if want == 0 {
				switch table.Lines() {
				case 207:
					want, wantTotal = 207, 1568
				case 166:
					want, wantTotal = 166, 1527
				default:
					t.Errorf("%s %s lines=%d want one of 207/166", name, path, table.Lines())
					want = table.Lines()
				}
			}
			if table.Lines() != want {
				t.Errorf("%s %s lines=%d want %d", name, path, table.Lines(), want)
			}
			total += table.Lines()
		}
		if total != wantTotal {
			t.Errorf("%s total=%d want %d", name, total, wantTotal)
		}
		t.Logf("%s: 16/16 tables, %d lines", name, total)
	}
}

// TestOriginalUITextWordSetExactDifferencesOverBothLawfulInstalls prints one
// row for every scalar leaf carried by ui.Words. The ordinary release gate sets
// one lawful root at a time and therefore executes the first half; the story's
// two-root gate also sets AGAINROM_ASSETS_RU and gets the exact cross-root
// report. Reflection is deliberately confined to this census: production uses
// named fields and finite arrays, never a run-time string key.
func TestOriginalUITextWordSetExactDifferencesOverBothLawfulInstalls(t *testing.T) {
	load := func(root string) ui.Words {
		archives, err := OpenArchives(root)
		if err != nil {
			t.Fatalf("OpenArchives(%s): %v", root, err)
		}
		return LoadInstallWords(archives.Containers).Words()
	}
	flatten := func(words ui.Words) map[string]string {
		out := make(map[string]string)
		var walk func(reflect.Value, string)
		walk = func(v reflect.Value, path string) {
			switch v.Kind() {
			case reflect.String:
				out[path] = v.String()
			case reflect.Array:
				for i := 0; i < v.Len(); i++ {
					walk(v.Index(i), path+"["+strconv.Itoa(i)+"]")
				}
			case reflect.Struct:
				t := v.Type()
				for i := 0; i < v.NumField(); i++ {
					walk(v.Field(i), t.Field(i).Name)
				}
			default:
				t.Fatalf("ui.Words census reached unsupported %s at %s", v.Kind(), path)
			}
		}
		walk(reflect.ValueOf(words), "")
		return out
	}

	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the original UI word-set census needs a lawful install")
	}
	en := flatten(load(root))
	t.Logf("%s: executed %d ui.Words scalar fields", filepath.Base(root), len(en))

	ruRoot := os.Getenv("AGAINROM_ASSETS_RU")
	if ruRoot == "" {
		t.Log("no AGAINROM_ASSETS_RU: exact EN/RU field differences were not selected")
		return
	}
	ru := flatten(load(ruRoot))
	if len(ru) != len(en) {
		t.Fatalf("ui.Words scalar population EN=%d RU=%d", len(en), len(ru))
	}
	paths := make([]string, 0, len(en))
	for path := range en {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	different := 0
	for _, path := range paths {
		ruValue, ok := ru[path]
		if !ok {
			t.Fatalf("RU ui.Words has no %s", path)
		}
		relation := "same"
		if en[path] != ruValue {
			relation = "different"
			different++
		}
		t.Logf("field %s: EN=%q RU=%q (%s)", path, en[path], ruValue, relation)
	}
	t.Logf("EN/RU exact field report: selected=%d executed=%d skipped=0 different=%d same=%d",
		len(paths), len(paths), different, len(paths)-different)
}

func TestOriginalConditionalPanelCaptionsRenderOverBothLawfulInstalls(t *testing.T) {
	run := func(t *testing.T, root string) {
		t.Helper()
		f, err := NewFrontEnd(root)
		if err != nil {
			t.Fatalf("NewFrontEnd(%q): %v", root, err)
		}
		font := f.tipFont()
		if font == nil {
			t.Fatal("front end has no compact-card font")
		}
		caster := f.Words.PanelCaptions[190]
		piercing := f.Words.PanelCaptions[191]
		if caster == "" || piercing == "" {
			t.Fatalf("installed conditional captions = %q/%q, want both nonempty", caster, piercing)
		}
		s := ui.PanelSubject{
			Words: f.Words, DetailLevel: 7, DetailSet: true,
			Char:   ui.UnitCharacter{Known: true, Mage: false, Experience: 900},
			Combat: ui.UnitCombat{Known: true, AlwaysHits: false},
			OriginalPanel: ui.OriginalPanelActor{Known: true, Flags: 0,
				XPValue: 9, Byte14A: 2},
			Weight: 35, WeightKnown: true,
		}
		layout := ui.CompactPanelLayout(nil)
		report := ui.CharacterPanelReport(layout, font, s)
		seenCaster, seenPiercing := false, false
		for _, row := range report {
			seenCaster = seenCaster || row.Label == caster
			seenPiercing = seenPiercing || row.RightLabel != "" && strings.HasPrefix(piercing, row.RightLabel)
		}
		if !seenCaster || !seenPiercing {
			t.Fatalf("installed report has slot190=%v slot191=%v: %+v", seenCaster, seenPiercing, report)
		}
		full := ui.RenderCharacterPanel(layout, font, s)
		mutations := []struct {
			name string
			edit func(*ui.PanelSubject)
		}{
			{"slot190 +0x1c zero", func(v *ui.PanelSubject) { v.OriginalPanel.XPValue = 0 }},
			{"slot191 +0x14a zero", func(v *ui.PanelSubject) { v.OriginalPanel.Byte14A = 0 }},
			{"player bit 0", func(v *ui.PanelSubject) { v.OriginalPanel.Flags = 0x1 }},
			{"human bit 4", func(v *ui.PanelSubject) { v.OriginalPanel.Flags = 0x10 }},
		}
		for _, mutation := range mutations {
			changed := s
			mutation.edit(&changed)
			if bytes.Equal(full.Pix, ui.RenderCharacterPanel(layout, font, changed).Pix) {
				t.Errorf("%s changed no rendered installed-text pixels", mutation.name)
			}
		}
		t.Logf("%s: rendered slots 190/191 and four independent actor-operand mutations",
			filepath.Base(root))
	}

	selected := 0
	for _, tc := range []struct{ env, name string }{
		{"AGAINROM_ASSETS", "primary"}, {"AGAINROM_ASSETS_RU", "second"},
	} {
		root := os.Getenv(tc.env)
		if root == "" {
			continue
		}
		selected++
		t.Run(tc.name, func(t *testing.T) { run(t, root) })
	}
	if selected == 0 {
		t.Skip("no AGAINROM_ASSETS root: installed conditional-caption rendering needs a lawful install")
	}
}

func TestMissionOutcomeTextComesOffTheViewer(t *testing.T) {
	if got := missionOutcomeText(nil, sim.OutcomeWon); got != MissionWonText {
		t.Fatalf("no viewer, won = %q, want %q", got, MissionWonText)
	}
	if got := missionOutcomeText(nil, sim.OutcomeLost); got != MissionLostText {
		t.Fatalf("no viewer, lost = %q, want %q", got, MissionLostText)
	}

	v := &ui.Viewer{}
	v.SetWords(LoadInstallWords(installFixture{
		MainTextPath: textFile(274, map[int]string{
			mainSlotMissionWon:  "Mission Completed",
			mainSlotMissionLost: "Mission Failed",
		}),
	}).Words())
	if got := missionOutcomeText(v, sim.OutcomeWon); got != "Mission Completed" {
		t.Fatalf("installed won = %q", got)
	}
	if got := missionOutcomeText(v, sim.OutcomeLost); got != "Mission Failed" {
		t.Fatalf("installed lost = %q", got)
	}
	// The two authored constants are the ui values under their old names.
	if MissionWonText != ui.MissionWonText || MissionLostText != ui.MissionLostText {
		t.Fatal("the pkg/game aliases drifted from the ui values")
	}
}

func TestTownDialogueButtonComesFromTheWordSet(t *testing.T) {
	installed := LoadInstallWords(installFixture{
		MainTextPath: textFile(274, map[int]string{mainSlotNoticeButton: "Ok"}),
	}).Words()
	for _, tc := range []struct {
		name  string
		words ui.Words
		want  string
	}{
		{"a hand-built front end", ui.Words{}, ui.AuthoredNoticeButton},
		{"an install stating slot 77", installed, "Ok"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.words.OnLayout(ui.AuthoredDialogueLayout())
			if got.ButtonLabel != tc.want {
				t.Fatalf("button = %q, want %q", got.ButtonLabel, tc.want)
			}
		})
	}
}
