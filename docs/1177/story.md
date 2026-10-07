# 1177 — stable actor identity after removal

A departed character's own entity id stays reserved once its body is gone, across
a native AGS save and cold load, so a later summon receives a fresh id instead of
that character's own. The mission-defeat guard keys on real party membership, not
on whichever actor currently holds a watched id: an unrelated actor reoccupying a
departed companion's id is no longer misread as that companion's own return or
death.

## Reproduced defect

`review/rc-entity-identity-current/README.md` (local, outside every repository,
not evidence itself) reproduces mission 130 on EN and RU: the authored betrayal
branch turns companion 116 hostile, it dies, its corpse fully decays with no
`SourceBinding` (the shipped shape of that placement), and `entityIDFloor`'s
removal-time bump was, until this story, the only thing standing between id 116
and a later allocation. An ordinary shop-sourced Control Spirit scroll, used on
an unrelated corpse, raises a Ghost that receives id 116 back — `pkg/sim/spell.go`
rule 25 calls `NextEntityID`, then removes the target corpse and appends the new
Ghost. `guardedCharacterLost` (`pkg/game/world.go`) then saw a self-owned actor
occupying the watched id, cleared the departed exemption, and read the Ghost's own
later death as the missing companion's — a false Failure.

## Authority

- `MISSION-DEFEAT-045` (promoted, High) is the existing authority `DIV-125` and
  `DIV-219` already cite for the primary-fall guard: one primary pointer, no
  second-character guarding established. This story's guard fix stays inside
  that same authored, research-silent area; it does not contradict the claim.
- `SAV-DEADLOAD-125/126/128/131/132` (promoted/active, High) govern the dead-actor
  stage/timer/rebind mechanics `pkg/sim/originaldead.go` already implements and
  this story does not change; they establish nothing about entity or slot
  allocation after removal.
- No promoted or provisional claim addresses ROM1's own entity/slot allocation
  policy after an actor is removed. `entityIDFloor` (`pkg/sim/entityidentity.go`)
  says so in its own doc comment: "This is native identity, not ROM1 provenance."
  This is an Open debt (below), not an inferred ROM1 fact.
- Knowledge pin: snapshot 37, `c4073aef7e22bbc849b0efdd2a04a13047e37ae5`.

## As-built behaviour

- `entityIDFloor` (`pkg/sim.World`) is bumped past every permanently removed
  entity's own id (`step.go`'s `remove`), persisted as the outermost wrap of the
  byte form (`appendEntityIDFloor`, the final step of `MarshalBinary`), and
  consulted by `NextEntityID`. `reserveObservableEntityIDs` additionally seeds it
  from every live entity, `originalDead` record and script-referenced id, so a
  legacy save upgraded through `UpgradeSaveFormChecked` reconstructs the floor
  from whatever the newly decoded world can still observe, with no fabricated
  ROM1 source binding.
- `pkg/game/entityidentity.go`'s `reserveLegacyMissionIDs`, called from
  `resume.go` only for a save whose `World[0] < 94`, additionally reserves ids
  from game-level residue (departed characters, commanded/swing/phase actors) and
  the current party/initial entities — the layer that protects a bare campaign
  companion with no `SourceBinding`, which the World form alone cannot.
- `pkg/game/world.go`'s `guardedCharacterLost` now keeps a watched id's departed
  exemption when that id is currently held by an actor outside the
  persisted-character type band (`sim.InPersistBand`, `0x21..0x40`) — a Ghost's
  `TypeID` is sourced from the world's declared `GhostTemplate`, never that band.
  A genuine returned party actor (persist-band `TypeID`) still clears the
  exemption and counts as a loss when dead, exactly as before.
- The byte-form version moved 93 to 94 for the new trailer. Every hand-built
  fixture across `pkg/sim`, `pkg/game`, `pkg/mapload`, `cmd/savemigrate` and
  `cmd/saverepair` that asserted an exact form length or version byte was
  updated; `pkg/sim/binary_test.go`'s own version-spelling test
  (`TestUnmarshalRefusesEveryVersionButTheCurrentOne`) was already renamed off
  the literal number by story 0106 (DD-11) and needed no further edit.

## Proof

Four regression tests cover the story's four requirements, each confirmed
load-bearing by reverting its production fix and observing the test fail:

- `TestDepartedCompanionIdentityReservationSurvivesColdLoadAndFreshSummon`
  (`pkg/sim/entityidentity1177_test.go`): a `SourceBinding`-less removal (the
  shipped companion's own shape) bumps `entityIDFloor` to 4; a native
  `MarshalBinary`/`UnmarshalBinary` round trip preserves it; a Control Spirit
  cast against the cold-loaded world mints id 4, never reoccupying id 3.
- `TestLegacySaveBoundedReconstructionProtectsOnlyObservableIdentity` (same
  file): a legacy form-93 fixture's `UpgradeSaveForm` protects a
  `SourceBinding`-backed removed id from World-level evidence alone, and does
  not protect a `SourceBinding`-less one — the exact bound `reserveLegacyMissionIDs`
  exists to close from the game layer's own residue.
- `TestGuardedCharacterLostDoesNotMisreadAReusedIdAsTheDepartedMembersDeath`
  (`pkg/game/entityidentity1177_test.go`): an unrelated non-persist-band actor
  reoccupying a departed companion's id does not clear the exemption or read as
  a loss; a genuine returned persist-band actor still does.
- `TestTransferredCharacterLossGuardSurvivesRemovalAndNativeEnvelope`
  (`pkg/game/traitor130_test.go`, pre-existing, fixture corrected to a
  persist-band `TypeID` this story required): the absent-id and returned-member
  cases around the same guard.

Full-repository evidence, gathered on the code landing candidate `d43b2e4`
(this documentation lands as a further commit on the same branch, changing no
Go source, test or fixture):

- `gofmt -l .`: clean.
- `go build ./...`: clean.
- `go test -trimpath -count=1 ./...`: 49/49 packages ok, 0 failures.
- `bash scripts/check-no-game-assets.sh`: clean.
- `check-div-claims.sh`: exit 0, 446 live rows unchanged (`DIV-125` amended in
  place, no row added).
- `check-milestone2-acceptance.sh` (EN and RU): 0 mismatches, 0 refused across
  every one of its instruments.
- `check-release-tests.sh` (EN and RU, one invocation): first pass caught a
  stale hardcoded byte-form-version assertion in
  `TestReleaseAreaDamageLifecycle1164/signed-poison-zero-crossing`
  (`pkg/game/area1164_release_test.go`), fixed (93 to 94, commit `d43b2e4`), and
  reconfirmed on the same commit: 282 of 282 gated tests ran on each of EN and
  RU, 0 lacked a subject.
- Mission census: `AGAINROM_ASSETS=<seat>/gameversions/en /tmp/mr -mission {10,20}
  -trace -ticks 1 | grep -c UNSUPPORTED` reports 0 and 0, unchanged from the
  standing baseline (`pipeline/LOG.md`, 2026-09-02: "0 UNSUPPORTED on missions
  10 and 20"). This story touches no simulation timing, pathing or script
  routing, so an unchanged census is the expected result, not a new measurement.

## Open debt

`DIV-1252`/`DIV-1253` were reserved for this story and are returned unused.
`DIV-125` ("campaign / mission-loss character set") already carried the guard's
own native-residue mechanism as an authored, research-silent (`UNKNOWN`)
divergence; this story's allocation-floor and persist-band refinements are
amended into that row in place rather than opened as new ones, since no new
disagreement with a promoted claim exists — only a correctness improvement to
the same already-tracked mechanism. `DIV-125`'s revisit condition now also names
original entity/slot allocation after removal as open evidence.

A save written before this story's format version (94) reconstructs a departed
id's reservation from three layers on legacy load: the World form's own
observable evidence (a `SourceBinding`-backed removal survives through
`originalDead`), `reserveLegacyMissionIDs`' reservation of the fresh mission's
whole authored initial entity set, and the same call's residue-based
reservation (departed/commanded/swing/phase ids). The authored initial-entity
layer already covers every authored placement regardless of residue —
including a bare campaign companion, which this paragraph previously named as
the uncovered case; it is not.

The id class none of the three layers reconstructs is a runtime-minted one: an
entity raised and then removed before the save, with no live record, no
`SourceBinding` (so no `originalDead` entry), no authored placement, and no
residue trace unless it happened to be commanded or mid-swing when it left.
Measured: a form-93 world whose true floor was 10 over live ids 1 and 2
reconstructs `NextEntityID` to only 3, reissuing ids 3 through 9. The player
effect is nil today: no `pkg/sim` or `pkg/game` consumer holds a reference to a
removed runtime id, and the id-keyed front-end caches that are never pruned
(`mw.prev`, `mw.figures`, `mw.art`, `mw.chars`) are not written for a summon,
so a reissued summon id inherits nothing. This class is not reproduced or
claimed fixed here.

A form-93 save where the collision this story fixes had already happened
before the save was written is also out of scope: the reservation this guard
now restores cannot evict the occupant already sitting on the watched id, so
`mapload.CarryParty`'s existing-member arm (`pkg/mapload/carry.go`) still tests
only existence and aliveness and would carry that occupant's `SkillXP`, worn
items and pack forward as the departed companion; `mapWorld.isGuarded`
(`pkg/game/world.go`) would still mark that id in the UI, and `scriptEntity`
would still resolve a compiled reference against the occupant. Unreachable
from a form-94 save, since the id cannot be reoccupied there at all; the
preserved owner `.ags` corpus holds no example, both of its saves being form
67. Left as named debt, not reproduced or claimed fixed here.
