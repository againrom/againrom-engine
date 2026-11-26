package game

import (
	"fmt"
	"strings"
	"testing"
)

// A message page is named and read through the row's message store, so a
// second-game map's pages, the sections of its mission text, have fields and
// take edits that are read back from the section, and the first game's page
// files keep their bytes.
func TestSecondMapMessagePageTakesFieldsAndEdits(t *testing.T) {
	const file = "main/text/mission10.txt"
	host, _ := secondEventTextHost(t, eventTextFixture)
	if fields := host.MessageFields(5); len(fields) != 2 || !strings.HasPrefix(fields[0].TextValue, "The gate opens.") {
		t.Fatalf("event 5 fields %+v", fields)
	}
	if parts := host.MessageParts(5); len(parts) == 0 {
		t.Fatal("event 5 has no parts")
	}
	if fields := host.MessageFields(6); len(fields) != 0 {
		t.Fatalf("event 6 is no section yet has fields %+v", fields)
	}
	if _, err := host.SetMessage(5, "Go on.", -1); err != nil {
		t.Fatalf("SetMessage: %v", err)
	}
	if section, ok := eventSection(string(host.workspace.Pages[file]), "event5"); !ok || !strings.Contains(section, "Go on.") || strings.Contains(section, "gate") {
		t.Fatalf("the section after SetMessage %q", section)
	}
	if section, _ := eventSection(string(host.workspace.Pages[file]), "event7"); !strings.Contains(section, "Nobody reads this.") {
		t.Fatalf("event 7 changed %q", section)
	}
	parts := host.MessageParts(5)
	parts[0].Body = "Again."
	if _, err := host.SetMessageParts(5, parts); err != nil {
		t.Fatalf("SetMessageParts: %v", err)
	}
	if section, _ := eventSection(string(host.workspace.Pages[file]), "event5"); !strings.Contains(section, "Again.") {
		t.Fatalf("the section after SetMessageParts %q", section)
	}
	if host.page != "" && host.page != "event5" {
		t.Fatalf("the open page %q is not restored", host.page)
	}
}

func TestFirstMapMessagePageKeepsItsFileBytes(t *testing.T) {
	e, _ := openGoalMap(t, emptyScriptMap(t, 40))
	s := formHost(e)
	workspacePage(t, e, "event01.txt", "<part=1>\nHello")
	if fields := s.MessageFields(1); len(fields) != 2 || fields[0].TextValue != "Hello" {
		t.Fatalf("event01 fields %+v", fields)
	}
	if _, err := s.SetMessage(1, "Bye", -1); err != nil {
		t.Fatal(err)
	}
	if got := string(s.workspace.Pages["event01.txt"]); got != "<part=1>\nBye" {
		t.Fatalf("event01.txt %q", got)
	}
}

// messageFilename is the first game's page file of message n.
func messageFilename(n uint32) string { return fmt.Sprintf(messagePageForm, n) }
