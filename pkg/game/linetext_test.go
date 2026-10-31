package game

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/locale"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// lineTextFixtures are the synthetic files every row is read over: missing,
// empty, with and without a final newline, blank, sectioned, Cyrillic in
// both code pages, bytes that are not UTF-8, oversized and over-long.
func lineTextFixtures() map[string][]byte {
	sections := "#briefing\r\n Find the gate. \r\n#event1\r\n<part=1>\r\nHello.\r\n" +
		"#subobjective0\r\n One \r\n#subobjective1\r\nTwo\r\n#failure2\r\nLost\r\n" +
		"#title21\r\nThe Ford\x00\x00\r\n#npc1talk1\r\nWelcome\r\n"
	return map[string][]byte{
		"missing":          nil,
		"empty":            {},
		"final newline":    []byte("first\r\nsecond\r\n"),
		"no final newline": []byte("first\r\nsecond"),
		"lf only":          []byte("first\nsecond\n"),
		"bare cr":          []byte("A\rXB\r\nC\r\n"),
		"blank":            []byte(" \r\n\t\r\n"),
		"sections":         []byte(sections),
		"cyrillic":         {0xcf, 0xf0, 0xe8, 0xe2, 0xe5, 0xf2, '\r', '\n', 0x8f, 0xe0, 0xa8, '\r', '\n', '#', 'e', 'v', 'e', 'n', 't', '1', '\r', '\n', 0xc0, 0xe1},
		"not utf-8":        {0xff, 0xfe, 'a', '\r', '\n'},
		"oversized":        bytes.Repeat([]byte("a\r\n"), 256*1024/3+1),
		"over-long":        bytes.Repeat([]byte("x\r\n"), 5000),
	}
}

var lineTextCodes = []TextCode{
	{},
	{From: locale.DOSCyrillic, To: locale.DOSCyrillic},
	{From: locale.WindowsCyrillic, To: locale.DOSCyrillic, Selector: 1},
}

// lineTextRow is one source row read through its reader, against the
// reader's own code as it stood before the row (want).
type lineTextRow struct {
	name      string
	address   string
	got, want func(src entrySource, held []byte, code TextCode) any
}

func lineTextRows() []lineTextRow {
	const (
		named    = "main/text/named.txt"
		brief    = "main/text/battle/m10/briefing.txt"
		event    = "main/text/battle/m10/event01.txt"
		title    = "main/text/battle/m21/title.txt"
		briefMap = "main/text/battle/m21/briefmap.txt"
	)
	return []lineTextRow{
		{"text table", named,
			func(src entrySource, _ []byte, code TextCode) any { return LoadTextTable(src, named, code) },
			func(src entrySource, _ []byte, code TextCode) any {
				b, err := src.ReadFile(named)
				if err != nil {
					return (*TextTable)(nil)
				}
				return SplitTextTable(code.Bytes(b))
			}},
		{"text table read error", named,
			func(src entrySource, _ []byte, code TextCode) any {
				return fmt.Sprint(installText(src, textTableSource.named(named), code).err())
			},
			func(src entrySource, _ []byte, code TextCode) any {
				_, err := src.ReadFile(named)
				return fmt.Sprint(err)
			}},
		{"help", HelpTextPath,
			func(src entrySource, _ []byte, code TextCode) any { return LoadInstallWords(src, code).help },
			func(src entrySource, _ []byte, code TextCode) any {
				if b, err := src.ReadFile(HelpTextPath); err == nil {
					return string(code.Bytes(b))
				}
				return ""
			}},
		{"tip", named,
			func(src entrySource, _ []byte, code TextCode) any {
				s, ok := ReadShopTip(src, named, code)
				return fmt.Sprintf("%q %v", s, ok)
			},
			func(src entrySource, _ []byte, code TextCode) any {
				b, err := src.ReadFile(named)
				if err != nil {
					return fmt.Sprintf("%q %v", "", false)
				}
				return fmt.Sprintf("%q %v", string(code.Bytes(b)), true)
			}},
		{"tip section", named,
			func(src entrySource, _ []byte, code TextCode) any {
				return readTipText(src, ui.TipText{File: named, Section: "npc1talk1"}, code)
			},
			func(src entrySource, _ []byte, code TextCode) any {
				b, err := src.ReadFile(named)
				if err != nil {
					return ""
				}
				text := string(code.Bytes(b))
				if text == "" {
					return text
				}
				return secondGameTextSection([]byte(text), "npc1talk1")
			}},
		{"briefing", brief,
			func(src entrySource, _ []byte, _ TextCode) any {
				return MissionObjectives(src, 10) + "|" + MissionObjectives(src, 0)
			},
			func(src entrySource, _ []byte, _ TextCode) any {
				raw, err := src.ReadFile(brief)
				if err != nil {
					return "|"
				}
				return strings.TrimSpace(vfs.DecodeText(raw)) + "|"
			}},
		{"event text", event,
			func(src entrySource, _ []byte, _ TextCode) any {
				b, ok := ReadEventText(src, 10, 1)
				_, zero := ReadEventText(src, 10, 0)
				return []any{b, ok, zero}
			},
			func(src entrySource, _ []byte, _ TextCode) any {
				b, err := src.ReadFile(event)
				if err != nil {
					return []any{[]byte(nil), false, false}
				}
				return []any{b, true, false}
			}},
		{"mission text", "main/text/mission10.txt",
			func(src entrySource, _ []byte, code TextCode) any {
				payload, ok := readSecondGameMissionText(src, code, 10)
				ev, evOK := readSecondGameEventText(src, code, 10, 1)
				labels, slots := secondGameFailureEntries(src, code, 10)
				return []any{payload, ok, ev, evOK, secondGameBriefing(src, base.GameROM2.Edition(), 10),
					secondGameObjectiveLabels(src, code, 10), labels, slots}
			},
			func(src entrySource, _ []byte, code TextCode) any {
				raw, err := src.ReadFile("main/text/mission10.txt")
				if err != nil {
					return []any{[]byte(nil), false, []byte(nil), false, "", []string(nil), []string(nil), []int32(nil)}
				}
				payload := code.Bytes(raw)
				ev := secondGameTextSection(payload, "event1")
				briefRaw := InstallTextCode(src, base.GameROM2.Edition()).Bytes(raw)
				briefing := strings.TrimSpace(vfs.DecodeText([]byte(secondGameTextSection(briefRaw, "briefing"))))
				var labels []string
				for i := 0; i < 1024-752; i++ {
					label := secondGameTextSection(payload, fmt.Sprintf("subobjective%d", i))
					if label == "" {
						break
					}
					labels = append(labels, strings.TrimSpace(vfs.DecodeText([]byte(label))))
				}
				words := LoadInstallWords(src, code)
				var failures []string
				var slots []int32
				for reason := 2; reason < 1024; reason++ {
					label := secondGameTextSection(payload, fmt.Sprintf("failure%d", reason))
					if label == "" {
						if reason > 4 {
							break
						}
						label, _ = words.Global(0x118 + reason)
						slots = append(slots, int32(reason))
					}
					failures = append(failures, label)
				}
				return []any{payload, true, []byte(ev), ev != "", briefing, labels, failures, slots}
			}},
		{"town text", secondTownText,
			func(src entrySource, _ []byte, code TextCode) any {
				town := installText(src, townTextSource.at(), code)
				return []any{fmt.Sprint(town.err()), town.section("npc1talk1")}
			},
			func(src entrySource, _ []byte, code TextCode) any {
				payload, err := src.ReadFile(secondTownText)
				if err != nil {
					return []any{fmt.Sprint(err), ""}
				}
				return []any{"<nil>", secondGameTextSection(code.Bytes(payload), "npc1talk1")}
			}},
		{"town dialogue", named,
			func(src entrySource, _ []byte, _ TextCode) any {
				return installText(src, townDialogueSource.named(named), TextCode{}).bytes()
			},
			func(src entrySource, _ []byte, _ TextCode) any {
				payload, _ := src.ReadFile(named)
				return payload
			}},
		{"world map title and briefing", title,
			func(src entrySource, _ []byte, _ TextCode) any {
				a := &worldMapAssets{texts: map[int]worldMapText{}, reads: map[string]int{}}
				return []any{a.missionText(src, 21), a.reads}
			},
			func(src entrySource, _ []byte, _ TextCode) any {
				reads := map[string]int{}
				read := func(p string) ([]byte, error) { reads[p]++; return src.ReadFile(p) }
				t := worldMapText{title: "Mission 21", brief: "Briefing unavailable"}
				if raw, err := read(title); err == nil && len(strings.TrimSpace(string(raw))) > 0 {
					t.title = string(raw)
				}
				if raw, err := read(briefMap); err == nil && len(strings.TrimSpace(string(raw))) > 0 {
					t.brief = string(raw)
				}
				return []any{t, reads}
			}},
		{"world map brief map", briefMap,
			func(src entrySource, _ []byte, _ TextCode) any {
				a := &worldMapAssets{texts: map[int]worldMapText{}, reads: map[string]int{}}
				return a.missionText(src, 21)
			},
			func(src entrySource, _ []byte, _ TextCode) any {
				t := worldMapText{title: "Mission 21", brief: "Briefing unavailable"}
				if raw, err := src.ReadFile(briefMap); err == nil && len(strings.TrimSpace(string(raw))) > 0 {
					t.brief = string(raw)
				}
				return t
			}},
		{"editor title", title,
			func(src entrySource, _ []byte, code TextCode) any {
				return installText(src, editorTitleSource.at(21), code).text()
			},
			func(src entrySource, _ []byte, code TextCode) any {
				raw, err := src.ReadFile(title)
				if err != nil || len(strings.TrimSpace(string(raw))) == 0 {
					return ""
				}
				return strings.TrimSpace(baseMessageUTF8(raw, code.Selector))
			}},
		{"world map sweep page", title,
			func(src entrySource, _ []byte, _ TextCode) any {
				return installText(src, worldMapPageSource.named(title), TextCode{}).present()
			},
			func(src entrySource, _ []byte, _ TextCode) any {
				raw, err := src.ReadFile(title)
				return !(err != nil || len(raw) == 0)
			}},
		{"world map records", named,
			func(src entrySource, _ []byte, code TextCode) any {
				if r := installText(src, worldMapRecordSource.named(named), code); r.present() {
					return string(r.bytes())
				}
				return ""
			},
			func(src entrySource, _ []byte, code TextCode) any {
				if raw, err := src.ReadFile(named); err == nil {
					return string(code.Bytes(raw))
				}
				return ""
			}},
		{"held sections", "",
			func(_ entrySource, held []byte, _ TextCode) any {
				return installText(nil, heldSectionSource.holding(held), TextCode{}).sectionCut("title21", 2)
			},
			func(_ entrySource, held []byte, _ TextCode) any {
				body := secondGameTextSection(held, "title21")
				if len(body) < 2 {
					return ""
				}
				return strings.TrimRight(body[:len(body)-2], "\x00")
			}},
		{"mission page", named,
			func(src entrySource, _ []byte, _ TextCode) any {
				return installText(src, missionPageSource.named(named), TextCode{}).present()
			},
			func(src entrySource, _ []byte, _ TextCode) any {
				_, err := src.ReadFile(named)
				return err == nil
			}},
		{"installed page", named,
			func(src entrySource, _ []byte, code TextCode) any {
				page := installText(src, installedPageSource.named(named), code)
				return []any{fmt.Sprint(page.err()), page.bytes()}
			},
			func(src entrySource, _ []byte, code TextCode) any {
				b, err := src.ReadFile(named)
				if err != nil {
					return []any{fmt.Sprint(err), []byte(nil)}
				}
				if u, ok := DecodeInstallTextFile(b, code.Selector); ok {
					b = u
				}
				return []any{"<nil>", b}
			}},
		{"editor page", "",
			func(_ entrySource, held []byte, code TextCode) any {
				return installText(nil, editorPageSource.holding(held), code).text()
			},
			func(_ entrySource, held []byte, code TextCode) any { return baseMessageUTF8(held, code.Selector) }},
		{"report part", "",
			func(_ entrySource, held []byte, _ TextCode) any { return decodeInstallText(string(held)) },
			func(_ entrySource, held []byte, _ TextCode) any { return vfs.DecodeText(held) }},
		{"credits roll", "main/text/credits.txt",
			func(src entrySource, _ []byte, code TextCode) any {
				if c := installText(src, creditsRollSource.at(), code); c.present() {
					return c.lines()
				}
				return []string(nil)
			},
			func(src entrySource, _ []byte, code TextCode) any {
				var lines []string
				if raw, err := src.ReadFile("main/text/credits.txt"); err == nil && len(raw) <= 256*1024 {
					lines = strings.Split(strings.TrimSuffix(strings.ReplaceAll(string(code.Bytes(raw)), "\r\n", "\n"), "\n"), "\n")
					if len(lines) > 4096 {
						lines = lines[:4096]
					}
				}
				return lines
			}},
		{"hall credits", "main/text/credits.txt",
			func(src entrySource, _ []byte, code TextCode) any {
				if c := installText(src, hallCreditsSource.at(), code); c.present() {
					return c.lines()
				}
				return []string(nil)
			},
			func(src entrySource, _ []byte, code TextCode) any {
				if raw, err := src.ReadFile("main/text/credits.txt"); err == nil {
					return strings.Split(strings.ReplaceAll(strings.TrimSpace(vfs.DecodeText(code.Bytes(raw))), "\r\n", "\n"), "\n")
				}
				return []string(nil)
			}},
		{"npc name lines", "main/text/npcnames.txt",
			func(src entrySource, _ []byte, code TextCode) any {
				return installText(src, npcNameLinesSource.at(), code).lines()
			},
			func(src entrySource, _ []byte, code TextCode) any {
				payload, err := src.ReadFile("main/text/npcnames.txt")
				if err != nil {
					return []string(nil)
				}
				return strings.Split(strings.ReplaceAll(string(code.Bytes(payload)), "\r\n", "\n"), "\n")
			}},
		{"cheat lines", MainTextPath,
			func(src entrySource, _ []byte, _ TextCode) any {
				return installText(src, cheatLinesSource.at(), TextCode{}).lines()
			},
			func(src entrySource, _ []byte, _ TextCode) any {
				raw, err := src.ReadFile(MainTextPath)
				if err != nil {
					return []string(nil)
				}
				return strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
			}},
		{"item name text", ItemNameTextAddress,
			func(src entrySource, _ []byte, code TextCode) any {
				txt := installText(src, itemNameTextSource.at(), code)
				return []any{txt.present(), txt.bytes()}
			},
			func(src entrySource, _ []byte, code TextCode) any {
				txt, err := src.ReadFile(ItemNameTextAddress)
				if err != nil {
					return []any{false, []byte(nil)}
				}
				return []any{true, code.Bytes(txt)}
			}},
		{"hero picture list", HeroPictureAddress,
			func(src entrySource, _ []byte, _ TextCode) any {
				list, ok := ReadBodyList(src)
				return []any{list, ok}
			},
			func(src entrySource, _ []byte, _ TextCode) any {
				b, err := src.ReadFile(HeroPictureAddress)
				if err != nil {
					return []any{data.BodyList{}, false}
				}
				return []any{data.ParseBodyList(b), true}
			}},
	}
}

// baseMessageUTF8 is the editor's page decode as it stood before its row.
func baseMessageUTF8(b []byte, selector int) string {
	if utf8.Valid(b) {
		return string(b)
	}
	if decoded, ok := DecodeInstallTextFile(b, selector); ok {
		return string(decoded)
	}
	return string(b)
}

// TestLineTextRows reads every source row over every fixture and text code,
// and finds the answer its reader gave before the row.
func TestLineTextRows(t *testing.T) {
	for _, row := range lineTextRows() {
		t.Run(row.name, func(t *testing.T) {
			for name, content := range lineTextFixtures() {
				src := missionSource{}
				if content != nil && row.address != "" {
					src[row.address] = content
				}
				for _, code := range lineTextCodes {
					got, want := row.got(src, content, code), row.want(src, content, code)
					if !reflect.DeepEqual(got, want) {
						t.Errorf("%s under %+v: got %#v, want %#v", name, code, clip(got), clip(want))
					}
				}
			}
		})
	}
}

// TestLineTextAbsentSource is every row with no archive and with a number
// that names no file: no text, and an error the presence checks report.
func TestLineTextAbsentSource(t *testing.T) {
	if got := installText(nil, townTextSource.at(), TextCode{}); got.present() || got.err() == nil {
		t.Fatalf("no archive: present=%v err=%v", got.present(), got.err())
	}
	if got := installText(missionSource{}, missionTextSource.at(0), TextCode{}); got.present() || got.err() == nil {
		t.Fatalf("mission 0: present=%v err=%v", got.present(), got.err())
	}
	if p, ok := EventTextPath(10, 1); !ok || p != "main/text/battle/m10/event01.txt" {
		t.Fatalf("EventTextPath = %q %v", p, ok)
	}
	if title, brief, ok := WorldMapTextPaths(0); ok || title != "" || brief != "" {
		t.Fatalf("WorldMapTextPaths(0) = %q %q %v", title, brief, ok)
	}
}

// TestLineTextFormIsTheRows: a row read in a form it does not state panics.
func TestLineTextFormIsTheRows(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("a whole-text row answered a section")
		}
	}()
	installText(missionSource{}, tipTextSource.named("x"), TextCode{}).section("key")
}

func clip(v any) string {
	s := fmt.Sprintf("%#v", v)
	if len(s) > 300 {
		return s[:300] + "..."
	}
	return s
}
