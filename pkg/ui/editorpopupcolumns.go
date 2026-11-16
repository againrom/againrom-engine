package ui

// popupRowPosition is row index's column and line. A row placed on the card
// takes no line: the lines count the rows before it that are not placed.
func popupRowPosition(s uiWindow, index int) (int, int) {
	if s.Body.Columns <= 1 {
		line := index
		for _, row := range s.Body.Rows[:index] {
			if row.placed() {
				line--
			}
		}
		return 0, line
	}
	column, line := max(0, min(s.Body.Rows[index].Column, s.Body.Columns-1)), 0
	for _, row := range s.Body.Rows[:index] {
		if max(0, min(row.Column, s.Body.Columns-1)) == column && !row.placed() {
			line++
		}
	}
	return column, line
}

// popupRowCount is the lines of the longest column; rows placed on the card
// take none.
func popupRowCount(s uiWindow) int {
	if s.Body.Columns <= 1 {
		n := len(s.Body.Rows) + bandLines(s)
		for _, row := range s.Body.Rows {
			if row.placed() {
				n--
			}
		}
		return n
	}
	counts := make([]int, s.Body.Columns)
	for _, row := range s.Body.Rows {
		if !row.placed() {
			counts[max(0, min(row.Column, s.Body.Columns-1))]++
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
func popupShownRows(s uiWindow, shown int) []int {
	var out []int
	line := 0
	for i, row := range s.Body.Rows {
		if row.placed() {
			out = append(out, i)
			continue
		}
		column := 0
		if s.Body.Columns > 1 {
			column, line = popupRowPosition(s, i)
		}
		top := popupColumnTop(s, column)
		if line >= top && line < top+shown {
			out = append(out, i)
		}
		if s.Body.Columns <= 1 {
			line++
		}
	}
	return out
}

// bandLines are the lines a scrolling card band takes after the rows; 0
// while the band stands fixed or there is none.
func bandLines(s uiWindow) int {
	if s.Card.Subject == nil || !s.Card.Below || s.Body.Columns > 1 {
		return 0
	}
	return max(0, s.Card.BandLines)
}

func popupColumnTop(s uiWindow, column int) int {
	if column < len(s.Body.ColumnTops) {
		return s.Body.ColumnTops[column]
	}
	return s.Body.Top
}
