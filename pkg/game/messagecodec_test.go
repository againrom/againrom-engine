package game

import (
	"strings"
	"testing"
)

func TestMessageCodecKeepsAttributesAndOtherPages(t *testing.T) {
	raw := "prefix\r\n<Part=1,npc=21,tips=7,iamfemale,custom=x>\r\nOne<control>x<part=2,npc=51>\nTwo<part=3>Three<broken"
	d := decodeMessageMarkup(raw)
	if d.encode() != raw {
		t.Fatal("round trip")
	}
	var edits []messagePageEdit
	for _, i := range d.pages() {
		tag := d.Tags[i]
		speaker := int64(-1)
		if n, ok := tagSpeaker(tag.Tag); ok {
			speaker = int64(n)
		}
		edits = append(edits, messagePageEdit{i, tag.Number, strings.TrimPrefix(strings.TrimPrefix(tag.Body, "\r\n"), "\n"), speaker})
	}
	if got, err := d.write(edits, false); err != nil || got != raw {
		t.Fatal("writer round trip", err, got)
	}
	edits[0].Speaker = 23
	if got, err := d.write(edits, false); err != nil || got != strings.Replace(raw, "npc=21", "npc=23", 1) {
		t.Fatal("speaker write changed an opaque attribute", err, got)
	}
	edits[0].Speaker = 21
	edits[1].Body = "Changed"
	got, err := d.write(edits, false)
	if err != nil || got != strings.Replace(raw, "Two", "Changed", 1) {
		t.Fatal("page 2 changed another byte", err, got)
	}
	edits = []messagePageEdit{edits[2], edits[0], edits[1]}
	got, err = d.write(edits, true)
	if err != nil || !strings.Contains(got, "<part=1>Three\r\n<broken<Part=2,npc=21,tips=7,iamfemale,custom=x>") || !strings.Contains(got, "<part=3,npc=51>") {
		t.Fatal("renumbering or opaque attribute lost", err, got)
	}
}

func TestMessageCodecRenumbersConditionalArmsTogether(t *testing.T) {
	d := decodeMessageMarkup("<part=1 iamfemale npc=21>\nHer<part=1 iammale npc=21>\nHis<part=2 tips=7>\nNext")
	got, err := d.write([]messagePageEdit{{2, 2, "Next", -1}, {0, 1, "Her", 21}, {1, 1, "His", 21}}, true)
	if err != nil || got != "<part=1 tips=7>\nNext\r\n<part=2 iamfemale npc=21>\nHer\r\n<part=2 iammale npc=21>\nHis" {
		t.Fatal(err, got)
	}
}

func TestMessageCodecRemovalKeepsOpaqueMarkup(t *testing.T) {
	d := decodeMessageMarkup("prefix<part=1>\nOne<opaque key=x>Keep<part=2>\nTwo<trailer>Tail")
	got, err := d.write([]messagePageEdit{{2, 2, "Two", -1}}, true)
	if err != nil || got != "prefix<opaque key=x>Keep<part=1>\nTwo<trailer>Tail" {
		t.Fatal("page removal discarded undecoded markup", err, got)
	}
}

func TestMessageCodecKeepsPageDelimitersWhenTextReplaced(t *testing.T) {
	raw := "<part=1 npc=21>\r\nOne\r\n<part=2 tips=7>\r\nTwo\r\n<part=3>\r\nThree"
	d := decodeMessageMarkup(raw)
	got, err := d.write([]messagePageEdit{{0, 1, "One", 21}, {1, 2, "Replacement", -1}, {2, 3, "Three", -1}}, false)
	if err != nil || got != strings.Replace(raw, "Two", "Replacement", 1) {
		t.Fatal("editing text changed a delimiter", err, got)
	}
	for i, want := range []string{"One", "Replacement", "Three"} {
		if body, ok := dialoguePart([]byte(got), i+1, EventAudience{}); !ok || body != want {
			t.Fatal("the live reader cannot show an edited page", i+1, body, ok)
		}
	}
}
