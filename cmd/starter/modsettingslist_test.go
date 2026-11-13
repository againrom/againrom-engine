package main

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/witness"
	basepkg "againrom/pkg/base"
	"againrom/pkg/installinfo"
	"againrom/pkg/mod"
)

// manySettings declares twelve settings, four of each type, so the panel holds
// more than settingRows of them.
const manySettings = `[gold]
label = { en = "Gold", ru = "Золото" }
type = "int"
default = 100
min = 0
max = 1000

[hard]
label = { en = "Hard fights", ru = "Тяжелые бои" }
type = "bool"
default = false

[mood]
label = { en = "Mood", ru = "Настроение" }
type = "choice"
choices = ["calm", "grim", "wild"]
default = "calm"

[speed]
label = { en = "Speed", ru = "Скорость" }
type = "int"
default = 5
min = 1
max = 10

[fog]
label = { en = "Fog", ru = "Туман" }
type = "bool"
default = true

[weather]
label = { en = "Weather", ru = "Погода" }
type = "choice"
choices = ["clear", "rain", "snow"]
default = "clear"

[lives]
label = { en = "Lives", ru = "Жизни" }
type = "int"
default = 3
min = 1
max = 9

[night]
label = { en = "Night", ru = "Ночь" }
type = "bool"
default = false

[music]
label = { en = "Music", ru = "Музыка" }
type = "choice"
choices = ["a", "b"]
default = "a"

[damage]
label = { en = "Damage", ru = "Урон" }
type = "int"
default = 0
min = -50
max = 50

[heal]
label = "Healing"
type = "bool"
default = true

[tier]
label = { en = "Tier" }
type = "choice"
choices = ["low", "mid", "high"]
default = "mid"
`

// manyRig is a starter folder with one base described by info and the mod
// "many" enabled, which declares manySettings.
func manyRig(t *testing.T, info installinfo.Info, extraIni string) (*rig, *app) {
	t.Helper()
	r := newRig(t, "[starter]\nlast-base = en\n[bases]\nen = {root}\n[mods]\nenabled = many\n"+extraIni)
	r.fakes.validRoots[r.root] = info
	mods := filepath.Join(r.dir, "mods")
	writeModFolder(t, mods, "many", goodMod("many"))
	if err := os.WriteFile(filepath.Join(mods, "many", "settings.toml"), []byte(manySettings), 0o644); err != nil {
		t.Fatal(err)
	}
	return r, r.app()
}

func manyDecl(t *testing.T) []mod.Setting {
	t.Helper()
	decl, err := mod.ParseSettings([]byte(manySettings))
	if err != nil || len(decl) != 12 {
		t.Fatalf("%d settings, %v", len(decl), err)
	}
	return decl
}

// panelPoint is a point over the settings panel's rows.
var panelPoint = image.Pt(100, panelTop+30)

// panelItems are the layout items inside the settings panel.
func panelItems(a *app) []item {
	var out []item
	for _, it := range a.layout() {
		if it.rect.Min.Y >= panelTop-3 && it.rect.Min.Y < 410+panelH {
			out = append(out, it)
		}
	}
	return out
}

func panelHas(a *app, text string, tn tone) bool {
	for _, it := range panelItems(a) {
		if it.text == text && it.tone == tn {
			return true
		}
	}
	return false
}

func TestEverySettingOfAModIsReachableByTheWheel(t *testing.T) {
	decl := manyDecl(t)
	for _, tc := range []struct {
		info     installinfo.Info
		lang     string
		top, end string
	}{
		{installinfo.Info{Language: "english"}, "en", "1-5 of 12, the wheel scrolls", "8-12 of 12, the wheel scrolls"},
		{installinfo.Info{Language: "russian"}, "ru", "1-5 из 12, листает колесо", "8-12 из 12, листает колесо"},
	} {
		_, a := manyRig(t, tc.info, "")
		seen := map[string]bool{}
		look := func() {
			for _, it := range panelItems(a) {
				seen[it.text] = true
			}
		}
		look()
		if !panelHas(a, tc.top, toneDim) {
			t.Fatalf("%s: no window line %q in %q", tc.lang, tc.top, texts(a))
		}
		for n := 0; n < 7; n++ {
			a.wheel(panelPoint, 1)
			look()
		}
		if !panelHas(a, tc.end, toneDim) {
			t.Fatalf("%s: no window line %q in %q", tc.lang, tc.end, texts(a))
		}
		a.wheel(panelPoint, 50)
		if !panelHas(a, tc.end, toneDim) {
			t.Fatalf("%s: the panel scrolled past its last setting", tc.lang)
		}
		for _, st := range decl {
			label := st.Label(tc.lang)
			if st.Kind == mod.KindInt {
				label = fmt.Sprintf("%s (%d..%d)", label, st.Min, st.Max)
			}
			if !seen[label] {
				t.Errorf("%s: setting %s (%q) never came into view", tc.lang, st.Key, label)
			}
		}
		a.wheel(panelPoint, -50)
		if !panelHas(a, tc.top, toneDim) {
			t.Fatalf("%s: the panel did not scroll back", tc.lang)
		}
	}
}

func TestEverySettingIsEditedStoredAndPassed(t *testing.T) {
	r, a := manyRig(t, installinfo.Info{Language: "english"}, "[mod.many]\nlives=4\n")
	typeInto(t, a, fSetting0, "")
	a.backspace()
	a.backspace()
	a.typed("50") // gold 100 -> 1 -> 150
	do(t, a, action{aSettingCycle, 1})
	do(t, a, action{aSettingCycle, 2})
	do(t, a, action{aSettingUp, 3})
	do(t, a, action{aSettingCycle, 4})
	a.wheel(panelPoint, 3)
	do(t, a, action{aSettingCycle, 5})
	do(t, a, action{aSettingCycle, 7})
	a.wheel(panelPoint, 4)
	do(t, a, action{aSettingCycle, 8})
	// damage is the row 2 in view of rows 7 to 11.
	if !a.hasField(fSetting0 + 2) {
		t.Fatal("the scrolled integer setting has no field")
	}
	typeInto(t, a, fSetting0+2, "")
	a.backspace()
	a.typed("-12")
	do(t, a, action{aSettingCycle, 10})
	if _, ok := a.s.modValue("many", "tier"); ok {
		t.Fatal("an untouched setting got a stored value")
	}
	a.save()
	data, err := os.ReadFile(r.ini)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, want := range []string{"\nlives=4\n", "gold = 150\n", "hard = true\n", "mood = grim\n", "speed = 6\n", "fog = false\n",
		"weather = rain\n", "night = true\n", "music = b\n", "damage = -12\n", "heal = false\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("starter.ini lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "tier") || strings.Contains(got, "lives = 4") {
		t.Errorf("an untouched setting changed in starter.ini:\n%s", got)
	}
	a.play()
	want := []string{filepath.Join(r.dir, "againrom.exe"), "-assets", r.root, "-mods", "many", "-mods-dir", filepath.Join(r.dir, "mods")}
	for _, kv := range []string{"gold=150", "hard=true", "mood=grim", "speed=6", "fog=false", "weather=rain", "lives=4", "night=true", "music=b", "damage=-12", "heal=false"} {
		want = append(want, "-mod-setting", "many."+kv)
	}
	if len(r.fakes.started) != 1 || !reflect.DeepEqual(r.fakes.started[0], want) {
		t.Fatalf("started %q\nwant    %q (status %q)", r.fakes.started, want, a.status)
	}
	// The stored values read back into the panel.
	b := r.app()
	b.wheel(panelPoint, 7)
	if !panelHas(b, "-12", toneNormal) {
		t.Fatalf("the panel does not show the stored value: %q", texts(b))
	}
}

func TestFocusOrTabWithoutTypingStoresNothing(t *testing.T) {
	r, a := manyRig(t, installinfo.Info{Language: "english"}, "")
	do(t, a, action{aFocus, int(fSetting0)})
	a.typed("")
	a.focus = fAgainrom
	for n := 0; n < int(fieldCount); n++ {
		a.tab()
	}
	a.save()
	data, _ := os.ReadFile(r.ini)
	if strings.Contains(string(data), "[mod.many]") || len(a.s.ModSettings) != 0 {
		t.Fatalf("a setting nobody typed into was stored:\n%s", data)
	}
}

func TestTheWheelOverThePanelDropsASettingFocus(t *testing.T) {
	_, a := manyRig(t, installinfo.Info{Language: "english"}, "")
	typeInto(t, a, fSetting0, "")
	a.wheel(panelPoint, 1)
	if a.focus != fNone {
		t.Fatalf("focus %v stayed on a row that now shows another setting", a.focus)
	}
	typeInto(t, a, fSetting0+2, "") // speed, declaration 3, now the row 2 in view
	a.typed("0")
	if v, _ := a.s.modValue("many", "speed"); v != "50" {
		t.Fatalf("speed %q", v)
	}
}

func TestOutOfBoundsValuesAreRefusedWithTheLoadersWords(t *testing.T) {
	r, a := manyRig(t, installinfo.Info{Language: "english"}, "[mod.many]\nnight = maybe\ntier = top\n")
	decl := manyDecl(t)
	typeInto(t, a, fSetting0, "1") // gold 1001
	rowWords := "1001 is outside 0..1000"
	if !panelHas(a, rowWords, toneBad) {
		t.Fatalf("the gold row does not refuse 1001: %q", texts(a))
	}
	if _, err := mod.ResolveSettings("many", decl, map[string]string{"gold": "1001"}); err == nil || !strings.HasSuffix(err.Error(), ": "+rowWords) {
		t.Fatalf("the loader says %v", err)
	}
	a.play()
	if !a.statusBad || !strings.Contains(a.status, `mod "many" setting gold: `+rowWords) || len(r.fakes.started) != 0 {
		t.Fatalf("status %q started %q", a.status, r.fakes.started)
	}
	do(t, a, action{aSettingUp, 0})
	do(t, a, action{aSettingUp, 0})
	if v, _ := a.s.modValue("many", "gold"); v != "1000" || panelHas(a, rowWords, toneBad) {
		t.Fatalf("up from 1001 gave %q", v)
	}
	for n := 0; n < 6; n++ {
		do(t, a, action{aSettingDown, 3})
	}
	if v, _ := a.s.modValue("many", "speed"); v != "1" {
		t.Fatalf("down from 5 six times gave %q, want the minimum 1", v)
	}
	do(t, a, action{aFocus, int(fSetting0) + 3})
	a.typed("x")
	if !panelHas(a, `"1x" is not an integer`, toneBad) {
		t.Fatalf("no refusal of 1x: %q", texts(a))
	}
	do(t, a, action{aSettingUp, 3})
	if v, _ := a.s.modValue("many", "speed"); v != "6" {
		t.Fatalf("up from a non-integer gave %q, want the default 5 plus one", v)
	}
	a.wheel(panelPoint, 7)
	for _, words := range []string{`"maybe" is not true or false`, `"top" is not one of low, mid, high`} {
		if !panelHas(a, words, toneBad) {
			t.Errorf("no refusal %q: %q", words, texts(a))
		}
	}
}

// TestThePanelIsTheSameOnBothGames compares the panel on a first-game and a
// second-game base of each language, at the top and scrolled to the end. With
// AGAINROM_STARTER_MOD_FRAMES set the frames are written there as PNGs.
func TestThePanelIsTheSameOnBothGames(t *testing.T) {
	dir := os.Getenv("AGAINROM_STARTER_MOD_FRAMES")
	for _, pair := range [][2]string{{basepkg.ROM1EN, basepkg.ROM2EN}, {basepkg.ROM1RU, basepkg.ROM2RU}} {
		var panels [2][2][]item
		for g, id := range pair {
			_, a := manyRig(t, profileInfo(id, true), "")
			for s, at := range []string{"top", "end"} {
				if at == "end" {
					a.wheel(panelPoint, 7)
				}
				panels[g][s] = panelItems(a)
				if dir != "" {
					witness.PNG(t, filepath.Join(dir, fmt.Sprintf("starter-settings-%s-%s.png", id, at)), a.render())
				}
			}
		}
		for s := range 2 {
			if !reflect.DeepEqual(panels[0][s], panels[1][s]) {
				t.Errorf("%s and %s draw different panels:\n%v\n%v", pair[0], pair[1], panels[0][s], panels[1][s])
			}
		}
	}
}

// TestTheListsKeepTheirRows pins the rows and wheel areas the base and mod
// lists had before the settings panel became the third scrolling list.
func TestTheListsKeepTheirRows(t *testing.T) {
	for k := range baseRows {
		if got, want := scrollLists[listBases].row(k), image.Rect(10, 66+18*k, winW-10, 84+18*k); got != want {
			t.Errorf("base row %d %v, want %v", k, got, want)
		}
	}
	for k := range modRows {
		if got, want := scrollLists[listMods].row(k), image.Rect(10, 250+18*k, winW-10, 268+18*k); got != want {
			t.Errorf("mod row %d %v, want %v", k, got, want)
		}
	}
	if scrollLists[listBases].area != image.Rect(10, 66, winW-10, 156) || scrollLists[listMods].area != image.Rect(10, 250, winW-10, 360) {
		t.Errorf("wheel areas %v %v", scrollLists[listBases].area, scrollLists[listMods].area)
	}
	_, a := manyRig(t, installinfo.Info{Language: "english"}, "")
	a.wheel(panelPoint, 3)
	if a.scroll != [listCount]int{0, 0, 3} {
		t.Fatalf("scroll %v: the wheel over the panel moved another list", a.scroll)
	}
	a.wheel(image.Pt(100, 300), 2)
	if a.scroll != [listCount]int{0, 2, 3} {
		t.Fatalf("scroll %v: the wheel over the mod list moved another list", a.scroll)
	}
}

func TestPanelCaptionsFit(t *testing.T) {
	for _, language := range []string{"english", "russian"} {
		_, a := manyRig(t, installinfo.Info{Language: language}, "")
		for _, it := range panelItems(a) {
			if it.kind == iText && textWidth(it.text) > it.rect.Dx() {
				t.Errorf("%s: %q is %d px wide in %d", language, it.text, textWidth(it.text), it.rect.Dx())
			}
		}
	}
}
