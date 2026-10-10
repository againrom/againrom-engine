package game

// secondGameCompletionDirectory is the movie stem a departure output selects:
// the cutpaths row the output indexes (R2-ENGINE-075). Only outputs 1..5
// are stored by a case body (R2-ENGINE-148); -1 and every other value
// select no movie (DIV-2628).
func secondGameCompletionDirectory(src entrySource, code TextCode, output int) string {
	if output < 1 || output > 5 {
		return ""
	}
	stem, _ := LoadTextTable(src, mainPrefix+"text/cutpaths.txt", code).At(output)
	return stem
}
