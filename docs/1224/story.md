# Two combat fixes to the generated mission SAV

## Result

The owner played the story1222 generated-mission SAV kit
(`review/owner-sav-story1222/game9248.sav`, mission 10 warrior;
`game9249.sav`, mission 20 archer) in the original EN `rom.exe` and found two
combat defects: the mission-10 warrior's attack order never went through,
and the mission-20 archer's first hit taken looked heavier (30+) than a bat
can deal. Both root causes are in the same `ExportCurrentSave` producer that
kit exercises. One is fixed outright (the hero's own record was caught in an
AI-only repair). One is a real, separate, confirmed defect fixed in the same
producer (the hero's own fighter/mage class flag), whose causal link to the
specific damage number stays open debt.

Base: engine main `c93f58a` (story1222 landed). Knowledge pin unchanged
from story1222 (k75).

## Symptom 1: the warrior's attack order never went through (mission 10)

**Mechanism.** DIV-1382 (story1222) added an AI-start repair — mover/order
post, `+0x08/0x09`=5,255, `+0x84/0x85`=0x80,0x80, `U50`=0x0b — to every
Human/Unit record on a fresh generated mission, sourced from `SAV-1096`
(Medium)/`AI-POST-042`/`AI-GRPGUARD-074`, whose own corpus
(game9232/9233/9234) is three AI-only mission-start saves. It applied to the
player's own hero too: `game9248.sav`'s hero record carried order
`+0x00/0x01` = `(17,66)`, the AI-start post cell, instead of staying at rest
for the player to command, with the matching mover bytes set the same way.
Against `game9237.sav` (`review/owner-sav-exp0397-r5/candidates/`, the
byte-corrected mission-10 reference that plays combat correctly) and three
real corpus mission-10 originals sharing the hero's own row
(`gameversions/saves/2026-08-02/game0006.sav`, `2026-08-12/game0011.sav`,
`game0012.sav`), the hero's own record carries zero at every one of these
offsets instead.

**Fix.** Two writers implement this repair, independently, and both now
exclude the party's own hero (`e.Owner != sim.SelfSlot`, `sim.SelfSlot`
being the player's own owner slot):

- `pkg/sim/currentmotion.go:65`, `ProjectActorMotion` — the writer whose
  output actually reaches the exported bytes, since `projectMotion`
  (`pkg/game/worldsavemotion.go`) runs after `savactorproject.go`'s own pass
  and does an unconditional whole-block `copy(mover, m.Mover[:])` from this
  function's return value, making any mover-touching write inside
  `savactorproject.go` dead code before this story.
- `pkg/game/savactorproject.go:217`, `savedActorValueRecord` — the order
  `+0x00/0x01` post write and the `U50` idle-state write, which are not
  shadowed by `projectMotion` (it never touches those offsets). The
  mover-touching lines this function used to also write here are removed as
  now-fully-dead code, since `currentmotion.go`'s own gated write is the one
  that reaches the document.

The `mover[5]` passability-mask repair (`TERR-PASS-051`) is unaffected and
stays unconditional on owner: every actor, including the hero, needs a real
block-test mask to move at all.

**Evidence.** `SAV-1096`, `AI-POST-042`, `AI-GRPGUARD-074` (`go run
./tools/claim <ID>` against the pinned snapshot). DIV-1388 records the
divergence: the hero-specific exclusion rests on direct file comparison
against `game9237.sav` and the corpus, not a promoted claim naming the hero
specifically, since SAV-1096's own corpus is entirely AI actors.

## Symptom 2: the archer's first hit looked too heavy (mission 20)

**Mechanism, confirmed part.** `MissionParty(nil, nil, nil)` — the
Table-less party the generated-mission-SAV path builds — resolves through
`chargenProfile`'s "no base row resolves" fallback (`missionHumans` returns
an empty `data.Collection` for a nil `*mapload.Table`, so `data.ChargenBase`
cannot resolve a row). That fallback (`pkg/game/hero.go`, before this story)
always returned Go's zero-value `data.Profile{}` — `Fighter: false` —
regardless of which archetype (`class bool`) the caller had actually
requested. `MissionParty` requests the fighter axis (`class=false`) for
every party it builds, warrior and archer alike, so both `game9248.sav` and
`game9249.sav` exported `U4C=0x6` (the mage/non-fighter bit) instead of
`0x0`. `HERO-HP-072` (High for condition/direction, Medium for naming)
establishes this bit gates health/mana doubling: the health arm doubles
exactly when the bit is clear (fighter), which a wrongly-set mage bit denies
the archer hero.

**Mechanism, unconfirmed part.** Whether this specific flag is the mechanism
behind the reported 30+ damage number is not established. `HERO-HP-072`'s
own scope is health/mana doubling, not damage or defence computation, and
no claim in evidence ties `U4C` to either. Ruled out as candidates: the
bats' own damage stats (`DamageBase=3`, `DamageSpread=3`, read directly off
the mission-20 world's own TypeID-74 units — far too low on their own to
explain 30+), and the hero's defence/protection byte layout (`UBE`), whose
`protection[0]=0`/uniform-rest pattern was cross-checked against a real,
correctly-functioning corpus hero
(`gameversions/saves/2026-08-02/game0000.sav`) and found structurally
identical, not a hole specific to the broken kit.

**Fix.** `chargenProfile` (`pkg/game/hero.go:257`) now takes the caller's
own requested archetype as a `class bool` parameter and returns
`data.Profile{Fighter: !class}` in the fallback arm, keeping the requested
archetype instead of silently naming every unresolvable request a mage. The
health/mana column fields stay `false` either way — R-3's own documented
"less accurate" fallback character — since no row was found to read a
column value from. `MissionPartyAs` (`pkg/game/hero.go:551`) and
`ChargenParty` (`pkg/game/chargen.go:409`) both pass their own already-known
`class` through. `MissionParty(nil, nil, nil)` is the only reachable caller
of the `!ok` arm: every production call site (`frontend.go`'s four
`MissionParty` sites, `cmd/missionrun` via `MissionPartyAs`,
`cmd/paneldump`, `cmd/presenceprobe`, `cmd/tooltipshot`) passes a real,
non-nil `*mapload.Table`, and `ChargenParty` always resolves a row from the
loaded install (`ok` is always true there), so this fix changes nothing for
the party every real player builds.

**Evidence.** `HERO-HP-072`. DIV-1389 records the divergence: no claim
addresses what a base-row lookup failure should report for archetype at
all; the fallback's own existence is the owner's R-3 rule ("a Humans
collection this tree cannot read still yields a playable, if less accurate
and less dressed, character"), and naming the requested archetype instead
of defaulting to mage is the narrower reading of that rule.

## Proof

- `TestAC7BaseRowSelectionReachesThePartysProfile`'s "none resolves,
  requested archetype's fighter bit" subtest and
  `TestMissionPartyWithNoHumansCollectionCarriesTheRequestedArchetype`
  (renamed from `...ZeroProfile`, `pkg/game/chargen_test.go`) assert
  `chargenProfile`'s fallback keeps the requested archetype's own `Fighter`
  bit, replacing the pre-existing assertion of the zero Profile the bug
  itself was.
- `TestReleaseGeneratedMissionHeroFighterClassFlags`
  (`pkg/game/generatedmissionherofighter_test.go`) opens missions 10 and 20
  through `MissionParty(nil, nil, nil)`/`App`/`OpenMission`, exports through
  the real `ExportCurrentSave` path and asserts the hero's own exported
  `U4C` is `0`, identifying the hero by cell position against the live
  `SelfSlot`-owned entity (the same join `generatedmissionsavfields_test.go`
  already uses, since `sav.DecodeDocumentData`'s own object numbering is not
  `sav.File.PartyWalk`'s `ArchiveIndex`).
- `TestReleaseChargenPartyFighterClassFlagsUnaffected` builds a party
  through `ChargenParty` (the path every real player takes) and asserts the
  same hero's `U4C` stays `0`, unaffected by this story.
- `TestReleaseGeneratedMissionSAVOriginalConstraints`
  (`pkg/game/generatedmissionsavfields_test.go`, extended by the prior
  story's own hero-exclusion assertions) continues to pass for both
  missions, asserting the hero's own mover/order/state fields at zero
  against the AI-owned population's own convention.
- A regenerated owner kit, `review/owner-sav-story1224/game9250.sav`
  (mission 10) and `game9251.sav` (mission 20), written through the same
  `MissionParty(nil, nil, nil)`/`ExportCurrentSave` path the original kit
  used, decoded and confirmed: hero `U4C=0x0` (was `0x6`), hero mover
  `+0x08/09`/`+0x82/83`/`+0x84/85` and order `+0x00/01` all zero (were
  `5,255`/post cell/`128,128`/post cell). `review/owner-sav-story1224/README.md`
  has hashes, sizes and predictions.

`go test -trimpath -count=1 ./...`, `gofmt`, `internal/archtest`,
`internal/gatedtests` and `internal/storyguard` (CommentBytes raised to
7748354 for this story's own new code and comments) all pass.
`scripts/check-no-game-assets.sh` is clean. `pipeline/check-release-tests.sh`
and `scripts/check-milestone2-acceptance.sh` results are in `verification.md`.

missionrun UNSUPPORTED census (`cmd/missionrun -trace -ticks 1 | grep -c
UNSUPPORTED`): mission 10, 0; mission 20, 0 -- unchanged from engine main.
This story is a current-SAV field-export and character-generation fix, not
script/trigger routing.

## Open debt

- DIV-1388 (`docs/divergences/persistence-current-sav.md`): the hero
  exclusion from the AI-start mover/order/state repair rests on direct file
  comparison (`game9237.sav`, three real corpus mission-10 originals), not a
  promoted claim naming a hero's own mission-start record. `SAV-1096`'s own
  corpus is entirely AI actors.
- DIV-1389 (`docs/divergences/character-generation.md`): no claim addresses
  what a base-row lookup failure should report for archetype in ROM1; this
  fix is an authored reading of the owner's own R-3 fallback rule, not a
  decoded fact.
- Symptom 2's causal mechanism stays partly open: the `U4C` fix is confirmed
  correct and real (a wrong class flag reaching the exported document), but
  whether it explains the specific 30+ damage number the owner reported is
  not established. `HERO-HP-072`'s own scope is health/mana doubling, not
  damage or defence computation. The owner kit's own README states this
  prediction as unconfirmed and asks for a yes/no/unclear report against
  `game9251.sav`.
- DIV-1381, DIV-1382 (story1222) are unaffected by this story and remain
  open on their own terms.
