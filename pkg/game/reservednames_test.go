package game

import "testing"

func TestWindowsReservedNames(t *testing.T) {
	cases := []struct {
		name     string
		reserved bool
	}{
		{"Con", true},
		{"pRn", true},
		{"AuX", true},
		{"nUl", true},
		{"Con.alm", true},
		{"pRn.txt", true},
		{"AuX.map", true},
		{"nUl.dat", true},
		{"Con .alm", true},

		{"Com1", true},
		{"cOm2", true},
		{"CoM3", true},
		{"com4", true},
		{"COM5", true},
		{"Com6", true},
		{"cOm7", true},
		{"CoM8", true},
		{"Com9", true},
		{"Com1.alm", true},
		{"cOm5.txt", true},
		{"CoM9.map", true},
		{"Lpt1", true},
		{"lPt2", true},
		{"LpT3", true},
		{"lpt4", true},
		{"LPT5", true},
		{"Lpt6", true},
		{"lPt7", true},
		{"LpT8", true},
		{"Lpt9", true},
		{"Lpt1.alm", true},
		{"lPt5.txt", true},
		{"LpT9.map", true},

		{"Com\u00b9", true},
		{"cOm\u00b2", true},
		{"CoM\u00b3", true},
		{"Lpt\u00b9", true},
		{"lPt\u00b2", true},
		{"LpT\u00b3", true},
		{"com\u00b9.alm", true},
		{"Com\u00b2.txt", true},
		{"COM\u00b3.map", true},
		{"lpt\u00b9.alm", true},
		{"Lpt\u00b2.txt", true},
		{"LPT\u00b3.map", true},

		{"ConIn$", true},
		{"cOnOuT$", true},
		{"conin$", true},
		{"conout$", true},
		{"ConIn$.alm", true},
		{"cOnOuT$.txt", true},

		{"console", false},
		{"com10", false},
		{"lpt10", false},
		{"com0", true},
		{"lpt0", true},
		{"nul_map", false},
		{"a.con", false},
		{"карта", false},
	}

	checks := []struct {
		label    string
		reserved func(string) bool
	}{
		{"hostFolderName", func(name string) bool { return !hostFolderName(name) }},
		{"editorOrdinaryFilename", func(name string) bool { return !editorOrdinaryFilename(name) }},
		{"namedSaveBase", func(name string) bool { _, err := namedSaveBase(name); return err != nil }},
		{"saveFileBase", func(name string) bool { return saveFileBase(name) != name }},
	}

	for _, tc := range cases {
		for _, chk := range checks {
			if got := chk.reserved(tc.name); got != tc.reserved {
				t.Errorf("%s(%q): reserved=%v, want %v", chk.label, tc.name, got, tc.reserved)
			}
		}
	}
}
