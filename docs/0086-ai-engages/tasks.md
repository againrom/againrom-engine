# Tasks — a group decides whom to fight

Kinds: **impl** — code and its tests, one commit each.

## T1 the relation, and the state that carries it

**Kind:** impl. **Covers:** FR-1, FR-2, FR-3, FR-4, FR-5, AC-1, AC-2, AC-14, P-2.
**Plan:** DD-1, DD-2, DD-3, DD-4.

Files: `pkg/sim/relations.go` (new), `pkg/sim/world.go`, `pkg/sim/binary.go`, plus tests.

The type, its constructor and its hostility predicate; the world field; the root constructor's new
parameter; the encoding and the decoding; the version. `TestTheCanonicalWorldsFieldSetsArePinned`'s
declaration gains the field, and every pinned byte form and digest in the package is recomputed.

**Fences:** nothing reads the relation for a decision in this task — no acquisition, no change to
`Step`. The three older constructors keep their signatures.

**Done when:** a world round-trips with an arbitrary relation, a rejected length and a wrong
version are refused, a world naming none and one naming the zero matrix are byte-identical, and
two worlds differing in one byte differ in their digests.

## T2 the decision

**Kind:** impl. **Covers:** FR-7, FR-8, FR-9, FR-10, FR-11, FR-12, FR-13, FR-14, FR-15, FR-16,
FR-17, FR-18, FR-19, FR-20, AC-3 … AC-13, P-1, P-3, P-4. **Plan:** DD-5, DD-6, DD-7, DD-8, DD-9,
DD-10, DD-11, DD-12, DD-13, DD-14, DD-16.

Files: `pkg/sim/engage.go` (new), `pkg/sim/step.go`, `pkg/sim/combat.go`, plus tests.

The grouping, the shared-sight population, the relation filter, the corpse park, the notice radius
and its clip, the two scorers and the preference matrix, and the engage. `orderAttack` is extracted
from `Step`'s `KindAttack` arm and called by both. The pass is wired in beside the script pass.

**Fences:** no new entity or world field; the four constants (sight, window, guard floor, notice
margin) live in this package and nothing reads a registry. No member without a candidate is
written to in any field.

**Done when:** two hostile entities with no commands fight and one dies; every boundary the ACs
name is witnessed on both sides; a decision taken twice on an unchanged world assigns the same
victims; and the pass fires on the decision phase and no other.

## T3 the roster's relation row

**Kind:** impl. **Covers:** FR-6 (the file half), AC-15. **Plan:** DD-15.

Files: `pkg/formats/alm/alm.go`, plus tests.

The type-5 record's sixteen 16-bit words at `+0x2c` reach the decoded roster entry raw, at their
own width, interpreted in no way.

**Fences:** no narrowing, no diagonal, no matrix — the store is the loader's. The record size and
every other field are unchanged.

**Done when:** a synthetic roster's words come back in order and at full width, and a map whose
type-5 record is absent still decodes with no roster.

## T4 the map authors the relation

**Kind:** impl. **Covers:** FR-6 (the store), AC-15. **Plan:** DD-15.

Files: `pkg/mapload/fromalm.go`, plus tests.

The store the engine's map-load path makes: for the 1-based slot of each roster entry, the sixteen
words' low bytes into columns 1 to 16 of that row, then the diagonal forced to 2. The result
reaches the world through the root constructor.

**Fences:** column 0 is never written; a slot outside the matrix contributes nothing; no other
loader behaviour moves.

**Done when:** a two-player synthetic map produces the two rows its records author, with both
diagonals forced, and a map with no roster produces the all-zero relation.

## T5 the relation survives the start path

**Kind:** impl. **Covers:** FR-6 (the mission's world), AC-15. **Plan:** DD-15.

Files: `pkg/mapload/start.go`, plus tests.

Both start entry points build a world and then build a second one out of its parts. The relation is
named in neither rebuild, so a started mission carries the all-zero one and nothing on any shipped
map ever acquires. Each rebuild takes it from the world already in hand.

**Fences:** the store is not run a second time — the relation comes off the loaded world, so a start
cannot come to disagree with a plain load. Nothing else about either rebuild moves.

**Done when:** the map's relation is intact at every shape that reaches the two rebuilds — party and
none, scripted and not — and a mission started the way `pkg/game` starts one fights with no command.

## Traceability

| Task | FR | AC | DD |
|---|---|---|---|
| T1 | FR-1 … FR-5 | AC-1, AC-2, AC-14 | DD-1 … DD-4 |
| T2 | FR-7 … FR-20 | AC-3 … AC-13 | DD-5 … DD-14, DD-16 |
| T3 | FR-6 | AC-15 | DD-15 |
| T4 | FR-6 | AC-15 | DD-15 |
| T5 | FR-6 | AC-15 | DD-15 |
