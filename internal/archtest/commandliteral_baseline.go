package archtest

// CommittedCommandLiterals is the one committed record of how many sim.Command
// composite literals the tests still write. Regenerate it with:
//
//	go run ./internal/archtest/cmd/commandliteral
//
// then read the number before pasting: a fall is the point of the ratchet and
// belongs in the commit that caused it, and a rise is a new test building a
// command by naming its fields instead of calling a constructor.
//
// 867 to 936 is a MEASUREMENT CORRECTION, not new debt. The walk this baseline
// used to be measured by was syntactic and resolved an elided element type
// one level deep, so it missed doubly nested elision - 69 literals in eleven
// files that were always there. The walk is now type-aware (go/types resolves
// each literal's own static type), and 936 is what it finds on the same tree
// this ratchet already covered, with no test file touched since.
//
// Production has no baseline because its rule is an absolute: no non-test file
// outside pkg/sim/command.go writes one.
var CommittedCommandLiterals = CommandLiteralBaseline{
	Tests: 912,
}
