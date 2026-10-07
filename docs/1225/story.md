# Generated hero health maximum reads the derivation, not a constant

## Result

The owner played the prior owner kit's `game9250.sav`/`game9251.sav`
(mission 10/20, `review/owner-sav-story1224/`) in the original EN `rom.exe`
and reported the hero's maximum health as a constant 100 in this engine, where
the original showed 137 for the identical fixture (Danath: body 43 reaction 26
mind 15 spirit 15 speed 17, skills `[0 10 0 0 0 0]`, skill XP
`[0 1593 0 0 0 0]`, definition row 26, fighter). A generated hero's health
maximum -- and current health at creation, which equals it -- now comes from
`HERO-HP-072`'s own derivation instead of the provisional constant. For this
exact fixture the derivation this tree already carries reads **145**.

**Correction (the story's sole review, `pipeline/reviews/story1225-review-
a63b74a.md`).** The first cut of this story reported an unexplained
eight-point gap against the owner's own witnessed 137 and left it open as
DIV-1390. The review found the actual cause: this engine's SAV writer stored
a freshly minted hero's aggregate experience (`U130`, `actor+0x130`) as 0
instead of the live six-slot skill-experience sum HERO-XP-077 names, for
every engine-minted Human. The original's own health/mana derive reads that
aggregate (`HERO-HP-005`, `HERO-GENERAL-091`), so it recomputed 137 -- the
formula's own output over the wrong operand -- not a second, lower formula
result. `Hero.Recompute` was correct throughout. `currentActorSource`
(`pkg/game/currentworldbuild.go`) now writes the aggregate from the live
`SkillXP` slots; DIV-1390 is closed (`docs/DIVERGENCES-CLOSED.md`); the owner
kit is regenerated and now predicts 145 in the original, not 137.

Base: engine main `1d18eae`. Knowledge pin unchanged (k75).

## Mechanism

`chargenProfile` (`pkg/game/hero.go`) is the one function `MissionParty`,
`MissionPartyAs` and `ChargenParty` all use to resolve a party member's
`data.Profile`. Its "no base row resolves" fallback -- the arm
`MissionParty(nil, nil, nil)` hits, the Table-less recipe both the owner's
witnessed kits and every generated-mission-SAV producer test use -- returned
`data.Profile{Fighter: !class}` (DIV-1389, the prior story's own fix for the
class bit), leaving every other field at Go's zero value, including
`HealthColumn`.

`mapload.PartySpawn`/`PartySpawnWithTable` (`pkg/mapload/start.go`) read
`Profile.HealthColumn` to decide whether to trust `Hero.Recompute`'s own
derived maximum or fall back to a provisional constant
(`mapload.SpawnHP`, 100) -- a gate written for a party member whose base row
genuinely carries no health column at all, not for one whose Profile was
never populated because no row resolved in the first place. `HERO-HP-072`'s
own derive reads only Body and the class bit, neither of which needs a
shipped row: a member built through this fallback already carries everything
the graph needs, so treating it as column-less was overcautious for the
health axis specifically.

## Fix

`chargenProfile`'s fallback now returns
`data.Profile{Fighter: !class, HealthColumn: true}`. `ManaColumn` stays
false: mana is out of this story's scope, and `HERO-HP-072` itself notes the
doubling bit is read from whether the ManaMax column is positive, a question
this fallback still cannot answer. `Hero.Recompute` itself is untouched --
it already implements `HERO-HP-072`'s text end to end (confirmed by reading
and by an existing test's own documented arithmetic, `TestHumanSaleChanged
QuotientDerivesWithoutTraining`) -- so every party member that already
resolved a real base row, every companion, every placed map unit, and every
resumed `OriginalHuman` actor is unaffected. For a lawful install,
`ChargenParty` always resolves a real row (`ok` is always true, all four
class/gender pairs resolving rows 26-29 on both EN and RU), so the ordinary
player-facing chargen path was never affected in play; the fallback is
reachable through the Table-less recipe on any install, and, separately,
through `ChargenParty` itself given a `Humans` collection that lacks the PC
rows (`TestAC7BaseRowSelectionReachesThePartysProfile`), which no lawful
install supplies.

## The 145-versus-137 gap was a SAV-writer bug, not a formula gap (DIV-1390, closed)

`Hero.Recompute`'s health-maximum term is: `body=43`, `mult=2` (clear class
bit), `experience = Σ SkillXPFor(level)` over six skill slots (1593 here, all
from Blade at level 10), `h = (int32)(body*mult)` = 86, then
`h = ftol(h + log₁.₁(experience/5000+1)*mult)` = 91, then
`healthMax = ftol(h * (1.1^body/100 + 1))` = **145**. This was always the
correct arithmetic for `HERO-HP-072`'s own text.

The first cut of this story treated the owner's witnessed 137 as possible
evidence against the experience term, since recomputing with
`experience = 0` reproduces 137 exactly. The review traced why the original
saw a zero operand: `currentActorSource` (`pkg/game/currentworldbuild.go`)
built a freshly minted Human or Humanoid's SAV record with aggregate
experience (`U130`, `actor+0x130`) hardcoded to 0, beside a nonzero six-slot
`SkillXP` sum -- for every engine-minted hero, not only this fixture.
`HERO-HP-005`, `HERO-XP-077` and `HERO-GENERAL-091` establish that `U130` is
exactly the running six-slot experience aggregate the original's own health
and mana formulas read; `SAV-HEROXP-063` establishes it is stored and
restored raw, never derived on load. The zero was a pre-existing SAV-writer
defect, not a fact about which XP-shaped field the derive reads.
`currentActorSource` now builds that aggregate from the live `SkillXP`
slots, the same sum this engine's own campaign score already uses
(`DIV-1267`), whenever a Human or Humanoid record has no source basis yet.
`TestSAVU130CorpusResaveKeepsTheLoadedAggregate` confirms this does not
touch a resumed/loaded original's own aggregate: across the full discovered
save corpus, 205 characters in 105 files, every resaved `U130` equals what
that file loaded. With the aggregate corrected, the owner kit now carries
1593, the original's own mission-10 saves' own total, so the original's
derive should read 145, matching `Hero.Recompute` exactly. DIV-1390 is
closed (`docs/DIVERGENCES-CLOSED.md`).

## A narrower, still-open finding: a sourced actor's health maximum does not live-recompute on equip (DIV-1391)

The owner's resave of the prior kit's own `game9250.sav` -- loaded once in
the original EN client, then played and saved back out -- reads health
maximum 137 where the freshly-exported file read 100 before this story's
health-column fix. The first cut of this story read that as the original
re-deriving on an ordinary LOAD/resume, with no claim tracing the load or
resume routine. The review found both halves of that reading wrong: the
resave is not a load-then-immediate-save -- it records play, including a
worn Weapon the kit's hero did not carry -- and `SAV-HUMLOAD-445`/
`SAV-HUMRESUME-460` do trace the original's own load and resume bodies,
showing neither one calls the derive. `HERO-EQUIP-017` instead establishes
that `Weapon::Equip` calls the derive after its own modifier stores, for any
actor. This engine's own equivalent, `Rearm`/`applyRearmLoadout`
(`pkg/game/rearm.go`), calls `Hero.Recompute` only for an actor with no
source basis; a sourced actor (any resumed or loaded original) returns its
stored combat block unchanged on every later equipment change, including a
weapon attach. With DIV-1390's own correction, a sourced actor's stored
`HealthMax` already carries what `Hero.Recompute` would give over its own
correct aggregate, so the STORED value does not diverge; only a LIVE
recompute after a later equip during play, which the original's own weapon
path performs and this tree's sourced-actor path does not, remains open.
This is recorded as DIV-1391, narrowed and left open; it is a
persistence-architecture question, not a fix this story implements.

## Scope

Only the player-side human's own health maximum on the generated/New-Game
Table-less fallback path is touched. Unaffected, confirmed by reading:

- Placed map units (`ConstructActorBasis`/`blockFor` and related loaders) --
  a different construction path entirely, not `chargenProfile`.
- `OriginalHuman`-backed resume (`originalHumanSpawn`) -- checked first in
  `partySpawn` and returns before the Table-less fallback is ever reached.
- Companions and mercenaries -- separate `Profile` construction sites, not
  `chargenProfile`'s fallback arm.
- `ChargenParty` on a lawful install (the path every real player takes) --
  always resolves a real base row there, so `ok` is always true and this
  fallback arm is never hit in play; `TestReleaseChargenPartyFighterClass
  FlagsUnaffected`-style coverage already established this for the class
  bit, and the same reasoning holds for `HealthColumn`.
  `TestAC7BaseRowSelectionReachesThePartysProfile` still drives `ChargenParty`
  itself into this arm with a `Humans` collection that lacks the PC rows, a
  condition no lawful install supplies; both statements hold for lawful
  installs only.

## Proof

- `TestMissionPartyHealthMaximumComesFromTheDerivation`
  (`pkg/game/herohealthmaximum_test.go`): fixture-only, no install needed.
  Builds `MissionParty(nil, nil, nil)`, asserts `Profile.HealthColumn` is
  now true, asserts `mapload.PartySpawn`'s returned health equals
  `Derived.HealthMax` and is not the `mapload.SpawnHP` constant, and pins the
  derivation's own number (145) against the fixture's Body/Blade skill so a
  fixture drift fails loudly rather than silently changing the pinned number.
- `TestReleaseGeneratedHeroHealthMaximumFollowsTheDerivation`
  (`pkg/game/herohealthmaximum_release_test.go`, both missions 10 and 20):
  builds the exact `MissionParty(nil, nil, nil)` recipe against the real
  installed table, opens the mission, exports the current save through the
  real `ExportCurrentSave` path, decodes it back with `sav.Open`, and asserts
  the exported current and maximum health both equal `mapload.PartySpawn`'s
  own derived maximum -- proving the fix reaches the document a player's
  client reads, not only `mapload`'s in-memory return value.
- `TestReleaseHeroHealthMaximumGroundTruthResave`
  (`pkg/game/herohealthmaximum_release_test.go`): the review's correction
  retargets this as the aggregate-0 witness. Pins the owner's resave
  (`groundCorpusFile`, `2026-09-24/exp-engine-lineage/game0000-original-
  resave9250.sav`, sha256 `a1bc3c65ba5b384dc0802449cba6d45f5a167eecd0ed3a1e
  707e34a9e0da071d`) -- Body/Reaction/Mind/Spirit, both skill arrays,
  aggregate experience (`U130`) 0, and health maximum 137 -- as what the
  formula gives over the wrong (zero) operand this engine's own SAV writer
  used to write, not a second, lower ground truth for the formula itself.
- `TestReleaseGeneratedHeroAggregateExperienceMatchesItsSkillSlots`
  (`pkg/game/herohealthmaximum_release_test.go`): the review's own
  correction test. Asserts each exported Human's `U130` equals the sum of
  its six skill-XP dwords, for the Table-less recipe the owner kit uses and
  the ordinary production fighter and mage recipes a real installed Table
  drives. Fails on every recipe before `currentActorSource`'s fix (`U130` 0
  against a nonzero sum).
- `TestSAVU130CorpusResaveKeepsTheLoadedAggregate`
  (`pkg/game/savu130corpus_test.go`, `sessioncorpusaudit`): proves the fix
  does not touch a resumed original's own aggregate. Across the full
  discovered save corpus, 205 name-matched characters in 105 files, every
  resaved `U130` equals what that file loaded; zero mismatches.
- A regenerated owner kit, `review/owner-sav-story1225/game9252.sav`
  (mission 10) and `game9253.sav` (mission 20), written through the same
  `MissionParty(nil, nil, nil)`/`ExportCurrentSave` path the prior kit used,
  after the `U130` correction. Both decode to hero health `145/145` and
  aggregate experience (`total XP`) `1593` (the first cut of this kit read
  `145/145` with aggregate `0`, `cmd/savtool party`; before this story,
  `100/100`). See `verification.md` for hashes, sizes and the prediction the
  owner's own run resolves.

`go test -trimpath -count=1 ./...`, `gofmt`, `internal/archtest`,
`internal/gatedtests` and `internal/storyguard` (`storymention` unchanged at
0; `CommentBytes` raised to 7793026 for this story's own new code and
comments, with the required paragraph in `internal/storyguard/baseline.go`)
all pass. `scripts/check-no-game-assets.sh` is clean.
`pipeline/check-release-tests.sh` and `scripts/check-milestone2-acceptance.sh`
results are in `verification.md`.

missionrun UNSUPPORTED census (`cmd/missionrun -trace -ticks 1 | grep -c
UNSUPPORTED`): mission 10, 0; mission 20, 0 -- unchanged from engine main.
This story is a character-generation and stat-derivation fix, not
script/trigger routing.

## Open debt

- DIV-1390 (`docs/DIVERGENCES-CLOSED.md`, CLOSED): the 145-versus-137 gap was
  the SAV writer's own aggregate-experience defect, not a formula gap; fixed
  in `currentActorSource` and proven across the full save corpus.
- DIV-1391 (`docs/divergences/persistence-current-sav.md`, OPEN, narrowed):
  a sourced actor's health maximum does not live-recompute on a later
  equipment change the way the original's own `Weapon::Equip` does
  (`HERO-EQUIP-017`); the stored value no longer diverges once DIV-1390's own
  fix is in, only a later live recompute during play would.
- Whether Armor/Shield equip or item removal also call the original's own
  derive for a sourced actor is untraced; `HERO-EQUIP-017` only establishes
  it for `Weapon::Equip` (DIV-1391).
- The owner kit's own prediction is now 145 in the original, pending the
  owner's own run of the regenerated `game9252.sav`/`game9253.sav`.
- `FrontEnd.RestoreOriginal` refuses every kit-path export with a
  roster/source count mismatch (the review's non-returning note 6):
  `game9252.sav`, and a fresh fighter or mage export with
  `NativeMissionTerrain` set, all reproduce this on engine main independent
  of this story's own change. Exports through the ordinary player-save path
  reload without it. This engine therefore cannot check an owner kit itself
  by loading it back; the owner's own play of the kit remains the only
  witness. Out of this story's scope; named here as open debt.
