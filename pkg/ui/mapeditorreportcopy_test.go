package ui

import (
	"slices"
	"testing"
)

// reportCopied sends the copy key with a fresh memory clipboard and answers
// what the clipboard then holds.
func reportCopied(t *testing.T, e *MapEditor) string {
	t.Helper()
	UseMemoryTextClipboard()
	if err := textClipboard.write("untouched"); err != nil {
		t.Fatal(err)
	}
	if err := e.Key("copy"); err != nil {
		t.Fatal(err)
	}
	got, err := ReadTextClipboard()
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestMCPConnectionPanelCopiesTheSelectedRowValue(t *testing.T) {
	const url, token, file = "http://127.0.0.1:41999/mcp", "0123456789abcdef0123456789abcdef", `C:\profile\mcp-token.txt`
	for _, withMap := range []bool{false, true} {
		var e *MapEditor
		if withMap {
			e = newMapEditorFixture(t)
		} else {
			e = NewMapEditor(panelFont(), nil, nil)
		}
		e.SetMCPConnection("MCP "+url,
			MCPDetail{Value: url},
			MCPDetail{Label: "Bearer token: ", Value: token},
			MCPDetail{Label: "Token file: ", Value: file})
		rows, open := e.ReportOpen()
		if !open || len(rows) != 3 || rows[1].Text != "Bearer token: "+token {
			t.Fatalf("withMap=%v: the connection panel shows %v", withMap, rows)
		}
		for i, want := range []string{url, token, file} {
			e.report.sel = 0
			for s := 0; s < i; s++ {
				if err := e.Key("down"); err != nil {
					t.Fatal(err)
				}
			}
			if e.ReportSelected() != i {
				t.Fatalf("withMap=%v: arrow keys left row %d selected, want %d", withMap, e.ReportSelected(), i)
			}
			if got := reportCopied(t, e); got != want {
				t.Fatalf("withMap=%v row %d: the clipboard holds %q, want %q", withMap, i, got, want)
			}
		}
		e.listChoose(1, false)
		if got := reportCopied(t, e); got != token {
			t.Fatalf("withMap=%v: after a click the clipboard holds %q, want the token", withMap, got)
		}
		spec, _ := e.listSpecBase()
		if !slices.Contains(spec.Notes, e.word("editor.report.copyhint")) {
			t.Fatalf("withMap=%v: the panel does not show the copy hint: %v", withMap, spec.Notes)
		}
	}
}

func TestReportPanelsCopyTheirSelectedRow(t *testing.T) {
	e := newMapEditorFixture(t)
	rows := []InspectionFinding{
		{Error: true, Rule: "rule-a", Kind: "Unit", Index: 3, Text: "first problem"},
		{Rule: "rule-b", Text: "second problem"},
	}
	e.openReport(rows, "Report", nil)
	if err := e.Key("down"); err != nil {
		t.Fatal(err)
	}
	if got, want := reportCopied(t, e), rows[1].Line(); got != want {
		t.Fatalf("a finding copies %q, want its line %q", got, want)
	}
	e.openReport([]InspectionFinding{{Rule: keyListRule, Text: "Ctrl+C: copy"}}, "Keys", nil)
	if got := reportCopied(t, e); got != "Ctrl+C: copy" {
		t.Fatalf("a key list row copies %q", got)
	}
	e.closeReport()
	if got := reportCopied(t, e); got != "untouched" {
		t.Fatalf("with no panel open and nothing selected the clipboard holds %q", got)
	}
}
