package game

import (
	"slices"
	"strings"
	"testing"

	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

const skrakanInnPath = "main/text/inn/npc/npc25m150.txt"

// Skrakan's portal-passage inn part (`DLG-VOICE-076`): EN 12 parts, part 3 of
// 199 characters and 3 sentence marks; RU 10 parts, 127 characters, 1 mark. The
// panel shows the whole installed part on both roots.
func TestReleaseSkrakanPortalPartShowsTheWholeInstalledText(t *testing.T) {
	f := releaseFront(t)
	font := f.Font.Value()
	if font == nil {
		t.Fatal("no installed font")
	}
	parts, length, marks := 12, 199, 3
	if font.Selector == text.SelectorConverting {
		parts, length, marks = 10, 127, 1
	}
	payload, err := f.Archives.Containers.ReadFile(skrakanInnPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(dialogueBlocks(payload)); got != parts {
		t.Fatalf("%s has %d parts, want %d", skrakanInnPath, got, parts)
	}

	// The only tag naming part 3, cut without the production scan.
	src := string(payload)
	var tags int
	var installed string
	for cursor := 0; ; {
		open := strings.IndexByte(src[cursor:], '<')
		if open < 0 {
			break
		}
		open += cursor
		end := open + strings.IndexByte(src[open:], '>')
		cursor = end + 1
		tag := strings.ToLower(src[open+1 : end])
		at := strings.Index(tag, "part=")
		if at < 0 {
			continue
		}
		number := tag[at+len("part="):]
		if digits := strings.IndexFunc(number, func(r rune) bool { return r < '0' || r > '9' }); digits >= 0 {
			number = number[:digits]
		}
		if number != "3" {
			continue
		}
		tags++
		body := src[cursor:]
		body = body[strings.IndexByte(body, '\n')+1:]
		body = body[:strings.IndexByte(body, '<')]
		installed = strings.TrimSuffix(body, "\r\n")
	}
	if tags != 1 {
		t.Fatalf("%d tags name part 3, want 1", tags)
	}
	if got := len(installed); got != length {
		t.Errorf("installed part 3 has %d characters, want %d", got, length)
	}
	if got := strings.Count(installed, ".") + strings.Count(installed, "!") + strings.Count(installed, "?"); got != marks {
		t.Errorf("installed part 3 has %d sentence marks, want %d", got, marks)
	}
	if last := installed[len(installed)-1]; !strings.ContainsRune(".!?", rune(last)) {
		t.Errorf("installed part 3 ends with %q, not a sentence mark", last)
	}

	// The town lookup delivers that part and the panel wraps every word.
	d := newTownDialogue(payload)
	if !d.lookup(3, HeroAudience(f.Carried)) {
		t.Fatal("the town dialogue finds no part 3")
	}
	if d.text != installed {
		t.Fatalf("the town dialogue delivers %d characters, the install holds %d", len(d.text), len(installed))
	}
	layout := f.Words.OnLayout(ui.AuthoredDialogueLayout()).WithPortrait(d.hasPortrait)
	lines := ui.NoticeLayoutOf(layout, font, d.text)
	if got, want := strings.Fields(strings.Join(lines, " ")), strings.Fields(installed); !slices.Equal(got, want) {
		t.Errorf("the panel shows %d words on %d lines, the installed part has %d", len(got), len(lines), len(want))
	}
}
