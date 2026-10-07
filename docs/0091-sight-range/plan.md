# Plan — 0091 sight range

## Approach

Four packages, in dependency order: `pkg/sim` gains the field and reads it in the two places that
held the constant; `pkg/data` gains the hero's own derivation; `pkg/mapload` fills the field on
every path a placement takes; `pkg/ui` takes the decoded column count. The `pkg/ui` half touches
none of the others and is done last so that a revert of either half is one commit.

The march is not touched. It already takes the range as an argument — the change is at the two call
sites that pass a constant into it.

## Design decisions

- **DD-1 — the field is `Entity.ScanRange uint8`, and the name is the column's (FR-1).** `pkg/data`
  already has an unrelated `UnitDef.Sight` — no column, no reader, default 0 — so naming the entity
  field `Sight` would put two names one letter apart on two different things and make
  `Sight: def.Sight` a plausible-looking loader line that is wrong. Naming it after the column both
  bands carry means the loader reads `ScanRange: uint8(def.ScanRange)` and the wrong line cannot be
  written.
- **DD-2 — one byte, not four (FR-1, FR-9).** The source field is a byte, its reader is a byte compare,
  and the group radius it feeds is a byte too. A four-byte field could hold ranges the rule that
  produces one cannot, which is the argument the group rate term and the facing already stand on. It
  also makes FR-8's truncation the conversion at the loader rather than a check somewhere later.
- **DD-3 — the narrowing lives in the loader, not in the definition tier (FR-8).** The definition
  tier carries every column whole and at full width, by policy, and changing that for one column
  would make the table's contract asymmetric. The conversion is therefore where the value crosses
  into the simulation, exactly as the relation's own low-byte narrowing is. It is a conversion and
  not a clamp: the streamer's store is a byte store after the empty-cell test, so truncation is what
  the original does with an oversized cell. No shipped row exercises it.
- **DD-4 — no refusal on decode (FR-10, FR-11, P-5).** Every byte is a range, so there is nothing to fold
  and nothing to reject; a refusal would make a state the constructor accepts a state the decoder
  will not read back. This is the facing's rule and it is taken for the same reason.
- **DD-5 — the constant is deleted, not defaulted (FR-2, FR-6).** `sightRadius` goes away entirely rather than
  becoming a fallback for an entity that names no range. A fallback would be a second source for the
  same number and would hide P-3's completeness claim behind a value that looks right. An entity
  naming no range has range 0, which is a range.
- **DD-6 — the notice radius reads each member's own range (FR-3, FR-4).** The published expression is
  the maximum of `distance + that member's sight`, which is what the existing code computes with the
  constant substituted for the sight. So the change is the substitution and the byte narrowing stays
  where it is — inside the maximum, per member, before the comparison.
- **DD-7 — the hero's derivation goes beside `Hero.Speed` and not on `Combat` (FR-7).** `Combat` is
  the eight numbers a blow reads; a sight range is not one of them, and folding it in would hand
  every consumer of a blow's numbers a term it has no use for. `Speed` already sits outside `Combat`
  for that reason and this is the same shape: one method, capped statistics, integer arithmetic.
- **DD-8 — the party's range is derived even though nothing in this build reads it.** An entity of
  roster slot 0 belongs to no group and takes no engagement decision, so no rule here consults a
  hero's range. It is filled anyway because it is hashed state that will be read as a statement
  about a hero the moment anything does, and 0 or 5 there would be an authored number of exactly the
  kind this story exists to remove.
- **DD-9 — the version's digest pin is recomputed outside the tree.** The pinned bytes are a hand
  transcription and the pinned digest is derived from them by an implementation of the hash that is
  not this package's, checked first against published vectors. The new byte is added to the
  transcription by hand, and a second test strips it back off every record and requires the previous
  version's pinned digest — so the transcription and the constant each have to be right.
- **DD-10 — the view test pins the figure to its derivation, not to itself (FR-12, FR-13, AC-13).** The count is
  the decoded rectangle's width divided by the native cell size: `(640 - 160) / 32`. Both terms are
  written into the test from the decoded viewport rather than read off the constant, so a test that
  computed the expectation from `AuthoredStartColumns()` — which is what the pre-existing test does
  in one place — is replaced rather than left to agree with any value the constant takes.

## Files

- `pkg/sim/world.go` — the field, in the entity's declaration order at the tail.
- `pkg/sim/sight.go` — the group march reads each member's range; the constant's home is gone.
- `pkg/sim/engage.go` — the constant is deleted; the notice radius reads each member's range.
- `pkg/sim/binary.go` — version 18, the record's new width and its tail byte, encode and decode.
- `pkg/sim/*_test.go` — the field-set pin, the byte-form pin, the digest pin and the two version
  assertions that name 16.
- `pkg/data/hero.go` — `Hero.Sight`.
- `pkg/mapload/fromalm.go` — the spawn block carries the range; all three arms fill it (FR-5, FR-6).
- `pkg/mapload/start.go` — a party member's range from his own derivation.
- `pkg/ui/viewer.go` — the column count and what its documentation now rests on.
- `pkg/ui/startview_test.go` — the derivation the count is pinned to.
- `cmd/classdump/databin.go` — the range in the per-placement template report, so the decode is
  witnessed against a lawful install rather than against the table read a second time.

## Success criteria

- **SC-1** Every acceptance criterion has a test, and the sight-range tests fail if the field is
  read from a constant anywhere.
- **SC-2** `go test ./...` is green with no game install present.
- **SC-3** A mutation battery over the new law — the seed, the per-member union, the maximum in the
  notice radius, the narrowing, the derivation's cap and its divisor, the record offset — reports
  its survivors rather than a score.
- **SC-4** The mission drive is run against BOTH roots and its output is recorded whichever way it
  goes; nothing is tuned to make it win.
- **SC-5** The corpus census behind the spec's 4-to-12 claim is reproduced from the shipped tool on
  both roots.

## Risks

- **R-1** *The mission drive still loses.* The story is the current hypothesis for that failure and
  may be wrong; the notice clip and the relation the map authors sit on the same decision. Mitigated
  by SC-4 stating the result as a result rather than as a pass condition.
- **R-2** *A concurrent lane also widens the entity record.* Version 17 is held elsewhere and the
  same file's tail is the natural place for both. Mitigated by taking 18 unconditionally and by not
  rebasing mid-story; the conflict is the orchestrator's to resolve at the merge.
- **R-3** *The digest pin is recomputed from the encoder by accident.* That would make the byte-form
  test agree with itself and witness nothing. Mitigated by DD-9's out-of-tree derivation and by the
  strip-back test, which cannot pass unless the transcription is right.
