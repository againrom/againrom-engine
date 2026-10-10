package ui

import (
	"reflect"
	"testing"

	"againrom/pkg/render/text"
)

// installedWords is a word set every field of which differs from the authored
// one, so a test cannot pass by accident on a field nothing wrote.
func installedWords() Words {
	return Words{
		NoticeButton:        "Ok",
		MissionWon:          "Mission Completed",
		MissionLost:         "Mission Failed",
		MenuSave:            "~Save Game",
		MenuLoad:            "~Load Game",
		MenuDiplomacy:       "Diplomacy",
		MenuGameOptions:     "Game ~Options",
		MenuSoundOptions:    "Sou~nd Options",
		MenuQuestObjectives: "~Quest Objectives",
		MenuEndQuest:        "~End Quest",
		MenuReturn:          "~Return to Game",
		MenuAbort:           "Abort Game",
		MenuChangeMap:       "Change Map",
		MenuVictory:         "~Victory!",
		OutcomeContinue:     "Continue",
		MenuExitMain:        "~Exit to Main Menu",
		MenuExitWindows:     "Exit to ~Windows",
	}
}

// Words keeps decoded fields and homogeneous index families explicit. The
// owner-directed copy, including the duration unit, has named fields.
// The name does not spell the field count: it went stale once already
// ("...Eleven" when this struct held eleven fields) the first time a
// decoded index was added, on the same failure `pkg/sim/binary_test.go`'s
// own doc block warns against for the byte-form version. The count lives
// only in len(want) below.
func TestWordsHoldsExactlyTheNamedVocabularyFields(t *testing.T) {
	ft := reflect.TypeOf(Words{})
	want := map[string]bool{
		"NoticeButton": false, "MissionWon": false, "MissionLost": false,
		"PauseNotice": false, "HelpText": false,
		"TipClose": false, "TipShowNext": false,
		"MenuSave": false, "MenuLoad": false, "MenuDiplomacy": false, "MenuGameOptions": false,
		"MenuSoundOptions": false, "MenuQuestObjectives": false, "QuestHeading": false,
		"MenuEndQuest": false, "MenuReturn": false, "MenuAbort": false,
		"MenuChangeMap": false, "MenuVictory": false,
		"OutcomeContinue": false,
		"MenuExitMain":    false, "MenuExitWindows": false,
		"ShopUndo": false, "ShopBuy": false, "ShopSell": false, "ShopExit": false,
		"SchoolTrain": false, "SchoolExit": false,
		"TavernHire": false, "TavernHired": false, "TavernFire": false, "TavernTalk": false, "TavernExit": false, "TavernSleep": false,
		"Command": false,
		"Hover":   false, "SiteHints": false,
		"UnitNames": false, "BuildingNames": false, "PanelCaptions": false, "ItemStats": false,
		"ItemCasts": false, "ItemDamage": false, "ItemRange": false, "ItemRays": false,
		"ItemMagic": false, "ItemSpellOf": false, "ItemSpellOfSuffix": false, "ItemSpellNames": false, "SpellBookNames": false,
		"DurationUnit": false,
		"MapListSize":  false, "MapListColumns": false,
		"SaveAcknowledgement": false, "SaveDialog": false, "Engine": false, "WorldHomeTitle": false,
		"WorldHomeDetail": false, "WorldPayment": false, "SelectionStatus": false,
		"SkillRaised": false,
		"PickedUp":    false, "PickedUpNow": false, "PickedUpPieces": false,
		"PickedUpGold": false, "PickedUpGoldUnit": false,
		"SettingNotice": false, "NoHeroSelected": false,
	}
	if ft.NumField() != len(want) {
		names := make([]string, ft.NumField())
		for i := range names {
			names[i] = ft.Field(i).Name
		}
		t.Fatalf("Words has %d fields %v, want exactly %d named vocabulary fields",
			ft.NumField(), names, len(want))
	}
	for i := 0; i < ft.NumField(); i++ {
		if _, ok := want[ft.Field(i).Name]; !ok {
			t.Errorf("unexpected field %q: every field needs a decoded source or an owner-authored reason",
				ft.Field(i).Name)
		}
	}
	// Individual decoded positions, including deliberately empty table slots,
	// are tested at their consumer. The scalar fallbacks must all be visible.
	a := AuthoredWords()
	for name, value := range map[string]string{"TipClose": a.TipClose, "TipShowNext": a.TipShowNext,
		"SchoolTrain": a.SchoolTrain, "SchoolExit": a.SchoolExit,
		"TavernHire": a.TavernHire, "TavernHired": a.TavernHired, "TavernFire": a.TavernFire,
		"TavernTalk": a.TavernTalk, "TavernExit": a.TavernExit,
		"SaveAcknowledgement": a.SaveAcknowledgement,
		"WorldHomeTitle":      a.WorldHomeTitle, "WorldHomeDetail": a.WorldHomeDetail,
		"WorldPayment": a.WorldPayment,
		"PickedUp":     a.PickedUp, "PickedUpNow": a.PickedUpNow, "PickedUpPieces": a.PickedUpPieces,
		"PickedUpGold": a.PickedUpGold, "PickedUpGoldUnit": a.PickedUpGoldUnit} {
		if value == "" {
			t.Errorf("authored %s is empty", name)
		}
	}
}

// A fighter's slot 1..5 reads lines 0..4, a mage's lines 5..9; slot 0 and an
// out-of-range slot have none; an empty resolved line takes the authored one.
func TestSkillRaisedLineSelectsByClassAndSlot(t *testing.T) {
	authored := AuthoredWords()
	w := AuthoredWords()
	for i := range w.SkillRaised {
		w.SkillRaised[i] = string(rune('a' + i))
	}
	for _, mage := range []bool{false, true} {
		for slot := 1; slot <= 5; slot++ {
			want := slot - 1
			if mage {
				want += 5
			}
			if got, ok := w.SkillRaisedLine(mage, slot); !ok || got != w.SkillRaised[want] {
				t.Errorf("mage=%v slot %d = %q/%v, want line %d", mage, slot, got, ok, want)
			}
		}
		for _, slot := range []int{-1, 0, 6} {
			if got, ok := w.SkillRaisedLine(mage, slot); ok || got != "" {
				t.Errorf("mage=%v slot %d = %q/%v, want no line", mage, slot, got, ok)
			}
		}
	}
	w.SkillRaised[7] = ""
	if got, ok := w.SkillRaisedLine(true, 3); !ok || got != authored.SkillRaised[7] || got == "" {
		t.Errorf("empty resolved line = %q/%v, want the authored %q", got, ok, authored.SkillRaised[7])
	}
	for i, s := range authored.SkillRaised {
		if s == "" {
			t.Errorf("authored SkillRaised[%d] is empty", i)
		}
	}
}

// AC-5: the outcome sentence follows the ending, from whichever set is held.
func TestOutcomeTextFollowsTheWordSet(t *testing.T) {
	authored := AuthoredWords()
	if got := authored.OutcomeText(false); got != MissionWonText {
		t.Fatalf("authored won = %q", got)
	}
	if got := authored.OutcomeText(true); got != MissionLostText {
		t.Fatalf("authored lost = %q", got)
	}
	w := installedWords()
	if got := w.OutcomeText(false); got != "Mission Completed" {
		t.Fatalf("installed won = %q", got)
	}
	if got := w.OutcomeText(true); got != "Mission Failed" {
		t.Fatalf("installed lost = %q", got)
	}
}

// The dialogue/failure word and the two success-control words reach only their
// own controls; geometry and paint do not move.
func TestSetWordsRewritesOnlyTheButtonWord(t *testing.T) {
	v := &Viewer{
		noticeLayouts: [4]NoticeLayout{AuthoredDialogueLayout(), AuthoredOutcomeLayout(), AuthoredSuccessLayout(), AuthoredFailureLayout()},
		words:         AuthoredWords(),
	}
	before := v.noticeLayouts
	v.SetWords(installedWords())
	for i := 0; i < 2; i++ {
		if v.noticeLayouts[i].ButtonLabel != "Ok" {
			t.Errorf("layout %d button = %q, want Ok", i, v.noticeLayouts[i].ButtonLabel)
		}
		want := before[i]
		want.ButtonLabel = "Ok"
		if v.noticeLayouts[i] != want {
			t.Errorf("layout %d changed beyond its button word", i)
		}
	}
	success := before[NoticeSuccess]
	success.ButtonLabel = "~Victory!"
	success.SecondaryButtonLabel = "Continue"
	if v.noticeLayouts[NoticeSuccess] != success {
		t.Errorf("success layout changed beyond its two control words")
	}
	if v.Words() != installedWords() {
		t.Fatal("the viewer did not keep the set it was given")
	}
}

// The authored layouts state the authored word, so a viewer never told an
// install's words draws exactly what it drew before this story.
func TestAuthoredLayoutsStateTheAuthoredButtonWord(t *testing.T) {
	if got := AuthoredDialogueLayout().ButtonLabel; got != AuthoredNoticeButton {
		t.Errorf("dialogue button = %q, want %q", got, AuthoredNoticeButton)
	}
	if got := AuthoredOutcomeLayout().ButtonLabel; got != AuthoredNoticeButton {
		t.Errorf("outcome button = %q, want %q", got, AuthoredNoticeButton)
	}
	success := AuthoredSuccessLayout()
	if success.ButtonLabel != AuthoredWords().MenuVictory ||
		success.SecondaryButtonLabel != AuthoredWords().OutcomeContinue {
		t.Errorf("success controls = %q/%q", success.ButtonLabel, success.SecondaryButtonLabel)
	}
}

// OnLayout is a value and an empty word leaves the layout alone — the zero
// Words is what a front end assembled by hand in a test carries.
func TestOnLayoutIsAValueAndSkipsAnEmptyWord(t *testing.T) {
	base := AuthoredDialogueLayout()
	got := installedWords().OnLayout(base)
	if got.ButtonLabel != "Ok" {
		t.Fatalf("button = %q", got.ButtonLabel)
	}
	if base.ButtonLabel != AuthoredNoticeButton {
		t.Fatal("OnLayout mutated the layout it was handed")
	}
	if (Words{}).OnLayout(base).ButtonLabel != AuthoredNoticeButton {
		t.Fatal("an empty word blanked the button")
	}
}

// AC-7: the two menu surfaces' rows, in screen order, from the word set.
func TestGameMenuRowsComeFromTheWordSet(t *testing.T) {
	t.Run("authored", func(t *testing.T) {
		w := AuthoredWords()
		wantMission := []string{"SAVE GAME", "LOAD GAME", "GAME OPTIONS", "SOUND OPTIONS",
			"QUEST OBJECTIVES", "END QUEST", "RETURN TO GAME"}
		checkRows(t, missionGameMenuRows(w, true, true, true, true), wantMission)
		wantTown := []string{"SAVE GAME", "LOAD GAME", "GAME OPTIONS", "SOUND OPTIONS", "ABORT GAME", "RETURN TO GAME"}
		checkRows(t, townGameMenuRows(w), wantTown)
	})
	t.Run("installed", func(t *testing.T) {
		w := installedWords()
		wantMission := []string{"Save Game", "Load Game", "Game Options", "Sound Options",
			"Quest Objectives", "End Quest", "Return to Game"}
		checkRows(t, missionGameMenuRows(w, true, true, true, true), wantMission)
		wantTown := []string{"Save Game", "Load Game", "Game Options", "Sound Options", "Abort Game", "Return to Game"}
		checkRows(t, townGameMenuRows(w), wantTown)
	})
	// AC-8 at the row level: a set with one installed label and ten authored
	// others draws exactly one installed row.
	t.Run("per row", func(t *testing.T) {
		w := AuthoredWords()
		w.MenuEndQuest = "~End Quest"
		rows := missionGameMenuRows(w, true, true, true, true)
		if got := gameMenuLabelText(rows[5].Label); got != "End Quest" {
			t.Fatalf("row 5 = %q", got)
		}
		if got := gameMenuLabelText(rows[0].Label); got != "SAVE GAME" {
			t.Fatalf("row 0 = %q", got)
		}
	})
}

func checkRows(t *testing.T, rows []gameMenuRow, want []string) {
	t.Helper()
	if len(rows) != len(want) {
		t.Fatalf("%d rows, want %d", len(rows), len(want))
	}
	for i := range rows {
		if got := gameMenuLabelText(rows[i].Label); got != want[i] {
			t.Errorf("row %d = %q, want %q", i, got, want[i])
		}
	}
}

// AC-9: the accelerator walk runs over the DRAWN label whichever source it
// came from, including a label whose marked byte is not ASCII, and folds it
// through the same CP866 lowercase step the original applies under the
// install's Russian selector (MENU-KEY-013, closing 0168 SC-3 at 1014).
func TestAcceleratorOverAnInstalledLabel(t *testing.T) {
	// The RU root's own shape: the mark is not on the first byte, and the
	// marked byte is CP866 (MENU-KEY-013 measures exactly this on rows 5, 6
	// and 7 of both surfaces). The bytes are written as hex, never as literal
	// non-ASCII text. Byte 0xa0 already sits in the CP866 lowercase range
	// (0xa0..0xaf), so the fold leaves it unchanged — this is the exact
	// shipped case, and it does not exercise either fold clause.
	label := string([]byte{0x87, '~', 0xa0, 0xa4})
	if got := gameMenuAccelerator(label, 'E', text.SelectorConverting); got != 0xa0 {
		t.Fatalf("accelerator = %#x, want 0xa0", got)
	}
	if got := gameMenuAcceleratorColumn(label); got != 1 {
		t.Fatalf("column = %d, want 1", got)
	}
	if got := gameMenuLabelText(label); got != string([]byte{0x87, 0xa0, 0xa4}) {
		t.Fatalf("drawn label = % x", got)
	}

	// MENU-KEY-013's own two fold clauses, each on a byte no shipped label
	// marks but the decoded routine folds all the same: 0x91 (uppercase С, the
	// 0x90..0x9f block, +0x50) and 0x82 (uppercase В, the 0x80..0x8f block,
	// +0x20).
	if got := gameMenuAccelerator(string([]byte{'~', 0x91}), 'E', text.SelectorConverting); got != 0xe1 {
		t.Fatalf("0x91 folds to %#x, want 0xe1", got)
	}
	if got := gameMenuAccelerator(string([]byte{'~', 0x82}), 'E', text.SelectorConverting); got != 0xa2 {
		t.Fatalf("0x82 folds to %#x, want 0xa2", got)
	}

	// Under every OTHER selector — 0 included, which is what a caller with no
	// install font passes — the byte is returned unfolded, exactly as this
	// build behaved before 1014.
	if got := gameMenuAccelerator(string([]byte{'~', 0x91}), 'E', 0); got != 0x91 {
		t.Fatalf("selector 0 folded the byte to %#x, want 0x91 unfolded", got)
	}

	// The English shape is unchanged, at either selector.
	for _, sel := range []int{0, text.SelectorConverting} {
		if got := gameMenuAccelerator("~ABC", 'Z', sel); got != 'a' {
			t.Fatalf("ASCII accelerator at selector %d = %q", sel, got)
		}
	}
	if got := gameMenuAcceleratorColumn("~ABC"); got != 0 {
		t.Fatalf("ASCII column = %d", got)
	}
}

// TestGameMenuLowerFoldsExactlyMENUKEY013sTwoRanges pins the fold's own edges
// (MENU-KEY-013): +0x20 over 0x80..0x8f, +0x50 over 0x90..0x9f, nothing
// outside them, and only under the Russian selector.
func TestGameMenuLowerFoldsExactlyMENUKEY013sTwoRanges(t *testing.T) {
	for _, tc := range []struct{ in, want byte }{
		{0x80, 0xa0}, {0x8f, 0xaf}, // first range's own edges
		{0x90, 0xe0}, {0x9f, 0xef}, // second range's own edges
		{0x7f, 0x7f},               // just below the first range
		{0xa0, 0xa0}, {0xaf, 0xaf}, // already lowercase, first block
		{0xe0, 0xe0}, {0xef, 0xef}, // already lowercase, second block
	} {
		if got := gameMenuLower(tc.in, text.SelectorConverting); got != tc.want {
			t.Errorf("gameMenuLower(%#x, converting) = %#x, want %#x", tc.in, got, tc.want)
		}
		if got := gameMenuLower(tc.in, 0); got != tc.in {
			t.Errorf("gameMenuLower(%#x, 0) = %#x, want it unfolded", tc.in, got)
		}
	}
	// ASCII still folds at every selector, as lowerASCII always did.
	if got := gameMenuLower('R', text.SelectorConverting); got != 'r' {
		t.Errorf("gameMenuLower('R', converting) = %q, want 'r'", got)
	}
}

// AC-11: the category-(c) words are unchanged. The panel is the largest of
// those surfaces and its captions are the owner's authored arrangement
// (UNIT-PANEL-011); this pins them so that a later story cannot quietly bind one
// to an index that merely resembles it.
func TestAuthoredPanelCaptionsAreUnchanged(t *testing.T) {
	want := []string{
		"SELECTED", "", "BODY", "HEALTH", "AGILITY", "", "MIND", "MANA",
		"SPIRIT", "", "DMG", "ABSORB", "ATTACK", "DEFENSE", "SKILLS", "RESISTANCE",
		"BLADE", "FIRE", "AXE", "WATER", "BLUDGEON", "AIR", "PIKE", "EARTH",
		"SHOOTING", "ASTRAL", "GENERAL", "WEIGHT", "SPELLCASTER", "XP", "ARMOR PIERCING",
		"SIGHT", "SPEED", "WEAPON", "WORN", "SWING", "CELL",
	}
	got := []string{}
	for _, row := range AuthoredPanelLayout().Rows {
		got = append(got, row.Label)
		if row.Right != nil {
			got = append(got, row.Right.Label)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("%d captions %q, want %d", len(got), got, len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("caption %d = %q, want %q", i, got[i], want[i])
		}
	}
	named := 0
	for _, s := range got {
		if s != "" {
			named++
		}
	}
	if named != 34 {
		t.Fatalf("%d named captions, want 34 including both conditional install captions", named)
	}
}

// The words travel with the viewer on the one statement a map screen is
// entered on: a map opened AFTER SetWords, and a viewer already open when
// SetWords is called, hold the same set.
func TestWordsReachEveryViewerAndTheMenu(t *testing.T) {
	seams := &[]*mapSeam{}
	a := newTestApp(t, appRows(3), seamLoader(t, seams))
	if got := a.flow.words; got != AuthoredWords() {
		t.Fatal("a fresh flow did not start on the authored words")
	}
	a.SetWords(installedWords(), nil, nil)

	a.flow.screen = ScreenPicker
	a.step(appInput{Enter: true}, affAt)
	if a.Screen() != ScreenMap {
		t.Fatalf("setup: screen = %v", a.Screen())
	}
	v := a.flow.viewer
	if v.Words() != installedWords() {
		t.Fatal("the map this flow opened did not get the words")
	}
	if got := v.noticeLayouts[1].ButtonLabel; got != "Ok" {
		t.Fatalf("outcome button on the opened map = %q", got)
	}

	// A viewer already open when the call lands is written too, so the call
	// cannot leave half the application on the old set.
	a.SetWords(AuthoredWords(), nil, nil)
	if v.Words() != AuthoredWords() || v.noticeLayouts[0].ButtonLabel != AuthoredNoticeButton {
		t.Fatal("an open viewer kept the previous words")
	}

	font := gameMenuTestFont()
	encode := func(r rune) (byte, bool) { return byte(r), true }
	a.SetWords(installedWords(), font, encode)
	if a.flow.menuFont != font {
		t.Fatal("the menu font did not arrive with the words")
	}
	if a.flow.encodeMenuKey == nil {
		t.Fatal("the key encoder did not arrive with the words")
	}
	a.flow.menuSurface = gameMenuMission
	if got := gameMenuLabelText(a.flow.menuRows()[0].Label); got != "Save Game" {
		t.Fatalf("mission menu row 0 = %q", got)
	}
	a.flow.menuSurface = gameMenuTown
	if got := gameMenuLabelText(a.flow.menuRows()[4].Label); got != "Abort Game" {
		t.Fatalf("town menu row 4 = %q", got)
	}
}
