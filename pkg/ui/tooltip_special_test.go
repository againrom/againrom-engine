package ui

import (
	"image"
	"reflect"
	"testing"
)

func TestTooltipPickerChoosesByColumn(t *testing.T) {
	rows := []PickerRow{
		{Text: "10.alm", Choosable: true},
		{Text: "Cross.ALM - Crossroads of Mystery", Choosable: true,
			Width: 256, Height: 256, Description: "By Andrey Makhovikov.#For any level."},
	}
	a := newTestApp(t, rows, okLoader(t))
	a.flow.screen = ScreenPicker
	a.flow.words.MapListSize = "Size of map"
	a.flow.words.MapListColumns = [2]string{"Players", "Level"}
	a.SetTooltipFont(panelFont())
	second := pickerTop + pickerLine + 1

	a.tooltipPoint = image.Pt(pickerLeft+1, pickerTop+1)
	if got := a.tooltipTarget(); got.key() != "" {
		t.Fatalf("row with no description got a hint: %+v", got)
	}
	a.tooltipPoint = image.Pt(pickerLeft+1, second)
	if got := a.tooltipTarget(); !reflect.DeepEqual(got.lines, []string{"By Andrey Makhovikov.", "For any level."}) {
		t.Fatalf("description hint = %v", got.lines)
	}
	for _, tc := range []struct {
		x    int
		want string
	}{
		{pickerLeft + pickerColumnSize - 1, ""},
		{pickerLeft + pickerColumnSize, "Size of map"},
		{pickerLeft + pickerColumnWord - 1, "Size of map"},
		{pickerLeft + pickerColumnWord, "Players"},
		{pickerLeft + pickerColumnLast - 1, "Players"},
		{pickerLeft + pickerColumnLast, "Level"},
		{pickerLeft + 600, "Level"},
	} {
		for _, y := range []int{pickerTop + 1, second, pickerTop + 5*pickerLine} {
			a.tooltipPoint = image.Pt(tc.x, y)
			got := a.tooltipTarget()
			if tc.want == "" {
				continue
			}
			if !reflect.DeepEqual(got.lines, []string{tc.want}) {
				t.Errorf("x %d y %d: hint %v, want %q whatever the row", tc.x, y, got.lines, tc.want)
			}
		}
	}
	a.tooltipPoint = image.Pt(pickerLeft+pickerColumnWord, pickerTop-1)
	if got := a.tooltipTarget(); got.key() != "" {
		t.Fatalf("above the list got a hint: %+v", got)
	}
}

func TestTooltipPickerColumnWithoutCaptionStatesNothing(t *testing.T) {
	rows := []PickerRow{{Text: "a.alm", Choosable: true, Width: 8, Height: 8}}
	a := newTestApp(t, rows, okLoader(t))
	a.flow.screen = ScreenPicker
	a.SetTooltipFont(panelFont())
	a.tooltipPoint = image.Pt(pickerLeft+pickerColumnWord+1, pickerTop+1)
	if got := a.tooltipTarget(); got.key() != "" {
		t.Fatalf("empty caption produced %+v", got)
	}
}

func TestPickerRowColumnTexts(t *testing.T) {
	r := PickerRow{Width: 64, Height: 48, Word70: 3, Word74: 12}
	if !r.HasColumns() || r.ColumnTexts() != [3]string{"48x32", "3", "12"} {
		t.Fatalf("columns = %v", r.ColumnTexts())
	}
	if (PickerRow{}).HasColumns() {
		t.Fatal("a row with no decoded size shows columns")
	}
}

func detailedChargen(t *testing.T) *Chargen {
	t.Helper()
	stats := make([]ChargenStat, 4)
	for i := range stats {
		stats[i] = ChargenStat{Floor: 0, Ceiling: 50, Start: 10 + i}
	}
	c := NewChargen(ChargenSetup{
		Name:      "Danath",
		PreCreate: &ChargenPreCreate{Art: &ChargenPresentation{Layout: testGenerator(), Font: messageFont()}},
		Choices:   []ChargenChoice{{Options: []string{"male", "female"}, Parent: -1}},
		Stats:     stats, Cost: triangular(50), Budget: 5000,
	})
	c.Forward()
	if c.Stage() != DetailedStage {
		t.Fatal("fixture did not reach DetailedStage")
	}
	return c
}

func TestTooltipChargenAttributeRows(t *testing.T) {
	a := NewApp("chargen", nil, nil, nil)
	a.Layout(640, 480)
	c := detailedChargen(t)
	a.flow.chargen, a.flow.screen = c, ScreenChargen
	for i := 0; i < 4; i++ {
		a.flow.words.Hover[155+i] = []string{"Body", "Reaction", "Mind", "Spirit"}[i] + " prose"
		a.flow.words.Hover[15+i] = []string{"Body", "Reaction", "Mind", "Spirit"}[i]
	}
	for i := 0; i < 4; i++ {
		value := c.statValue[i]
		row := chargenStatValueBox[i]
		cases := []struct {
			name string
			at   image.Point
			want []string
		}{
			{"plate prose", image.Pt(20, row.Min.Y+3), []string{a.flow.words.Hover[155+i]}},
			{"label and value", row.Min.Add(image.Pt(3, 3)), []string{a.flow.words.Hover[15+i] + " = " + itoa(value)}},
		}
		raise, _ := c.StatStepText(i, true)
		lower, _ := c.StatStepText(i, false)
		cases = append(cases,
			struct {
				name string
				at   image.Point
				want []string
			}{"next point cost", chargenStatPlusBox[i].Min.Add(image.Pt(3, 3)), []string{raise}},
			struct {
				name string
				at   image.Point
				want []string
			}{"refund", chargenStatMinusBox[i].Min.Add(image.Pt(3, 3)), []string{lower}})
		for _, tc := range cases {
			if got := a.chargenTooltip(tc.at); !reflect.DeepEqual(got.lines, tc.want) {
				t.Errorf("row %d %s: hint %v, want %v", i, tc.name, got.lines, tc.want)
			}
		}
	}
}

func itoa(n int) string { return GroupSigned(int64(n))[1:] }

func TestTooltipChargenSignedTextsAreGrouped(t *testing.T) {
	c := detailedChargen(t)
	raise, _ := c.StatStepText(0, true)
	lower, _ := c.StatStepText(0, false)
	if raise[0] != '-' || lower[0] != '+' {
		t.Fatalf("signs: raise %q lower %q", raise, lower)
	}
	if got := GroupSigned(-1234); got != "-1,234" {
		t.Fatalf("GroupSigned(-1234) = %q", got)
	}
}
