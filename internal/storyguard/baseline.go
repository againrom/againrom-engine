package storyguard

// Committed is the one committed record every ratcheted count in this
// package is checked against. Regenerate it with:
//
//	go run ./internal/storyguard/cmd/measure
//
// then read every number before pasting it: measure regenerates all of them
// at once, so a routine paste can carry a rise nobody looked at. A count may
// only fall, with one stated exception: CommentBytes may rise for a doc
// comment on genuinely new code. Every other rise is a regression Check
// reports. Any fall requires this file to be updated in the commit that
// caused it.
//
// The six zero comment forms are absolutes: the only way their count can move
// is a regression. A comment-only change proves it moved no code with
// scripts/check-comment-only-change.sh.
//
// This file holds the numbers only. The reason a number moved is in the
// history of this file.
var Committed = Baseline{
	TestIdentCount: 4454,
	TestFileCount:  264,
	CommentForms: map[string]int{
		"specclause":      0,
		"storymention":    0,
		"calendardate":    0,
		"rom1address":     0,
		"funaddr":         0,
		"expmention":      0,
		"acclause":        1855,
		"scclause":        500,
		"barestorynumber": 1671,
	},
	Counts: map[string]int{
		"dirnames":               0,
		"nongofilenames":         45,
		"structtags.nontest":     0,
		"structtags.test":        0,
		"stringliterals.nontest": 1,
		"stringliterals.test":    28,
		"longcommentgroups":      1631,
	},
	CommentBytes: 10789440,
}
