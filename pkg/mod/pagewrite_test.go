package mod

import (
	"strings"
	"testing"
)

const screensSource = "# keep me\n[[screen]]\nkey = \"about\"  # note\nkind = \"info\"\ntitle = \"t1\"\nmenu = \"m1\"\nplace = \"main\"\ntext = [\"p1\", \"p2\"]\n\n" +
	"[[screen]]\nkey = \"facts\"\nkind = \"table\"\ntitle = \"t1\"\nmenu = \"m2\"\nplace = \"game\"\nrows = [[\"a\", \"b\"], [\"c\", \"d\"]]\n\n" +
	"[[action]]\nkey = \"giveup\"\naction = \"abandon\"\nmenu = \"m3\"\n"

func readScreensData(t *testing.T, src string) Data {
	t.Helper()
	var d Data
	text := func(string) (string, bool) { return "x", true }
	if err := readScreens(DataReader{Mod: "m", Text: text}, &d, ScreensFile, []byte(src)); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestScreensWriterKeepsFileAndRoundTripsEdits(t *testing.T) {
	d := readScreensData(t, screensSource)
	if same, err := writeScreens(ScreensFile, []byte(screensSource), d); err != nil || string(same) != screensSource {
		t.Fatalf("an unchanged file is rewritten: %q, %v", same, err)
	}
	d.Screens.Screens[0].Game = true
	d.Screens.Screens[0].TextKeys = append(d.Screens.Screens[0].TextKeys, "p3")
	d.Screens.Screens = append(d.Screens.Screens[:1], d.Screens.Screens[2:]...)
	d.Screens.Screens = append(d.Screens.Screens, Screen{Key: "more", Kind: ScreenList, Main: true, TitleKey: "t", MenuKey: "m", ItemKeys: []string{"i"}})
	out, err := writeScreens(ScreensFile, []byte(screensSource), d)
	if err != nil || !strings.HasPrefix(string(out), "# keep me\n[[screen]]\nkey = \"about\"  # note\n") || strings.Contains(string(out), "facts") {
		t.Fatalf("%q, %v", out, err)
	}
	back := readScreensData(t, string(out))
	if len(back.Screens.Screens) != 3 || !back.Screens.Screens[0].Game || len(back.Screens.Screens[0].TextKeys) != 3 || back.Screens.Screens[2].Key != "more" {
		t.Fatalf("read back %+v", back.Screens.Screens)
	}
	d.Screens.Screens[0].Main, d.Screens.Screens[0].Game = false, false
	if _, err := writeScreens(ScreensFile, []byte(screensSource), d); err == nil {
		t.Fatal("a screen that opens from no menu was written")
	}
}

const settingsSource = "# knobs\n[speed]  # how fast\nlabel = \"Speed\"\ntype = \"int\"\ndefault = 3\nmin = 1\nmax = 9\n\n[hard]\nlabel = { en = \"Hard\", ru = \"Трудно\" }\ntype = \"bool\"\ndefault = false\n"

func TestSettingsWriterKeepsFileAndRoundTripsEdits(t *testing.T) {
	var d Data
	if err := readSettings(DataReader{}, &d, SettingsName, []byte(settingsSource)); err != nil {
		t.Fatal(err)
	}
	if same, err := writeSettings(SettingsName, []byte(settingsSource), d); err != nil || string(same) != settingsSource {
		t.Fatalf("an unchanged file is rewritten: %q, %v", same, err)
	}
	d.Settings.Settings[0].Default.Int = 5
	d.Settings.Settings[1].Labels["ru"] = "Сложно"
	d.Settings.Settings = append(d.Settings.Settings, Setting{Key: "mode", Labels: map[string]string{"en": "Mode"}, Kind: KindChoice,
		Default: Value{Kind: KindChoice, Str: "a"}, Choices: []string{"a", "b"}})
	out, err := writeSettings(SettingsName, []byte(settingsSource), d)
	if err != nil || !strings.HasPrefix(string(out), "# knobs\n[speed]  # how fast\n") {
		t.Fatalf("%q, %v", out, err)
	}
	back, err := ParseSettings(out)
	if err != nil || len(back) != 3 || back[0].Default.Int != 5 || back[1].Labels["ru"] != "Сложно" || back[2].Choices[1] != "b" {
		t.Fatalf("read back %+v, %v", back, err)
	}
	d.Settings.Settings = d.Settings.Settings[2:]
	out, err = writeSettings(SettingsName, out, d)
	if err != nil || strings.Contains(string(out), "[speed]") || strings.Contains(string(out), "[hard]") {
		t.Fatalf("removed: %q, %v", out, err)
	}
}

func TestSpeakersWriterKeepsOffRowsAndRenames(t *testing.T) {
	src := "# faces\n[[speaker]]\nid = 20000\nname = 'Karl'  # lord\npicture = \"a.png\"\n\n[[speaker]]\nid = 20001\noff = true\n"
	d, err := ParseSpeakers("m", SpeakersFile, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	data := Data{Speakers: d}
	if same, err := writeSpeakers(SpeakersFile, []byte(src), data); err != nil || string(same) != src {
		t.Fatalf("an unchanged file is rewritten: %q, %v", same, err)
	}
	data.Speakers.Speakers[0].Name = "Ann"
	data.Speakers.Speakers[0].Picture = "b.png"
	out, err := writeSpeakers(SpeakersFile, []byte(src), data)
	if err != nil || !strings.Contains(string(out), "name = \"Ann\"  # lord") || !strings.Contains(string(out), "off = true") || !strings.Contains(string(out), "b.png") {
		t.Fatalf("%q, %v", out, err)
	}
	data.Speakers.Speakers = data.Speakers.Speakers[1:]
	out, err = writeSpeakers(SpeakersFile, out, data)
	if err != nil || strings.Contains(string(out), "20000") || !strings.Contains(string(out), "# faces") {
		t.Fatalf("removed: %q, %v", out, err)
	}
}
