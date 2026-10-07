# Plan — 0103

Three tasks, bottom up, each landing in one tier: **T1** the format leaf and the dump, **T2** the
simulation's representation and byte form, **T3** the two consumers — placement and the sack query.
The order is what lets T3 witness every behaviour through a real map and a real script instead of
through a hand-built world.

## The leaf

**P-1 `Loot()` is a lazy decoder over the raw body, not a field on `Map`.** `Map.Script()` already
does exactly this over the type-7 body: the section stays a retained byte slice, the grammar is a
separate file, and the decode is run per call. Copying that shape buys FR-1's per-call read and FR-4 for nothing — the
document's write-back is `Document.Write()` returning the retained copy verbatim, and `encode.go`
only rebuilds the two type-0 strings, so neither is touched. A decoded field on `Map` would put a
second representation of the same bytes in the reader and invite the two to disagree.

**P-2 The type-8 field is renamed and its comment rewritten.** It is called `Markers` today and its
doc calls it "the type8 condition/marker-region tree ... the leaf grammar is undecoded". Two of
those three clauses are now known false, and the claim they came from is retracted in exactly those
clauses. It becomes `LootSection` with a comment saying what it is. Nothing outside the package
reads it. **`TileMarkers` (type-9) is left alone** — its grammar is published, its meaning is not,
and renaming it would replace one guess with another.

**P-3 FR-6's item-code split is two free functions on a `uint16`, in the leaf.** `alm.TileIndex(cell)`
and `alm.Impassable(cell)` are already this shape — a packed word from the file and a function per
field — so `ItemClass` and `ItemIndex` sit beside them. It also has to be here: the class table is
the executable's, but the layering runs formats -> data, so a leaf cannot reach the tier that would
otherwise own it, and this story adds no tier. The functions split bits and nothing else; the
class-14 widening (bits 0..7 instead of 0..4) is a branch inside `ItemIndex`.

**P-4 FR-2's record words are carried whole and range-checked nowhere.** `Owner`, `X`, `Y`, `Gold`
and the element's leading word stay `uint32` at the file's own width, following `Unit.Owner`'s
precedent in this same reader. `CellX()`/`CellY()` are the methods that shift (FR-7); `Ground()` is the
predicate for `Owner == 0`. `ItemCode()` on an element takes the low 16 bits. **No field is added
for the code's high half or for the parts of the code the split does not name**: the raw word is
there, and a named field with no consumer is what this reader's own comments refuse.

**P-5 FR-2's two head widths are one branch at the top of the walk, not two decoders.** The head
is 16 or 20 bytes on `FormatVersion >= 989`; on the short head `Gold` stays zero. FR-5's absent
or empty section needs no branch of its own: a walk of zero records over a zero-length body
consumes it exactly. Everything after the
head is identical, so the branch is a width and an offset.

**P-6 Errors name the record.** FR-3's two failures — a record running past the payload, and bytes
left over — are wrapped errors carrying the record index and the offset, in the style
`readScriptNodes` already uses. Nothing is returned beside an error.

**P-7 FR-8's `almtool loot` is a new file beside `script.go` in `cmd/almtool`**, one `case` in the existing
dispatch and one line of usage. It prints per record and then a census, because the census is what
AC-1 reports and a human-readable dump is what makes a disagreement locatable.

## The simulation

**P-8 A sack is a list beside the entities, not an entity.** It does not tick, holds no timer, takes
no order and is in no actor list, so making it an `Entity` would hand it every entity invariant —
health, movement domain, facing, decay — and force each to be defended as inapplicable. A new
`pkg/sim/sack.go` holds FR-9's type and its rules and FR-13's copying accessor; `World` gains
one field.

**P-9 One new constructor, `NewLootWorld`, and the funnel gains a parameter.** `newWorld` is the one
body all four public constructors reach, and that is the property FR-12 keeps: it gains a `sacks`
argument, the four existing constructors pass `nil`, and `NewLootWorld` takes what `NewRelatedWorld`
takes plus the sacks. Fifty-eight call sites of the existing constructors are untouched. An options
struct or a post-construction setter is refused for the reason the constructor's own comment already
gives: either lets a world exist in a state its constructor did not choose.

**P-10 Normalisation lives in one function the constructor calls, and the decoder does not.** The
merge (FR-10) and the sort (FR-11) are a *constructor* act: they turn a caller's list into the one
legal representation. The decoder must instead **refuse** an unsorted or duplicated list (FR-15),
because a form that accepts two spellings of one world is not injective and the digest is taken over
the form. So: `normaliseSacks(b, in) ([]Sack, error)` merges, sorts and refuses out-of-bounds;
`sackFault(b, s)` is the per-sack predicate both it and the decoder call. This is the split
`patrolFault` already models, with the difference that here the constructor also folds.

**P-11 The lookup is a binary search over the ordered list.** The list is ascending by (Y, X), which
is also the key order the original packs, so `sackAt(x, y) bool` — the one lookup FR-16 needs — is a
search on that key and needs no index, no map and no plane. A map would be a second representation to keep in step and would
iterate non-deterministically if anything ever ranged it.

**P-12 FR-14's byte form: version 22, a counted section between the groups and the script.** That is
where the group section already sits and for the same reason — it is variable-length and
self-measuring, so it must be inside the region the tail walk consumes in order, not after the fixed
2500-byte relation block that closes the form. Layout: `uint32` count, then per sack `int32 X`,
`int32 Y`, `uint32 Gold`, `uint32` item count, then that many `uint16` codes. The offset-table
comment and the version paragraph above `formatVersion` both gain an entry; in this file the comment
is the specification of the form, so it is not optional.

**P-13 The digest needs no work and every pinned digest moves.** `Hash()` is FNV-1a over `encode()`,
so the section enters the digest by construction. The version byte is at offset 0, so **every
literal digest in the tree changes whether or not a world holds a sack** — the same churn 0099 paid.
Re-pin from a run and record old -> new in the commit body.

## The consumers

**P-14 FR-16's arm goes in the FIRST switch of `runCheck`, beside the variable read.** FR-17's
low-byte truncation and FR-18's out-of-bounds zero are two lines inside it. The
second switch is reached only after `w.scriptEntity(c.Unit, c.HasUnit)` resolves, and a check-14
node names **no unit** — it carries two plain X/Y parameters and nothing else. An arm placed in the
second switch would be correct code that never runs: every node would leave at the unresolved
reference with a recorded silence. The first switch is where `ScriptCheckVariable` sits for the same
reason — it reads `Args` and no entity.

**P-15 The gap report is not edited.** `NewScript` builds its unsupported set by calling
`scriptCheckSupported`, so adding 14 to that one table moves the report and the dispatch together.
FR-19 is satisfied by *not* writing a second list.

**P-16 Placement is a pass in `FromALMWith` beside the existing ones.** `sacksFrom(m) []sim.Sack`
decodes the section, keeps `Ground()` records (FR-20) and so places nothing for a stock one
(FR-21), drops an out-of-bounds cell without failing the load (FR-22), and maps each record to
one `sim.Sack`; `FromALMWith` calls `sim.NewLootWorld` instead of `sim.NewRelatedWorld`. **A decode
error yields no sacks and no load failure** (FR-23): the loader's existing arms already prefer a
playable map to a refused one, and loot is not what makes a map playable.

**P-17 Duplicate cells are left for the constructor to fold.** `sacksFrom` emits one entry per ground
record in file order and does not merge; the merge is FR-10's rule and belongs where FR-10 put it,
so there is exactly one implementation of it and the file order that feeds it is the order the
elements join in.

## Files

| Tier | Files |
|---|---|
| leaf | new `pkg/formats/alm/loot.go`, `alm.go` (the rename and its comment), new `pkg/formats/alm/loot_test.go`, `alm_test.go` (three `Markers.Body` assertions), new `cmd/almtool/loot.go`, `cmd/almtool/main.go` |
| sim | new `pkg/sim/sack.go`, `world.go`, `binary.go`, new `pkg/sim/sackform_test.go`, `binary_test.go` |
| consumers | `pkg/sim/script.go`, `pkg/mapload/fromalm.go`, new `pkg/sim/sackcheck_test.go`, `pkg/sim/script_test.go`, `pkg/mapload/script_test.go`, new `pkg/mapload/loot_test.go` |

`internal/archtest`'s allow-map needs **no** new edge: every import this story adds already exists
(`mapload -> sim`, `mapload -> formats/alm`, `cmd/almtool -> formats/alm`).

## What goes red, and is meant to

Every literal digest and every byte-form offset or length pin, in `pkg/sim/*_test.go` and
`pkg/mapload/{fromalm,gridform}_test.go` — the version byte moves them all (P-13). The
unimplemented-check reports in `pkg/sim/script_test.go` and `pkg/mapload/script_test.go`, which name
14 today and must stop. The three `Markers.Body` assertions in `pkg/formats/alm/alm_test.go`, which
change name only. **A test red for any other reason is a stop-and-report, not an edit** — in
particular `TestTheTenthMissionIsDrivenToAWin` must not change state, since mission 10's script
carries no check-14 node.

## Testing

SC-1 is the repo gate, run in the verification stage and pasted there. Fixtures are bytes built
in test code (SC-2). The leaf's tests build a type-8 payload against the
grammar in `spec.md`, including the short head, an empty section, a record with zero elements, a
record that overruns, and a payload with a trailing byte. The simulation's tests build worlds
directly: the merge, the scramble-equals-sorted digest property, the four decoder refusals, and a
marshal/unmarshal round trip. The consumers' tests build a map fixture carrying both kinds of record
and assert the placed set, and drive a compiled check-14 node over a world with and without a sack.

The corpus evidence for AC-1 and AC-8 is `almtool loot` over both installed roots, run in the
verification stage and pasted there — never asserted in a test.
