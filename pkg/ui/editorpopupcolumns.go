package ui

// popupRowPosition is row index's column and line. A row placed on the card
// takes no line: the lines count the rows before it that are not placed.
func popupRowPosition(s popupSpec, index int) (int, int) {
	if s.Columns <= 1 {
		line := index
		for _, row := range s.Rows[:index] {
			if row.placed() {
				line--
			}
		}
		return 0, line
	}
	column, line := max(0, min(s.Rows[index].Column, s.Columns-1)), 0
	for _, row := range s.Rows[:index] {
		if max(0, min(row.Column, s.Columns-1)) == column && !row.placed() {
			line++
		}
	}
	return column, line
}

// popupRowCount is the lines of the longest column; rows placed on the card
// take none.
func popupRowCount(s popupSpec) int {
	if s.Columns <= 1 {
		n := len(s.Rows)
		for _, row := range s.Rows {
			if row.placed() {
				n--
			}
		}
		return n
	}
	counts := make([]int, s.Columns)
	for _, row := range s.Rows {
		if !row.placed() {
			counts[max(0, min(row.Column, s.Columns-1))]++
		}
	}
	total := 0
	for _, count := range counts {
		total = max(total, count)
	}
	return total
}

// popupShownRows are the rows a popup shows shown lines of from its tops,
// in row order; a row placed on the card always shows.
func popupShownRows(s popupSpec, shown int) []int {
	var out []int
	line := 0
	for i, row := range s.Rows {
		if row.placed() {
			out = append(out, i)
			continue
		}
		column := 0
		if s.Columns > 1 {
			column, line = popupRowPosition(s, i)
		}
		top := popupColumnTop(s, column)
		if line >= top && line < top+shown {
			out = append(out, i)
		}
		if s.Columns <= 1 {
			line++
		}
	}
	return out
}

func popupColumnTop(s popupSpec, column int) int {
	if column < len(s.ColumnTops) {
		return s.ColumnTops[column]
	}
	return s.Top
}
