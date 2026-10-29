package ui

func popupRowPosition(s popupSpec, index int) (int, int) {
	if s.Columns <= 1 {
		return 0, index
	}
	column, line := max(0, min(s.Rows[index].Column, s.Columns-1)), 0
	for _, row := range s.Rows[:index] {
		if max(0, min(row.Column, s.Columns-1)) == column {
			line++
		}
	}
	return column, line
}

func popupRowCount(s popupSpec) int {
	if s.Columns <= 1 {
		return len(s.Rows)
	}
	counts := make([]int, s.Columns)
	for _, row := range s.Rows {
		counts[max(0, min(row.Column, s.Columns-1))]++
	}
	total := 0
	for _, count := range counts {
		total = max(total, count)
	}
	return total
}

func popupShownRows(s popupSpec, shown int) []int {
	var out []int
	for i := range s.Rows {
		_, line := popupRowPosition(s, i)
		if line >= s.Top && line < s.Top+shown {
			out = append(out, i)
		}
	}
	return out
}
