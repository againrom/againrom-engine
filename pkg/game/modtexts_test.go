package game

import (
	"againrom/internal/synth"
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/mod"
)

func TestEncodeInstallTextFileWritesTheInstallsBytes(t *testing.T) {
	const utf = "\xef\xbb\xbfПривет, \u00b0 caf\u00e9\r\nTab\there\r\n"
	ru := EncodeInstallTextFile([]byte(utf), 1)
	// "Привет" is 8F E0 A8 A2 A5 E2 in the DOS layout; the accented e has no
	// byte and is dropped, the line ends and the tab stay.
	if !bytes.HasPrefix(ru, []byte("\x8f\xe0\xa8\xa2\xa5\xe2, ")) {
		t.Errorf("selector 1: % x", ru)
	}
	if !bytes.HasSuffix(ru, []byte(" caf\r\nTab\there\r\n")) {
		t.Errorf("selector 1 tail: %q", ru)
	}
	en := EncodeInstallTextFile([]byte(utf), 0)
	if !bytes.HasPrefix(en, []byte("\xcf\xf0\xe8\xe2\xe5\xf2, \xb0 caf")) || !bytes.HasSuffix(en, []byte("\r\nTab\there\r\n")) {
		t.Errorf("selector 0: % x", en)
	}
}

func TestEncodeInstallTextFileServesAFileThatIsNotUTF8Unchanged(t *testing.T) {
	legacy := []byte("\x8f\xe0\xa8\xa2\xa5\xe2 \xe9\r\n")
	for _, selector := range []int{0, 1} {
		got := EncodeInstallTextFile(legacy, selector)
		if !bytes.Equal(got, legacy) {
			t.Errorf("selector %d: % x, want the file unchanged", selector, got)
		}
		got[0] ^= 0xff
		if legacy[0] != 0x8f {
			t.Fatal("the served bytes alias the file")
		}
	}
	ascii := []byte("Plain\r\n")
	if got := EncodeInstallTextFile(ascii, 1); !bytes.Equal(got, ascii) {
		t.Errorf("ASCII: %q", got)
	}
}

func TestDecodeInstallTextFileRefusesWhatItCannotGiveBack(t *testing.T) {
	if got, ok := DecodeInstallTextFile([]byte("\x8f\xe0\r\n"), 1); !ok || string(got) != "Пр\r\n" {
		t.Errorf("%q %v", got, ok)
	}
	if _, ok := DecodeInstallTextFile([]byte("a\x98b"), 0); ok {
		t.Error("an unassigned byte decoded")
	}
}

func TestSetModMissionsEncodesTextFilesTitleAndBriefingOnce(t *testing.T) {
	f := guestFixtureFront(t)
	m := guestDecl()
	m.Title, m.Brief = "Привет", "Бой"
	m.TextFiles = map[string][]byte{
		"event01.txt":  []byte("Привет\r\n"),
		"briefmap.txt": []byte("Карта\r\n"),
		"event02.txt":  []byte("\x8f\xe0\r\n"),
		"tips01.txt":   []byte("Tip\r\n"),
	}
	if _, err := f.SetModMissions(mod.MissionData{Missions: []mod.Mission{m}}); err != nil {
		t.Fatal(err)
	}
	selector := LanguageSelector(f.Archives.Containers)
	for name, want := range map[string][]byte{
		"event01.txt":  EncodeInstallTextFile([]byte("Привет\r\n"), selector),
		"briefmap.txt": EncodeInstallTextFile([]byte("Карта\r\n"), selector),
		"event02.txt":  []byte("\x8f\xe0\r\n"),
		"tips01.txt":   []byte("Tip\r\n"),
		"title.txt":    []byte(EncodeInstallText("Привет", selector)),
		"briefing.txt": []byte(EncodeInstallText("Бой", selector)),
	} {
		got, err := f.Archives.Containers.ReadFile("main/text/battle/m25/" + name)
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("%s: % x %v, want % x", name, got, err, want)
		}
	}
	if len(EncodeInstallTextFile([]byte("Привет"), selector)) != 6 {
		t.Error("the Cyrillic file did not encode to six bytes")
	}
}

// messageScript is a type-7 payload with one trigger that sends the messages
// numbered by events, one send-message action each.
func messageScript(events ...uint32) []byte {
	var actions []scriptNode
	for i, e := range events {
		actions = append(actions, scriptNode{label: "message", opcode: 2, id: uint32(10 + i), values: []uint32{e}, tags: []uint32{1}})
	}
	b := scriptPayload(actions, []scriptNode{{label: "always", opcode: 1, id: 1}})
	b = b[:len(b)-4]
	b = binary.LittleEndian.AppendUint32(b, 1)
	rec := make([]byte, 184)
	binary.LittleEndian.PutUint32(rec[0x80:], 1)
	binary.LittleEndian.PutUint32(rec[0x84:], 1)
	for i := range events {
		if i < 4 {
			binary.LittleEndian.PutUint32(rec[0x98+4*i:], uint32(10+i))
		}
	}
	return append(b, rec...)
}

func TestMessageScriptRaisesItsEvents(t *testing.T) {
	m := built(t, withScript(cleanOptions(), messageScript(3, 7)), nil, nil)
	got, err := m.Map()
	if err != nil {
		t.Fatal(err)
	}
	_, rep, err := tableFiles(checkInspector().table).compileScript(got, campaignScriptRefs(got, checkInspector().table, nil, 0))
	if err != nil || len(rep.Raises) != 2 || rep.Raises[0].Event != 3 || rep.Raises[1].Event != 7 {
		t.Fatalf("raises %+v %v", rep.Raises, err)
	}
}

func withScript(o synth.ALMOptions, payload []byte) synth.ALMOptions {
	o.Type7Payload = payload
	return o
}

func TestMapRuleModMessageTextNamesAMessageTheModGivesNoPage(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "t"), 0o755); err != nil {
		t.Fatal(err)
	}
	host := filepath.Join(dir, "map.alm")
	for name, body := range map[string]string{"map.alm": "x", "t/event03.txt": "<part=1>\r\nHi\r\n"} {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := built(t, withScript(cleanOptions(), messageScript(3, 7, 255, 7)), nil, nil)
	x := checkInspector()
	if rows := ofRule(findings(t, x, m, host), "mod-message-text"); len(rows) != 2 || !strings.Contains(rows[0].Text, "map ") {
		t.Fatalf("loose map without pages: %+v", rows)
	}
	x.mod = &editorMod{dir: dir, id: "m", missions: mod.MissionData{Missions: []mod.Mission{
		{Mod: "m", Key: "k", Number: 25, Map: "map.alm", Texts: "t"}}}}
	rows := ofRule(findings(t, x, m, host), "mod-message-text")
	if len(rows) != 1 || rows[0].Error || !strings.Contains(rows[0].Text, "message 7") || !strings.Contains(rows[0].Text, "event07.txt") || !strings.Contains(rows[0].Text, `mod "m"`) {
		t.Fatalf("one warning for message 7 expected, got %+v", rows)
	}
	// An unbound campaign map uses its own page source.
	if rows := ofRule(findings(t, x, m, "scenario/9.alm"), "mod-message-text"); len(rows) != 2 {
		t.Fatalf("campaign map without pages: %+v", rows)
	}
	// A mission with no texts folder gives no page for any message.
	x.mod.missions.Missions[0].Texts = ""
	if rows := ofRule(findings(t, x, m, host), "mod-message-text"); len(rows) != 2 {
		t.Fatalf("no texts folder: %+v", rows)
	}
	if err := os.WriteFile(filepath.Join(dir, "t", "event07.txt"), []byte("<part=1>\r\nHi\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	x.mod.missions.Missions[0].Texts = "t"
	if rows := ofRule(findings(t, x, m, host), "mod-message-text"); len(rows) != 0 {
		t.Fatalf("both pages given: %+v", rows)
	}
}
