package ui

import (
	"strings"
	"testing"
)

// askingEdits is a host that asks before it saves a map with errors: the save
// writes only after AllowErrors, and then answers its note.
type askingEdits struct {
	reportModEdits
	allowed bool
	written []string
	note    string
}

func (p *askingEdits) AllowErrors()     { p.allowed = true }
func (p *askingEdits) SaveNote() string { return p.note }
func (p *askingEdits) hasErrors() bool  { n, _ := CountFindings(p.rows); return n > 0 }
func (p *askingEdits) SaveAs(path string) (string, error) {
	defer func() { p.allowed = false }()
	if p.hasErrors() && !p.allowed {
		return "", &ReportRefusal{Findings: p.rows, Ask: true}
	}
	p.written = append(p.written, path)
	p.dirty = false
	return path, nil
}
func (p *askingEdits) SaveIntoMod() (string, error) { return p.SaveAs("mods/x/map.alm") }

func TestSaveWithErrorsAsksAndEnterWritesIt(t *testing.T) {
	for _, key := range []string{"enter", "sack"} {
		edits := &askingEdits{reportModEdits: reportModEdits{reportEdits{rows: reportRows, dirty: true}}, note: "Texts not written: x"}
		e := reportEditor(t, edits)
		_ = e.Key("saveas") // a mod is named, so Save goes to the mod
		if rows, open := e.ReportOpen(); !open || len(rows) != 4 || len(edits.written) != 0 || e.report.confirm == nil {
			t.Fatalf("%s: open %v rows %d written %v", key, open, len(rows), edits.written)
		}
		if !strings.Contains(e.report.note, "3 errors") || !strings.Contains(e.report.note, "Enter or S") {
			t.Fatalf("note %q", e.report.note)
		}
		if err := e.Key(key); err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		if len(edits.written) != 1 || e.Dirty() || e.doc.Source != "mods/x/map.alm" {
			t.Fatalf("%s: written %v dirty %v source %q", key, edits.written, e.Dirty(), e.doc.Source)
		}
		if !strings.HasPrefix(e.report.note, "Saved with 3 errors") {
			t.Fatalf("report note %q", e.report.note)
		}
		if line := e.StatusLine(); !strings.Contains(line, "Saved with 3 errors, R shows the report") || !strings.Contains(line, "Texts not written: x") {
			t.Fatalf("status %q", line)
		}
		if _, open := e.ReportOpen(); !open {
			t.Fatal("the report is not available after the save")
		}
	}
}

func TestSaveWithErrorsEscapeWritesNothing(t *testing.T) {
	edits := &askingEdits{reportModEdits: reportModEdits{reportEdits{rows: reportRows, dirty: true}}}
	e := reportEditor(t, edits)
	_ = e.Key("saveas")
	_ = e.Key("escape")
	if _, open := e.ReportOpen(); open || e.report.confirm != nil {
		t.Fatalf("Escape left the question open: %v", open)
	}
	_ = e.Key("sack")
	if len(edits.written) != 0 || !e.Dirty() {
		t.Fatalf("written %v dirty %v", edits.written, e.Dirty())
	}
}

func TestSaveAsWithErrorsAsksOnceAndWarningsDoNot(t *testing.T) {
	edits := &askingEdits{reportModEdits: reportModEdits{reportEdits{rows: reportRows, dirty: true}}}
	e := reportEditor(t, edits)
	e.doc.Edits = struct {
		InspectionEdits
		InspectionChecker
		InspectionErrorSave
		InspectionSaveNote
	}{edits, edits, edits, edits}
	_ = e.Key("saveas")
	e.TextInput("out.alm")
	_ = e.Key("enter")
	if len(edits.written) != 0 || e.report.confirm == nil {
		t.Fatalf("Save As wrote %v before the answer", edits.written)
	}
	if err := e.Key("enter"); err != nil || len(edits.written) != 1 || edits.written[0] != "out.alm" || e.doc.Source != "out.alm" {
		t.Fatalf("Save As: %v written %v source %q", err, edits.written, e.doc.Source)
	}
	// Warnings only: no question.
	edits.rows, edits.dirty = reportRows[3:], true
	_ = e.Key("saveas")
	e.TextInput("warn.alm")
	if err := e.Key("enter"); err != nil || len(edits.written) != 2 || e.report.confirm != nil {
		t.Fatalf("warnings only: %v written %v", err, edits.written)
	}
}
