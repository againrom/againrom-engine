# Tasks — 0094 the party's roster slot

Legend: **kind** is `impl` (one commit). Every entry states its `Done when:`.

## T1 — one slot constant, exported `impl`

Covers FR-2, AC-3, DD-1.
Files: `pkg/sim/engage.go`, `pkg/sim/engage_test.go`.

Scope fence: a rename and its doc comment. No behaviour changes; no other package is edited in this
commit, and `pkg/mapload` does not yet reference the symbol.

**Done when:** the constant is exported and untyped, its doc says it is the slot a start's party
takes as well as the slot whose groups stand their ground, no unexported spelling survives anywhere,
and the suite is green with no test edited beyond the identifier.

## T2 — the party carries the slot, and three comments stop asserting it does not `impl`

Covers FR-1, FR-3, FR-5, AC-1, AC-2, AC-8, AC-8a, DD-2.
Files: `pkg/mapload/start.go`, `pkg/mapload/start_test.go`, `pkg/mapload/fromalm.go` (comment only),
`pkg/sim/engage.go` (comment only).

Scope fence: no executable line of `fromalm.go` or `engage.go` changes — the edits there are the two
comments that name the party as an entity of slot 0. The group word is not written. No other field
of a party entity moves.

**Done when:** a start's party entities carry the exported constant and group word 0; a world built
from a map with no party is unchanged field for field; `TestAStartedMissionFightsWithoutBeingTold`
states that the hostile takes the nearer of two slot-1 candidates and that the party is the other,
so it can no longer pass by cost ordering alone; and any other test that turns red is classified in
the commit body as the contract arriving or as a defect, never silenced.

## T3 — the party is acquired, acquires, and loses its walk when it does `impl`

Covers FR-4, FR-6, FR-6a, AC-4, AC-5, AC-6, AC-7, AC-7a, DD-5.
Files: a new test file in `pkg/sim`.

Fixture worlds only: a relation over two slots, one entity at slot 1, one at slot `k`, built in test
code. No install, no `.alm`.

Scope fence: **no production file in `pkg/sim` changes.** If one must, the plan is wrong and this is
a revision, not an edit.

**Done when:** the asymmetric case is covered in both directions independently — a slot hostile to 1
that 1 is not hostile to, and the reverse; the out-of-reach case shows the slot-1 entity given no
order and standing still; and the single-order case is pinned beside the re-issued one and beside
the same drive at slot 0, so the disclosed cost is a measurement rather than a sentence.

## T4 — the front end is told which slot is the local participant's `impl`

Covers FR-10, AC-14, DD-3.
Files: `pkg/game/world.go`, `pkg/ui/viewer.go` (a read accessor), a test file in `pkg/game`.

Scope fence: one pushed value, the comment that explains it, and the accessor that makes the value
observable at all — `pkg/ui` gains no behaviour, and neither its arming gate nor its numeral rule
is edited.

**Done when:** the value pushed is the exported constant rather than a literal; the comment no
longer says this build cannot know the answer; and a test on a world opened WITH a party reads the
pushed value back and shows it equals the slot the party carries, so the two cannot drift into the
state that flips a numeral's drift.

## T5 — a started world round-trips with the party's slot `impl`

Covers FR-7, AC-10, DD-6.
Files: a test file in `pkg/mapload`.

Scope fence: `pkg/sim/binary.go` is not edited. If the round trip needs a change there, the premise
that the slot is already encoded is false and this is a revision.

**Done when:** a world built by a start round-trips byte-identically and its party entities come back
carrying the slot — asserted on the decoded entity, never on the version byte.

## T6 — the roster readout `impl`

Covers FR-8, FR-9, AC-12, AC-13, DD-4.
Files: `cmd/almtool/roster.go` (new), `cmd/almtool/main.go` (one case, one usage line),
`cmd/almtool/roster_test.go` (new).

Scope fence: `main.go` gains nothing but the dispatch case and the usage string. No other verb's
output changes. Nothing is read from an install in the test, and no roster name is written as
literal non-ASCII text.

**Done when:** the readout prints per record the 1-based slot, the name, the count of placed units
naming it, the sixteen raw words and the effective row; a synthetic document with roster and unit
records exercises the low-byte narrowing, the forced diagonal and the owner count; a document with
no roster record prints the header alone and returns nil; and a stream that will not decode returns
an error with nothing on standard output.
